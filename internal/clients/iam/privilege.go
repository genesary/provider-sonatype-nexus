package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/security/privilege"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

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

// PrivilegeClient manages Nexus privileges.
type PrivilegeClient interface {
	Get(name string) (*security.Privilege, error)
	Create(privCR *iamv1alpha1.Privilege) error
	Update(name string, privCR *iamv1alpha1.Privilege) error
	Delete(name string) error
}

// privilegeClientImpl dispatches privilege CRUD to the correct sub-service.
type privilegeClientImpl struct {
	svc *privilege.SecurityPrivilegeService
}

// NewPrivilegeClient returns a new PrivilegeClient.
func NewPrivilegeClient(creds nexus.Credentials) (PrivilegeClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &privilegeClientImpl{svc: nc.Security.Privilege}, nil
}

// Get returns the privilege with the given name.
func (c *privilegeClientImpl) Get(name string) (*security.Privilege, error) {
	return c.svc.Get(name)
}

// Create creates a new privilege from the CR spec.
func (c *privilegeClientImpl) Create(privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return c.svc.Application.Create(buildApplicationPrivilege(privCR))
	case privilegeTypeRepoView:
		return c.svc.RepositoryView.Create(buildRepoViewPrivilege(privCR))
	case privilegeTypeRepoAdmin:
		return c.svc.RepositoryAdmin.Create(buildRepoAdminPrivilege(privCR))
	case privilegeTypeRepoContentSelector:
		return c.svc.RepositoryContentSelector.Create(buildRepoContentSelectorPrivilege(privCR))
	case privilegeTypeScript:
		return c.svc.Script.Create(buildScriptPrivilege(privCR))
	case privilegeTypeWildcard:
		return c.svc.Wildcard.Create(buildWildcardPrivilege(privCR))
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// Update updates the named privilege from the CR spec.
func (c *privilegeClientImpl) Update(name string, privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return c.svc.Application.Update(name, buildApplicationPrivilege(privCR))
	case privilegeTypeRepoView:
		return c.svc.RepositoryView.Update(name, buildRepoViewPrivilege(privCR))
	case privilegeTypeRepoAdmin:
		return c.svc.RepositoryAdmin.Update(name, buildRepoAdminPrivilege(privCR))
	case privilegeTypeRepoContentSelector:
		return c.svc.RepositoryContentSelector.Update(name, buildRepoContentSelectorPrivilege(privCR))
	case privilegeTypeScript:
		return c.svc.Script.Update(name, buildScriptPrivilege(privCR))
	case privilegeTypeWildcard:
		return c.svc.Wildcard.Update(name, buildWildcardPrivilege(privCR))
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// Delete deletes the privilege with the given name.
func (c *privilegeClientImpl) Delete(name string) error {
	return c.svc.Delete(name)
}

// GeneratePrivilegeObservation returns the observed Privilege state.
func GeneratePrivilegeObservation(observed *security.Privilege) iamv1alpha1.PrivilegeObservation {
	if observed == nil {
		return iamv1alpha1.PrivilegeObservation{}
	}

	return iamv1alpha1.PrivilegeObservation{
		ReadOnly:    &observed.ReadOnly,
		Description: observed.Description,
		Actions:     observed.Actions,
	}
}

// IsPrivilegeUpToDate reports whether the CR spec matches observed.
func IsPrivilegeUpToDate(privCR *iamv1alpha1.Privilege) bool {
	obs := privCR.Status.AtProvider

	if !helpers.IsComparablePtrEqualComparable(privCR.Spec.ForProvider.Description, obs.Description) {
		return false
	}

	if !helpers.AreStringSlicesEqual(privCR.Spec.ForProvider.Actions, obs.Actions) {
		return false
	}

	return true
}

// repoPrivilegeFields holds common fields shared by repository privilege types.
type repoPrivilegeFields struct {
	name        string
	description string
	format      string
	repository  string
}

// extractRepoFields extracts common repository privilege fields
// from the CR spec.
func extractRepoFields(privCR *iamv1alpha1.Privilege) repoPrivilegeFields {
	fields := repoPrivilegeFields{name: privCR.Spec.ForProvider.Name}

	helpers.AssignIfNonNil(&fields.description, privCR.Spec.ForProvider.Description)
	helpers.AssignIfNonNil(&fields.format, privCR.Spec.ForProvider.Format)
	helpers.AssignIfNonNil(&fields.repository, privCR.Spec.ForProvider.Repository)

	return fields
}

// buildApplicationPrivilege builds a PrivilegeApplication from the CR spec.
func buildApplicationPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeApplication {
	privObj := security.PrivilegeApplication{
		Name:    privCR.Spec.ForProvider.Name,
		Actions: toApplicationActions(privCR.Spec.ForProvider.Actions),
	}

	helpers.AssignIfNonNil(&privObj.Description, privCR.Spec.ForProvider.Description)
	helpers.AssignIfNonNil(&privObj.Domain, privCR.Spec.ForProvider.Domain)

	return privObj
}

// buildRepoViewPrivilege builds a PrivilegeRepositoryView from the CR spec.
func buildRepoViewPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeRepositoryView {
	fields := extractRepoFields(privCR)

	return security.PrivilegeRepositoryView{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryViewActions(privCR.Spec.ForProvider.Actions),
	}
}

// buildRepoAdminPrivilege builds a PrivilegeRepositoryAdmin from the CR spec.
func buildRepoAdminPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeRepositoryAdmin {
	fields := extractRepoFields(privCR)

	return security.PrivilegeRepositoryAdmin{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryAdminActions(privCR.Spec.ForProvider.Actions),
	}
}

// buildRepoContentSelectorPrivilege builds a
// PrivilegeRepositoryContentSelector from the CR spec.
func buildRepoContentSelectorPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeRepositoryContentSelector {
	fields := extractRepoFields(privCR)

	privObj := security.PrivilegeRepositoryContentSelector{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryContentSelectorActions(privCR.Spec.ForProvider.Actions),
	}

	helpers.AssignIfNonNil(&privObj.ContentSelector, privCR.Spec.ForProvider.ContentSelector)

	return privObj
}

// buildScriptPrivilege builds a PrivilegeScript from the CR spec.
func buildScriptPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeScript {
	privObj := security.PrivilegeScript{
		Name:    privCR.Spec.ForProvider.Name,
		Actions: toScriptActions(privCR.Spec.ForProvider.Actions),
	}

	helpers.AssignIfNonNil(&privObj.Description, privCR.Spec.ForProvider.Description)
	helpers.AssignIfNonNil(&privObj.ScriptName, privCR.Spec.ForProvider.ScriptName)

	return privObj
}

// buildWildcardPrivilege builds a PrivilegeWildcard from the CR spec.
func buildWildcardPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeWildcard {
	privObj := security.PrivilegeWildcard{
		Name: privCR.Spec.ForProvider.Name,
	}

	helpers.AssignIfNonNil(&privObj.Description, privCR.Spec.ForProvider.Description)
	helpers.AssignIfNonNil(&privObj.Pattern, privCR.Spec.ForProvider.Pattern)

	return privObj
}

// toApplicationActions converts string action names to
// PrivilegeApplication action types.
func toApplicationActions(actions []string) []security.SecurityPrivilegeApplicationActions {
	result := make([]security.SecurityPrivilegeApplicationActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeApplicationActions(act)
	}

	return result
}

// toRepositoryViewActions converts string action names to
// PrivilegeRepositoryView action types.
func toRepositoryViewActions(actions []string) []security.SecurityPrivilegeRepositoryViewActions {
	result := make([]security.SecurityPrivilegeRepositoryViewActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryViewActions(act)
	}

	return result
}

// toRepositoryAdminActions converts string action names to
// PrivilegeRepositoryAdmin action types.
func toRepositoryAdminActions(actions []string) []security.SecurityPrivilegeRepositoryAdminActions {
	result := make([]security.SecurityPrivilegeRepositoryAdminActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryAdminActions(act)
	}

	return result
}

// toRepositoryContentSelectorActions converts strings to
// PrivilegeRepositoryContentSelector action types.
func toRepositoryContentSelectorActions(actions []string) []security.SecurityPrivilegeRepositoryContentSelectorActions {
	result := make([]security.SecurityPrivilegeRepositoryContentSelectorActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryContentSelectorActions(act)
	}

	return result
}

// toScriptActions converts string action names to PrivilegeScript action types.
func toScriptActions(actions []string) []security.SecurityPrivilegeScriptActions {
	result := make([]security.SecurityPrivilegeScriptActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeScriptActions(act)
	}

	return result
}
