// Package saml manages SAML SSO configuration resources.
package saml

import (
	"context"
	"slices"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/document"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// idpMetadataKey is the data key used when caching the identity provider
// metadata in the writeConnectionSecretToRef secret.
const idpMetadataKey = "idpMetadata"

const (
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetSAML is returned when retrieving SAML configuration fails.
	errGetSAML = "cannot get SAML configuration from Nexus"
	// errApplySAML is returned when applying SAML configuration fails.
	errApplySAML = "cannot apply SAML configuration in Nexus"
	// errDeleteSAML is returned when deleting SAML configuration fails.
	errDeleteSAML = "cannot delete SAML configuration from Nexus"
	// errReadIdPMetadata is returned when reading the identity provider
	// metadata from its source fails.
	errReadIdPMetadata = "cannot read SAML identity provider metadata"
)

// Setup adds a controller that reconciles SAML resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(iamv1alpha1.SAMLGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*iamv1alpha1.SAML](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.SAMLGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &iamv1alpha1.SAMLList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.SAML{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, samlCR *iamv1alpha1.SAML) (managed.TypedExternalClient[*iamv1alpha1.SAML], error) {
	err := c.usage.Track(ctx, samlCR)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, samlCR)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	samlClient, err := iamclient.NewSAMLClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: samlClient, kube: c.kube}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client iamclient.SAMLClient
	kube   client.Client
}

// Observe checks whether the external resource exists and is up-to-date. A
// SAML being deleted, or whose management policies forbid updating it, is
// always up to date: nothing would act on its drift, so the source of its
// metadata being unavailable must not fail its reconcile.
func (e *external) Observe(ctx context.Context, samlCR *iamv1alpha1.SAML) (managed.ExternalObservation, error) {
	observed, err := e.client.Read(ctx)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetSAML)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	samlCR.Status.AtProvider = iamclient.GenerateSAMLObservation(observed)
	samlCR.SetConditions(nexusv1alpha1.Available())

	if meta.WasDeleted(samlCR) || !mayUpdate(samlCR) {
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	idpMetadata, err := e.readIdPMetadata(ctx, samlCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	observation := managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsSAMLUpToDate(samlCR, string(idpMetadata.Data)),
	}

	// Refresh the cache only with the metadata Nexus accepted, so that it
	// follows the endpoint when the URL changes but the metadata does not.
	// Outdated metadata is cached by Create or Update once Nexus takes it.
	if observation.ResourceUpToDate {
		observation.ConnectionDetails = idpMetadata.ConnectionDetails
	}

	return observation, nil
}

// mayUpdate reports whether the management policies of samlCR let the
// reconciler update the SAML configuration. The reconciler rejects
// non-default policies before Observe while the management policies feature
// is disabled, so the policies need no feature check here.
func mayUpdate(samlCR *iamv1alpha1.SAML) bool {
	policies := samlCR.GetManagementPolicies()

	return len(policies) == 0 ||
		slices.Contains(policies, xpv2.ManagementActionUpdate) ||
		slices.Contains(policies, xpv2.ManagementActionAll)
}

// Create creates the desired SAML configuration in Nexus.
func (e *external) Create(ctx context.Context, samlCR *iamv1alpha1.SAML) (managed.ExternalCreation, error) {
	details, err := e.apply(ctx, samlCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{ConnectionDetails: details}, nil
}

// Update reconciles the SAML configuration to the desired state.
func (e *external) Update(ctx context.Context, samlCR *iamv1alpha1.SAML) (managed.ExternalUpdate, error) {
	details, err := e.apply(ctx, samlCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: details}, nil
}

// Delete removes the SAML configuration from Nexus.
func (e *external) Delete(ctx context.Context, _ *iamv1alpha1.SAML) (managed.ExternalDelete, error) {
	err := e.client.Delete(ctx)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteSAML)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// apply reads the identity provider metadata, applies the desired SAML
// configuration to Nexus, and returns the ConnectionDetails that cache the
// metadata.
func (e *external) apply(ctx context.Context, samlCR *iamv1alpha1.SAML) (managed.ConnectionDetails, error) {
	idpMetadata, err := e.readIdPMetadata(ctx, samlCR)
	if err != nil {
		return nil, err
	}

	err = e.client.Apply(ctx, iamclient.GenerateSAML(samlCR, string(idpMetadata.Data)))
	if err != nil {
		return nil, errors.Wrap(err, errApplySAML)
	}

	return idpMetadata.ConnectionDetails, nil
}

// readIdPMetadata reads the identity provider metadata from idpMetadata or
// idpMetadataFrom.
func (e *external) readIdPMetadata(ctx context.Context, samlCR *iamv1alpha1.SAML) (document.Document, error) {
	source := document.Source{
		Inline: samlCR.Spec.ForProvider.IdpMetadata,
		From:   samlCR.Spec.ForProvider.IdpMetadataFrom,
		Parse:  iamclient.ParseIdPMetadata,
		Key:    idpMetadataKey,
	}

	idpMetadata, err := source.Read(ctx, e.kube, samlCR)

	return idpMetadata, errors.Wrap(err, errReadIdPMetadata)
}
