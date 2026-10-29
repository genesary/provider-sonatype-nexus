package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// RoleClient manages Nexus roles.
type RoleClient interface {
	Get(ctx context.Context, id string) (*v395.RoleXOResponse, error)
	Create(ctx context.Context, role v395.RoleXORequest) error
	Update(ctx context.Context, id string, role v395.RoleXORequest) error
	Delete(ctx context.Context, id string) error
}

// roleClient implements RoleClient over the Nexus API.
type roleClient struct {
	api *v395.SecurityManagementRolesAPIService
}

// NewRoleClient returns a new RoleClient.
func NewRoleClient(creds nexus.Credentials) (RoleClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &roleClient{api: nc.SecurityManagementRolesAPI}, nil
}

// Get returns the role with the given ID.
func (c *roleClient) Get(ctx context.Context, id string) (*v395.RoleXOResponse, error) {
	role, resp, err := c.api.GetSecurityRoles(ctx, id).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return role, nil
}

// Create creates a role.
func (c *roleClient) Create(ctx context.Context, role v395.RoleXORequest) error {
	_, resp, err := c.api.CreateSecurityRoles(ctx).RoleXORequest(role).Execute()

	return nexus.APIError(resp, err)
}

// Update updates the role with the given ID.
func (c *roleClient) Update(ctx context.Context, id string, role v395.RoleXORequest) error {
	return nexus.APIError(c.api.UpdateSecurityRoles(ctx, id).RoleXORequest(role).Execute())
}

// Delete deletes the role with the given ID.
func (c *roleClient) Delete(ctx context.Context, id string) error {
	return nexus.APIError(c.api.DeleteSecurityRoles(ctx, id).Execute())
}

// GenerateRole converts a Role CR to the Nexus API type.
//
// Privileges and roles are always submitted, empty included, so that removing
// every entry from the spec removes it from Nexus too.
func GenerateRole(roleRes *iamv1alpha1.Role) v395.RoleXORequest {
	spec := roleRes.Spec.ForProvider

	return v395.RoleXORequest{
		Id:          spec.ID,
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Privileges:  nonNilStrings(spec.Privileges),
		Roles:       nonNilStrings(spec.Roles),
	}
}

// GenerateRoleObservation returns the observed Role state.
func GenerateRoleObservation(observed *v395.RoleXOResponse) iamv1alpha1.RoleObservation {
	if observed == nil {
		return iamv1alpha1.RoleObservation{}
	}

	return iamv1alpha1.RoleObservation{
		ID:          observed.GetId(),
		Name:        observed.GetName(),
		Description: observed.GetDescription(),
		Privileges:  observed.Privileges,
		Roles:       observed.Roles,
		Source:      observed.Source,
		ReadOnly:    observed.ReadOnly,
	}
}

// nonNilStrings returns values, or an empty slice when values is nil, so that
// the list is submitted as [] rather than left out of the payload.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

// IsRoleUpToDate reports whether the CR spec matches the observed Role.
func IsRoleUpToDate(roleRes *iamv1alpha1.Role) bool {
	obs := roleRes.Status.AtProvider

	if roleRes.Spec.ForProvider.Name != obs.Name {
		return false
	}

	// GenerateRole submits an empty description when the spec sets none, and
	// Nexus stores it verbatim, so dropping the description is drift.
	if ptr.Deref(roleRes.Spec.ForProvider.Description, "") != obs.Description {
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
