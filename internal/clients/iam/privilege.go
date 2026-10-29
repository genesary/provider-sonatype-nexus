package iam

import (
	"context"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// privilegeTypeApplication is the privilege type for application privileges.
	privilegeTypeApplication = "application"
	// privilegeTypeRepoView is the privilege type for repository-view privileges.
	privilegeTypeRepoView = "repository-view"
	// privilegeTypeRepoAdmin is the privilege type for repository-admin privileges.
	privilegeTypeRepoAdmin = "repository-admin"
	// privilegeTypeRepoContentSelector is the privilege type for repository
	// content-selector privileges.
	privilegeTypeRepoContentSelector = "repository-content-selector"
	// privilegeTypeScript is the privilege type for script privileges.
	privilegeTypeScript = "script"
	// privilegeTypeWildcard is the privilege type for wildcard privileges.
	privilegeTypeWildcard = "wildcard"
	// errUnknownPrivilegeType is the error message for unsupported privilege types.
	errUnknownPrivilegeType = "unknown privilege type"
)

// Privilege is the Nexus representation of a privilege of any type.
type Privilege = v395.ApiPrivilegeRequest

// PrivilegeClient manages Nexus privileges.
type PrivilegeClient interface {
	Get(ctx context.Context, name string) (*Privilege, error)
	Create(ctx context.Context, privCR *iamv1alpha1.Privilege) error
	Update(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error
	Delete(ctx context.Context, name string) error
}

// privilegeClient dispatches privilege writes to the endpoint of their type.
type privilegeClient struct {
	api *v395.SecurityManagementPrivilegesAPIService
}

// NewPrivilegeClient returns a new PrivilegeClient.
func NewPrivilegeClient(creds nexus.Credentials) (PrivilegeClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &privilegeClient{api: nc.SecurityManagementPrivilegesAPI}, nil
}

// Get returns the privilege with the given name.
func (c *privilegeClient) Get(ctx context.Context, name string) (*Privilege, error) {
	priv, resp, err := c.api.GetSecurityPrivileges(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return priv, nil
}

// Create creates a new privilege from the CR spec.
func (c *privilegeClient) Create(ctx context.Context, privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return nexus.APIError(c.api.CreateApplicationPrivilege(ctx).ApiPrivilegeApplicationRequest(buildApplicationPrivilege(privCR)).Execute())
	case privilegeTypeRepoView:
		return nexus.APIError(c.api.CreateRepositoryViewPrivilege(ctx).ApiPrivilegeRepositoryViewRequest(buildRepoViewPrivilege(privCR)).Execute())
	case privilegeTypeRepoAdmin:
		return nexus.APIError(c.api.CreateRepositoryAdminPrivilege(ctx).ApiPrivilegeRepositoryAdminRequest(buildRepoAdminPrivilege(privCR)).Execute())
	case privilegeTypeRepoContentSelector:
		return nexus.APIError(c.api.CreateRepositoryContentSelectorPrivilege(ctx).ApiPrivilegeRepositoryContentSelectorRequest(buildRepoContentSelectorPrivilege(privCR)).Execute())
	case privilegeTypeScript:
		return nexus.APIError(c.api.CreateScriptPrivilege(ctx).ApiPrivilegeScriptRequest(buildScriptPrivilege(privCR)).Execute())
	case privilegeTypeWildcard:
		return nexus.APIError(c.api.CreateWildcardPrivilege(ctx).ApiPrivilegeWildcardRequest(buildWildcardPrivilege(privCR)).Execute())
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// Update updates the named privilege from the CR spec.
func (c *privilegeClient) Update(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return nexus.APIError(c.api.UpdateApplicationPrivilege(ctx, name).ApiPrivilegeApplicationRequest(buildApplicationPrivilege(privCR)).Execute())
	case privilegeTypeRepoView:
		return nexus.APIError(c.api.UpdateRepositoryViewPrivilege(ctx, name).ApiPrivilegeRepositoryViewRequest(buildRepoViewPrivilege(privCR)).Execute())
	case privilegeTypeRepoAdmin:
		return nexus.APIError(c.api.UpdateRepositoryAdminPrivilege(ctx, name).ApiPrivilegeRepositoryAdminRequest(buildRepoAdminPrivilege(privCR)).Execute())
	case privilegeTypeRepoContentSelector:
		return nexus.APIError(c.api.UpdateRepositoryContentSelectorPrivilege(ctx, name).ApiPrivilegeRepositoryContentSelectorRequest(buildRepoContentSelectorPrivilege(privCR)).Execute())
	case privilegeTypeScript:
		return nexus.APIError(c.api.UpdateScriptPrivilege(ctx, name).ApiPrivilegeScriptRequest(buildScriptPrivilege(privCR)).Execute())
	case privilegeTypeWildcard:
		return nexus.APIError(c.api.UpdateWildcardPrivilege(ctx, name).ApiPrivilegeWildcardRequest(buildWildcardPrivilege(privCR)).Execute())
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// Delete deletes the privilege with the given name.
func (c *privilegeClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteSecurityPrivileges(ctx, name).Execute())
}

// GeneratePrivilegeObservation returns the observed Privilege state.
//
// Every field Nexus reports is recorded, including the type specific ones, so
// that drift detection can compare them: a privilege whose repository or
// content selector was changed out of band is otherwise indistinguishable from
// one still matching its spec.
func GeneratePrivilegeObservation(observed *Privilege) iamv1alpha1.PrivilegeObservation {
	if observed == nil {
		return iamv1alpha1.PrivilegeObservation{}
	}

	return iamv1alpha1.PrivilegeObservation{
		ReadOnly:        new(observed.GetReadOnly()),
		Name:            observed.Name,
		Type:            observed.Type,
		Description:     observed.GetDescription(),
		Actions:         observed.Actions,
		Domain:          observed.GetDomain(),
		Format:          observed.GetFormat(),
		Repository:      observed.GetRepository(),
		ContentSelector: observed.GetContentSelector(),
		ScriptName:      observed.GetScriptName(),
		Pattern:         observed.GetPattern(),
	}
}

// IsPrivilegeUpToDate reports whether the CR spec matches observed.
//
// Only the fields the privilege type actually carries are asserted: the payload
// builders submit nothing else, and Nexus reports the rest as empty.
func IsPrivilegeUpToDate(privCR *iamv1alpha1.Privilege) bool {
	spec := &privCR.Spec.ForProvider
	obs := &privCR.Status.AtProvider

	if spec.Type != obs.Type {
		return false
	}

	// The payload builders submit an empty description when the spec sets
	// none, which is how Nexus is told to drop it, so a description removed
	// from the spec is drift.
	if ptr.Deref(spec.Description, "") != obs.Description {
		return false
	}

	if !helpers.AreStringSlicesEqual(spec.Actions, obs.Actions) {
		return false
	}

	return arePrivilegeTypeFieldsUpToDate(spec, obs)
}

// arePrivilegeTypeFieldsUpToDate reports whether the fields specific to the
// privilege type match the observed privilege.
func arePrivilegeTypeFieldsUpToDate(spec *iamv1alpha1.PrivilegeParameters, obs *iamv1alpha1.PrivilegeObservation) bool {
	switch spec.Type {
	case privilegeTypeApplication:
		return helpers.IsComparablePtrEqualComparable(spec.Domain, obs.Domain)
	case privilegeTypeRepoView, privilegeTypeRepoAdmin:
		return areRepoPrivilegeFieldsUpToDate(spec, obs)
	case privilegeTypeRepoContentSelector:
		return areRepoPrivilegeFieldsUpToDate(spec, obs) &&
			helpers.IsComparablePtrEqualComparable(spec.ContentSelector, obs.ContentSelector)
	case privilegeTypeScript:
		return helpers.IsComparablePtrEqualComparable(spec.ScriptName, obs.ScriptName)
	case privilegeTypeWildcard:
		return helpers.IsComparablePtrEqualComparable(spec.Pattern, obs.Pattern)
	default:
		// An unsupported type cannot be created either, so there is nothing
		// left to compare; Create and Update report the error.
		return true
	}
}

// areRepoPrivilegeFieldsUpToDate reports whether the format and repository
// shared by every repository privilege type match the observed privilege.
func areRepoPrivilegeFieldsUpToDate(spec *iamv1alpha1.PrivilegeParameters, obs *iamv1alpha1.PrivilegeObservation) bool {
	return helpers.IsComparablePtrEqualComparable(spec.Format, obs.Format) &&
		helpers.IsComparablePtrEqualComparable(spec.Repository, obs.Repository)
}

// The payload builders below always submit a description, empty when the spec
// sets none, which is how Nexus is told to drop it.

// buildApplicationPrivilege builds an application privilege from the CR spec.
func buildApplicationPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeApplicationRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeApplicationRequest{
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Actions:     nonNilStrings(spec.Actions),
		Domain:      ptr.Deref(spec.Domain, ""),
	}
}

// buildRepoViewPrivilege builds a repository-view privilege from the CR spec.
func buildRepoViewPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeRepositoryViewRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeRepositoryViewRequest{
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Actions:     nonNilStrings(spec.Actions),
		Format:      ptr.Deref(spec.Format, ""),
		Repository:  ptr.Deref(spec.Repository, ""),
	}
}

// buildRepoAdminPrivilege builds a repository-admin privilege from the CR spec.
func buildRepoAdminPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeRepositoryAdminRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeRepositoryAdminRequest{
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Actions:     nonNilStrings(spec.Actions),
		Format:      ptr.Deref(spec.Format, ""),
		Repository:  ptr.Deref(spec.Repository, ""),
	}
}

// buildRepoContentSelectorPrivilege builds a repository-content-selector
// privilege from the CR spec.
func buildRepoContentSelectorPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeRepositoryContentSelectorRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeRepositoryContentSelectorRequest{
		Name:            spec.Name,
		Description:     new(ptr.Deref(spec.Description, "")),
		Actions:         nonNilStrings(spec.Actions),
		Format:          ptr.Deref(spec.Format, ""),
		Repository:      ptr.Deref(spec.Repository, ""),
		ContentSelector: ptr.Deref(spec.ContentSelector, ""),
	}
}

// buildScriptPrivilege builds a script privilege from the CR spec.
func buildScriptPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeScriptRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeScriptRequest{
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Actions:     nonNilStrings(spec.Actions),
		ScriptName:  ptr.Deref(spec.ScriptName, ""),
	}
}

// buildWildcardPrivilege builds a wildcard privilege from the CR spec.
func buildWildcardPrivilege(privCR *iamv1alpha1.Privilege) v395.ApiPrivilegeWildcardRequest {
	spec := privCR.Spec.ForProvider

	return v395.ApiPrivilegeWildcardRequest{
		Name:        spec.Name,
		Description: new(ptr.Deref(spec.Description, "")),
		Pattern:     ptr.Deref(spec.Pattern, ""),
	}
}
