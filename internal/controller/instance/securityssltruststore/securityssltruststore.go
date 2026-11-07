// Package securityssltruststore manages SSL truststore certificate resources.
package securityssltruststore

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
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
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.SecuritySSLTruststoreGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.SecuritySSLTruststore](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.SecuritySSLTruststoreGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.SecuritySSLTruststoreList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.SecuritySSLTruststore{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.TypedExternalClient[*instancev1alpha1.SecuritySSLTruststore], error) {
	err := c.usage.Track(ctx, truststoreCR)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, truststoreCR)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	sslClient, err := instanceclient.NewSSLTruststoreClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: sslClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.SSLTruststoreClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalObservation, error) {
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

	truststoreCR.Status.AtProvider = instanceclient.CertToObservation(cert)

	truststoreCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsCertUpToDate(truststoreCR),
	}, nil
}

// Create creates the desired certificate in the Nexus truststore.
func (e *external) Create(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalCreation, error) {
	err := e.client.AddCertificate(ctx, truststoreCR.Spec.ForProvider.Pem)
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

	meta.SetExternalName(truststoreCR, added.GetId())

	return managed.ExternalCreation{}, nil
}

// Update reconciles the certificate to the desired state.
func (e *external) Update(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalUpdate, error) {
	oldID := meta.GetExternalName(truststoreCR)
	if oldID != "" {
		err := e.client.RemoveCertificate(ctx, oldID)
		if err != nil && !nexus.IsNotFound(err) {
			return managed.ExternalUpdate{}, errors.Wrap(err, errRemoveCert)
		}
	}

	err := e.client.AddCertificate(ctx, truststoreCR.Spec.ForProvider.Pem)
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

	meta.SetExternalName(truststoreCR, added.GetId())

	return managed.ExternalUpdate{}, nil
}

// Delete removes the certificate from the Nexus truststore.
func (e *external) Delete(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalDelete, error) {
	certID := meta.GetExternalName(truststoreCR)
	if certID == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.RemoveCertificate(ctx, certID)
	if err != nil {
		if nexus.IsNotFound(err) {
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
func (e *external) findCertificateByID(ctx context.Context, certID string) (*v395.ApiCertificate, error) {
	certs, err := e.client.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for idx := range certs {
		if certs[idx].GetId() == certID {
			return &certs[idx], nil
		}
	}

	return nil, errCertNotFound
}

// findCertificateByPem finds a certificate in the truststore by its PEM
// content.
func (e *external) findCertificateByPem(ctx context.Context, pem string) (*v395.ApiCertificate, error) {
	certs, err := e.client.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	normalizedPem := strings.TrimSpace(pem)

	for idx := range certs {
		if strings.TrimSpace(certs[idx].GetPem()) == normalizedPem {
			return &certs[idx], nil
		}
	}

	return nil, errors.New(errFindCert)
}
