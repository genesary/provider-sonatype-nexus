// Package blobstore contains the controller for BlobStore resources.
package blobstore

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotBlobStore    = "managed resource is not a BlobStore custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errNewClient       = "cannot create new Nexus client"
	errGetBlobStore    = "cannot get blob store from Nexus"
	errCreateBlobStore = "cannot create blob store in Nexus"
	errUpdateBlobStore = "cannot update blob store in Nexus"
	errDeleteBlobStore = "cannot delete blob store from Nexus"
)

// Setup adds a controller that reconciles BlobStore managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.BlobStoreGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.BlobStoreGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.BlobStore{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.BlobStore)
	if !ok {
		return nil, errors.New(errNotBlobStore)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
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

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.BlobStore)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	var exists bool
	var upToDate bool

	switch cr.Spec.ForProvider.Type {
	case "File":
		bs, err := e.client.BlobStore().GetFile(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return managed.ExternalObservation{ResourceExists: false}, nil
			}
			return managed.ExternalObservation{}, errors.Wrap(err, errGetBlobStore)
		}
		if bs != nil {
			exists = true
			upToDate = isFileBlobStoreUpToDate(cr, bs)
		}
	case "S3":
		bs, err := e.client.BlobStore().GetS3(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return managed.ExternalObservation{ResourceExists: false}, nil
			}
			return managed.ExternalObservation{}, errors.Wrap(err, errGetBlobStore)
		}
		if bs != nil {
			exists = true
			upToDate = isS3BlobStoreUpToDate(cr, bs)
		}
	default:
		// Default to File type
		bs, err := e.client.BlobStore().GetFile(ctx, name)
		if err != nil {
			if isNotFound(err) {
				return managed.ExternalObservation{ResourceExists: false}, nil
			}
			return managed.ExternalObservation{}, errors.Wrap(err, errGetBlobStore)
		}
		if bs != nil {
			exists = true
			upToDate = isFileBlobStoreUpToDate(cr, bs)
		}
	}

	if !exists {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.BlobStore)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBlobStore)
	}

	switch cr.Spec.ForProvider.Type {
	case "File":
		bs := generateFileBlobStore(cr)
		if err := e.client.BlobStore().CreateFile(ctx, bs); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	case "S3":
		bs := generateS3BlobStore(cr)
		if err := e.client.BlobStore().CreateS3(ctx, bs); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	default:
		// Default to File type
		bs := generateFileBlobStore(cr)
		if err := e.client.BlobStore().CreateFile(ctx, bs); err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errCreateBlobStore)
		}
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.BlobStore)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	switch cr.Spec.ForProvider.Type {
	case "File":
		bs := generateFileBlobStore(cr)
		if err := e.client.BlobStore().UpdateFile(ctx, name, bs); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	case "S3":
		bs := generateS3BlobStore(cr)
		if err := e.client.BlobStore().UpdateS3(ctx, name, bs); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	default:
		// Default to File type
		bs := generateFileBlobStore(cr)
		if err := e.client.BlobStore().UpdateFile(ctx, name, bs); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateBlobStore)
		}
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.BlobStore)
	if !ok {
		return errors.New(errNotBlobStore)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	if err := e.client.BlobStore().Delete(ctx, name); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeleteBlobStore)
	}

	return nil
}

// generateFileBlobStore generates a File blob store from the CR spec.
func generateFileBlobStore(cr *v1alpha1.BlobStore) *blobstore.File {
	bs := &blobstore.File{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.Path != nil {
		bs.Path = *cr.Spec.ForProvider.Path
	}

	if cr.Spec.ForProvider.SoftQuota != nil {
		bs.SoftQuota = &blobstore.SoftQuota{}
		if cr.Spec.ForProvider.SoftQuota.Type != nil {
			bs.SoftQuota.Type = *cr.Spec.ForProvider.SoftQuota.Type
		}
		if cr.Spec.ForProvider.SoftQuota.Limit != nil {
			bs.SoftQuota.Limit = *cr.Spec.ForProvider.SoftQuota.Limit
		}
	}

	return bs
}

// generateS3BlobStore generates an S3 blob store from the CR spec.
func generateS3BlobStore(cr *v1alpha1.BlobStore) *blobstore.S3 {
	bs := &blobstore.S3{
		Name: cr.Spec.ForProvider.Name,
	}

	if cr.Spec.ForProvider.SoftQuota != nil {
		bs.SoftQuota = &blobstore.SoftQuota{}
		if cr.Spec.ForProvider.SoftQuota.Type != nil {
			bs.SoftQuota.Type = *cr.Spec.ForProvider.SoftQuota.Type
		}
		if cr.Spec.ForProvider.SoftQuota.Limit != nil {
			bs.SoftQuota.Limit = *cr.Spec.ForProvider.SoftQuota.Limit
		}
	}

	if cr.Spec.ForProvider.S3Config != nil {
		bs.BucketConfiguration = blobstore.S3BucketConfiguration{
			Bucket: blobstore.S3Bucket{
				Name: cr.Spec.ForProvider.S3Config.Bucket,
			},
		}
		if cr.Spec.ForProvider.S3Config.Region != nil {
			bs.BucketConfiguration.Bucket.Region = *cr.Spec.ForProvider.S3Config.Region
		}
		if cr.Spec.ForProvider.S3Config.Prefix != nil {
			bs.BucketConfiguration.Bucket.Prefix = *cr.Spec.ForProvider.S3Config.Prefix
		}
		if cr.Spec.ForProvider.S3Config.ExpirationDays != nil {
			bs.BucketConfiguration.Bucket.Expiration = *cr.Spec.ForProvider.S3Config.ExpirationDays
		}
	}

	return bs
}

// isFileBlobStoreUpToDate checks if a File blob store is up to date.
func isFileBlobStoreUpToDate(cr *v1alpha1.BlobStore, bs *blobstore.File) bool {
	if cr.Spec.ForProvider.Path != nil && bs.Path != *cr.Spec.ForProvider.Path {
		return false
	}

	if cr.Spec.ForProvider.SoftQuota != nil {
		if bs.SoftQuota == nil {
			return false
		}
		if cr.Spec.ForProvider.SoftQuota.Type != nil && bs.SoftQuota.Type != *cr.Spec.ForProvider.SoftQuota.Type {
			return false
		}
		if cr.Spec.ForProvider.SoftQuota.Limit != nil && bs.SoftQuota.Limit != *cr.Spec.ForProvider.SoftQuota.Limit {
			return false
		}
	}

	return true
}

// isS3BlobStoreUpToDate checks if an S3 blob store is up to date.
func isS3BlobStoreUpToDate(cr *v1alpha1.BlobStore, bs *blobstore.S3) bool {
	if cr.Spec.ForProvider.SoftQuota != nil {
		if bs.SoftQuota == nil {
			return false
		}
		if cr.Spec.ForProvider.SoftQuota.Type != nil && bs.SoftQuota.Type != *cr.Spec.ForProvider.SoftQuota.Type {
			return false
		}
		if cr.Spec.ForProvider.SoftQuota.Limit != nil && bs.SoftQuota.Limit != *cr.Spec.ForProvider.SoftQuota.Limit {
			return false
		}
	}

	if cr.Spec.ForProvider.S3Config != nil {
		if bs.BucketConfiguration.Bucket.Name != cr.Spec.ForProvider.S3Config.Bucket {
			return false
		}
		if cr.Spec.ForProvider.S3Config.Region != nil && bs.BucketConfiguration.Bucket.Region != *cr.Spec.ForProvider.S3Config.Region {
			return false
		}
		if cr.Spec.ForProvider.S3Config.Prefix != nil && bs.BucketConfiguration.Bucket.Prefix != *cr.Spec.ForProvider.S3Config.Prefix {
			return false
		}
	}

	return true
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
