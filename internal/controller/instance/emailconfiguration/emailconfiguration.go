// Package emailconfiguration manages EmailConfiguration resources.
package emailconfiguration

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

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// Error message constants for reconcile operations.
const (
	// errNotEmailConfig is returned for wrong managed resource type.
	errNotEmailConfig = "managed resource is not an EmailConfiguration custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetEmailConfig is returned when retrieving email config fails.
	errGetEmailConfig = "cannot get email configuration from Nexus"
	// errUpdateEmailConfig is returned when updating email config fails.
	errUpdateEmailConfig = "cannot update email configuration in Nexus"
	// errGetPassword is returned when resolving the password secret fails.
	errGetPassword = "cannot get password from secret"
)

// Setup adds a controller that reconciles EmailConfiguration resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(instancev1alpha1.EmailConfigurationGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(instancev1alpha1.EmailConfigurationGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ClusterProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.EmailConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, ok := managedRes.(*instancev1alpha1.EmailConfiguration)
	if !ok {
		return nil, errors.New(errNotEmailConfig)
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

	emailClient, err := instanceclient.NewEmailConfigurationClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: emailClient, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client instanceclient.EmailConfigurationClient
	kube   client.Client
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	emailCfg, ok := managedRes.(*instancev1alpha1.EmailConfiguration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotEmailConfig)
	}

	// EmailConfiguration is a singleton; report absent when being deleted so
	// the finalizer can be cleared.
	if emailCfg.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	config, err := e.client.GetEmailConfiguration(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetEmailConfig)
	}

	emailCfg.Status.AtProvider = instanceclient.GenerateEmailConfigurationObservation(config)
	emailCfg.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsEmailConfigurationUpToDate(emailCfg),
	}, nil
}

// Create applies the desired email configuration.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	emailCfg, ok := managedRes.(*instancev1alpha1.EmailConfiguration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotEmailConfig)
	}

	err := e.syncEmailConfig(ctx, emailCfg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the desired email configuration.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	emailCfg, ok := managedRes.(*instancev1alpha1.EmailConfiguration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotEmailConfig)
	}

	err := e.syncEmailConfig(ctx, emailCfg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete is a no-op; EmailConfiguration is a singleton.
func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// syncEmailConfig resolves the password and pushes the desired config to Nexus.
// Shared by Create and Update.
func (e *external) syncEmailConfig(ctx context.Context, emailCfg *instancev1alpha1.EmailConfiguration) error {
	password, err := e.resolvePassword(ctx, emailCfg)
	if err != nil {
		return errors.Wrap(err, errGetPassword)
	}

	cfg := instanceclient.GenerateEmailConfiguration(emailCfg, password)

	err = e.client.UpdateEmailConfiguration(ctx, cfg)
	if err != nil {
		return errors.Wrap(err, errUpdateEmailConfig)
	}

	return nil
}

// resolvePassword retrieves the SMTP password from the referenced Secret,
// returning an empty string when no secret reference is configured.
func (e *external) resolvePassword(
	ctx context.Context,
	emailCfg *instancev1alpha1.EmailConfiguration,
) (string, error) {
	if emailCfg.Spec.ForProvider.PasswordSecretRef == nil {
		return "", nil
	}

	return nexus.GetSecretValue(ctx, e.kube, emailCfg.Spec.ForProvider.PasswordSecretRef)
}
