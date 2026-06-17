// Package privilege contains the controller for Privilege resources.
package privilege

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
	// errNotPrivilege is returned when the managed resource is not a Privilege.
	errNotPrivilege = "managed resource is not a Privilege custom resource"
	// errTrackPCUsage is returned when ProviderConfig usage tracking fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when the ProviderConfig cannot be retrieved.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when the Nexus client cannot be created.
	errNewClient = "cannot create new Nexus client"
	// errGetPrivilege is returned when a privilege cannot be fetched from Nexus.
	errGetPrivilege = "cannot get privilege from Nexus"
	// errCreatePrivilege is returned when a privilege cannot be created.
	errCreatePrivilege = "cannot create privilege in Nexus"
	// errUpdatePrivilege is returned when a privilege cannot be updated.
	errUpdatePrivilege = "cannot update privilege in Nexus"
	// errDeletePrivilege is returned when a privilege cannot be deleted.
	errDeletePrivilege = "cannot delete privilege from Nexus"
	// errUnknownType is returned when the privilege type is not recognized.
	errUnknownType = "unknown privilege type"

	// privilegeTypeApplication is the application privilege type.
	privilegeTypeApplication = "application"
	// privilegeTypeRepoView is the repository-view privilege type.
	privilegeTypeRepoView = "repository-view"
	// privilegeTypeRepoAdmin is the repository-admin privilege type.
	privilegeTypeRepoAdmin = "repository-admin"
	// privilegeTypeRepoContentSelector is the repo-content-selector privilege type.
	privilegeTypeRepoContentSelector = "repository-content-selector"
	// privilegeTypeScript is the script privilege type.
	privilegeTypeScript = "script"
	// privilegeTypeWildcard is the wildcard privilege type.
	privilegeTypeWildcard = "wildcard"
)

// repoPrivilegeFields holds fields common to repository-type privileges.
type repoPrivilegeFields struct {
	name        string
	description string
	format      string
	repository  string
}

// extractRepoFields extracts common repository privilege fields from the
// CR spec.
func extractRepoFields(privCR *v1alpha1.Privilege) repoPrivilegeFields {
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

// Setup adds a controller that reconciles Privilege managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.PrivilegeGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.PrivilegeGroupVersionKind),
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
		For(&v1alpha1.Privilege{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isPrivilege := managedRes.(*v1alpha1.Privilege)
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

// Observe the external resource.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	privCR, isPrivilege := managedRes.(*v1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalObservation{}, errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(privCR)
	if name == "" {
		name = privCR.Spec.ForProvider.Name
	}

	priv, err := e.client.Security().GetPrivilege(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetPrivilege)
	}

	if priv == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	privCR.SetConditions(v1alpha1.Available())

	upToDate := isPrivilegeUpToDate(privCR, priv)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	privCR, isPrivilege := managedRes.(*v1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalCreation{}, errors.New(errNotPrivilege)
	}

	err := createPrivilege(ctx, e.client, privCR)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePrivilege)
	}

	meta.SetExternalName(privCR, privCR.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	privCR, isPrivilege := managedRes.(*v1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalUpdate{}, errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(privCR)
	if name == "" {
		name = privCR.Spec.ForProvider.Name
	}

	err := updatePrivilege(ctx, e.client, name, privCR)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePrivilege)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	privCR, isPrivilege := managedRes.(*v1alpha1.Privilege)
	if !isPrivilege {
		return managed.ExternalDelete{}, errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(privCR)
	if name == "" {
		name = privCR.Spec.ForProvider.Name
	}

	err := e.client.Security().DeletePrivilege(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePrivilege)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

// toApplicationActions converts string slice to application action types.
func toApplicationActions(actions []string) []security.SecurityPrivilegeApplicationActions {
	result := make([]security.SecurityPrivilegeApplicationActions, len(actions))
	for idx, act := range actions {
		result[idx] = security.SecurityPrivilegeApplicationActions(act)
	}

	return result
}

// toRepositoryViewActions converts string slice to repository view action
// types.
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

// toRepositoryContentSelectorActions converts string slice to
// repository content selector action types.
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

// createPrivilege creates a privilege based on its type.
func createPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return createAppPrivilege(ctx, nexusClient, privCR)
	case privilegeTypeRepoView:
		return createRepoViewPrivilege(ctx, nexusClient, privCR)
	case privilegeTypeRepoAdmin:
		return createRepoAdminPrivilege(ctx, nexusClient, privCR)
	case privilegeTypeRepoContentSelector:
		return createRepoContentSelectorPrivilege(ctx, nexusClient, privCR)
	case privilegeTypeScript:
		return createScriptPrivilege(ctx, nexusClient, privCR)
	case privilegeTypeWildcard:
		return createWildcardPrivilege(ctx, nexusClient, privCR)
	default:
		return errors.New(errUnknownType)
	}
}

// buildApplicationPrivilege builds a PrivilegeApplication from the CR spec.
func buildApplicationPrivilege(privCR *v1alpha1.Privilege) security.PrivilegeApplication {
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

// createAppPrivilege creates an application privilege.
func createAppPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
	return nexusClient.Security().CreatePrivilegeApplication(ctx, buildApplicationPrivilege(privCR))
}

// createRepoViewPrivilege creates a repository-view privilege.
func createRepoViewPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
	fields := extractRepoFields(privCR)
	privObj := security.PrivilegeRepositoryView{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryViewActions(privCR.Spec.ForProvider.Actions),
	}

	return nexusClient.Security().CreatePrivilegeRepositoryView(ctx, privObj)
}

// createRepoAdminPrivilege creates a repository-admin privilege.
func createRepoAdminPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
	fields := extractRepoFields(privCR)
	privObj := security.PrivilegeRepositoryAdmin{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryAdminActions(privCR.Spec.ForProvider.Actions),
	}

	return nexusClient.Security().CreatePrivilegeRepositoryAdmin(ctx, privObj)
}

// createRepoContentSelectorPrivilege creates a repo-content-selector
// privilege.
func createRepoContentSelectorPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
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

	return nexusClient.Security().CreatePrivilegeRepositoryContentSelector(ctx, privObj)
}

// createScriptPrivilege creates a script privilege.
func createScriptPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
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

	return nexusClient.Security().CreatePrivilegeScript(ctx, privObj)
}

// createWildcardPrivilege creates a wildcard privilege.
func createWildcardPrivilege(ctx context.Context, nexusClient nexus.Client, privCR *v1alpha1.Privilege) error {
	privObj := security.PrivilegeWildcard{
		Name: privCR.Spec.ForProvider.Name,
	}
	if privCR.Spec.ForProvider.Description != nil {
		privObj.Description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.Pattern != nil {
		privObj.Pattern = *privCR.Spec.ForProvider.Pattern
	}

	return nexusClient.Security().CreatePrivilegeWildcard(ctx, privObj)
}

// updatePrivilege updates a privilege based on its type.
func updatePrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
	switch privCR.Spec.ForProvider.Type {
	case privilegeTypeApplication:
		return updateAppPrivilege(ctx, nexusClient, name, privCR)
	case privilegeTypeRepoView:
		return updateRepoViewPrivilege(ctx, nexusClient, name, privCR)
	case privilegeTypeRepoAdmin:
		return updateRepoAdminPrivilege(ctx, nexusClient, name, privCR)
	case privilegeTypeRepoContentSelector:
		return updateRepoContentSelectorPrivilege(ctx, nexusClient, name, privCR)
	case privilegeTypeScript:
		return updateScriptPrivilege(ctx, nexusClient, name, privCR)
	case privilegeTypeWildcard:
		return updateWildcardPrivilege(ctx, nexusClient, name, privCR)
	default:
		return errors.New(errUnknownType)
	}
}

// updateAppPrivilege updates an application privilege.
func updateAppPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
	return nexusClient.Security().UpdatePrivilegeApplication(ctx, name, buildApplicationPrivilege(privCR))
}

// updateRepoViewPrivilege updates a repository-view privilege.
func updateRepoViewPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
	fields := extractRepoFields(privCR)
	privObj := security.PrivilegeRepositoryView{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryViewActions(privCR.Spec.ForProvider.Actions),
	}

	return nexusClient.Security().UpdatePrivilegeRepositoryView(ctx, name, privObj)
}

// updateRepoAdminPrivilege updates a repository-admin privilege.
func updateRepoAdminPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
	fields := extractRepoFields(privCR)
	privObj := security.PrivilegeRepositoryAdmin{
		Name:        fields.name,
		Description: fields.description,
		Format:      fields.format,
		Repository:  fields.repository,
		Actions:     toRepositoryAdminActions(privCR.Spec.ForProvider.Actions),
	}

	return nexusClient.Security().UpdatePrivilegeRepositoryAdmin(ctx, name, privObj)
}

// updateRepoContentSelectorPrivilege updates a repo-content-selector privilege.
func updateRepoContentSelectorPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
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

	return nexusClient.Security().UpdatePrivilegeRepositoryContentSelector(ctx, name, privObj)
}

// updateScriptPrivilege updates a script privilege.
func updateScriptPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
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

	return nexusClient.Security().UpdatePrivilegeScript(ctx, name, privObj)
}

// updateWildcardPrivilege updates a wildcard privilege.
func updateWildcardPrivilege(ctx context.Context, nexusClient nexus.Client, name string, privCR *v1alpha1.Privilege) error {
	privObj := security.PrivilegeWildcard{
		Name: privCR.Spec.ForProvider.Name,
	}
	if privCR.Spec.ForProvider.Description != nil {
		privObj.Description = *privCR.Spec.ForProvider.Description
	}

	if privCR.Spec.ForProvider.Pattern != nil {
		privObj.Pattern = *privCR.Spec.ForProvider.Pattern
	}

	return nexusClient.Security().UpdatePrivilegeWildcard(ctx, name, privObj)
}

// isPrivilegeUpToDate checks if a Privilege is up to date.
func isPrivilegeUpToDate(privCR *v1alpha1.Privilege, priv *security.Privilege) bool {
	if privCR.Spec.ForProvider.Description != nil && *privCR.Spec.ForProvider.Description != priv.Description {
		return false
	}

	if !stringSlicesEqual(privCR.Spec.ForProvider.Actions, priv.Actions) {
		return false
	}

	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for idx := range left {
		if left[idx] != right[idx] {
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
