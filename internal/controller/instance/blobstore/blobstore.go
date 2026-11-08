// Package blobstore contains the controller for BlobStore resources.
package blobstore

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
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
	// errGetAccountKey is returned when reading the Azure account key fails.
	errGetAccountKey = "cannot get Azure account key from secret"
	// errFingerprint is returned when the Azure account key Secret cannot be
	// fingerprinted.
	errFingerprint = "cannot fingerprint Azure account key secret"

	// blobStoreTypeS3 is the string identifier for S3-type blob stores.
	blobStoreTypeS3 = "S3"
	// blobStoreTypeAzure is the string identifier for Azure-type blob stores.
	blobStoreTypeAzure = "Azure"

	// defaultS3Region is the region Nexus is asked for when the spec sets
	// none; Nexus then uses the region of the bucket.
	defaultS3Region = "DEFAULT"
)

// Setup creates a controller for BlobStore resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.BlobStoreGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.BlobStore](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *instancev1alpha1.BlobStore) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	rec := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.BlobStoreGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.BlobStoreList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.BlobStore{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates a TypedExternalClient for the BlobStore controller.
func (c *connector) Connect(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.TypedExternalClient[*instancev1alpha1.BlobStore], error) {
	err := c.usage.Track(ctx, blobStore)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, blobStore)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	clients, err := instance.NewBlobStoreClients(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		genericClient: clients.Generic,
		s3Client:      clients.S3,
		fileClient:    clients.File,
		azureClient:   clients.Azure,
		kube:          c.kube,
	}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	genericClient instance.BlobStoreClient
	s3Client      instance.BlobStoreS3Client
	fileClient    instance.BlobStoreFileClient
	azureClient   instance.BlobStoreAzureClient
	kube          client.Client
}

// Observe checks if the BlobStore resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeS3:
		return e.observeS3BlobStore(ctx, blobStore)
	case blobStoreTypeAzure:
		return e.observeAzureBlobStore(ctx, blobStore)
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
	name := helpers.ExternalNameOr(blobStore, blobStore.Spec.ForProvider.Name)

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
	// behind for the up to date check to match against. The credentials
	// fingerprint is not reported by Nexus but recorded by the provider, so
	// it is carried over.
	appliedFingerprint := blobStore.Status.AtProvider.CredentialsFingerprint
	blobStore.Status.AtProvider = generate(name, result)
	blobStore.Status.AtProvider.CredentialsFingerprint = appliedFingerprint
	ext.populateBlobStoreStats(ctx, blobStore)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: checker(blobStore),
	}, nil
}

// Create creates a new BlobStore resource.
func (e *external) Create(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalCreation, error) {
	var err error

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeS3:
		err = e.s3Client.Create(ctx, generateS3BlobStore(blobStore))
	case blobStoreTypeAzure:
		return e.createAzureBlobStore(ctx, blobStore)
	default:
		err = e.fileClient.Create(ctx, generateFileBlobStore(blobStore))
	}

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
	}

	meta.SetExternalName(blobStore, blobStore.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing BlobStore resource.
func (e *external) Update(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalUpdate, error) {
	name := helpers.ExternalNameOr(blobStore, blobStore.Spec.ForProvider.Name)

	var err error

	switch blobStore.Spec.ForProvider.Type {
	case blobStoreTypeS3:
		err = e.s3Client.Update(ctx, name, generateS3BlobStore(blobStore))
	case blobStoreTypeAzure:
		return e.updateAzureBlobStore(ctx, name, blobStore)
	default:
		err = e.fileClient.Update(ctx, name, generateFileBlobStoreUpdate(blobStore))
	}

	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing BlobStore resource.
func (e *external) Delete(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalDelete, error) {
	name := helpers.ExternalNameOr(blobStore, blobStore.Spec.ForProvider.Name)

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

// createAzureBlobStore creates an Azure blob store with the account key read
// from its Secret, and records the fingerprint of the applied key.
func (e *external) createAzureBlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalCreation, error) {
	model, fingerprint, err := e.resolveAzureBlobStore(ctx, blobStore)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = e.azureClient.Create(ctx, model)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
	}

	meta.SetExternalName(blobStore, blobStore.Spec.ForProvider.Name)

	if fingerprint != "" {
		// Non-fatal: without a recorded fingerprint the next Observe reports
		// drift and Update re-applies the account key.
		_ = nexus.PatchStatus(ctx, e.kube, blobStore, func(bs *instancev1alpha1.BlobStore) {
			bs.Status.AtProvider.CredentialsFingerprint = fingerprint
		})
	}

	return managed.ExternalCreation{}, nil
}

// updateAzureBlobStore updates an Azure blob store with the account key read
// from its Secret, and records the fingerprint of the applied key.
func (e *external) updateAzureBlobStore(ctx context.Context, name string, blobStore *instancev1alpha1.BlobStore) (managed.ExternalUpdate, error) {
	model, fingerprint, err := e.resolveAzureBlobStore(ctx, blobStore)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = e.azureClient.Update(ctx, name, model)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
	}

	blobStore.Status.AtProvider.CredentialsFingerprint = fingerprint

	return managed.ExternalUpdate{}, nil
}

// observeFileBlobStore handles Observe for File-type blob stores.
func (e *external) observeFileBlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.fileClient.Get, generateFileBlobStoreObservation, isFileBlobStoreUpToDate)
}

// observeS3BlobStore handles Observe for S3-type blob stores.
func (e *external) observeS3BlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	return observeTypedBlobStore(ctx, blobStore, e, e.s3Client.Get, generateS3BlobStoreObservation, isS3BlobStoreUpToDate)
}

// observeAzureBlobStore handles Observe for Azure-type blob stores. Nexus
// never reports the account key, so the blob store is only up to date when
// the fingerprint of the account key Secret is the one last applied.
func (e *external) observeAzureBlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (managed.ExternalObservation, error) {
	credentialsUpToDate, err := e.azureCredentialsUpToDate(ctx, blobStore)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	return observeTypedBlobStore(ctx, blobStore, e, e.azureClient.Get, generateAzureBlobStoreObservation,
		func(cr *instancev1alpha1.BlobStore) bool {
			return credentialsUpToDate(cr.Status.AtProvider.CredentialsFingerprint) && isAzureBlobStoreUpToDate(cr)
		})
}

// azureCredentialsUpToDate fingerprints the account key Secret of an Azure
// blob store and returns a check of the fingerprint last applied against it.
// A blob store being deleted is always up to date, so that a deleted Secret
// cannot block its deletion.
func (e *external) azureCredentialsUpToDate(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (func(applied string) bool, error) {
	if meta.WasDeleted(blobStore) {
		return func(string) bool { return true }, nil
	}

	fingerprint, err := azureCredentialsFingerprint(ctx, e.kube, blobStore)
	if err != nil {
		return nil, err
	}

	return func(applied string) bool { return applied == fingerprint }, nil
}

// resolveAzureBlobStore reads the account key of an Azure blob store and
// builds the blob store Nexus is sent. It returns the fingerprint of the
// account key it read.
func (e *external) resolveAzureBlobStore(ctx context.Context, blobStore *instancev1alpha1.BlobStore) (v395.AzureBlobStoreApiModel, string, error) {
	// One read of the Secret serves both the fingerprint and the account
	// key, so the recorded fingerprint describes exactly the key applied.
	secrets := nexus.MemoizeSecrets(e.kube)

	fingerprint, err := azureCredentialsFingerprint(ctx, secrets, blobStore)
	if err != nil {
		return v395.AzureBlobStoreApiModel{}, "", err
	}

	var accountKey string

	ref := accountKeySecretRef(blobStore)
	if ref != nil {
		accountKey, err = nexus.GetSecretValue(ctx, secrets, blobStore.GetNamespace(), ref)
		if err != nil {
			return v395.AzureBlobStoreApiModel{}, "", errors.Wrap(err, errGetAccountKey)
		}
	}

	return generateAzureBlobStore(blobStore, accountKey), fingerprint, nil
}

// azureCredentialsFingerprint fingerprints the account key Secret an Azure
// blob store references, reading it through kube. It is empty when the blob
// store references none.
func azureCredentialsFingerprint(ctx context.Context, kube client.Reader, blobStore *instancev1alpha1.BlobStore) (string, error) {
	fingerprint, err := nexus.CredentialsFingerprint(ctx, kube, blobStore.GetNamespace(), nil, accountKeySecretRef(blobStore))

	return fingerprint, errors.Wrap(err, errFingerprint)
}

// accountKeySecretRef returns the account key Secret reference of an Azure
// blob store, or nil when it has none.
func accountKeySecretRef(blobStore *instancev1alpha1.BlobStore) *xpv2.LocalSecretKeySelector {
	azureConfig := blobStore.Spec.ForProvider.AzureConfig
	if azureConfig == nil {
		return nil
	}

	return azureConfig.AccountKeySecretRef
}

// populateBlobStoreStats fetches the stats and the type from the list
// endpoint and adds them to Status.AtProvider: neither is reported by the
// typed endpoint the observation is otherwise built from. Errors are silently
// ignored — these fields are best-effort and should not block reconciliation.
func (e *external) populateBlobStoreStats(ctx context.Context, blobStore *instancev1alpha1.BlobStore) {
	name := helpers.ExternalNameOr(blobStore, blobStore.Spec.ForProvider.Name)

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

// generateAzureBlobStore generates an Azure blob store from the CR spec and
// the account key read from its Secret, empty when the spec references none.
func generateAzureBlobStore(blobStoreCR *instancev1alpha1.BlobStore, accountKey string) v395.AzureBlobStoreApiModel {
	azureBlobStore := v395.AzureBlobStoreApiModel{
		Name:      blobStoreCR.Spec.ForProvider.Name,
		SoftQuota: generateSoftQuota(blobStoreCR),
	}

	azureConfig := blobStoreCR.Spec.ForProvider.AzureConfig
	if azureConfig == nil {
		return azureBlobStore
	}

	azureBlobStore.BucketConfiguration = v395.AzureBlobStoreApiBucketConfiguration{
		AccountName:   azureConfig.AccountName,
		ContainerName: azureConfig.ContainerName,
		Authentication: v395.AzureBlobStoreApiAuthentication{
			AuthenticationMethod: azureConfig.AuthenticationMethod,
		},
		PreSignedUrlEnabled: azureConfig.PreSignedURLEnabled,
	}

	if accountKey != "" {
		azureBlobStore.BucketConfiguration.Authentication.AccountKey = &accountKey
	}

	return azureBlobStore
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

// generateAzureBlobStoreObservation renders the Azure blob store Nexus
// reported as the observed state of the managed resource.
func generateAzureBlobStoreObservation(_ string, azureBlobStore *v395.AzureBlobStoreApiModel) instancev1alpha1.BlobStoreObservation {
	obs := instancev1alpha1.BlobStoreObservation{Name: azureBlobStore.Name}

	setSoftQuotaObservation(&obs, azureBlobStore.SoftQuota)

	bucket := azureBlobStore.BucketConfiguration
	if bucket.AccountName != "" {
		obs.AccountName = new(bucket.AccountName)
	}

	if bucket.ContainerName != "" {
		obs.ContainerName = new(bucket.ContainerName)
	}

	if bucket.Authentication.AuthenticationMethod != "" {
		obs.AuthenticationMethod = new(bucket.Authentication.AuthenticationMethod)
	}

	obs.PreSignedURLEnabled = bucket.PreSignedUrlEnabled

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

// isAzureBlobStoreUpToDate checks if an Azure blob store matches observed.
// The account key is not compared: Nexus never reports it, so its changes are
// detected through the credentials fingerprint.
func isAzureBlobStoreUpToDate(blobStoreCR *instancev1alpha1.BlobStore) bool {
	if !isSoftQuotaUpToDate(blobStoreCR) {
		return false
	}

	azureConfig := blobStoreCR.Spec.ForProvider.AzureConfig
	if azureConfig == nil {
		return true
	}

	obs := blobStoreCR.Status.AtProvider

	if ptr.Deref(obs.AccountName, "") != azureConfig.AccountName ||
		ptr.Deref(obs.ContainerName, "") != azureConfig.ContainerName ||
		ptr.Deref(obs.AuthenticationMethod, "") != azureConfig.AuthenticationMethod {
		return false
	}

	if azureConfig.PreSignedURLEnabled != nil &&
		*azureConfig.PreSignedURLEnabled != ptr.Deref(obs.PreSignedURLEnabled, false) {
		return false
	}

	return true
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
