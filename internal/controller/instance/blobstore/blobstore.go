// Package blobstore contains the controller for BlobStore resources.
package blobstore

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotBlobStore is returned when the managed resource is not a BlobStore.
	errNotBlobStore = "managed resource is not a BlobStore custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when getting the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetBlobStore is returned when getting the blob store from Nexus fails.
	errGetBlobStore = "cannot get blob store from Nexus"
	// errCreateBlobStore is returned when creating the blob store in Nexus fails.
	errCreateBlobStore = "cannot create blob store in Nexus"
	// errUpdateBlobStore is returned when updating the blob store in Nexus fails.
	errUpdateBlobStore = "cannot update blob store in Nexus"
	// errDeleteBlobStore is returned when deleting the blob store from Nexus fails.
	errDeleteBlobStore = "cannot delete blob store from Nexus"

	// blobStoreTypeFile is the string identifier for File-type blob stores.
	blobStoreTypeFile = "File"

	// defaultS3Region is the region Nexus is asked for when the spec sets
	// none; Nexus then uses the region of the bucket.
	defaultS3Region = "DEFAULT"
)

// Setup creates a controller for BlobStore resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(instancev1alpha1.BlobStoreGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(instancev1alpha1.BlobStoreGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.BlobStore{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the BlobStore controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isBlobStore := managedRes.(*instancev1alpha1.BlobStore)
	if !isBlobStore {
		return nil, errors.New(errNotBlobStore)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	generic, file, s3Client, err := instance.NewBlobStoreClients(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		genericClient: generic,
		s3Client:      s3Client,
		fileClient:    file,
	}, nil
}

// external implements managed.ExternalClient.
type external struct {
	genericClient instance.BlobStoreClient
	s3Client      instance.BlobStoreS3Client
	fileClient    instance.BlobStoreFileClient
}

// Observe checks if the BlobStore resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	blobStore, isBlobStore := managedRes.(*instancev1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalObservation{}, errors.New(errNotBlobStore)
	}

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeFile:
		return e.observeFileBlobStore(ctx, blobStore)
	case "S3":
		return e.observeS3BlobStore(ctx, blobStore)
	default:
		return e.observeFileBlobStore(ctx, blobStore)
	}
}

// observeTypedBlobStore fetches a typed blob store, populates
// observation, and returns whether the resource is up to date.
func observeTypedBlobStore[T any](
	ctx context.Context,
	blobStore *instancev1alpha1.BlobStore,
	ext *external,
	getter func(context.Context, string) (*T, error),
	generate func(string, *T) instancev1alpha1.BlobStoreObservation,
	checker func(*instancev1alpha1.BlobStore) bool,
) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	result, err := getter(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetBlobStore)
	}

	if result == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	blobStore.SetConditions(nexusv1alpha1.Available())

	// The observation is replaced rather than updated field by field, so a
	// setting removed from the blob store out of band leaves no stale value
	// behind for the up to date check to match against.
	blobStore.Status.AtProvider = generate(name, result)
	ext.populateBlobStoreStats(ctx, blobStore)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: checker(blobStore),
	}, nil
}

// Create creates a new BlobStore resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	blobStore, isBlobStore := managedRes.(*instancev1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalCreation{}, errors.New(errNotBlobStore)
	}

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeFile:
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.fileClient.Create(ctx, fileBlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	case "S3":
		s3BlobStore := generateS3BlobStore(blobStore)

		err := e.s3Client.Create(ctx, s3BlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	default:
		// Default to File type
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.fileClient.Create(ctx, fileBlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	}

	meta.SetExternalName(blobStore, blobStore.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing BlobStore resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	blobStore, isBlobStore := managedRes.(*instancev1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalUpdate{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeFile:
		fileBlobStore := generateFileBlobStoreUpdate(blobStore)

		err := e.fileClient.Update(ctx, name, fileBlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	case "S3":
		s3BlobStore := generateS3BlobStore(blobStore)

		err := e.s3Client.Update(ctx, name, s3BlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	default:
		// Default to File type
		fileBlobStore := generateFileBlobStoreUpdate(blobStore)

		err := e.fileClient.Update(ctx, name, fileBlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing BlobStore resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	blobStore, isBlobStore := managedRes.(*instancev1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalDelete{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	err := e.genericClient.Delete(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteBlobStore)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

// observeFileBlobStore handles Observe for File-type blob stores.
func (e *external) observeFileBlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.fileClient.Get, generateFileBlobStoreObservation, isFileBlobStoreUpToDate)
}

// observeS3BlobStore handles Observe for S3-type blob stores.
func (e *external) observeS3BlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.s3Client.Get, generateS3BlobStoreObservation, isS3BlobStoreUpToDate)
}

// populateBlobStoreStats fetches the stats and the type from the list
// endpoint and adds them to Status.AtProvider: neither is reported by the
// typed endpoint the observation is otherwise built from. Errors are silently
// ignored — these fields are best-effort and should not block reconciliation.
func (e *external) populateBlobStoreStats(ctx context.Context, blobStore *instancev1alpha1.BlobStore) {
	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	generics, err := e.genericClient.List(ctx)
	if err != nil {
		return
	}

	for _, generic := range generics {
		if generic.GetName() != name {
			continue
		}

		blobStore.Status.AtProvider.Type = generic.GetType()
		blobStore.Status.AtProvider.AvailableSpaceInBytes = new(generic.GetAvailableSpaceInBytes())
		blobStore.Status.AtProvider.TotalSizeInBytes = new(generic.GetTotalSizeInBytes())
		blobStore.Status.AtProvider.BlobCount = new(generic.GetBlobCount())

		break
	}
}

// generateFileBlobStore generates a File blob store creation request from the
// CR spec.
func generateFileBlobStore(blobStoreCR *instancev1alpha1.BlobStore) v395.FileBlobStoreApiCreateRequest {
	return v395.FileBlobStoreApiCreateRequest{
		Name:      blobStoreCR.Spec.ForProvider.Name,
		Path:      ptr.Deref(blobStoreCR.Spec.ForProvider.Path, ""),
		SoftQuota: generateSoftQuota(blobStoreCR),
	}
}

// generateFileBlobStoreUpdate generates a File blob store update request from
// the CR spec.
func generateFileBlobStoreUpdate(blobStoreCR *instancev1alpha1.BlobStore) v395.FileBlobStoreApiUpdateRequest {
	create := generateFileBlobStore(blobStoreCR)

	return v395.FileBlobStoreApiUpdateRequest{
		Path:      create.Path,
		SoftQuota: create.SoftQuota,
	}
}

// generateS3BlobStore generates an S3 blob store from the CR spec.
//
// The spec's expirationDays is not submitted: Nexus no longer accepts a bucket
// expiration.
func generateS3BlobStore(blobStoreCR *instancev1alpha1.BlobStore) v395.S3BlobStoreApiModel {
	s3BlobStore := v395.S3BlobStoreApiModel{
		Name:      blobStoreCR.Spec.ForProvider.Name,
		SoftQuota: generateSoftQuota(blobStoreCR),
	}

	s3Config := blobStoreCR.Spec.ForProvider.S3Config
	if s3Config != nil {
		s3BlobStore.BucketConfiguration.Bucket = v395.S3BlobStoreApiBucket{
			Name:   s3Config.Bucket,
			Region: ptr.Deref(s3Config.Region, defaultS3Region),
			Prefix: s3Config.Prefix,
		}
	}

	return s3BlobStore
}

// generateSoftQuota generates the soft quota of a blob store, nil when the
// spec sets none.
func generateSoftQuota(blobStoreCR *instancev1alpha1.BlobStore) *v395.BlobStoreApiSoftQuota {
	spec := blobStoreCR.Spec.ForProvider.SoftQuota
	if spec == nil {
		return nil
	}

	return &v395.BlobStoreApiSoftQuota{
		Type:  ptr.Deref(spec.Type, ""),
		Limit: new(ptr.Deref(spec.Limit, 0)),
	}
}

// generateFileBlobStoreObservation renders the File blob store Nexus reported
// as the observed state of the managed resource. The File endpoint does not
// report the name, so it is the one the blob store was read under.
func generateFileBlobStoreObservation(name string, fileBlobStore *v395.FileBlobStoreApiModel) instancev1alpha1.BlobStoreObservation {
	obs := instancev1alpha1.BlobStoreObservation{Name: name}

	if fileBlobStore.Path != "" {
		obs.Path = new(fileBlobStore.Path)
	}

	setSoftQuotaObservation(&obs, fileBlobStore.SoftQuota)

	return obs
}

// generateS3BlobStoreObservation renders the S3 blob store Nexus reported as
// the observed state of the managed resource.
func generateS3BlobStoreObservation(_ string, s3BlobStore *v395.S3BlobStoreApiModel) instancev1alpha1.BlobStoreObservation {
	obs := instancev1alpha1.BlobStoreObservation{Name: s3BlobStore.Name}

	setSoftQuotaObservation(&obs, s3BlobStore.SoftQuota)

	bucket := s3BlobStore.BucketConfiguration.Bucket
	if bucket.Name != "" {
		obs.BucketName = new(bucket.Name)
	}

	if bucket.Region != "" {
		obs.BucketRegion = new(bucket.Region)
	}

	if bucket.GetPrefix() != "" {
		obs.BucketPrefix = bucket.Prefix
	}

	return obs
}

// setSoftQuotaObservation records the soft quota of a blob store, leaving the
// observation untouched when the blob store has none.
func setSoftQuotaObservation(obs *instancev1alpha1.BlobStoreObservation, softQuota *v395.BlobStoreApiSoftQuota) {
	if softQuota == nil {
		return
	}

	obs.SoftQuotaType = new(softQuota.Type)
	obs.SoftQuotaLimit = new(softQuota.GetLimit())
}

// isFileBlobStoreUpToDate checks if a File blob store matches observed.
func isFileBlobStoreUpToDate(blobStoreCR *instancev1alpha1.BlobStore) bool {
	obs := blobStoreCR.Status.AtProvider

	if blobStoreCR.Spec.ForProvider.Path != nil {
		if obs.Path == nil || *blobStoreCR.Spec.ForProvider.Path != *obs.Path {
			return false
		}
	}

	return isSoftQuotaUpToDate(blobStoreCR)
}

// isS3BlobStoreUpToDate checks if an S3 blob store matches observed.
func isS3BlobStoreUpToDate(blobStoreCR *instancev1alpha1.BlobStore) bool {
	return isSoftQuotaUpToDate(blobStoreCR) && isS3BucketConfigUpToDate(blobStoreCR)
}

// isSoftQuotaUpToDate checks if a blob store soft quota spec matches.
//
// A spec without a soft quota asks for a blob store without one: the payload
// builders omit the block entirely, which is how Nexus is told to drop it.
func isSoftQuotaUpToDate(blobStoreCR *instancev1alpha1.BlobStore) bool {
	obs := blobStoreCR.Status.AtProvider

	if blobStoreCR.Spec.ForProvider.SoftQuota == nil {
		return obs.SoftQuotaType == nil && obs.SoftQuotaLimit == nil
	}

	if obs.SoftQuotaType == nil || obs.SoftQuotaLimit == nil {
		return false
	}

	if blobStoreCR.Spec.ForProvider.SoftQuota.Type != nil &&
		*blobStoreCR.Spec.ForProvider.SoftQuota.Type != *obs.SoftQuotaType {
		return false
	}

	if blobStoreCR.Spec.ForProvider.SoftQuota.Limit != nil &&
		*blobStoreCR.Spec.ForProvider.SoftQuota.Limit != *obs.SoftQuotaLimit {
		return false
	}

	return true
}

// isS3BucketConfigUpToDate checks if S3 blob store bucket config matches.
func isS3BucketConfigUpToDate(blobStoreCR *instancev1alpha1.BlobStore) bool {
	s3Config := blobStoreCR.Spec.ForProvider.S3Config
	if s3Config == nil {
		return true
	}

	obs := blobStoreCR.Status.AtProvider

	if obs.BucketName == nil || *obs.BucketName != s3Config.Bucket {
		return false
	}

	return matchesOptionalString(s3Config.Region, obs.BucketRegion) &&
		matchesOptionalString(s3Config.Prefix, obs.BucketPrefix)
}

// matchesOptionalString reports whether an observed value satisfies a spec
// field that carries intent only when it is set.
func matchesOptionalString(desired, observed *string) bool {
	if desired == nil {
		return true
	}

	return observed != nil && *observed == *desired
}
