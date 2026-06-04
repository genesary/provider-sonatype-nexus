// Package securityssltruststore contains the controller for SecuritySSLTruststore resources.
package securityssltruststore

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotTruststore = "managed resource is not a SecuritySSLTruststore custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Nexus client"
	errListCerts     = "cannot list certificates from Nexus truststore"
	errAddCert       = "cannot add certificate to Nexus truststore"
	errRemoveCert    = "cannot remove certificate from Nexus truststore"
	errFindCert      = "cannot find certificate after adding to truststore"
)

// Setup adds a controller that reconciles SecuritySSLTruststore managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SecuritySSLTruststoreGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SecuritySSLTruststoreGroupVersionKind),
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
		For(&v1alpha1.SecuritySSLTruststore{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.SecuritySSLTruststore)
	if !ok {
		return nil, errors.New(errNotTruststore)
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
	cr, ok := mg.(*v1alpha1.SecuritySSLTruststore)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTruststore)
	}

	certID := meta.GetExternalName(cr)
	if certID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cert, err := e.findCertificateByID(ctx, certID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errListCerts)
	}

	if cert == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Populate status
	cr.Status.AtProvider = certToObservation(cert)

	cr.SetConditions(v1alpha1.Available())

	upToDate := isCertUpToDate(cr, cert)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.SecuritySSLTruststore)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTruststore)
	}

	cert := &security.SSLCertificate{
		Pem: cr.Spec.ForProvider.Pem,
	}

	if err := e.client.SSL().AddCertificate(ctx, cert); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddCert)
	}

	// Find the certificate we just added to get its ID
	added, err := e.findCertificateByPem(ctx, cr.Spec.ForProvider.Pem)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errFindCert)
	}

	if added == nil {
		return managed.ExternalCreation{}, errors.New(errFindCert)
	}

	meta.SetExternalName(cr, added.Id)

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.SecuritySSLTruststore)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTruststore)
	}

	// Remove old certificate
	oldID := meta.GetExternalName(cr)
	if oldID != "" {
		err := e.client.SSL().RemoveCertificate(ctx, oldID)
		if err != nil {
			if !isNotFound(err) {
				return managed.ExternalUpdate{}, errors.Wrap(err, errRemoveCert)
			}
		}
	}

	// Add new certificate
	cert := &security.SSLCertificate{
		Pem: cr.Spec.ForProvider.Pem,
	}

	if err := e.client.SSL().AddCertificate(ctx, cert); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errAddCert)
	}

	// Find the new certificate to get its ID
	added, err := e.findCertificateByPem(ctx, cr.Spec.ForProvider.Pem)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errFindCert)
	}

	if added == nil {
		return managed.ExternalUpdate{}, errors.New(errFindCert)
	}

	meta.SetExternalName(cr, added.Id)

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.SecuritySSLTruststore)
	if !ok {
		return errors.New(errNotTruststore)
	}

	certID := meta.GetExternalName(cr)
	if certID == "" {
		return nil
	}

	err := e.client.SSL().RemoveCertificate(ctx, certID)
	if err != nil {
		if isNotFound(err) {
			return nil
		}

		return errors.Wrap(err, errRemoveCert)
	}

	return nil
}

// findCertificateByID finds a certificate in the truststore by its ID.
func (e *external) findCertificateByID(ctx context.Context, id string) (*security.SSLCertificate, error) {
	certs, err := e.client.SSL().ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for i := range certs {
		if certs[i].Id == id {
			return &certs[i], nil
		}
	}

	return nil, nil
}

// findCertificateByPem finds a certificate in the truststore by its PEM content.
func (e *external) findCertificateByPem(ctx context.Context, pem string) (*security.SSLCertificate, error) {
	certs, err := e.client.SSL().ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	normalizedPem := strings.TrimSpace(pem)
	for i := range certs {
		if strings.TrimSpace(certs[i].Pem) == normalizedPem {
			return &certs[i], nil
		}
	}

	return nil, nil
}

// certToObservation converts an SSLCertificate to an observation.
func certToObservation(cert *security.SSLCertificate) v1alpha1.SecuritySSLTruststoreObservation {
	obs := v1alpha1.SecuritySSLTruststoreObservation{}
	if cert.Id != "" {
		obs.ID = &cert.Id
	}

	if cert.Fingerprint != "" {
		obs.Fingerprint = &cert.Fingerprint
	}

	if cert.SerialNumber != "" {
		obs.SerialNumber = &cert.SerialNumber
	}

	if cert.IssuerCommonName != "" {
		obs.IssuerCommonName = &cert.IssuerCommonName
	}

	if cert.IssuerOrganization != "" {
		obs.IssuerOrganization = &cert.IssuerOrganization
	}

	if cert.IssuerOrganizationUnit != "" {
		obs.IssuerOrganizationUnit = &cert.IssuerOrganizationUnit
	}

	if cert.SubjectCommonName != "" {
		obs.SubjectCommonName = &cert.SubjectCommonName
	}

	if cert.SubjectOrganization != "" {
		obs.SubjectOrganization = &cert.SubjectOrganization
	}

	if cert.SubjectOrganizationUnit != "" {
		obs.SubjectOrganizationUnit = &cert.SubjectOrganizationUnit
	}

	if cert.IssuedOn != 0 {
		obs.IssuedOn = &cert.IssuedOn
	}

	if cert.ExpiresOn != 0 {
		obs.ExpiresOn = &cert.ExpiresOn
	}

	return obs
}

// isCertUpToDate checks if the certificate PEM matches.
func isCertUpToDate(cr *v1alpha1.SecuritySSLTruststore, cert *security.SSLCertificate) bool {
	return strings.TrimSpace(cr.Spec.ForProvider.Pem) == strings.TrimSpace(cert.Pem)
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
