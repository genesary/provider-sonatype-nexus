// Package securityrealm contains the controller for SecurityRealm resources.
package securityrealm

import (
	"context"
	"reflect"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotSecurityRealm = "managed resource is not a SecurityRealm custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create new Nexus client"
	errGetRealms        = "cannot get active realms from Nexus"
	errActivateRealms   = "cannot activate realms in Nexus"
)

// Setup adds a controller that reconciles SecurityRealm managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SecurityRealmGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SecurityRealmGroupVersionKind),
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
		For(&v1alpha1.SecurityRealm{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.SecurityRealm)
	if !ok {
		return nil, errors.New(errNotSecurityRealm)
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
	cr, ok := mg.(*v1alpha1.SecurityRealm)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSecurityRealm)
	}

	activeRealms, err := e.client.Security().ListActiveRealms(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRealms)
	}

	// SecurityRealm is a singleton that always exists (realms are always configured)
	cr.SetConditions(v1alpha1.Available())

	// Update observation with available realms
	availableRealms, _ := e.client.Security().ListAvailableRealms(ctx)
	if availableRealms != nil {
		realmInfos := make([]v1alpha1.RealmInfo, len(availableRealms))
		for i, r := range availableRealms {
			realmInfos[i] = v1alpha1.RealmInfo{
				ID:   r.ID,
				Name: r.Name,
			}
		}
		cr.Status.AtProvider.AvailableRealms = realmInfos
	}

	upToDate := reflect.DeepEqual(cr.Spec.ForProvider.ActiveRealms, activeRealms)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.SecurityRealm)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSecurityRealm)
	}

	if err := e.client.Security().ActivateRealms(ctx, cr.Spec.ForProvider.ActiveRealms); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.SecurityRealm)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSecurityRealm)
	}

	if err := e.client.Security().ActivateRealms(ctx, cr.Spec.ForProvider.ActiveRealms); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	// SecurityRealm is a singleton; we don't delete it, just leave it as-is
	// In real-world usage, you might want to restore default realms
	return nil
}
