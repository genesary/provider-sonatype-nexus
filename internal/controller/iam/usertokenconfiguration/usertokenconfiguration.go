// Package usertokenconfiguration manages UserTokenConfiguration resources.
package usertokenconfiguration

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
	// errNotUserTokenConfig is returned for wrong managed resource type.
	errNotUserTokenConfig = "managed resource is not a UserTokenConfiguration custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetUserTokenConfig is returned when retrieving user token config fails.
	errGetUserTokenConfig = "cannot get user token configuration from Nexus"
	// errUpdateUserTokenConfig is returned when updating user token config fails.
	errUpdateUserTokenConfig = "cannot update user token configuration in Nexus"
)

// Setup adds a controller that reconciles UserTokenConfiguration resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.UserTokenConfigurationGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.UserTokenConfigurationGroupVersionKind),
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
		For(&iamv1alpha1.UserTokenConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isUserTokenConfig := managedRes.(*iamv1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return nil, errors.New(errNotUserTokenConfig)
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

	tokenClient, err := iamclient.NewUserTokenConfigurationClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: tokenClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.UserTokenConfigurationClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*iamv1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalObservation{}, errors.New(errNotUserTokenConfig)
	}

	// UserTokenConfiguration is a singleton; report absent when being deleted
	// so the finalizer can be cleared.
	if userTokenCfg.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, err := e.client.GetUserTokenConfiguration(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUserTokenConfig)
	}

	userTokenCfg.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsUserTokenConfigUpToDate(userTokenCfg, config),
	}, nil
}

// Create applies the desired user token configuration.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*iamv1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalCreation{}, errors.New(errNotUserTokenConfig)
	}

	config := iamclient.GenerateUserTokenConfiguration(userTokenCfg)

	err := e.client.UpdateUserTokenConfiguration(ctx, config)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the desired user token configuration.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*iamv1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalUpdate{}, errors.New(errNotUserTokenConfig)
	}

	config := iamclient.GenerateUserTokenConfiguration(userTokenCfg)

	err := e.client.UpdateUserTokenConfiguration(ctx, config)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete is a no-op; UserTokenConfiguration is a singleton.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
