// Package ldap contains the controller for LDAP resources.
package ldap

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotLDAP      = "managed resource is not an LDAP custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Nexus client"
	errGetLDAP      = "cannot get LDAP server from Nexus"
	errCreateLDAP   = "cannot create LDAP server in Nexus"
	errUpdateLDAP   = "cannot update LDAP server in Nexus"
	errDeleteLDAP   = "cannot delete LDAP server from Nexus"
)

// Setup adds a controller that reconciles LDAP managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.LDAPGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.LDAPGroupVersionKind),
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
		For(&v1alpha1.LDAP{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.LDAP)
	if !ok {
		return nil, errors.New(errNotLDAP)
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

	return &external{client: nc, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
	kube   client.Client
}

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.LDAP)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	ldap, err := e.client.Security().GetLDAP(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLDAP)
	}

	if ldap == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	// Store the ID in observation
	if ldap.ID != "" {
		cr.Status.AtProvider.ID = &ldap.ID
	}

	upToDate := isLDAPUpToDate(cr, ldap)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.LDAP)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLDAP)
	}

	ldap, err := generateLDAP(ctx, e.kube, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.Security().CreateLDAP(ctx, ldap); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLDAP)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.LDAP)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	ldap, err := generateLDAP(ctx, e.kube, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.Security().UpdateLDAP(ctx, name, ldap); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLDAP)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.LDAP)
	if !ok {
		return errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	if err := e.client.Security().DeleteLDAP(ctx, name); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeleteLDAP)
	}

	return nil
}

// generateLDAP generates an LDAP configuration from the CR spec.
func generateLDAP(ctx context.Context, kube client.Client, cr *v1alpha1.LDAP) (security.LDAP, error) {
	ldap := security.LDAP{
		Name:       cr.Spec.ForProvider.Name,
		Protocol:   cr.Spec.ForProvider.Protocol,
		Host:       cr.Spec.ForProvider.Host,
		Port:       cr.Spec.ForProvider.Port,
		SearchBase: cr.Spec.ForProvider.SearchBase,
		AuthSchema: cr.Spec.ForProvider.AuthScheme,
		UserBaseDN: cr.Spec.ForProvider.UserBaseDN,
	}

	if cr.Spec.ForProvider.AuthUsername != nil {
		ldap.AuthUserName = *cr.Spec.ForProvider.AuthUsername
	}

	// Handle auth password from secret
	if cr.Spec.ForProvider.AuthPasswordSecretRef != nil {
		password, err := nexus.GetSecretValue(ctx, kube, cr.Spec.ForProvider.AuthPasswordSecretRef)
		if err != nil {
			return ldap, errors.Wrap(err, "cannot get auth password from secret")
		}
		ldap.AuthPassword = password
	}

	if cr.Spec.ForProvider.AuthRealm != nil {
		ldap.AuthRealm = *cr.Spec.ForProvider.AuthRealm
	}
	if cr.Spec.ForProvider.ConnectionTimeoutSeconds != nil {
		ldap.ConnectionTimeoutSeconds = *cr.Spec.ForProvider.ConnectionTimeoutSeconds
	}
	if cr.Spec.ForProvider.ConnectionRetryDelaySeconds != nil {
		ldap.ConnectionRetryDelaySeconds = *cr.Spec.ForProvider.ConnectionRetryDelaySeconds
	}
	if cr.Spec.ForProvider.MaxIncidentCount != nil {
		ldap.MaxIncidentCount = *cr.Spec.ForProvider.MaxIncidentCount
	}
	if cr.Spec.ForProvider.UseTrustStore != nil {
		ldap.UseTrustStore = *cr.Spec.ForProvider.UseTrustStore
	}
	if cr.Spec.ForProvider.UserSubtree != nil {
		ldap.UserSubtree = *cr.Spec.ForProvider.UserSubtree
	}
	if cr.Spec.ForProvider.UserObjectClass != nil {
		ldap.UserObjectClass = *cr.Spec.ForProvider.UserObjectClass
	}
	if cr.Spec.ForProvider.UserIDAttribute != nil {
		ldap.UserIDAttribute = *cr.Spec.ForProvider.UserIDAttribute
	}
	if cr.Spec.ForProvider.UserRealNameAttribute != nil {
		ldap.UserRealNameAttribute = *cr.Spec.ForProvider.UserRealNameAttribute
	}
	if cr.Spec.ForProvider.UserEmailAddressAttribute != nil {
		ldap.UserEmailAddressAttribute = *cr.Spec.ForProvider.UserEmailAddressAttribute
	}
	if cr.Spec.ForProvider.UserPasswordAttribute != nil {
		ldap.UserPasswordAttribute = *cr.Spec.ForProvider.UserPasswordAttribute
	}
	if cr.Spec.ForProvider.UserMemberOfAttribute != nil {
		ldap.UserMemberOfAttribute = *cr.Spec.ForProvider.UserMemberOfAttribute
	}
	if cr.Spec.ForProvider.UserLDAPFilter != nil {
		ldap.UserLDAPFilter = *cr.Spec.ForProvider.UserLDAPFilter
	}
	if cr.Spec.ForProvider.LDAPGroupsAsRoles != nil {
		ldap.LDAPGroupsAsRoles = *cr.Spec.ForProvider.LDAPGroupsAsRoles
	}
	if cr.Spec.ForProvider.GroupType != nil {
		ldap.GroupType = *cr.Spec.ForProvider.GroupType
	}
	if cr.Spec.ForProvider.GroupBaseDN != nil {
		ldap.GroupBaseDn = *cr.Spec.ForProvider.GroupBaseDN
	}
	if cr.Spec.ForProvider.GroupSubtree != nil {
		ldap.GroupSubtree = *cr.Spec.ForProvider.GroupSubtree
	}
	if cr.Spec.ForProvider.GroupObjectClass != nil {
		ldap.GroupObjectClass = *cr.Spec.ForProvider.GroupObjectClass
	}
	if cr.Spec.ForProvider.GroupIDAttribute != nil {
		ldap.GroupIDAttribute = *cr.Spec.ForProvider.GroupIDAttribute
	}
	if cr.Spec.ForProvider.GroupMemberAttribute != nil {
		ldap.GroupMemberAttribute = *cr.Spec.ForProvider.GroupMemberAttribute
	}
	if cr.Spec.ForProvider.GroupMemberFormat != nil {
		ldap.GroupMemberFormat = *cr.Spec.ForProvider.GroupMemberFormat
	}

	return ldap, nil
}

// isLDAPUpToDate checks if an LDAP configuration is up to date.
func isLDAPUpToDate(cr *v1alpha1.LDAP, ldap *security.LDAP) bool {
	if cr.Spec.ForProvider.Protocol != ldap.Protocol {
		return false
	}
	if cr.Spec.ForProvider.Host != ldap.Host {
		return false
	}
	if cr.Spec.ForProvider.Port != ldap.Port {
		return false
	}
	if cr.Spec.ForProvider.SearchBase != ldap.SearchBase {
		return false
	}
	if cr.Spec.ForProvider.AuthScheme != ldap.AuthSchema {
		return false
	}
	if cr.Spec.ForProvider.UserBaseDN != ldap.UserBaseDN {
		return false
	}
	// Additional checks can be added here for other fields
	return true
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
