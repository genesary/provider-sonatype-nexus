// Package anonymousaccess manages AnonymousAccess resources.
package anonymousaccess

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotAnonymousAccess means the managed resource is not an AnonymousAccess.
	errNotAnonymousAccess = "managed resource is not an AnonymousAccess custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetAnonymous is returned when retrieving anonymous access fails.
	errGetAnonymous = "cannot get anonymous access settings from Nexus"
	// errUpdateAnonymous is returned when updating anonymous access fails.
	errUpdateAnonymous = "cannot update anonymous access settings in Nexus"
)

// Setup adds a controller that reconciles AnonymousAccess resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.AnonymousAccessGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.AnonymousAccessGroupVersionKind),
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
		For(&iamv1alpha1.AnonymousAccess{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isAnonymousAccess := managedRes.(*iamv1alpha1.AnonymousAccess)
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

	anonClient, err := iamclient.NewAnonymousAccessClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: anonClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.AnonymousAccessClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	anonAccess, isAnonymousAccess := managedRes.(*iamv1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalObservation{}, errors.New(errNotAnonymousAccess)
	}

	// AnonymousAccess is a singleton; report absent when being deleted so the
	// finalizer can be cleared.
	if anonAccess.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	settings, err := e.client.GetAnonymousAccess(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAnonymous)
	}

	anonAccess.Status.AtProvider = iamclient.GenerateAnonymousAccessObservation(settings)
	anonAccess.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsAnonymousAccessUpToDate(anonAccess),
	}, nil
}

// Create activates the desired anonymous access settings.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	anonAccess, isAnonymousAccess := managedRes.(*iamv1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalCreation{}, errors.New(errNotAnonymousAccess)
	}

	settings := iamclient.GenerateAnonymousAccessSettings(anonAccess)

	err := e.client.UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the desired anonymous access settings.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	anonAccess, isAnonymousAccess := managedRes.(*iamv1alpha1.AnonymousAccess)
	if !isAnonymousAccess {
		return managed.ExternalUpdate{}, errors.New(errNotAnonymousAccess)
	}

	settings := iamclient.GenerateAnonymousAccessSettings(anonAccess)

	err := e.client.UpdateAnonymousAccess(ctx, settings)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateAnonymous)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete is a no-op; AnonymousAccess is a singleton and cannot be deleted.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
