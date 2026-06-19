// Package cleanuppolicy contains the controller for CleanupPolicy resources.
package cleanuppolicy

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
	// errNotCleanupPolicy means the managed resource is not a CleanupPolicy.
	errNotCleanupPolicy = "managed resource is not a CleanupPolicy custom resource"
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
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(contentv1alpha1.CleanupPolicyGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(contentv1alpha1.CleanupPolicyGroupVersionKind),
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
		For(&contentv1alpha1.CleanupPolicy{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isCleanupPolicy := managedRes.(*contentv1alpha1.CleanupPolicy)
	if !isCleanupPolicy {
		return nil, errors.New(errNotCleanupPolicy)
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

	cleanupClient, err := contentclient.NewCleanupPolicyClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: cleanupClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client contentclient.CleanupPolicyClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	cleanupPolicy, isCleanupPolicy := managedRes.(*contentv1alpha1.CleanupPolicy)
	if !isCleanupPolicy {
		return managed.ExternalObservation{}, errors.New(errNotCleanupPolicy)
	}

	name := meta.GetExternalName(cleanupPolicy)
	if name == "" {
		name = cleanupPolicy.Spec.ForProvider.Name
	}

	observed, err := e.client.GetCleanupPolicy(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
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
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	cleanupPolicy, isCleanupPolicy := managedRes.(*contentv1alpha1.CleanupPolicy)
	if !isCleanupPolicy {
		return managed.ExternalCreation{}, errors.New(errNotCleanupPolicy)
	}

	err := e.client.CreateCleanupPolicy(ctx, contentclient.GenerateCleanupPolicy(cleanupPolicy))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateCleanupPolicy)
	}

	meta.SetExternalName(cleanupPolicy, cleanupPolicy.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	cleanupPolicy, isCleanupPolicy := managedRes.(*contentv1alpha1.CleanupPolicy)
	if !isCleanupPolicy {
		return managed.ExternalUpdate{}, errors.New(errNotCleanupPolicy)
	}

	err := e.client.UpdateCleanupPolicy(ctx, contentclient.GenerateCleanupPolicy(cleanupPolicy))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateCleanupPolicy)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	cleanupPolicy, isCleanupPolicy := managedRes.(*contentv1alpha1.CleanupPolicy)
	if !isCleanupPolicy {
		return managed.ExternalDelete{}, errors.New(errNotCleanupPolicy)
	}

	name := meta.GetExternalName(cleanupPolicy)
	if name == "" {
		name = cleanupPolicy.Spec.ForProvider.Name
	}

	err := e.client.DeleteCleanupPolicy(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
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
