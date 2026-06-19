// Package privilege manages Privilege resources.
package privilege

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

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errNotPrivilege means the managed resource is not a Privilege.
	errNotPrivilege = "managed resource is not a Privilege custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetPrivilege is returned when retrieving a Privilege fails.
	errGetPrivilege = "cannot get privilege from Nexus"
	// errCreatePrivilege is returned when creating a Privilege fails.
	errCreatePrivilege = "cannot create privilege in Nexus"
	// errUpdatePrivilege is returned when updating a Privilege fails.
	errUpdatePrivilege = "cannot update privilege in Nexus"
	// errDeletePrivilege is returned when deleting a Privilege fails.
	errDeletePrivilege = "cannot delete privilege from Nexus"
)

// Setup adds a controller that reconciles Privilege resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.PrivilegeGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.PrivilegeGroupVersionKind),
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
		For(&iamv1alpha1.Privilege{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isPrivilege := managedRes.(*iamv1alpha1.Privilege)
	if !isPrivilege {
		return nil, errors.New(errNotPrivilege)
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

	privilegeClient, err := iamclient.NewPrivilegeClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: privilegeClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.PrivilegeClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	privRes, isPrivilege := managedRes.(*iamv1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalObservation{}, errors.New(errNotPrivilege)
	}

	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	observed, err := e.client.GetPrivilege(ctx, privName)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetPrivilege)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	privRes.SetConditions(nexusv1alpha1.Available())

	privRes.Status.AtProvider = iamclient.GeneratePrivilegeObservation(observed)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsPrivilegeUpToDate(privRes),
	}, nil
}

// Create creates the desired Privilege in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	privRes, isPrivilege := managedRes.(*iamv1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalCreation{}, errors.New(errNotPrivilege)
	}

	err := e.client.CreatePrivilege(ctx, privRes)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePrivilege)
	}

	meta.SetExternalName(privRes, privRes.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Privilege to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	privRes, isPrivilege := managedRes.(*iamv1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalUpdate{}, errors.New(errNotPrivilege)
	}

	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	err := e.client.UpdatePrivilege(ctx, privName, privRes)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePrivilege)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Privilege from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	privRes, isPrivilege := managedRes.(*iamv1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalDelete{}, errors.New(errNotPrivilege)
	}

	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	err := e.client.DeletePrivilege(ctx, privName)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePrivilege)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
