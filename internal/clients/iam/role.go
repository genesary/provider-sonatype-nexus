package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// RoleClient manages Nexus roles.
type RoleClient interface {
	GetRole(ctx context.Context, id string) (*security.Role, error)
	CreateRole(ctx context.Context, role security.Role) error
	UpdateRole(ctx context.Context, id string, role security.Role) error
	DeleteRole(ctx context.Context, id string) error
}

// NewRoleClient returns a new RoleClient.
func NewRoleClient(creds nexus.Credentials) (RoleClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
}

// GenerateRole converts a Role CR to the Nexus API type.
func GenerateRole(roleRes *iamv1alpha1.Role) security.Role {
	roleData := security.Role{
		ID:         roleRes.Spec.ForProvider.ID,
		Name:       roleRes.Spec.ForProvider.Name,
		Privileges: roleRes.Spec.ForProvider.Privileges,
		Roles:      roleRes.Spec.ForProvider.Roles,
	}

	helpers.AssignIfNonNil(&roleData.Description, roleRes.Spec.ForProvider.Description)

	return roleData
}

// GenerateRoleObservation returns the observed Role state.
// Note: the upstream go-nexus-client security.Role struct does not expose
// the readOnly or source fields returned by the Nexus REST API, so
// RoleObservation.Source and RoleObservation.ReadOnly remain nil until
// the upstream library is updated.
func GenerateRoleObservation(observed *security.Role) iamv1alpha1.RoleObservation {
	if observed == nil {
		return iamv1alpha1.RoleObservation{}
	}

	return iamv1alpha1.RoleObservation{
		Name:        observed.Name,
		Description: observed.Description,
		Privileges:  observed.Privileges,
		Roles:       observed.Roles,
	}
}

// IsRoleUpToDate reports whether the CR spec matches the observed Role.
func IsRoleUpToDate(roleRes *iamv1alpha1.Role) bool {
	obs := roleRes.Status.AtProvider

	if roleRes.Spec.ForProvider.Name != obs.Name {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(roleRes.Spec.ForProvider.Description, obs.Description) {
		return false
	}

	if !helpers.AreStringSlicesEqual(roleRes.Spec.ForProvider.Privileges, obs.Privileges) {
		return false
	}

	if !helpers.AreStringSlicesEqual(roleRes.Spec.ForProvider.Roles, obs.Roles) {
		return false
	}

	return true
}
