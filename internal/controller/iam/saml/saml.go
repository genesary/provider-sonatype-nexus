// Package saml manages SAML SSO configuration resources.
package saml

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotSAML means the managed resource is not a SAML custom resource.
	errNotSAML = "managed resource is not a SAML custom resource"
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
)

// Setup adds a controller that reconciles SAML resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.SAMLGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.SAMLGroupVersionKind),
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
		For(&iamv1alpha1.SAML{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isSAML := managedRes.(*iamv1alpha1.SAML)
	if !isSAML {
		return nil, errors.New(errNotSAML)
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

	samlClient, err := iamclient.NewSAMLClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: samlClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.SAMLClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	samlCR, isSAML := managedRes.(*iamv1alpha1.SAML)
	if !isSAML {
		return managed.ExternalObservation{}, errors.New(errNotSAML)
	}

	observed, err := e.client.GetSAML(ctx)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetSAML)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	samlCR.Status.AtProvider = iamclient.GenerateSAMLObservation(observed)
	samlCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsSAMLUpToDate(samlCR),
	}, nil
}

// Create creates the desired SAML configuration in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	samlCR, isSAML := managedRes.(*iamv1alpha1.SAML)
	if !isSAML {
		return managed.ExternalCreation{}, errors.New(errNotSAML)
	}

	err := e.client.ApplySAML(ctx, iamclient.GenerateSAML(samlCR))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalCreation{}, nil
}

// Update reconciles the SAML configuration to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	samlCR, isSAML := managedRes.(*iamv1alpha1.SAML)
	if !isSAML {
		return managed.ExternalUpdate{}, errors.New(errNotSAML)
	}

	err := e.client.ApplySAML(ctx, iamclient.GenerateSAML(samlCR))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the SAML configuration from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	_, isSAML := managedRes.(*iamv1alpha1.SAML)
	if !isSAML {
		return managed.ExternalDelete{}, errors.New(errNotSAML)
	}

	err := e.client.DeleteSAML(ctx)
	if err != nil {
		if iamclient.IsNotFound(err) {
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
