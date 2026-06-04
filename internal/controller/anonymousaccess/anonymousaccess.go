// Package anonymousaccess contains the controller for AnonymousAccess resources.
package anonymousaccess

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
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
	errNotAnonymousAccess = "managed resource is not an AnonymousAccess custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Nexus client"
	errGetAnonymous       = "cannot get anonymous access settings from Nexus"
	errUpdateAnonymous    = "cannot update anonymous access settings in Nexus"
)

// Setup adds a controller that reconciles AnonymousAccess managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.AnonymousAccessGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.AnonymousAccessGroupVersionKind),
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
		For(&v1alpha1.AnonymousAccess{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.AnonymousAccess)
	if !ok {
		return nil, errors.New(errNotAnonymousAccess)
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
	cr, ok := mg.(*v1alpha1.AnonymousAccess)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAnonymousAccess)
	}

	// AnonymousAccess is a singleton in Nexus (cannot be truly deleted).
	// When the CR is being deleted, report the resource as absent so the
	// managed reconciler can remove the finalizer and complete deletion.
	if cr.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	settings, err := e.client.Security().GetAnonymousAccess(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAnonymous)
	}

	cr.SetConditions(v1alpha1.Available())

	upToDate := isAnonymousAccessUpToDate(cr, settings)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.AnonymousAccess)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAnonymousAccess)
	}

	settings := generateAnonymousAccessSettings(cr)

	err := e.client.Security().UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.AnonymousAccess)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAnonymousAccess)
	}

	settings := generateAnonymousAccessSettings(cr)

	err := e.client.Security().UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	// AnonymousAccess is a singleton; we don't delete it
	// Optionally could disable anonymous access on delete
	return nil
}

// generateAnonymousAccessSettings generates settings from the CR spec.
func generateAnonymousAccessSettings(cr *v1alpha1.AnonymousAccess) security.AnonymousAccessSettings {
	return security.AnonymousAccessSettings{
		Enabled:   cr.Spec.ForProvider.Enabled,
		UserID:    cr.Spec.ForProvider.UserID,
		RealmName: cr.Spec.ForProvider.RealmName,
	}
}

// isAnonymousAccessUpToDate checks if AnonymousAccess settings are up to date.
func isAnonymousAccessUpToDate(cr *v1alpha1.AnonymousAccess, settings *security.AnonymousAccessSettings) bool {
	if cr.Spec.ForProvider.Enabled != settings.Enabled {
		return false
	}

	if cr.Spec.ForProvider.UserID != settings.UserID {
		return false
	}

	if cr.Spec.ForProvider.RealmName != settings.RealmName {
		return false
	}

	return true
}
