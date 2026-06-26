// Package iqserverconfiguration manages IQServerConfiguration resources.
package iqserverconfiguration

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

const (
	// errNotIQServerConfiguration means managed resource
	// is not an IQServerConfiguration.
	errNotIQServerConfiguration = "managed resource is not an IQServerConfiguration custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetIQServer is returned when retrieving IQ Server config fails.
	errGetIQServer = "cannot get IQ Server configuration from Nexus"
	// errDeleteIQServer is returned when IQ Server configuration disable fails.
	errDeleteIQServer = "cannot disable IQ Server configuration in Nexus"
	// errUpdateIQServer is returned when IQ Server configuration update fails.
	errUpdateIQServer = "cannot update IQ Server configuration in Nexus"
	// errGetPassword is returned when reading the IQ Server password fails.
	errGetPassword = "cannot get IQ Server password from secret"
	// errGetUsername is returned when reading the IQ Server username fails.
	errGetUsername = "cannot get IQ Server username from secret"
)

// Setup adds a controller that reconciles IQServerConfiguration resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(instancev1alpha1.IQServerConfigurationGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(instancev1alpha1.IQServerConfigurationGroupVersionKind),
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
		For(&instancev1alpha1.IQServerConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector creates an external client for IQServerConfiguration resources.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates the external IQ Server client.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return nil, errors.New(errNotIQServerConfiguration)
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

	iqClient, err := instanceclient.NewIQServerClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: iqClient, kube: c.kube}, nil
}

// external implements managed.ExternalClient for IQServerConfiguration.
type external struct {
	client instanceclient.IQServerClient
	kube   client.Client
}

// Observe checks the current IQ Server configuration against desired state.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	iqConfig, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIQServerConfiguration)
	}

	observed, err := e.client.Get()
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetIQServer)
	}

	if !iqConfig.GetDeletionTimestamp().IsZero() && observed != nil && !observed.Enabled {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	iqConfig.Status.AtProvider = instanceclient.GenerateIQServerObservation(observed)
	iqConfig.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsIQServerUpToDate(iqConfig, observed),
	}, nil
}

// Create applies the IQ Server configuration in Nexus.
// IQ Server config is a singleton; Create simply applies the spec.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	iqConfig, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIQServerConfiguration)
	}

	err := e.applyConfig(ctx, iqConfig)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the IQ Server configuration in Nexus.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	iqConfig, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIQServerConfiguration)
	}

	err := e.applyConfig(ctx, iqConfig)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete disables the IQ Server integration in Nexus.
// IQ Server config is a singleton; Delete disables rather than removes.
func (e *external) Delete(_ context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	_, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIQServerConfiguration)
	}

	err := e.client.Disable()
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIQServer)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// applyConfig resolves credentials and pushes the config to Nexus.
func (e *external) applyConfig(ctx context.Context, iqConfig *instancev1alpha1.IQServerConfiguration) error {
	username, password, err := e.resolveCredentials(ctx, iqConfig)
	if err != nil {
		return err
	}

	config := instanceclient.GenerateIQServerUpdate(&iqConfig.Spec.ForProvider, username, password)

	err = e.client.Update(config)
	if err != nil {
		return errors.Wrap(err, errUpdateIQServer)
	}

	return nil
}

// resolveCredentials reads username and password from Kubernetes Secrets.
func (e *external) resolveCredentials(ctx context.Context, iqConfig *instancev1alpha1.IQServerConfiguration) (username, password string, err error) {
	if iqConfig.Spec.ForProvider.UsernameSecretRef != nil {
		username, err = nexus.GetSecretValue(ctx, e.kube, iqConfig.Spec.ForProvider.UsernameSecretRef)
		if err != nil {
			return "", "", errors.Wrap(err, errGetUsername)
		}
	}

	if iqConfig.Spec.ForProvider.PasswordSecretRef != nil {
		password, err = nexus.GetSecretValue(ctx, e.kube, iqConfig.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return "", "", errors.Wrap(err, errGetPassword)
		}
	}

	return username, password, nil
}
