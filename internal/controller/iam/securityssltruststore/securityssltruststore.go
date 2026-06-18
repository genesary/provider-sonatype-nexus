// Package securityssltruststore manages SSL truststore certificate resources.
package securityssltruststore

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotTruststore means the managed resource is not a
	// SecuritySSLTruststore custom resource.
	errNotTruststore = "managed resource is not a SecuritySSLTruststore custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when getting the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errListCerts is returned when listing certificates from Nexus fails.
	errListCerts = "cannot list certificates from Nexus truststore"
	// errAddCert is returned when adding a certificate to Nexus fails.
	errAddCert = "cannot add certificate to Nexus truststore"
	// errRemoveCert is returned when removing a certificate from Nexus fails.
	errRemoveCert = "cannot remove certificate from Nexus truststore"
	// errFindCert is returned when a certificate cannot be found after adding.
	errFindCert = "cannot find certificate after adding to truststore"
)

// errCertNotFound is a sentinel error returned by findCertificateByID when
// the requested certificate ID is not present in the truststore list.
var errCertNotFound = errors.New("certificate not found in truststore")

// Setup adds a controller that reconciles SecuritySSLTruststore resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.SecuritySSLTruststoreGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.SecuritySSLTruststoreGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))) //nolint:deprecated // no replacement yet

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.SecuritySSLTruststore{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isTruststore := managedRes.(*iamv1alpha1.SecuritySSLTruststore)
	if !isTruststore {
		return nil, errors.New(errNotTruststore)
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

	sslClient, err := iamclient.NewSSLTruststoreClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: sslClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.SSLTruststoreClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	truststoreCR, isTruststore := managedRes.(*iamv1alpha1.SecuritySSLTruststore)
	if !isTruststore {
		return managed.ExternalObservation{}, errors.New(errNotTruststore)
	}

	certID := meta.GetExternalName(truststoreCR)
	if certID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cert, err := e.findCertificateByID(ctx, certID)
	if err != nil {
		if stderrors.Is(err, errCertNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errListCerts)
	}

	truststoreCR.Status.AtProvider = iamclient.CertToObservation(cert)

	truststoreCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsCertUpToDate(truststoreCR),
	}, nil
}

// Create creates the desired certificate in the Nexus truststore.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	truststoreCR, isTruststore := managedRes.(*iamv1alpha1.SecuritySSLTruststore)
	if !isTruststore {
		return managed.ExternalCreation{}, errors.New(errNotTruststore)
	}

	err := e.client.AddCertificate(ctx, &security.SSLCertificate{
		Pem: truststoreCR.Spec.ForProvider.Pem,
	})
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errAddCert)
	}

	added, err := e.findCertificateByPem(ctx, truststoreCR.Spec.ForProvider.Pem)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errFindCert)
	}

	if added == nil {
		return managed.ExternalCreation{}, errors.New(errFindCert)
	}

	meta.SetExternalName(truststoreCR, added.Id)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the certificate to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	truststoreCR, isTruststore := managedRes.(*iamv1alpha1.SecuritySSLTruststore)
	if !isTruststore {
		return managed.ExternalUpdate{}, errors.New(errNotTruststore)
	}

	oldID := meta.GetExternalName(truststoreCR)
	if oldID != "" {
		err := e.client.RemoveCertificate(ctx, oldID)
		if err != nil && !iamclient.IsNotFound(err) {
			return managed.ExternalUpdate{}, errors.Wrap(err, errRemoveCert)
		}
	}

	err := e.client.AddCertificate(ctx, &security.SSLCertificate{
		Pem: truststoreCR.Spec.ForProvider.Pem,
	})
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errAddCert)
	}

	added, err := e.findCertificateByPem(ctx, truststoreCR.Spec.ForProvider.Pem)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errFindCert)
	}

	if added == nil {
		return managed.ExternalUpdate{}, errors.New(errFindCert)
	}

	meta.SetExternalName(truststoreCR, added.Id)

	return managed.ExternalUpdate{}, nil
}

// Delete removes the certificate from the Nexus truststore.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	truststoreCR, isTruststore := managedRes.(*iamv1alpha1.SecuritySSLTruststore)
	if !isTruststore {
		return managed.ExternalDelete{}, errors.New(errNotTruststore)
	}

	certID := meta.GetExternalName(truststoreCR)
	if certID == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.RemoveCertificate(ctx, certID)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errRemoveCert)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// findCertificateByID finds a certificate in the truststore by its ID.
// Returns errCertNotFound when the ID is not present in the list.
func (e *external) findCertificateByID(ctx context.Context, certID string) (*security.SSLCertificate, error) {
	certs, err := e.client.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for idx := range certs {
		if certs[idx].Id == certID {
			return &certs[idx], nil
		}
	}

	return nil, errCertNotFound
}

// findCertificateByPem finds a certificate in the truststore by its PEM
// content.
func (e *external) findCertificateByPem(ctx context.Context, pem string) (*security.SSLCertificate, error) {
	certs, err := e.client.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	normalizedPem := strings.TrimSpace(pem)

	for idx := range certs {
		if strings.TrimSpace(certs[idx].Pem) == normalizedPem {
			return &certs[idx], nil
		}
	}

	return nil, errors.New(errFindCert)
}
