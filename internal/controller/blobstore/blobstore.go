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
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
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
)

// Setup creates a controller for BlobStore resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(repositoryv1alpha1.BlobStoreGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(repositoryv1alpha1.BlobStoreGroupVersionKind),
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
		For(&repositoryv1alpha1.BlobStore{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the BlobStore controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isBlobStore := managedRes.(*repositoryv1alpha1.BlobStore)
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

	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nc}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
}

// Observe checks if the BlobStore resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	blobStore, isBlobStore := managedRes.(*repositoryv1alpha1.BlobStore)
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
	blobStore *repositoryv1alpha1.BlobStore,
	ext *external,
	getter func(context.Context, string) (*T, error),
	populate func(*repositoryv1alpha1.BlobStore, *T),
	checker func(*repositoryv1alpha1.BlobStore) bool,
) (managed.ExternalObservation, error) {
	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	result, err := getter(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetBlobStore)
	}

	if result == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	blobStore.SetConditions(nexusv1alpha1.Available())
	populate(blobStore, result)
	ext.populateBlobStoreStats(ctx, blobStore)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: checker(blobStore),
	}, nil
}

// Create creates a new BlobStore resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	blobStore, isBlobStore := managedRes.(*repositoryv1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalCreation{}, errors.New(errNotBlobStore)
	}

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeFile:
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.client.BlobStore().CreateFile(ctx, fileBlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	case "S3":
		s3BlobStore := generateS3BlobStore(blobStore)

		err := e.client.BlobStore().CreateS3(ctx, s3BlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	default:
		// Default to File type
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.client.BlobStore().CreateFile(ctx, fileBlobStore)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	}

	meta.SetExternalName(blobStore, blobStore.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing BlobStore resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	blobStore, isBlobStore := managedRes.(*repositoryv1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalUpdate{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeFile:
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.client.BlobStore().UpdateFile(ctx, name, fileBlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	case "S3":
		s3BlobStore := generateS3BlobStore(blobStore)

		err := e.client.BlobStore().UpdateS3(ctx, name, s3BlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	default:
		// Default to File type
		fileBlobStore := generateFileBlobStore(blobStore)

		err := e.client.BlobStore().UpdateFile(ctx, name, fileBlobStore)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing BlobStore resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	blobStore, isBlobStore := managedRes.(*repositoryv1alpha1.BlobStore)
	if !isBlobStore {
		return managed.ExternalDelete{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	err := e.client.BlobStore().Delete(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
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
func (e *external) observeFileBlobStore(ctx context.Context, blobStore *repositoryv1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.client.BlobStore().GetFile, populateFileBlobStoreObservation, isFileBlobStoreUpToDate)
}

// observeS3BlobStore handles Observe for S3-type blob stores.
func (e *external) observeS3BlobStore(ctx context.Context, blobStore *repositoryv1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.client.BlobStore().GetS3, populateS3BlobStoreObservation, isS3BlobStoreUpToDate)
}

// populateBlobStoreStats fetches stats from the list endpoint and
// populates Status.AtProvider. Errors are silently ignored — stats are
// best-effort and should not block reconciliation.
func (e *external) populateBlobStoreStats(ctx context.Context, blobStore *repositoryv1alpha1.BlobStore) {
	name := meta.GetExternalName(blobStore)
	if name == "" {
		name = blobStore.Spec.ForProvider.Name
	}

	generics, err := e.client.BlobStore().List(ctx)
	if err != nil {
		return
	}

	for _, generic := range generics {
		if generic.Name != name {
			continue
		}

		available := int64(generic.AvailableSpaceInBytes)
		total := int64(generic.TotalSizeInBytes)
		count := int64(generic.BlobCount)

		blobStore.Status.AtProvider.AvailableSpaceInBytes = &available
		blobStore.Status.AtProvider.TotalSizeInBytes = &total
		blobStore.Status.AtProvider.BlobCount = &count

		break
	}
}

// generateFileBlobStore generates a File blob store from the CR spec.
func generateFileBlobStore(blobStoreCR *repositoryv1alpha1.BlobStore) *blobstore.File {
	fileBlobStore := &blobstore.File{
		Name: blobStoreCR.Spec.ForProvider.Name,
	}

	if blobStoreCR.Spec.ForProvider.Path != nil {
		fileBlobStore.Path = *blobStoreCR.Spec.ForProvider.Path
	}

	if blobStoreCR.Spec.ForProvider.SoftQuota != nil {
		fileBlobStore.SoftQuota = &blobstore.SoftQuota{}
		if blobStoreCR.Spec.ForProvider.SoftQuota.Type != nil {
			fileBlobStore.SoftQuota.Type = *blobStoreCR.Spec.ForProvider.SoftQuota.Type
		}

		if blobStoreCR.Spec.ForProvider.SoftQuota.Limit != nil {
			fileBlobStore.SoftQuota.Limit = *blobStoreCR.Spec.ForProvider.SoftQuota.Limit
		}
	}

	return fileBlobStore
}

// generateS3BlobStore generates an S3 blob store from the CR spec.
func generateS3BlobStore(blobStoreCR *repositoryv1alpha1.BlobStore) *blobstore.S3 {
	s3BlobStore := &blobstore.S3{
		Name: blobStoreCR.Spec.ForProvider.Name,
	}

	if blobStoreCR.Spec.ForProvider.SoftQuota != nil {
		s3BlobStore.SoftQuota = &blobstore.SoftQuota{}
		if blobStoreCR.Spec.ForProvider.SoftQuota.Type != nil {
			s3BlobStore.SoftQuota.Type = *blobStoreCR.Spec.ForProvider.SoftQuota.Type
		}

		if blobStoreCR.Spec.ForProvider.SoftQuota.Limit != nil {
			s3BlobStore.SoftQuota.Limit = *blobStoreCR.Spec.ForProvider.SoftQuota.Limit
		}
	}

	if blobStoreCR.Spec.ForProvider.S3Config != nil {
		s3BlobStore.BucketConfiguration = blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{
				Name: blobStoreCR.Spec.ForProvider.S3Config.Bucket,
			},
		}
		if blobStoreCR.Spec.ForProvider.S3Config.Region != nil {
			s3BlobStore.BucketConfiguration.Bucket.Region = *blobStoreCR.Spec.ForProvider.S3Config.Region
		}

		if blobStoreCR.Spec.ForProvider.S3Config.Prefix != nil {
			s3BlobStore.BucketConfiguration.Bucket.Prefix = *blobStoreCR.Spec.ForProvider.S3Config.Prefix
		}

		if blobStoreCR.Spec.ForProvider.S3Config.ExpirationDays != nil {
			s3BlobStore.BucketConfiguration.Bucket.Expiration = *blobStoreCR.Spec.ForProvider.S3Config.ExpirationDays
		}
	}

	return s3BlobStore
}

// populateFileBlobStoreObservation copies File blob store config
// fields into the observation for IsUpToDate comparison.
func populateFileBlobStoreObservation(blobStore *repositoryv1alpha1.BlobStore, fileBlobStore *blobstore.File) {
	if fileBlobStore.Path != "" {
		path := fileBlobStore.Path
		blobStore.Status.AtProvider.Path = &path
	}

	if fileBlobStore.SoftQuota != nil {
		quotaType := fileBlobStore.SoftQuota.Type
		quotaLimit := fileBlobStore.SoftQuota.Limit
		blobStore.Status.AtProvider.SoftQuotaType = &quotaType
		blobStore.Status.AtProvider.SoftQuotaLimit = &quotaLimit
	}
}

// populateS3BlobStoreObservation copies S3 blob store config fields into
// the observation so that IsUpToDate can compare spec to observation.
func populateS3BlobStoreObservation(blobStore *repositoryv1alpha1.BlobStore, s3BlobStore *blobstore.S3) {
	if s3BlobStore.SoftQuota != nil {
		quotaType := s3BlobStore.SoftQuota.Type
		quotaLimit := s3BlobStore.SoftQuota.Limit
		blobStore.Status.AtProvider.SoftQuotaType = &quotaType
		blobStore.Status.AtProvider.SoftQuotaLimit = &quotaLimit
	}

	bucketName := s3BlobStore.BucketConfiguration.Bucket.Name
	if bucketName != "" {
		blobStore.Status.AtProvider.BucketName = &bucketName
	}

	bucketRegion := s3BlobStore.BucketConfiguration.Bucket.Region
	if bucketRegion != "" {
		blobStore.Status.AtProvider.BucketRegion = &bucketRegion
	}

	bucketPrefix := s3BlobStore.BucketConfiguration.Bucket.Prefix
	if bucketPrefix != "" {
		blobStore.Status.AtProvider.BucketPrefix = &bucketPrefix
	}
}

// isFileBlobStoreUpToDate checks if a File blob store matches observed.
func isFileBlobStoreUpToDate(blobStoreCR *repositoryv1alpha1.BlobStore) bool {
	obs := blobStoreCR.Status.AtProvider

	if blobStoreCR.Spec.ForProvider.Path != nil {
		if obs.Path == nil || *blobStoreCR.Spec.ForProvider.Path != *obs.Path {
			return false
		}
	}

	return isSoftQuotaUpToDate(blobStoreCR)
}

// isS3BlobStoreUpToDate checks if an S3 blob store matches observed.
func isS3BlobStoreUpToDate(blobStoreCR *repositoryv1alpha1.BlobStore) bool {
	return isSoftQuotaUpToDate(blobStoreCR) && isS3BucketConfigUpToDate(blobStoreCR)
}

// isSoftQuotaUpToDate checks if a blob store soft quota spec matches.
func isSoftQuotaUpToDate(blobStoreCR *repositoryv1alpha1.BlobStore) bool {
	if blobStoreCR.Spec.ForProvider.SoftQuota == nil {
		return true
	}

	obs := blobStoreCR.Status.AtProvider

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
func isS3BucketConfigUpToDate(blobStoreCR *repositoryv1alpha1.BlobStore) bool {
	if blobStoreCR.Spec.ForProvider.S3Config == nil {
		return true
	}

	obs := blobStoreCR.Status.AtProvider

	if obs.BucketName == nil || *obs.BucketName != blobStoreCR.Spec.ForProvider.S3Config.Bucket {
		return false
	}

	if blobStoreCR.Spec.ForProvider.S3Config.Region != nil {
		if obs.BucketRegion == nil || *obs.BucketRegion != *blobStoreCR.Spec.ForProvider.S3Config.Region {
			return false
		}
	}

	if blobStoreCR.Spec.ForProvider.S3Config.Prefix != nil {
		if obs.BucketPrefix == nil || *obs.BucketPrefix != *blobStoreCR.Spec.ForProvider.S3Config.Prefix {
			return false
		}
	}

	return true
}
