package iam

import (
	"context"
	"strings"

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

	if roleRes.Spec.ForProvider.Description != nil {
		roleData.Description = *roleRes.Spec.ForProvider.Description
	}

	return roleData
}

// GenerateRoleObservation returns the observed Role state.
// Note: the upstream go-nexus-client security.Role struct does not expose
// the readOnly or source fields returned by the Nexus REST API, so
// RoleObservation.Source and RoleObservation.ReadOnly remain nil until
// the upstream library is updated.
func GenerateRoleObservation(_ *security.Role) iamv1alpha1.RoleObservation {
	return iamv1alpha1.RoleObservation{}
}

// IsRoleUpToDate reports whether the CR matches the observed Role.
func IsRoleUpToDate(roleRes *iamv1alpha1.Role, observed *security.Role) bool {
	if roleRes.Spec.ForProvider.Name != observed.Name {
		return false
	}

	if roleRes.Spec.ForProvider.Description != nil &&
		*roleRes.Spec.ForProvider.Description != observed.Description {
		return false
	}

	if !helpers.AreStringSlicesEqual(roleRes.Spec.ForProvider.Privileges, observed.Privileges) {
		return false
	}

	if !helpers.AreStringSlicesEqual(roleRes.Spec.ForProvider.Roles, observed.Roles) {
		return false
	}

	return true
}

// IsNotFound reports whether an error indicates the resource was not found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
