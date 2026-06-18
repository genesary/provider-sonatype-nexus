// Package anonymousaccess contains the controller for AnonymousAccess
// resources.
package anonymousaccess

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
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
	// errNotAnonymousAccess is returned when the managed resource is not
	// an AnonymousAccess.
	errNotAnonymousAccess = "managed resource is not an AnonymousAccess custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetAnonymous is returned when retrieving anonymous access settings fails.
	errGetAnonymous = "cannot get anonymous access settings from Nexus"
	// errUpdateAnonymous is returned when updating anonymous access settings fails.
	errUpdateAnonymous = "cannot update anonymous access settings in Nexus"
)

// Setup creates a controller for AnonymousAccess resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.AnonymousAccessGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.AnonymousAccessGroupVersionKind),
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
		For(&v1alpha1.AnonymousAccess{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the AnonymousAccess controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isAnonymousAccess := managedRes.(*v1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return nil, errors.New(errNotAnonymousAccess)
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

// Observe checks if the AnonymousAccess resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	anonAccess, isAnonymousAccess := managedRes.(*v1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalObservation{}, errors.New(errNotAnonymousAccess)
	}

	// AnonymousAccess is a singleton in Nexus (cannot be truly deleted).
	// When the CR is being deleted, report the resource as absent so the
	// managed reconciler can remove the finalizer and complete deletion.
	if anonAccess.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	settings, err := e.client.Security().GetAnonymousAccess(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAnonymous)
	}

	anonAccess.SetConditions(v1alpha1.Available())

	upToDate := isAnonymousAccessUpToDate(anonAccess, settings)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new AnonymousAccess resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	anonAccess, isAnonymousAccess := managedRes.(*v1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalCreation{}, errors.New(errNotAnonymousAccess)
	}

	settings := generateAnonymousAccessSettings(anonAccess)

	err := e.client.Security().UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing AnonymousAccess resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	anonAccess, isAnonymousAccess := managedRes.(*v1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalUpdate{}, errors.New(errNotAnonymousAccess)
	}

	settings := generateAnonymousAccessSettings(anonAccess)

	err := e.client.Security().UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing AnonymousAccess resource.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	// AnonymousAccess is a singleton; we don't delete it
	// Optionally could disable anonymous access on delete
	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateAnonymousAccessSettings generates settings from the CR spec.
func generateAnonymousAccessSettings(anonAccess *v1alpha1.AnonymousAccess) security.AnonymousAccessSettings {
	return security.AnonymousAccessSettings{
		Enabled:   anonAccess.Spec.ForProvider.Enabled,
		UserID:    anonAccess.Spec.ForProvider.UserID,
		RealmName: anonAccess.Spec.ForProvider.RealmName,
	}
}

// isAnonymousAccessUpToDate checks if AnonymousAccess settings are up to date.
func isAnonymousAccessUpToDate(
	anonAccess *v1alpha1.AnonymousAccess,
	settings *security.AnonymousAccessSettings,
) bool {
	if anonAccess.Spec.ForProvider.Enabled != settings.Enabled {
		return false
	}

	if anonAccess.Spec.ForProvider.UserID != settings.UserID {
		return false
	}

	if anonAccess.Spec.ForProvider.RealmName != settings.RealmName {
		return false
	}

	return true
}
