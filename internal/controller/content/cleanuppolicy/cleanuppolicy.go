// Package cleanuppolicy contains the controller for CleanupPolicy resources.
package cleanuppolicy

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
	// errGetCleanupPolicy is returned when retrieving a CleanupPolicy fails.
	errGetCleanupPolicy = "cannot get cleanup policy from Nexus"
	// errCreateCleanupPolicy is returned when creating a CleanupPolicy fails.
	errCreateCleanupPolicy = "cannot create cleanup policy in Nexus"
	// errUpdateCleanupPolicy is returned when updating a CleanupPolicy fails.
	errUpdateCleanupPolicy = "cannot update cleanup policy in Nexus"
	// errDeleteCleanupPolicy is returned when deleting a CleanupPolicy fails.
	errDeleteCleanupPolicy = "cannot delete cleanup policy from Nexus"
)

// Setup adds a controller that reconciles CleanupPolicy managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(contentv1alpha1.CleanupPolicyGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*contentv1alpha1.CleanupPolicy](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *contentv1alpha1.CleanupPolicy) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(contentv1alpha1.CleanupPolicyGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &contentv1alpha1.CleanupPolicyList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&contentv1alpha1.CleanupPolicy{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, cleanupPolicy *contentv1alpha1.CleanupPolicy) (managed.TypedExternalClient[*contentv1alpha1.CleanupPolicy], error) {
	err := c.usage.Track(ctx, cleanupPolicy)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, cleanupPolicy)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cleanupClient, err := contentclient.NewCleanupPolicyClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: cleanupClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client contentclient.CleanupPolicyClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, cleanupPolicy *contentv1alpha1.CleanupPolicy) (managed.ExternalObservation, error) {
	name := helpers.ExternalNameOr(cleanupPolicy, cleanupPolicy.Spec.ForProvider.Name)

	observed, err := e.client.Get(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetCleanupPolicy)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cleanupPolicy.Status.AtProvider = contentclient.GenerateCleanupPolicyObservation(observed)
	cleanupPolicy.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: contentclient.IsCleanupPolicyUpToDate(cleanupPolicy, observed),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, cleanupPolicy *contentv1alpha1.CleanupPolicy) (managed.ExternalCreation, error) {
	err := e.client.Create(ctx, contentclient.GenerateCleanupPolicy(cleanupPolicy))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCleanupPolicy)
	}

	meta.SetExternalName(cleanupPolicy, cleanupPolicy.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, cleanupPolicy *contentv1alpha1.CleanupPolicy) (managed.ExternalUpdate, error) {
	err := e.client.Update(ctx, contentclient.GenerateCleanupPolicy(cleanupPolicy))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCleanupPolicy)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, cleanupPolicy *contentv1alpha1.CleanupPolicy) (managed.ExternalDelete, error) {
	name := helpers.ExternalNameOr(cleanupPolicy, cleanupPolicy.Spec.ForProvider.Name)

	err := e.client.Delete(ctx, name)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteCleanupPolicy)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
