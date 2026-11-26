// Package securityssltruststore manages SSL truststore certificate resources.
package securityssltruststore

import (
	"context"
	stderrors "errors"
	"slices"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/document"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// pemKey is the data key used when caching the certificate in the
// writeConnectionSecretToRef secret.
const pemKey = "pem"

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
	// errFindCert is returned when the certificate the truststore reports
	// holding cannot be found.
	errFindCert = "cannot find existing certificate in truststore"
	// errReadCert is returned when reading the certificate from its source
	// fails.
	errReadCert = "cannot read certificate"
	// errRecordCert is returned when recording the ID of a replaced
	// certificate fails.
	errRecordCert = "cannot record the ID of the replaced certificate"
)

// errCertNotFound is a sentinel error returned by findCertificateByID and
// findCertificateByContent when the truststore does not hold the certificate.
var errCertNotFound = errors.New("certificate not found in truststore")

// Setup adds a controller that reconciles SecuritySSLTruststore resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.SecuritySSLTruststoreGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.SecuritySSLTruststore](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithServerAssignedExternalName(),
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

// Connect produces a TypedExternalClient for the given managed resource.
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

	return &external{client: sslClient, kube: c.kube}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.SSLTruststoreClient
	kube   client.Client
}

// Observe checks whether the external resource exists and is up-to-date. A
// certificate being deleted, or one the management policies do not let the
// reconciler update, is always up to date, so that its source being gone
// cannot block its deletion or its observation.
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

	if meta.WasDeleted(truststoreCR) || !mayUpdate(truststoreCR) {
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	desired, err := e.readCertificate(ctx, truststoreCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	observation := managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsCertUpToDate(truststoreCR, string(desired.Data)),
	}

	// Refresh the cache only with the certificate Nexus trusts, so that it
	// follows the endpoint when the URL changes but the certificate does not.
	// A new certificate is cached by Create or Update once Nexus adds it.
	if observation.ResourceUpToDate {
		observation.ConnectionDetails = desired.ConnectionDetails
	}

	return observation, nil
}

// mayUpdate reports whether the management policies of truststoreCR let the
// reconciler update the certificate. The reconciler rejects non-default
// policies before Observe while the management policies feature is disabled,
// so the policies need no feature check here.
func mayUpdate(truststoreCR *instancev1alpha1.SecuritySSLTruststore) bool {
	policies := truststoreCR.GetManagementPolicies()

	return len(policies) == 0 ||
		slices.Contains(policies, xpv2.ManagementActionUpdate) ||
		slices.Contains(policies, xpv2.ManagementActionAll)
}

// Create adds the desired certificate to the Nexus truststore. When the
// truststore already holds it, Create adopts it.
func (e *external) Create(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalCreation, error) {
	desired, err := e.readCertificate(ctx, truststoreCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	certID, err := e.addCertificate(ctx, string(desired.Data))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(truststoreCR, certID)

	return managed.ExternalCreation{ConnectionDetails: desired.ConnectionDetails}, nil
}

// Update replaces the certificate with the desired one. Nexus cannot update a
// truststore certificate in place, and its ID is the certificate fingerprint,
// so a different certificate always has a different ID: the new one is added
// and the old one removed. In that order, a source serving a certificate that
// Nexus rejects leaves the old one trusted.
func (e *external) Update(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (managed.ExternalUpdate, error) {
	desired, err := e.readCertificate(ctx, truststoreCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	certID, err := e.addCertificate(ctx, string(desired.Data))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	oldID := meta.GetExternalName(truststoreCR)
	if oldID == certID {
		return managed.ExternalUpdate{ConnectionDetails: desired.ConnectionDetails}, nil
	}

	if oldID != "" {
		err = e.client.RemoveCertificate(ctx, oldID)
		if err != nil && !nexus.IsNotFound(err) {
			return managed.ExternalUpdate{}, errors.Wrap(err, errRemoveCert)
		}
	}

	err = e.recordExternalName(ctx, truststoreCR, certID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: desired.ConnectionDetails}, nil
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

// readCertificate reads the desired certificate from pem or pemFrom.
func (e *external) readCertificate(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore) (document.Document, error) {
	source := document.Source{
		Inline: truststoreCR.Spec.ForProvider.Pem,
		From:   truststoreCR.Spec.ForProvider.PemFrom,
		Parse:  instanceclient.ParseCertificate,
		Key:    pemKey,
	}

	certificate, err := source.Read(ctx, e.kube, truststoreCR)

	return certificate, errors.Wrap(err, errReadCert)
}

// recordExternalName persists certID as the external name.
//
// The managed reconciler only persists the external name set by Create, and
// the status update that follows Update overwrites the in-memory one. Left
// unrecorded, the new certificate would stay in Nexus if the resource were
// deleted before its next reconcile.
func (e *external) recordExternalName(ctx context.Context, truststoreCR *instancev1alpha1.SecuritySSLTruststore, certID string) error {
	// Patch a copy: the response overwrites the status Observe generated.
	patched := truststoreCR.DeepCopy()
	meta.SetExternalName(patched, certID)

	err := e.kube.Patch(ctx, patched, client.MergeFrom(truststoreCR))
	if err != nil {
		return errors.Wrap(err, errRecordCert)
	}

	meta.SetExternalName(truststoreCR, certID)
	truststoreCR.SetResourceVersion(patched.GetResourceVersion())

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

// addCertificate adds the PEM encoded certificate to the truststore and
// returns its ID. When the truststore already holds the certificate, it
// returns the ID of the one there, so a certificate added out of band, or by
// an earlier attempt whose result was lost, is adopted instead of failing
// every retry with a conflict.
func (e *external) addCertificate(ctx context.Context, pem string) (string, error) {
	added, err := e.client.AddCertificate(ctx, pem)
	if err == nil && added.GetId() != "" {
		return added.GetId(), nil
	}

	if err != nil && !stderrors.Is(err, instanceclient.ErrCertificateExists) {
		return "", errors.Wrap(err, errAddCert)
	}

	existing, err := e.findCertificateByContent(ctx, pem)
	if err != nil {
		return "", errors.Wrap(err, errFindCert)
	}

	return existing.GetId(), nil
}

// findCertificateByContent finds the truststore certificate that the PEM text
// encodes, whatever its line endings or wrapping. Returns errCertNotFound when
// the truststore does not hold it.
func (e *external) findCertificateByContent(ctx context.Context, pem string) (*v395.ApiCertificate, error) {
	certs, err := e.client.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for idx := range certs {
		if instanceclient.SameCertificate(certs[idx].GetPem(), pem) {
			return &certs[idx], nil
		}
	}

	return nil, errCertNotFound
}
