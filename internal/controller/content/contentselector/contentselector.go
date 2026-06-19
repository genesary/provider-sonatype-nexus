// Package contentselector manages ContentSelector resources.
package contentselector

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errNotContentSelector means the managed resource is not a ContentSelector.
	errNotContentSelector = "managed resource is not a ContentSelector custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetContentSelector is returned when retrieving a ContentSelector fails.
	errGetContentSelector = "cannot get content selector from Nexus"
	// errCreateContentSelector means creating the content selector failed.
	errCreateContentSelector = "cannot create content selector in Nexus"
	// errUpdateContentSelector means updating the content selector failed.
	errUpdateContentSelector = "cannot update content selector in Nexus"
	// errDeleteContentSelector means deleting the content selector failed.
	errDeleteContentSelector = "cannot delete content selector from Nexus"
)

// Setup adds a controller that reconciles ContentSelector managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(contentv1alpha1.ContentSelectorGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(contentv1alpha1.ContentSelectorGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))) //nolint:deprecated // no replacement yet

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&contentv1alpha1.ContentSelector{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isContentSelector := managedRes.(*contentv1alpha1.ContentSelector)
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

	csClient, err := contentclient.NewContentSelectorClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: csClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client contentclient.ContentSelectorClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	contentSelector, isContentSelector := managedRes.(*contentv1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalObservation{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSelector)
	if name == "" {
		name = contentSelector.Spec.ForProvider.Name
	}

	observed, err := e.client.GetContentSelector(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetContentSelector)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	contentSelector.Status.AtProvider = contentclient.GenerateContentSelectorObservation(observed)
	contentSelector.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: contentclient.IsContentSelectorUpToDate(contentSelector, observed),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	contentSelector, isContentSelector := managedRes.(*contentv1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalCreation{}, errors.New(errNotContentSelector)
	}

	err := e.client.CreateContentSelector(ctx, contentclient.GenerateContentSelector(contentSelector))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateContentSelector)
	}

	meta.SetExternalName(contentSelector, contentSelector.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	contentSelector, isContentSelector := managedRes.(*contentv1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalUpdate{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSelector)
	if name == "" {
		name = contentSelector.Spec.ForProvider.Name
	}

	err := e.client.UpdateContentSelector(ctx, name, contentclient.GenerateContentSelector(contentSelector))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateContentSelector)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	contentSelector, isContentSelector := managedRes.(*contentv1alpha1.ContentSelector)
	if !isContentSelector {
		return managed.ExternalDelete{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(contentSelector)
	if name == "" {
		name = contentSelector.Spec.ForProvider.Name
	}

	err := e.client.DeleteContentSelector(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteContentSelector)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
