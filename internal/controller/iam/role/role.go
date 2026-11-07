// Package role manages Role resources.
package role

import (
	"context"

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
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(iamv1alpha1.RoleGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*iamv1alpha1.Role](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.RoleGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &iamv1alpha1.RoleList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.Role{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, roleRes *iamv1alpha1.Role) (managed.TypedExternalClient[*iamv1alpha1.Role], error) {
	err := c.usage.Track(ctx, roleRes)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, roleRes)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	roleClient, err := iamclient.NewRoleClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: roleClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client iamclient.RoleClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, roleRes *iamv1alpha1.Role) (managed.ExternalObservation, error) {
	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	observed, err := e.client.Get(ctx, roleID)
	if err != nil {
		if nexus.IsNotFound(err) {
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
func (e *external) Create(ctx context.Context, roleRes *iamv1alpha1.Role) (managed.ExternalCreation, error) {
	roleData := iamclient.GenerateRole(roleRes)

	err := e.client.Create(ctx, roleData)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRole)
	}

	meta.SetExternalName(roleRes, roleRes.Spec.ForProvider.ID)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Role to the desired state.
func (e *external) Update(ctx context.Context, roleRes *iamv1alpha1.Role) (managed.ExternalUpdate, error) {
	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	roleData := iamclient.GenerateRole(roleRes)

	err := e.client.Update(ctx, roleID, roleData)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRole)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Role from Nexus.
func (e *external) Delete(ctx context.Context, roleRes *iamv1alpha1.Role) (managed.ExternalDelete, error) {
	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	err := e.client.Delete(ctx, roleID)
	if err != nil {
		if nexus.IsNotFound(err) {
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
