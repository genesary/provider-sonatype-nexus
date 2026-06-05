// Package ldap manages LDAP server configuration resources.
package ldap

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
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
)

const (
	// errNotLDAP means the managed resource is not an LDAP custom resource.
	errNotLDAP = "managed resource is not an LDAP custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetCreds is returned when retrieving credentials fails.
	errGetCreds = "cannot get credentials"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetLDAP is returned when retrieving an LDAP server fails.
	errGetLDAP = "cannot get LDAP server from Nexus"
	// errCreateLDAP is returned when creating an LDAP server fails.
	errCreateLDAP = "cannot create LDAP server in Nexus"
	// errUpdateLDAP is returned when updating an LDAP server fails.
	errUpdateLDAP = "cannot update LDAP server in Nexus"
	// errDeleteLDAP is returned when deleting an LDAP server fails.
	errDeleteLDAP = "cannot delete LDAP server from Nexus"
	// errGetAuthPassword is returned when reading the auth password secret
	// fails.
	errGetAuthPassword = "cannot get auth password from secret"
)

// Setup adds a controller that reconciles LDAP resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.LDAPGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.LDAPGroupVersionKind),
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
		For(&iamv1alpha1.LDAP{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isLDAP := managedRes.(*iamv1alpha1.LDAP)
	if !isLDAP {
		return nil, errors.New(errNotLDAP)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	providerConfig := &nexusv1alpha1.ProviderConfig{}

	err = c.kube.Get(ctx, client.ObjectKey{Name: modernMG.GetProviderConfigReference().Name}, providerConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, providerConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	ldapClient, err := iamclient.NewLDAPClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: ldapClient, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.LDAPClient
	kube   client.Client
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	ldapCR, isLDAP := managedRes.(*iamv1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalObservation{}, errors.New(errNotLDAP)
	}

	ldapName := meta.GetExternalName(ldapCR)
	if ldapName == "" {
		ldapName = ldapCR.Spec.ForProvider.Name
	}

	observed, err := e.client.GetLDAP(ctx, ldapName)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetLDAP)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ldapCR.SetConditions(nexusv1alpha1.Available())

	if observed.ID != "" {
		ldapCR.Status.AtProvider.ID = &observed.ID
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsLDAPUpToDate(ldapCR, observed),
	}, nil
}

// Create creates the desired LDAP server in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	ldapCR, isLDAP := managedRes.(*iamv1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalCreation{}, errors.New(errNotLDAP)
	}

	password, err := e.getAuthPassword(ctx, ldapCR)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetAuthPassword)
	}

	ldapCfg := iamclient.GenerateLDAP(ldapCR, password)

	err = e.client.CreateLDAP(ctx, ldapCfg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLDAP)
	}

	meta.SetExternalName(ldapCR, ldapCR.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the LDAP server to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	ldapCR, isLDAP := managedRes.(*iamv1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalUpdate{}, errors.New(errNotLDAP)
	}

	ldapName := meta.GetExternalName(ldapCR)
	if ldapName == "" {
		ldapName = ldapCR.Spec.ForProvider.Name
	}

	password, err := e.getAuthPassword(ctx, ldapCR)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGetAuthPassword)
	}

	ldapCfg := iamclient.GenerateLDAP(ldapCR, password)

	err = e.client.UpdateLDAP(ctx, ldapName, ldapCfg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLDAP)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the LDAP server from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	ldapCR, isLDAP := managedRes.(*iamv1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalDelete{}, errors.New(errNotLDAP)
	}

	ldapName := meta.GetExternalName(ldapCR)
	if ldapName == "" {
		ldapName = ldapCR.Spec.ForProvider.Name
	}

	err := e.client.DeleteLDAP(ctx, ldapName)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLDAP)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// getAuthPassword retrieves the LDAP auth password from the referenced secret.
func (e *external) getAuthPassword(
	ctx context.Context,
	ldapCR *iamv1alpha1.LDAP,
) (string, error) {
	if ldapCR.Spec.ForProvider.AuthPasswordSecretRef == nil {
		return "", nil
	}

	return nexus.GetSecretValue(ctx, e.kube, ldapCR.Spec.ForProvider.AuthPasswordSecretRef)
}
