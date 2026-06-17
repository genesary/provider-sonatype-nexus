// Package ldap contains the controller for LDAP resources.
package ldap

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
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
	// errNotLDAP is returned when the managed resource is not an LDAP.
	errNotLDAP = "managed resource is not an LDAP custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
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
)

// Setup creates a controller for LDAP resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.LDAPGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.LDAPGroupVersionKind),
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
		For(&v1alpha1.LDAP{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the LDAP controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isLDAP := managedRes.(*v1alpha1.LDAP)
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

	creds, err := nexus.GetCredentials(ctx, c.kube, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nexusClient, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
	kube   client.Client
}

// Observe checks if the LDAP resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	ldapCR, isLDAP := managedRes.(*v1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalObservation{}, errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(ldapCR)
	if name == "" {
		name = ldapCR.Spec.ForProvider.Name
	}

	ldapResult, err := e.client.Security().GetLDAP(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetLDAP)
	}

	if ldapResult == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ldapCR.SetConditions(v1alpha1.Available())

	// Store the ID in observation
	if ldapResult.ID != "" {
		ldapCR.Status.AtProvider.ID = &ldapResult.ID
	}

	upToDate := isLDAPUpToDate(ldapCR, ldapResult)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new LDAP resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	ldapCR, isLDAP := managedRes.(*v1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalCreation{}, errors.New(errNotLDAP)
	}

	ldapCfg, err := generateLDAP(ctx, e.kube, ldapCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = e.client.Security().CreateLDAP(ctx, ldapCfg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateLDAP)
	}

	meta.SetExternalName(ldapCR, ldapCR.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing LDAP resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	ldapCR, isLDAP := managedRes.(*v1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalUpdate{}, errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(ldapCR)
	if name == "" {
		name = ldapCR.Spec.ForProvider.Name
	}

	ldapCfg, err := generateLDAP(ctx, e.kube, ldapCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = e.client.Security().UpdateLDAP(ctx, name, ldapCfg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateLDAP)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing LDAP resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	ldapCR, isLDAP := managedRes.(*v1alpha1.LDAP)
	if !isLDAP {
		return managed.ExternalDelete{}, errors.New(errNotLDAP)
	}

	name := meta.GetExternalName(ldapCR)
	if name == "" {
		name = ldapCR.Spec.ForProvider.Name
	}

	err := e.client.Security().DeleteLDAP(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLDAP)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateLDAP generates an LDAP configuration from the CR spec.
func generateLDAP(ctx context.Context, kube client.Client, ldapCR *v1alpha1.LDAP) (security.LDAP, error) {
	cfg := security.LDAP{
		Name:       ldapCR.Spec.ForProvider.Name,
		Protocol:   ldapCR.Spec.ForProvider.Protocol,
		Host:       ldapCR.Spec.ForProvider.Host,
		Port:       ldapCR.Spec.ForProvider.Port,
		SearchBase: ldapCR.Spec.ForProvider.SearchBase,
		AuthSchema: ldapCR.Spec.ForProvider.AuthScheme,
		UserBaseDN: ldapCR.Spec.ForProvider.UserBaseDN,
	}

	// Handle auth password from secret
	if ldapCR.Spec.ForProvider.AuthPasswordSecretRef != nil {
		password, err := nexus.GetSecretValue(ctx, kube, ldapCR.Spec.ForProvider.AuthPasswordSecretRef)
		if err != nil {
			return cfg, errors.Wrap(err, "cannot get auth password from secret")
		}

		cfg.AuthPassword = password
	}

	applyLDAPConnection(&cfg, &ldapCR.Spec.ForProvider)
	applyLDAPUserConfig(&cfg, &ldapCR.Spec.ForProvider)
	applyLDAPGroupConfig(&cfg, &ldapCR.Spec.ForProvider)

	return cfg, nil
}

// applyLDAPConnection applies connection-related fields from the spec to the
// LDAP config.
func applyLDAPConnection(cfg *security.LDAP, spec *v1alpha1.LDAPParameters) {
	if spec.AuthUsername != nil {
		cfg.AuthUserName = *spec.AuthUsername
	}

	if spec.AuthRealm != nil {
		cfg.AuthRealm = *spec.AuthRealm
	}

	if spec.ConnectionTimeoutSeconds != nil {
		cfg.ConnectionTimeoutSeconds = *spec.ConnectionTimeoutSeconds
	}

	if spec.ConnectionRetryDelaySeconds != nil {
		cfg.ConnectionRetryDelaySeconds = *spec.ConnectionRetryDelaySeconds
	}

	if spec.MaxIncidentCount != nil {
		cfg.MaxIncidentCount = *spec.MaxIncidentCount
	}

	if spec.UseTrustStore != nil {
		cfg.UseTrustStore = *spec.UseTrustStore
	}
}

// applyLDAPUserConfig applies user-mapping fields from the spec to the LDAP
// config.
func applyLDAPUserConfig(cfg *security.LDAP, spec *v1alpha1.LDAPParameters) {
	if spec.UserSubtree != nil {
		cfg.UserSubtree = *spec.UserSubtree
	}

	if spec.UserObjectClass != nil {
		cfg.UserObjectClass = *spec.UserObjectClass
	}

	if spec.UserIDAttribute != nil {
		cfg.UserIDAttribute = *spec.UserIDAttribute
	}

	if spec.UserRealNameAttribute != nil {
		cfg.UserRealNameAttribute = *spec.UserRealNameAttribute
	}

	if spec.UserEmailAddressAttribute != nil {
		cfg.UserEmailAddressAttribute = *spec.UserEmailAddressAttribute
	}

	if spec.UserPasswordAttribute != nil {
		cfg.UserPasswordAttribute = *spec.UserPasswordAttribute
	}

	if spec.UserMemberOfAttribute != nil {
		cfg.UserMemberOfAttribute = *spec.UserMemberOfAttribute
	}

	if spec.UserLDAPFilter != nil {
		cfg.UserLDAPFilter = *spec.UserLDAPFilter
	}
}

// applyLDAPGroupConfig applies group-mapping fields from the spec to the LDAP
// config. Group fields are only meaningful when LDAPGroupsAsRoles is enabled.
func applyLDAPGroupConfig(cfg *security.LDAP, spec *v1alpha1.LDAPParameters) {
	if spec.LDAPGroupsAsRoles == nil {
		return
	}

	cfg.LDAPGroupsAsRoles = *spec.LDAPGroupsAsRoles

	if spec.GroupType != nil {
		cfg.GroupType = *spec.GroupType
	}

	if spec.GroupBaseDN != nil {
		cfg.GroupBaseDn = *spec.GroupBaseDN
	}

	if spec.GroupSubtree != nil {
		cfg.GroupSubtree = *spec.GroupSubtree
	}

	if spec.GroupObjectClass != nil {
		cfg.GroupObjectClass = *spec.GroupObjectClass
	}

	if spec.GroupIDAttribute != nil {
		cfg.GroupIDAttribute = *spec.GroupIDAttribute
	}

	if spec.GroupMemberAttribute != nil {
		cfg.GroupMemberAttribute = *spec.GroupMemberAttribute
	}

	if spec.GroupMemberFormat != nil {
		cfg.GroupMemberFormat = *spec.GroupMemberFormat
	}
}

// isLDAPUpToDate checks if an LDAP configuration is up to date.
func isLDAPUpToDate(ldapCR *v1alpha1.LDAP, ldapResult *security.LDAP) bool {
	if ldapCR.Spec.ForProvider.Protocol != ldapResult.Protocol {
		return false
	}

	if ldapCR.Spec.ForProvider.Host != ldapResult.Host {
		return false
	}

	if ldapCR.Spec.ForProvider.Port != ldapResult.Port {
		return false
	}

	if ldapCR.Spec.ForProvider.SearchBase != ldapResult.SearchBase {
		return false
	}

	if ldapCR.Spec.ForProvider.AuthScheme != ldapResult.AuthSchema {
		return false
	}

	if ldapCR.Spec.ForProvider.UserBaseDN != ldapResult.UserBaseDN {
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
