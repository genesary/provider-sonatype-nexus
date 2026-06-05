// Package usertokenconfiguration contains the controller for
// UserTokenConfiguration resources.
package usertokenconfiguration

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
	// errNotUserTokenConfig is returned when the managed resource is not
	// a UserTokenConfiguration.
	errNotUserTokenConfig = "managed resource is not a UserTokenConfiguration custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetCreds is returned when retrieving credentials fails.
	errGetCreds = "cannot get credentials"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetUserTokenConfig is returned when retrieving user token configuration
	// fails.
	errGetUserTokenConfig = "cannot get user token configuration from Nexus"
	// errUpdateUserTokenConfig is returned when updating user token configuration
	// fails.
	errUpdateUserTokenConfig = "cannot update user token configuration in Nexus"
)

// Setup creates a controller for UserTokenConfiguration resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserTokenConfigurationGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserTokenConfigurationGroupVersionKind),
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
		For(&v1alpha1.UserTokenConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the UserTokenConfiguration controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isUserTokenConfig := managedRes.(*v1alpha1.UserTokenConfiguration)
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

// Observe checks if the UserTokenConfiguration resource exists and is
// up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*v1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalObservation{}, errors.New(errNotUserTokenConfig)
	}

	// UserTokenConfiguration is a singleton in Nexus (cannot be truly deleted).
	// When the CR is being deleted, report the resource as absent so the
	// managed reconciler can remove the finalizer and complete deletion.
	if userTokenCfg.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, err := e.client.Security().GetUserTokenConfiguration(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUserTokenConfig)
	}

	userTokenCfg.SetConditions(v1alpha1.Available())

	upToDate := isUserTokenConfigUpToDate(userTokenCfg, config)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new UserTokenConfiguration resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*v1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalCreation{}, errors.New(errNotUserTokenConfig)
	}

	config := generateUserTokenConfiguration(userTokenCfg)

	err := e.client.Security().UpdateUserTokenConfiguration(ctx, config)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing UserTokenConfiguration resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	userTokenCfg, isUserTokenConfig := managedRes.(*v1alpha1.UserTokenConfiguration)
	if !isUserTokenConfig {
		return managed.ExternalUpdate{}, errors.New(errNotUserTokenConfig)
	}

	config := generateUserTokenConfiguration(userTokenCfg)

	err := e.client.Security().UpdateUserTokenConfiguration(ctx, config)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing UserTokenConfiguration resource.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	// UserTokenConfiguration is a singleton; we don't delete it
	// Optionally could disable tokens on delete
	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateUserTokenConfiguration generates configuration from the CR spec.
func generateUserTokenConfiguration(userTokenCfg *v1alpha1.UserTokenConfiguration) security.UserTokenConfiguration {
	config := security.UserTokenConfiguration{
		Enabled: userTokenCfg.Spec.ForProvider.Enabled,
	}

	if userTokenCfg.Spec.ForProvider.ProtectContent != nil {
		config.ProtectContent = *userTokenCfg.Spec.ForProvider.ProtectContent
	}

	if userTokenCfg.Spec.ForProvider.ExpirationEnabled != nil {
		config.ExpirationEnabled = *userTokenCfg.Spec.ForProvider.ExpirationEnabled
	}

	if userTokenCfg.Spec.ForProvider.ExpirationDays != nil {
		config.ExpirationDays = int(*userTokenCfg.Spec.ForProvider.ExpirationDays)
	}

	return config
}

// isUserTokenConfigUpToDate checks if UserTokenConfiguration is up to date.
func isUserTokenConfigUpToDate(
	userTokenCfg *v1alpha1.UserTokenConfiguration,
	config *security.UserTokenConfiguration,
) bool {
	if userTokenCfg.Spec.ForProvider.Enabled != config.Enabled {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ProtectContent != nil &&
		*userTokenCfg.Spec.ForProvider.ProtectContent != config.ProtectContent {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ExpirationEnabled != nil &&
		*userTokenCfg.Spec.ForProvider.ExpirationEnabled != config.ExpirationEnabled {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ExpirationDays != nil &&
		int(*userTokenCfg.Spec.ForProvider.ExpirationDays) != config.ExpirationDays {
		return false
	}

	return true
}
