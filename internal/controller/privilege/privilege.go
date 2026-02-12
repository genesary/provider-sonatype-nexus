// Package privilege contains the controller for Privilege resources.
package privilege

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotPrivilege    = "managed resource is not a Privilege custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errNewClient       = "cannot create new Nexus client"
	errGetPrivilege    = "cannot get privilege from Nexus"
	errCreatePrivilege = "cannot create privilege in Nexus"
	errUpdatePrivilege = "cannot update privilege in Nexus"
	errDeletePrivilege = "cannot delete privilege from Nexus"
	errUnknownType     = "unknown privilege type"
)

// Setup adds a controller that reconciles Privilege managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.PrivilegeGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.PrivilegeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Privilege{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Privilege)
	if !ok {
		return nil, errors.New(errNotPrivilege)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nc}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
}

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Privilege)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
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

	cr.SetConditions(v1alpha1.Available())

	upToDate := isPrivilegeUpToDate(cr, priv)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Privilege)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPrivilege)
	}

	if err := createPrivilege(ctx, e.client, cr); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePrivilege)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Privilege)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	if err := updatePrivilege(ctx, e.client, name, cr); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePrivilege)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Privilege)
	if !ok {
		return errors.New(errNotPrivilege)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	if err := e.client.Security().DeletePrivilege(ctx, name); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeletePrivilege)
	}

	return nil
}

// toApplicationActions converts string slice to application action types.
func toApplicationActions(actions []string) []security.SecurityPrivilegeApplicationActions {
	result := make([]security.SecurityPrivilegeApplicationActions, len(actions))
	for i, a := range actions {
		result[i] = security.SecurityPrivilegeApplicationActions(a)
	}
	return result
}

// toRepositoryViewActions converts string slice to repository view action types.
func toRepositoryViewActions(actions []string) []security.SecurityPrivilegeRepositoryViewActions {
	result := make([]security.SecurityPrivilegeRepositoryViewActions, len(actions))
	for i, a := range actions {
		result[i] = security.SecurityPrivilegeRepositoryViewActions(a)
	}
	return result
}

// toRepositoryAdminActions converts string slice to repository admin action types.
func toRepositoryAdminActions(actions []string) []security.SecurityPrivilegeRepositoryAdminActions {
	result := make([]security.SecurityPrivilegeRepositoryAdminActions, len(actions))
	for i, a := range actions {
		result[i] = security.SecurityPrivilegeRepositoryAdminActions(a)
	}
	return result
}

// toRepositoryContentSelectorActions converts string slice to repository content selector action types.
func toRepositoryContentSelectorActions(actions []string) []security.SecurityPrivilegeRepositoryContentSelectorActions {
	result := make([]security.SecurityPrivilegeRepositoryContentSelectorActions, len(actions))
	for i, a := range actions {
		result[i] = security.SecurityPrivilegeRepositoryContentSelectorActions(a)
	}
	return result
}

// toScriptActions converts string slice to script action types.
func toScriptActions(actions []string) []security.SecurityPrivilegeScriptActions {
	result := make([]security.SecurityPrivilegeScriptActions, len(actions))
	for i, a := range actions {
		result[i] = security.SecurityPrivilegeScriptActions(a)
	}
	return result
}

// createPrivilege creates a privilege based on its type.
func createPrivilege(ctx context.Context, client nexus.Client, cr *v1alpha1.Privilege) error {
	switch cr.Spec.ForProvider.Type {
	case "application":
		p := security.PrivilegeApplication{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toApplicationActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Domain != nil {
			p.Domain = *cr.Spec.ForProvider.Domain
		}
		return client.Security().CreatePrivilegeApplication(ctx, p)

	case "repository-view":
		p := security.PrivilegeRepositoryView{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryViewActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		return client.Security().CreatePrivilegeRepositoryView(ctx, p)

	case "repository-admin":
		p := security.PrivilegeRepositoryAdmin{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryAdminActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		return client.Security().CreatePrivilegeRepositoryAdmin(ctx, p)

	case "repository-content-selector":
		p := security.PrivilegeRepositoryContentSelector{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryContentSelectorActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		if cr.Spec.ForProvider.ContentSelector != nil {
			p.ContentSelector = *cr.Spec.ForProvider.ContentSelector
		}
		return client.Security().CreatePrivilegeRepositoryContentSelector(ctx, p)

	case "script":
		p := security.PrivilegeScript{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toScriptActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.ScriptName != nil {
			p.ScriptName = *cr.Spec.ForProvider.ScriptName
		}
		return client.Security().CreatePrivilegeScript(ctx, p)

	case "wildcard":
		p := security.PrivilegeWildcard{
			Name: cr.Spec.ForProvider.Name,
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Pattern != nil {
			p.Pattern = *cr.Spec.ForProvider.Pattern
		}
		return client.Security().CreatePrivilegeWildcard(ctx, p)

	default:
		return errors.New(errUnknownType)
	}
}

// updatePrivilege updates a privilege based on its type.
func updatePrivilege(ctx context.Context, client nexus.Client, name string, cr *v1alpha1.Privilege) error {
	switch cr.Spec.ForProvider.Type {
	case "application":
		p := security.PrivilegeApplication{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toApplicationActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Domain != nil {
			p.Domain = *cr.Spec.ForProvider.Domain
		}
		return client.Security().UpdatePrivilegeApplication(ctx, name, p)

	case "repository-view":
		p := security.PrivilegeRepositoryView{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryViewActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		return client.Security().UpdatePrivilegeRepositoryView(ctx, name, p)

	case "repository-admin":
		p := security.PrivilegeRepositoryAdmin{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryAdminActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		return client.Security().UpdatePrivilegeRepositoryAdmin(ctx, name, p)

	case "repository-content-selector":
		p := security.PrivilegeRepositoryContentSelector{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toRepositoryContentSelectorActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Format != nil {
			p.Format = *cr.Spec.ForProvider.Format
		}
		if cr.Spec.ForProvider.Repository != nil {
			p.Repository = *cr.Spec.ForProvider.Repository
		}
		if cr.Spec.ForProvider.ContentSelector != nil {
			p.ContentSelector = *cr.Spec.ForProvider.ContentSelector
		}
		return client.Security().UpdatePrivilegeRepositoryContentSelector(ctx, name, p)

	case "script":
		p := security.PrivilegeScript{
			Name:    cr.Spec.ForProvider.Name,
			Actions: toScriptActions(cr.Spec.ForProvider.Actions),
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.ScriptName != nil {
			p.ScriptName = *cr.Spec.ForProvider.ScriptName
		}
		return client.Security().UpdatePrivilegeScript(ctx, name, p)

	case "wildcard":
		p := security.PrivilegeWildcard{
			Name: cr.Spec.ForProvider.Name,
		}
		if cr.Spec.ForProvider.Description != nil {
			p.Description = *cr.Spec.ForProvider.Description
		}
		if cr.Spec.ForProvider.Pattern != nil {
			p.Pattern = *cr.Spec.ForProvider.Pattern
		}
		return client.Security().UpdatePrivilegeWildcard(ctx, name, p)

	default:
		return errors.New(errUnknownType)
	}
}

// isPrivilegeUpToDate checks if a Privilege is up to date.
func isPrivilegeUpToDate(cr *v1alpha1.Privilege, priv *security.Privilege) bool {
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != priv.Description {
		return false
	}
	if !stringSlicesEqual(cr.Spec.ForProvider.Actions, priv.Actions) {
		return false
	}
	// Additional type-specific checks could be added here
	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
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
