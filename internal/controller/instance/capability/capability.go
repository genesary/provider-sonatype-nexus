// Package capability contains the controller for Capability resources.
package capability

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
	// errGetCapability is returned when retrieving a Capability fails.
	errGetCapability = "cannot get capability from Nexus"
	// errListCapabilities is returned when listing capabilities fails.
	errListCapabilities = "cannot list capabilities from Nexus"
	// errCreateCap is returned when creating a Capability fails.
	errCreateCap = "cannot create capability in Nexus"
	// errUpdateCap is returned when updating a Capability fails.
	errUpdateCap = "cannot update capability in Nexus"
	// errDeleteCap is returned when deleting a Capability fails.
	errDeleteCap = "cannot delete capability from Nexus"
)

// Setup adds a controller that reconciles Capability managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.CapabilityGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.Capability](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithServerAssignedExternalName(),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.CapabilityGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.CapabilityList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.Capability{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, capabilityCR *instancev1alpha1.Capability) (managed.TypedExternalClient[*instancev1alpha1.Capability], error) {
	err := c.usage.Track(ctx, capabilityCR)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, capabilityCR)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	capClient, err := instanceclient.NewCapabilityClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: capClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.CapabilityClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, capabilityCR *instancev1alpha1.Capability) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(capabilityCR)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, err := e.client.Get(ctx, id)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetCapability)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	capabilityCR.Status.AtProvider = instanceclient.GenerateCapabilityObservation(observed)
	capabilityCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsCapabilityUpToDate(capabilityCR, observed),
	}, nil
}

// Create creates the external resource. When Nexus refuses it because the
// type allows only one instance and that instance exists - as it always does
// for types like Audit, which Nexus creates itself - Create adopts it.
// Capabilities of types that allow several instances are never adopted.
func (e *external) Create(ctx context.Context, capabilityCR *instancev1alpha1.Capability) (managed.ExternalCreation, error) {
	params := &capabilityCR.Spec.ForProvider

	created, err := e.client.Create(ctx, instanceclient.GenerateCapabilityCreate(params))
	if err == nil {
		meta.SetExternalName(capabilityCR, created.GetId())

		return managed.ExternalCreation{}, nil
	}

	if !errors.Is(err, instanceclient.ErrCapabilityExists) {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCap)
	}

	existing, listErr := e.client.List(ctx)
	if listErr != nil {
		return managed.ExternalCreation{}, errors.Wrap(listErr, errListCapabilities)
	}

	for idx := range existing {
		if instanceclient.IsSameCapabilityInstance(params, &existing[idx]) {
			meta.SetExternalName(capabilityCR, existing[idx].GetId())

			return managed.ExternalCreation{}, nil
		}
	}

	return managed.ExternalCreation{}, errors.Wrap(err, errCreateCap)
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, capabilityCR *instancev1alpha1.Capability) (managed.ExternalUpdate, error) {
	id := meta.GetExternalName(capabilityCR)

	err := e.client.Update(ctx, id, instanceclient.GenerateCapabilityUpdate(&capabilityCR.Spec.ForProvider, id))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCap)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, capabilityCR *instancev1alpha1.Capability) (managed.ExternalDelete, error) {
	id := meta.GetExternalName(capabilityCR)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.Delete(ctx, id)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteCap)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
