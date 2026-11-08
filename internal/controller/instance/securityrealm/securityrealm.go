// Package securityrealm manages SecurityRealm resources.
package securityrealm

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
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
	// errGetRealms is returned when retrieving active realms fails.
	errGetRealms = "cannot get active realms from Nexus"
	// errGetAvailableRealms is returned when retrieving available realms fails.
	errGetAvailableRealms = "cannot get available realms from Nexus"
	// errActivateRealms is returned when activating realms fails.
	errActivateRealms = "cannot activate realms in Nexus"
)

// Setup adds a controller that reconciles SecurityRealm resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.SecurityRealmGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.SecurityRealm](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.SecurityRealmGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.SecurityRealmList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.SecurityRealm{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, securityRealm *instancev1alpha1.SecurityRealm) (managed.TypedExternalClient[*instancev1alpha1.SecurityRealm], error) {
	err := c.usage.Track(ctx, securityRealm)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, securityRealm)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	realmClient, err := instanceclient.NewSecurityRealmClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: realmClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.SecurityRealmClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, securityRealm *instancev1alpha1.SecurityRealm) (managed.ExternalObservation, error) {
	// SecurityRealm is a singleton; report absent when being deleted so the
	// finalizer can be cleared.
	if securityRealm.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	activeRealms, err := e.client.ListActive(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRealms)
	}

	availableRealms, err := e.client.ListAvailable(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetAvailableRealms)
	}

	securityRealm.Status.AtProvider = instanceclient.GenerateSecurityRealmObservation(availableRealms, activeRealms)
	securityRealm.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsSecurityRealmUpToDate(securityRealm),
	}, nil
}

// Create activates the desired security realms.
func (e *external) Create(ctx context.Context, securityRealm *instancev1alpha1.SecurityRealm) (managed.ExternalCreation, error) {
	err := e.client.Activate(ctx, securityRealm.Spec.ForProvider.ActiveRealms)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalCreation{}, nil
}

// Update activates the desired security realms.
func (e *external) Update(ctx context.Context, securityRealm *instancev1alpha1.SecurityRealm) (managed.ExternalUpdate, error) {
	err := e.client.Activate(ctx, securityRealm.Spec.ForProvider.ActiveRealms)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errActivateRealms)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete is a no-op; SecurityRealm is a singleton and cannot be deleted.
func (e *external) Delete(_ context.Context, _ *instancev1alpha1.SecurityRealm) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
