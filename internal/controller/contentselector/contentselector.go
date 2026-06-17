// Package contentselector contains the controller for ContentSelector
// resources.
package contentselector

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
	// errNotContentSelector is returned when the managed resource is not
	// a ContentSelector.
	errNotContentSelector = "managed resource is not a ContentSelector custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetContentSelector is returned when retrieving a ContentSelector fails.
	errGetContentSelector = "cannot get content selector from Nexus"
	// errCreateContentSelector is returned when creating a ContentSelector fails.
	errCreateContentSelector = "cannot create content selector in Nexus"
	// errUpdateContentSelector is returned when updating a ContentSelector fails.
	errUpdateContentSelector = "cannot update content selector in Nexus"
	// errDeleteContentSelector is returned when deleting a ContentSelector fails.
	errDeleteContentSelector = "cannot delete content selector from Nexus"
)

// Setup creates a controller for ContentSelector resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.ContentSelectorGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ContentSelectorGroupVersionKind),
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
		For(&v1alpha1.ContentSelector{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the ContentSelector controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isContentSelector := managedRes.(*v1alpha1.ContentSelector)
	if !isContentSelector {
		return nil, errors.New(errNotContentSelector)
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

// Observe checks if the ContentSelector resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	contentSel, isContentSelector := managedRes.(*v1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalObservation{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSel)
	if name == "" {
		name = contentSel.Spec.ForProvider.Name
	}

	csResult, err := e.client.Security().GetContentSelector(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetContentSelector)
	}

	if csResult == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	contentSel.SetConditions(v1alpha1.Available())

	upToDate := isContentSelectorUpToDate(contentSel, csResult)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new ContentSelector resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	contentSel, isContentSelector := managedRes.(*v1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalCreation{}, errors.New(errNotContentSelector)
	}

	csData := generateContentSelector(contentSel)

	err := e.client.Security().CreateContentSelector(ctx, csData)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateContentSelector)
	}

	meta.SetExternalName(contentSel, contentSel.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing ContentSelector resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	contentSel, isContentSelector := managedRes.(*v1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalUpdate{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSel)
	if name == "" {
		name = contentSel.Spec.ForProvider.Name
	}

	csData := generateContentSelector(contentSel)

	err := e.client.Security().UpdateContentSelector(ctx, name, csData)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateContentSelector)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing ContentSelector resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	contentSel, isContentSelector := managedRes.(*v1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalDelete{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSel)
	if name == "" {
		name = contentSel.Spec.ForProvider.Name
	}

	err := e.client.Security().DeleteContentSelector(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteContentSelector)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateContentSelector generates a ContentSelector from the CR spec.
func generateContentSelector(contentSel *v1alpha1.ContentSelector) security.ContentSelector {
	csData := security.ContentSelector{
		Name:       contentSel.Spec.ForProvider.Name,
		Expression: contentSel.Spec.ForProvider.Expression,
	}

	if contentSel.Spec.ForProvider.Description != nil {
		csData.Description = *contentSel.Spec.ForProvider.Description
	}

	return csData
}

// isContentSelectorUpToDate checks if a ContentSelector is up to date.
func isContentSelectorUpToDate(contentSel *v1alpha1.ContentSelector, csData *security.ContentSelector) bool {
	if contentSel.Spec.ForProvider.Expression != csData.Expression {
		return false
	}

	if contentSel.Spec.ForProvider.Description != nil &&
		*contentSel.Spec.ForProvider.Description != csData.Description {
		return false
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
