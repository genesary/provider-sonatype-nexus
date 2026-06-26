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
	errNotIQServerConfiguration = "managed resource is not an IQServerConfiguration custom resource"
	errTrackPCUsage             = "cannot track ProviderConfig usage"
	errGetPC                    = "cannot get ProviderConfig"
	errNewClient                = "cannot create new Nexus client"
	errGetIQServer              = "cannot get IQ Server configuration from Nexus"
	errUpdateIQServer           = "cannot update IQ Server configuration in Nexus"
	errGetPassword              = "cannot get IQ Server password from secret"
)

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

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

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

type external struct {
	client instanceclient.IQServerClient
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIQServerConfiguration)
	}

	if cr.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	observed, err := e.client.Get()
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetIQServer)
	}

	cr.Status.AtProvider = instanceclient.GenerateIQServerObservation(observed)
	cr.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsIQServerUpToDate(cr, observed),
	}, nil
}

func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIQServerConfiguration)
	}

	password, err := e.resolvePassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	config := instanceclient.GenerateIQServerUpdate(&cr.Spec.ForProvider, password)

	if err := e.client.Update(config); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUpdateIQServer)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := managedRes.(*instancev1alpha1.IQServerConfiguration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIQServerConfiguration)
	}

	password, err := e.resolvePassword(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	config := instanceclient.GenerateIQServerUpdate(&cr.Spec.ForProvider, password)

	if err := e.client.Update(config); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIQServer)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error {
	return nil
}

func (e *external) resolvePassword(ctx context.Context, cr *instancev1alpha1.IQServerConfiguration) (string, error) {
	if cr.Spec.ForProvider.PasswordSecretRef == nil {
		return "", nil
	}

	password, err := nexus.GetSecretValue(ctx, e.kube, cr.Spec.ForProvider.PasswordSecretRef)
	if err != nil {
		return "", errors.Wrap(err, errGetPassword)
	}

	return password, nil
}
