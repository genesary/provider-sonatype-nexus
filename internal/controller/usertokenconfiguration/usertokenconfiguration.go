// Package usertokenconfiguration contains the controller for UserTokenConfiguration resources.
package usertokenconfiguration

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
	errNotUserTokenConfig    = "managed resource is not a UserTokenConfiguration custom resource"
	errTrackPCUsage          = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errGetCreds              = "cannot get credentials"
	errNewClient             = "cannot create new Nexus client"
	errGetUserTokenConfig    = "cannot get user token configuration from Nexus"
	errUpdateUserTokenConfig = "cannot update user token configuration in Nexus"
)

// Setup adds a controller that reconciles UserTokenConfiguration managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserTokenConfigurationGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserTokenConfigurationGroupVersionKind),
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
		For(&v1alpha1.UserTokenConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.UserTokenConfiguration)
	if !ok {
		return nil, errors.New(errNotUserTokenConfig)
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
	cr, ok := mg.(*v1alpha1.UserTokenConfiguration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserTokenConfig)
	}

	// UserTokenConfiguration is a singleton in Nexus (cannot be truly deleted).
	// When the CR is being deleted, report the resource as absent so the
	// managed reconciler can remove the finalizer and complete deletion.
	if cr.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, err := e.client.Security().GetUserTokenConfiguration(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUserTokenConfig)
	}

	cr.SetConditions(v1alpha1.Available())

	upToDate := isUserTokenConfigUpToDate(cr, config)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.UserTokenConfiguration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserTokenConfig)
	}

	config := generateUserTokenConfiguration(cr)
	if err := e.client.Security().UpdateUserTokenConfiguration(ctx, config); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.UserTokenConfiguration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserTokenConfig)
	}

	config := generateUserTokenConfiguration(cr)
	if err := e.client.Security().UpdateUserTokenConfiguration(ctx, config); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUserTokenConfig)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	// UserTokenConfiguration is a singleton; we don't delete it
	// Optionally could disable tokens on delete
	return nil
}

// generateUserTokenConfiguration generates configuration from the CR spec.
func generateUserTokenConfiguration(cr *v1alpha1.UserTokenConfiguration) security.UserTokenConfiguration {
	config := security.UserTokenConfiguration{
		Enabled: cr.Spec.ForProvider.Enabled,
	}

	if cr.Spec.ForProvider.ProtectContent != nil {
		config.ProtectContent = *cr.Spec.ForProvider.ProtectContent
	}
	if cr.Spec.ForProvider.ExpirationEnabled != nil {
		config.ExpirationEnabled = *cr.Spec.ForProvider.ExpirationEnabled
	}
	if cr.Spec.ForProvider.ExpirationDays != nil {
		config.ExpirationDays = int(*cr.Spec.ForProvider.ExpirationDays)
	}

	return config
}

// isUserTokenConfigUpToDate checks if UserTokenConfiguration is up to date.
func isUserTokenConfigUpToDate(cr *v1alpha1.UserTokenConfiguration, config *security.UserTokenConfiguration) bool {
	if cr.Spec.ForProvider.Enabled != config.Enabled {
		return false
	}
	if cr.Spec.ForProvider.ProtectContent != nil && *cr.Spec.ForProvider.ProtectContent != config.ProtectContent {
		return false
	}
	if cr.Spec.ForProvider.ExpirationEnabled != nil && *cr.Spec.ForProvider.ExpirationEnabled != config.ExpirationEnabled {
		return false
	}
	if cr.Spec.ForProvider.ExpirationDays != nil && int(*cr.Spec.ForProvider.ExpirationDays) != config.ExpirationDays {
		return false
	}
	return true
}
