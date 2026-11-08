// Package routingrule contains the controller for RoutingRule resources.
package routingrule

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
	// errTrackPCUsage is used when tracking the ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is used when getting the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is used when creating a new Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetRoutingRule is used when getting the routing rule from Nexus fails.
	errGetRoutingRule = "cannot get routing rule from Nexus"
	// errCreateRoutingRule is used when creating the routing rule in Nexus fails.
	errCreateRoutingRule = "cannot create routing rule in Nexus"
	// errUpdateRoutingRule is used when updating the routing rule in Nexus fails.
	errUpdateRoutingRule = "cannot update routing rule in Nexus"
	// errDeleteRoutingRule is used when deleting the routing rule from Nexus fails.
	errDeleteRoutingRule = "cannot delete routing rule from Nexus"
)

// Setup adds a controller that reconciles RoutingRule managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(contentv1alpha1.RoutingRuleGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*contentv1alpha1.RoutingRule](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *contentv1alpha1.RoutingRule) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(contentv1alpha1.RoutingRuleGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &contentv1alpha1.RoutingRuleList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&contentv1alpha1.RoutingRule{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, routingRule *contentv1alpha1.RoutingRule) (managed.TypedExternalClient[*contentv1alpha1.RoutingRule], error) {
	err := c.usage.Track(ctx, routingRule)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, routingRule)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	rrClient, err := contentclient.NewRoutingRuleClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: rrClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client contentclient.RoutingRuleClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, routingRule *contentv1alpha1.RoutingRule) (managed.ExternalObservation, error) {
	name := helpers.ExternalNameOr(routingRule, routingRule.Spec.ForProvider.Name)

	observed, err := e.client.Get(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetRoutingRule)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	routingRule.Status.AtProvider = contentclient.GenerateRoutingRuleObservation(observed)
	routingRule.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: contentclient.IsRoutingRuleUpToDate(&routingRule.Spec.ForProvider, &routingRule.Status.AtProvider),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, routingRule *contentv1alpha1.RoutingRule) (managed.ExternalCreation, error) {
	err := e.client.Create(ctx, contentclient.GenerateRoutingRule(routingRule))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRoutingRule)
	}

	meta.SetExternalName(routingRule, routingRule.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, routingRule *contentv1alpha1.RoutingRule) (managed.ExternalUpdate, error) {
	err := e.client.Update(ctx, contentclient.GenerateRoutingRule(routingRule))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRoutingRule)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, routingRule *contentv1alpha1.RoutingRule) (managed.ExternalDelete, error) {
	name := helpers.ExternalNameOr(routingRule, routingRule.Spec.ForProvider.Name)

	err := e.client.Delete(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRoutingRule)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
