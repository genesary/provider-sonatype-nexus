// Package privilege manages Privilege resources.
package privilege

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
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
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetPrivilege is returned when retrieving a Privilege fails.
	errGetPrivilege = "cannot get privilege from Nexus"
	// errCreatePrivilege is returned when creating a Privilege fails.
	errCreatePrivilege = "cannot create privilege in Nexus"
	// errUpdatePrivilege is returned when updating a Privilege fails.
	errUpdatePrivilege = "cannot update privilege in Nexus"
	// errDeletePrivilege is returned when deleting a Privilege fails.
	errDeletePrivilege = "cannot delete privilege from Nexus"
)

// Setup adds a controller that reconciles Privilege resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(iamv1alpha1.PrivilegeGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*iamv1alpha1.Privilege](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *iamv1alpha1.Privilege) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.PrivilegeGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &iamv1alpha1.PrivilegeList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.Privilege{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, privRes *iamv1alpha1.Privilege) (managed.TypedExternalClient[*iamv1alpha1.Privilege], error) {
	err := c.usage.Track(ctx, privRes)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, privRes)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	privilegeClient, err := iamclient.NewPrivilegeClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: privilegeClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client iamclient.PrivilegeClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, privRes *iamv1alpha1.Privilege) (managed.ExternalObservation, error) {
	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	observed, err := e.client.Get(ctx, privName)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetPrivilege)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	privRes.SetConditions(nexusv1alpha1.Available())

	privRes.Status.AtProvider = iamclient.GeneratePrivilegeObservation(observed)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsPrivilegeUpToDate(privRes),
	}, nil
}

// Create creates the desired Privilege in Nexus.
func (e *external) Create(ctx context.Context, privRes *iamv1alpha1.Privilege) (managed.ExternalCreation, error) {
	err := e.client.Create(ctx, privRes)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreatePrivilege)
	}

	meta.SetExternalName(privRes, privRes.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Privilege to the desired state.
func (e *external) Update(ctx context.Context, privRes *iamv1alpha1.Privilege) (managed.ExternalUpdate, error) {
	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	err := e.client.Update(ctx, privName, privRes)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdatePrivilege)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Privilege from Nexus.
func (e *external) Delete(ctx context.Context, privRes *iamv1alpha1.Privilege) (managed.ExternalDelete, error) {
	privName := meta.GetExternalName(privRes)
	if privName == "" {
		privName = privRes.Spec.ForProvider.Name
	}

	err := e.client.Delete(ctx, privName)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeletePrivilege)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
