// Package ssrfprotection manages SSRFProtection resources.
package ssrfprotection

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
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
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetSSRF is returned when retrieving the SSRF protection settings
	// fails.
	errGetSSRF = "cannot get SSRF protection settings from Nexus"
	// errUpdateSSRF is returned when updating the SSRF protection settings
	// fails.
	errUpdateSSRF = "cannot update SSRF protection settings in Nexus"
	// errResetSSRF is returned when restoring the default SSRF protection
	// settings fails.
	errResetSSRF = "cannot restore default SSRF protection settings in Nexus"
)

// Setup adds a controller that reconciles SSRFProtection resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.SSRFProtectionGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.SSRFProtection](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.SSRFProtectionGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.SSRFProtectionList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.SSRFProtection{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector creates an external client for SSRFProtection resources.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates the external SSRF protection client.
func (c *connector) Connect(ctx context.Context, ssrf *instancev1alpha1.SSRFProtection) (managed.TypedExternalClient[*instancev1alpha1.SSRFProtection], error) {
	err := c.usage.Track(ctx, ssrf)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, ssrf)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	ssrfClient, err := instanceclient.NewSSRFProtectionClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: ssrfClient}, nil
}

// external implements managed.TypedExternalClient for SSRFProtection.
type external struct {
	client instanceclient.SSRFProtectionClient
}

// Observe checks the Nexus SSRF protection settings against desired state.
// The settings always exist; while the resource is being deleted they count
// as absent once they are back to the Nexus default.
func (e *external) Observe(ctx context.Context, ssrf *instancev1alpha1.SSRFProtection) (managed.ExternalObservation, error) {
	observed, err := e.client.Get(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetSSRF)
	}

	ssrf.Status.AtProvider = instanceclient.GenerateSSRFProtectionObservation(observed)

	if meta.WasDeleted(ssrf) && instanceclient.IsDefaultSSRFProtection(observed) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ssrf.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsSSRFProtectionUpToDate(ssrf, observed),
	}, nil
}

// Create applies the SSRF protection settings in Nexus. They are a
// singleton, so Create applies the spec like Update.
func (e *external) Create(ctx context.Context, ssrf *instancev1alpha1.SSRFProtection) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, e.apply(ctx, ssrf)
}

// Update applies the SSRF protection settings in Nexus.
func (e *external) Update(ctx context.Context, ssrf *instancev1alpha1.SSRFProtection) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, e.apply(ctx, ssrf)
}

// Delete restores the Nexus default SSRF protection settings: protection
// enabled, with no exceptions. The settings themselves cannot be removed.
func (e *external) Delete(ctx context.Context, _ *instancev1alpha1.SSRFProtection) (managed.ExternalDelete, error) {
	err := e.client.Update(ctx, instanceclient.DefaultSSRFProtection())
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errResetSSRF)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// apply replaces the Nexus SSRF protection settings with the spec.
func (e *external) apply(ctx context.Context, ssrf *instancev1alpha1.SSRFProtection) error {
	return errors.Wrap(e.client.Update(ctx, instanceclient.GenerateSSRFProtectionUpdate(&ssrf.Spec.ForProvider)), errUpdateSSRF)
}
