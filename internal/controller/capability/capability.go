// Package capability contains the controller for Capability resources.
package capability

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	capabilityclient "github.com/genesary/provider-sonatype-nexus/internal/clients/capability"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errNotCapability means the managed resource is not a Capability.
	errNotCapability = "managed resource is not a Capability custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetCapability is returned when retrieving a Capability fails.
	errGetCapability = "cannot get capability from Nexus"
	// errCreateCap is returned when creating a Capability fails.
	errCreateCap = "cannot create capability in Nexus"
	// errUpdateCap is returned when updating a Capability fails.
	errUpdateCap = "cannot update capability in Nexus"
	// errDeleteCap is returned when deleting a Capability fails.
	errDeleteCap = "cannot delete capability from Nexus"
)

// Setup adds a controller that reconciles Capability managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(instancev1alpha1.CapabilityGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(instancev1alpha1.CapabilityGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.Capability{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isCapability := managedRes.(*instancev1alpha1.Capability)
	if !isCapability {
		return nil, errors.New(errNotCapability)
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

	capClient, err := capabilityclient.NewCapabilityClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: capClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client capabilityclient.CapabilityClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	capabilityCR, isCapability := managedRes.(*instancev1alpha1.Capability)
	if !isCapability {
		return managed.ExternalObservation{}, errors.New(errNotCapability)
	}

	id := meta.GetExternalName(capabilityCR)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, err := e.client.GetCapability(ctx, id)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetCapability)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	capabilityCR.Status.AtProvider = capabilityclient.GenerateCapabilityObservation(observed)
	capabilityCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: capabilityclient.IsCapabilityUpToDate(capabilityCR, observed),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	capabilityCR, isCapability := managedRes.(*instancev1alpha1.Capability)
	if !isCapability {
		return managed.ExternalCreation{}, errors.New(errNotCapability)
	}

	created, err := e.client.CreateCapability(ctx, capabilityclient.GenerateCapabilityCreate(capabilityCR))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCap)
	}

	meta.SetExternalName(capabilityCR, created.ID)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	capabilityCR, isCapability := managedRes.(*instancev1alpha1.Capability)
	if !isCapability {
		return managed.ExternalUpdate{}, errors.New(errNotCapability)
	}

	id := meta.GetExternalName(capabilityCR)

	err := e.client.UpdateCapability(ctx, id, capabilityclient.GenerateCapabilityUpdate(capabilityCR, id))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCap)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	capabilityCR, isCapability := managedRes.(*instancev1alpha1.Capability)
	if !isCapability {
		return managed.ExternalDelete{}, errors.New(errNotCapability)
	}

	id := meta.GetExternalName(capabilityCR)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	err := e.client.DeleteCapability(ctx, id)
	if err != nil {
		if helpers.IsNotFound(err) {
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
