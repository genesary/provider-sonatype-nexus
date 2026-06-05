// Package securityrealm contains the controller for SecurityRealm resources.
package securityrealm

import (
	"context"
	"reflect"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotSecurityRealm is returned when the managed resource is not
	// a SecurityRealm.
	errNotSecurityRealm = "managed resource is not a SecurityRealm custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetCreds is returned when retrieving credentials fails.
	errGetCreds = "cannot get credentials"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetRealms is returned when retrieving active realms fails.
	errGetRealms = "cannot get active realms from Nexus"
	// errActivateRealms is returned when activating realms fails.
	errActivateRealms = "cannot activate realms in Nexus"
)

// Setup creates a controller for SecurityRealm resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.SecurityRealmGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SecurityRealmGroupVersionKind),
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
		For(&v1alpha1.SecurityRealm{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the SecurityRealm controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isSecurityRealm := managedRes.(*v1alpha1.SecurityRealm)
	if !isSecurityRealm {
		return nil, errors.New(errNotSecurityRealm)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	provConfig := &v1alpha1.ProviderConfig{}

	err = c.kube.Get(ctx, client.ObjectKey{Name: modernMG.GetProviderConfigReference().Name}, provConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, provConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
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

// Observe checks if the SecurityRealm resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	secRealm, isSecurityRealm := managedRes.(*v1alpha1.SecurityRealm)
	if !isSecurityRealm {
		return managed.ExternalObservation{}, errors.New(errNotSecurityRealm)
	}

	// SecurityRealm is a singleton in Nexus (cannot be truly deleted).
	// When the CR is being deleted, report the resource as absent so the
	// managed reconciler can remove the finalizer and complete deletion.
	if secRealm.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	activeRealms, err := e.client.Security().ListActiveRealms(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRealms)
	}

	secRealm.SetConditions(v1alpha1.Available())

	// Update observation with available realms
	availableRealms, _ := e.client.Security().ListAvailableRealms(ctx)
	if availableRealms != nil {
		realmInfos := make([]v1alpha1.RealmInfo, len(availableRealms))
		for idx, realmItem := range availableRealms {
			realmInfos[idx] = v1alpha1.RealmInfo{
				ID:   realmItem.ID,
				Name: realmItem.Name,
			}
		}

		secRealm.Status.AtProvider.AvailableRealms = realmInfos
	}

	upToDate := reflect.DeepEqual(secRealm.Spec.ForProvider.ActiveRealms, activeRealms)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new SecurityRealm resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	secRealm, isSecurityRealm := managedRes.(*v1alpha1.SecurityRealm)
	if !isSecurityRealm {
		return managed.ExternalCreation{}, errors.New(errNotSecurityRealm)
	}

	err := e.client.Security().ActivateRealms(ctx, secRealm.Spec.ForProvider.ActiveRealms)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing SecurityRealm resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	secRealm, isSecurityRealm := managedRes.(*v1alpha1.SecurityRealm)
	if !isSecurityRealm {
		return managed.ExternalUpdate{}, errors.New(errNotSecurityRealm)
	}

	err := e.client.Security().ActivateRealms(ctx, secRealm.Spec.ForProvider.ActiveRealms)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing SecurityRealm resource.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	// SecurityRealm is a singleton; we don't delete it, just leave it as-is
	// In real-world usage, you might want to restore default realms
	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
