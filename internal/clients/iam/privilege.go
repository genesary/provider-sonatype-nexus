package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// privilegeTypeApplication is the application privilege type.
	privilegeTypeApplication = "application"
	// privilegeTypeRepoView is the repository-view privilege type.
	privilegeTypeRepoView = "repository-view"
	// privilegeTypeRepoAdmin is the repository-admin privilege type.
	privilegeTypeRepoAdmin = "repository-admin"
	// privilegeTypeRepoContentSelector is the repository-content-selector type.
	privilegeTypeRepoContentSelector = "repository-content-selector"
	// privilegeTypeScript is the script privilege type.
	privilegeTypeScript = "script"
	// privilegeTypeWildcard is the wildcard privilege type.
	privilegeTypeWildcard = "wildcard"
	// errUnknownPrivilegeType is returned for unrecognised privilege types.
	errUnknownPrivilegeType = "unknown privilege type"
)

// PrivilegeClient manages Nexus privileges.
type PrivilegeClient interface {
	GetPrivilege(ctx context.Context, name string) (*security.Privilege, error)
	CreatePrivilege(ctx context.Context, privCR *iamv1alpha1.Privilege) error
	UpdatePrivilege(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error
	DeletePrivilege(ctx context.Context, name string) error
}

// privilegeClientImpl is the concrete implementation of PrivilegeClient.
type privilegeClientImpl struct {
	nexusClient nexus.Client
}

// NewPrivilegeClient returns a new PrivilegeClient.
func NewPrivilegeClient(creds nexus.Credentials) (PrivilegeClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return &privilegeClientImpl{nexusClient: nexusClient}, nil
}

// GetPrivilege retrieves a privilege by name.
func (c *privilegeClientImpl) GetPrivilege(ctx context.Context, name string) (*security.Privilege, error) {
	return c.nexusClient.Security().GetPrivilege(ctx, name)
}

// CreatePrivilege creates a privilege of the appropriate type.
func (c *privilegeClientImpl) CreatePrivilege(ctx context.Context, privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return c.nexusClient.Security().CreatePrivilegeApplication(ctx, buildApplicationPrivilege(privCR))
	case privilegeTypeRepoView:
		return c.nexusClient.Security().CreatePrivilegeRepositoryView(ctx, buildRepoViewPrivilege(privCR))
	case privilegeTypeRepoAdmin:
		return c.nexusClient.Security().CreatePrivilegeRepositoryAdmin(ctx, buildRepoAdminPrivilege(privCR))
	case privilegeTypeRepoContentSelector:
		return c.nexusClient.Security().CreatePrivilegeRepositoryContentSelector(ctx, buildRepoContentSelectorPrivilege(privCR))
	case privilegeTypeScript:
		return c.nexusClient.Security().CreatePrivilegeScript(ctx, buildScriptPrivilege(privCR))
	case privilegeTypeWildcard:
		return c.nexusClient.Security().CreatePrivilegeWildcard(ctx, buildWildcardPrivilege(privCR))
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// UpdatePrivilege updates a privilege of the appropriate type.
func (c *privilegeClientImpl) UpdatePrivilege(ctx context.Context, name string, privCR *iamv1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return c.nexusClient.Security().UpdatePrivilegeApplication(ctx, name, buildApplicationPrivilege(privCR))
	case privilegeTypeRepoView:
		return c.nexusClient.Security().UpdatePrivilegeRepositoryView(ctx, name, buildRepoViewPrivilege(privCR))
	case privilegeTypeRepoAdmin:
		return c.nexusClient.Security().UpdatePrivilegeRepositoryAdmin(ctx, name, buildRepoAdminPrivilege(privCR))
	case privilegeTypeRepoContentSelector:
		return c.nexusClient.Security().UpdatePrivilegeRepositoryContentSelector(ctx, name, buildRepoContentSelectorPrivilege(privCR))
	case privilegeTypeScript:
		return c.nexusClient.Security().UpdatePrivilegeScript(ctx, name, buildScriptPrivilege(privCR))
	case privilegeTypeWildcard:
		return c.nexusClient.Security().UpdatePrivilegeWildcard(ctx, name, buildWildcardPrivilege(privCR))
	default:
		return errors.New(errUnknownPrivilegeType)
	}
}

// DeletePrivilege removes a privilege by name.
func (c *privilegeClientImpl) DeletePrivilege(ctx context.Context, name string) error {
	return c.nexusClient.Security().DeletePrivilege(ctx, name)
}

// IsPrivilegeUpToDate reports whether the CR matches the observed Privilege.
func IsPrivilegeUpToDate(privCR *iamv1alpha1.Privilege, observed *security.Privilege) bool {
	if privCR.Spec.ForProvider.Description != nil &&
		*privCR.Spec.ForProvider.Description != observed.Description {
		return false
	}

	if !StringSlicesEqual(privCR.Spec.ForProvider.Actions, observed.Actions) {
		return false
	}

	return true
}

// repoPrivilegeFields holds fields common to repository-type privileges.
type repoPrivilegeFields struct {
	name        string
	description string
	format      string
	repository  string
}

// extractRepoFields extracts common repository privilege fields.
func extractRepoFields(privCR *iamv1alpha1.Privilege) repoPrivilegeFields {
	fields := repoPrivilegeFields{name: privCR.Spec.ForProvider.Name}

	if privCR.Spec.ForProvider.Description != nil {
		fields.description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.Format != nil {
		fields.format = *privCR.Spec.ForProvider.Format
	}

	if privCR.Spec.ForProvider.Repository != nil {
		fields.repository = *privCR.Spec.ForProvider.Repository
	}

	return fields
}

// buildApplicationPrivilege builds a PrivilegeApplication from the CR spec.
func buildApplicationPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeApplication {
	privObj := security.PrivilegeApplication{
		Name:    privCR.Spec.ForProvider.Name,
		Actions: toApplicationActions(privCR.Spec.ForProvider.Actions),
	}

	if privCR.Spec.ForProvider.Description != nil {
		privObj.Description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.Domain != nil {
		privObj.Domain = *privCR.Spec.ForProvider.Domain
	}

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

	if privCR.Spec.ForProvider.ContentSelector != nil {
		privObj.ContentSelector = *privCR.Spec.ForProvider.ContentSelector
	}

	return privObj
}

// buildScriptPrivilege builds a PrivilegeScript from the CR spec.
func buildScriptPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeScript {
	privObj := security.PrivilegeScript{
		Name:    privCR.Spec.ForProvider.Name,
		Actions: toScriptActions(privCR.Spec.ForProvider.Actions),
	}

	if privCR.Spec.ForProvider.Description != nil {
		privObj.Description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.ScriptName != nil {
		privObj.ScriptName = *privCR.Spec.ForProvider.ScriptName
	}

	return privObj
}

// buildWildcardPrivilege builds a PrivilegeWildcard from the CR spec.
func buildWildcardPrivilege(privCR *iamv1alpha1.Privilege) security.PrivilegeWildcard {
	privObj := security.PrivilegeWildcard{
		Name: privCR.Spec.ForProvider.Name,
	}

	if privCR.Spec.ForProvider.Description != nil {
		privObj.Description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.Pattern != nil {
		privObj.Pattern = *privCR.Spec.ForProvider.Pattern
	}

	return privObj
}

// toApplicationActions converts string slice to application action types.
func toApplicationActions(actions []string) []security.SecurityPrivilegeApplicationActions {
	result := make([]security.SecurityPrivilegeApplicationActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeApplicationActions(act)
	}

	return result
}

// toRepositoryViewActions converts string slice to repository view
// action types.
func toRepositoryViewActions(actions []string) []security.SecurityPrivilegeRepositoryViewActions {
	result := make([]security.SecurityPrivilegeRepositoryViewActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryViewActions(act)
	}

	return result
}

// toRepositoryAdminActions converts string slice to repository admin action
// types.
func toRepositoryAdminActions(actions []string) []security.SecurityPrivilegeRepositoryAdminActions {
	result := make([]security.SecurityPrivilegeRepositoryAdminActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryAdminActions(act)
	}

	return result
}

// toRepositoryContentSelectorActions converts string slice to repository
// content selector action types.
func toRepositoryContentSelectorActions(actions []string) []security.SecurityPrivilegeRepositoryContentSelectorActions {
	result := make([]security.SecurityPrivilegeRepositoryContentSelectorActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeRepositoryContentSelectorActions(act)
	}

	return result
}

// toScriptActions converts string slice to script action types.
func toScriptActions(actions []string) []security.SecurityPrivilegeScriptActions {
	result := make([]security.SecurityPrivilegeScriptActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeScriptActions(act)
	}

	return result
}
