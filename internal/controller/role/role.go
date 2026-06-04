// Package role contains the controller for Role resources.
package role

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotRole is returned when the managed resource is not a Role.
	errNotRole = "managed resource is not a Role custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetCreds is returned when retrieving credentials fails.
	errGetCreds = "cannot get credentials"
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

// Setup creates a controller for Role resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.RoleGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RoleGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Role{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the Role controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isRole := managedRes.(*v1alpha1.Role)
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

	provConfig := &v1alpha1.ProviderConfig{}

	err = c.kube.Get(ctx, client.ObjectKey{Name: modernMG.GetProviderConfigReference().Name}, provConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, provConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nexusClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
}

// Observe checks if the Role resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	roleRes, isRole := managedRes.(*v1alpha1.Role)
	if !isRole {
		return managed.ExternalObservation{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	roleResult, err := e.client.Security().GetRole(ctx, roleID)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetRole)
	}

	if roleResult == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	roleRes.SetConditions(v1alpha1.Available())

	upToDate := isRoleUpToDate(roleRes, roleResult)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new Role resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	roleRes, isRole := managedRes.(*v1alpha1.Role)
	if !isRole {
		return managed.ExternalCreation{}, errors.New(errNotRole)
	}

	roleData := generateRole(roleRes)

	err := e.client.Security().CreateRole(ctx, roleData)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRole)
	}

	meta.SetExternalName(roleRes, roleRes.Spec.ForProvider.ID)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing Role resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	roleRes, isRole := managedRes.(*v1alpha1.Role)
	if !isRole {
		return managed.ExternalUpdate{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	roleData := generateRole(roleRes)

	err := e.client.Security().UpdateRole(ctx, roleID, roleData)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRole)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing Role resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	roleRes, isRole := managedRes.(*v1alpha1.Role)
	if !isRole {
		return managed.ExternalDelete{}, errors.New(errNotRole)
	}

	roleID := meta.GetExternalName(roleRes)
	if roleID == "" {
		roleID = roleRes.Spec.ForProvider.ID
	}

	err := e.client.Security().DeleteRole(ctx, roleID)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRole)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateRole generates a Role from the CR spec.
func generateRole(roleRes *v1alpha1.Role) security.Role {
	roleData := security.Role{
		ID:         roleRes.Spec.ForProvider.ID,
		Name:       roleRes.Spec.ForProvider.Name,
		Privileges: roleRes.Spec.ForProvider.Privileges,
		Roles:      roleRes.Spec.ForProvider.Roles,
	}

	if roleRes.Spec.ForProvider.Description != nil {
		roleData.Description = *roleRes.Spec.ForProvider.Description
	}

	return roleData
}

// isRoleUpToDate checks if a Role is up to date.
func isRoleUpToDate(roleRes *v1alpha1.Role, roleData *security.Role) bool {
	if roleRes.Spec.ForProvider.Name != roleData.Name {
		return false
	}

	if roleRes.Spec.ForProvider.Description != nil &&
		*roleRes.Spec.ForProvider.Description != roleData.Description {
		return false
	}

	if !stringSlicesEqual(roleRes.Spec.ForProvider.Privileges, roleData.Privileges) {
		return false
	}

	if !stringSlicesEqual(roleRes.Spec.ForProvider.Roles, roleData.Roles) {
		return false
	}

	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(sliceA, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}

	for idx := range sliceA {
		if sliceA[idx] != sliceB[idx] {
			return false
		}
	}

	return true
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
