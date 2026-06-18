// Package role manages Role resources.
package role

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
)

const (
	// errNotRole means the managed resource is not a Role custom resource.
	errNotRole = "managed resource is not a Role custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetRole is returned when retrieving a Role fails.
	errGetRole = "cannot get role from Nexus"
	// errCreateRole is returned when creating a Role fails.
	errCreateRole = "cannot create role in Nexus"
	// errUpdateRole is returned when updating a Role fails.
	errUpdateRole = "cannot update role in Nexus"
	// errDeleteRole is returned when deleting a Role fails.
	errDeleteRole = "cannot delete role from Nexus"
)

// Setup adds a controller that reconciles Role resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.RoleGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.RoleGroupVersionKind),
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
		For(&iamv1alpha1.Role{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isRole := managedRes.(*iamv1alpha1.Role)
	if !isRole {
		return nil, errors.New(errNotRole)
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

	roleClient, err := iamclient.NewRoleClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: roleClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.RoleClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	roleRes, isRole := managedRes.(*iamv1alpha1.Role)
	if !isRole {
		return managed.ExternalObservation{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	observed, err := e.client.GetRole(ctx, roleID)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetRole)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	roleRes.SetConditions(nexusv1alpha1.Available())

	roleRes.Status.AtProvider = iamclient.GenerateRoleObservation(observed)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsRoleUpToDate(roleRes),
	}, nil
}

// Create creates the desired Role in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	roleRes, isRole := managedRes.(*iamv1alpha1.Role)
	if !isRole {
		return managed.ExternalCreation{}, errors.New(errNotRole)
	}

	roleData := iamclient.GenerateRole(roleRes)

	err := e.client.CreateRole(ctx, roleData)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRole)
	}

	meta.SetExternalName(roleRes, roleRes.Spec.ForProvider.ID)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Role to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	roleRes, isRole := managedRes.(*iamv1alpha1.Role)
	if !isRole {
		return managed.ExternalUpdate{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	roleData := iamclient.GenerateRole(roleRes)

	err := e.client.UpdateRole(ctx, roleID, roleData)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRole)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Role from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	roleRes, isRole := managedRes.(*iamv1alpha1.Role)
	if !isRole {
		return managed.ExternalDelete{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	err := e.client.DeleteRole(ctx, roleID)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRole)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
