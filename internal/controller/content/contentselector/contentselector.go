// Package contentselector manages ContentSelector resources.
package contentselector

import (
	"context"

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
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(contentv1alpha1.ContentSelectorGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*contentv1alpha1.ContentSelector](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *contentv1alpha1.ContentSelector) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(contentv1alpha1.ContentSelectorGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &contentv1alpha1.ContentSelectorList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&contentv1alpha1.ContentSelector{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, contentSelector *contentv1alpha1.ContentSelector) (managed.TypedExternalClient[*contentv1alpha1.ContentSelector], error) {
	err := c.usage.Track(ctx, contentSelector)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, contentSelector)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	csClient, err := contentclient.NewContentSelectorClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: csClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client contentclient.ContentSelectorClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, contentSelector *contentv1alpha1.ContentSelector) (managed.ExternalObservation, error) {
	name := helpers.ExternalNameOr(contentSelector, contentSelector.Spec.ForProvider.Name)

	observed, err := e.client.Get(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
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
func (e *external) Create(ctx context.Context, contentSelector *contentv1alpha1.ContentSelector) (managed.ExternalCreation, error) {
	err := e.client.Create(ctx, contentclient.GenerateContentSelector(contentSelector))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateContentSelector)
	}

	meta.SetExternalName(contentSelector, contentSelector.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, contentSelector *contentv1alpha1.ContentSelector) (managed.ExternalUpdate, error) {
	name := helpers.ExternalNameOr(contentSelector, contentSelector.Spec.ForProvider.Name)

	err := e.client.Update(ctx, name, contentclient.GenerateContentSelectorUpdate(contentSelector))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateContentSelector)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, contentSelector *contentv1alpha1.ContentSelector) (managed.ExternalDelete, error) {
	name := helpers.ExternalNameOr(contentSelector, contentSelector.Spec.ForProvider.Name)

	err := e.client.Delete(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
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
