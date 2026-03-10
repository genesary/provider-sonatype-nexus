// Package license contains the controller for License resources.
package license

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotLicense    = "managed resource is not a License custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Nexus client"
	errGetLicense    = "cannot get license from Nexus"
	errInstallLic    = "cannot install license in Nexus"
	errDeleteLic     = "cannot delete license from Nexus"
	errReadSecret    = "cannot read license secret"
	errSecretKeyMiss = "license secret key not found"

	annotationContentHash = "nexus.crossplane.io/license-content-hash"
)

// Setup adds a controller that reconciles License managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.LicenseGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.LicenseGroupVersionKind),
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
		For(&v1alpha1.License{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.License)
	if !ok {
		return nil, errors.New(errNotLicense)
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

	return &external{client: nc, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
	kube   client.Client
}

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLicense)
	}

	license, err := e.client.License().GetLicense(ctx)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLicense)
	}

	if license == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Populate status.atProvider from the API response
	cr.Status.AtProvider = licenseToObservation(license)

	cr.SetConditions(v1alpha1.Available())

	// Check if the license secret content has changed by comparing hashes
	upToDate, err := e.isUpToDate(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLicense)
	}

	licenseBytes, err := e.readLicenseSecret(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if _, err := e.client.License().InstallLicense(ctx, licenseBytes); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errInstallLic)
	}

	// Store hash of the license content for drift detection
	setContentHashAnnotation(cr, licenseBytes)

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.License)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLicense)
	}

	licenseBytes, err := e.readLicenseSecret(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if _, err := e.client.License().InstallLicense(ctx, licenseBytes); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errInstallLic)
	}

	// Update hash annotation
	setContentHashAnnotation(cr, licenseBytes)

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	_, ok := mg.(*v1alpha1.License)
	if !ok {
		return errors.New(errNotLicense)
	}

	if err := e.client.License().DeleteLicense(ctx); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeleteLic)
	}

	return nil
}

// readLicenseSecret reads the license binary content from the referenced Kubernetes Secret.
func (e *external) readLicenseSecret(ctx context.Context, cr *v1alpha1.License) ([]byte, error) {
	ref := cr.Spec.ForProvider.LicenseSecretRef
	secret := &corev1.Secret{}
	if err := e.kube.Get(ctx, types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}, secret); err != nil {
		return nil, errors.Wrap(err, errReadSecret)
	}

	data, ok := secret.Data[ref.Key]
	if !ok {
		return nil, errors.New(errSecretKeyMiss)
	}

	return data, nil
}

// isUpToDate checks if the license content hash matches the stored annotation.
func (e *external) isUpToDate(ctx context.Context, cr *v1alpha1.License) (bool, error) {
	licenseBytes, err := e.readLicenseSecret(ctx, cr)
	if err != nil {
		return false, err
	}

	currentHash := computeHash(licenseBytes)
	storedHash := ""
	if cr.GetAnnotations() != nil {
		storedHash = cr.GetAnnotations()[annotationContentHash]
	}

	return currentHash == storedHash, nil
}

// licenseToObservation converts a LicenseDetails to a LicenseObservation.
func licenseToObservation(l *nexus.LicenseDetails) v1alpha1.LicenseObservation {
	obs := v1alpha1.LicenseObservation{}
	if l.ContactCompany != "" {
		obs.ContactCompany = &l.ContactCompany
	}
	if l.ContactEmail != "" {
		obs.ContactEmail = &l.ContactEmail
	}
	if l.ContactName != "" {
		obs.ContactName = &l.ContactName
	}
	if l.EffectiveDate != "" {
		obs.EffectiveDate = &l.EffectiveDate
	}
	if l.ExpirationDate != "" {
		obs.ExpirationDate = &l.ExpirationDate
	}
	if l.Features != "" {
		obs.Features = &l.Features
	}
	if l.Fingerprint != "" {
		obs.Fingerprint = &l.Fingerprint
	}
	if l.LicenseType != "" {
		obs.LicenseType = &l.LicenseType
	}
	if l.LicensedUsers != "" {
		obs.LicensedUsers = &l.LicensedUsers
	}
	return obs
}

// setContentHashAnnotation sets the content hash annotation on the CR.
func setContentHashAnnotation(cr *v1alpha1.License, content []byte) {
	annotations := cr.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[annotationContentHash] = computeHash(content)
	cr.SetAnnotations(annotations)
}

// computeHash computes the SHA-256 hash of the given data.
func computeHash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
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
