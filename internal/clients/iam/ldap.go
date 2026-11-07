package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// Nexus defaults for the LDAP settings its API requires on every write. They
// mirror the CRD defaults, which the API server applies before the provider
// ever sees the spec.
const (
	defaultLDAPConnectionTimeoutSeconds    = 30
	defaultLDAPConnectionRetryDelaySeconds = 300
	defaultLDAPMaxIncidentsCount           = 3
	defaultLDAPUserObjectClass             = "inetOrgPerson"
	defaultLDAPUserIDAttribute             = "uid"
	defaultLDAPUserRealNameAttribute       = "cn"
	defaultLDAPUserEmailAddressAttribute   = "mail"
)

// LDAPClient manages Nexus LDAP server configurations.
type LDAPClient interface {
	Get(ctx context.Context, name string) (*v395.ReadLdapServerXo, error)
	Create(ctx context.Context, ldap v395.CreateLdapServerXo) error
	Update(ctx context.Context, name string, ldap v395.UpdateLdapServerXo) error
	Delete(ctx context.Context, name string) error
}

// ldapClient implements LDAPClient over the Nexus API.
type ldapClient struct {
	api *v395.SecurityManagementLDAPAPIService
}

// NewLDAPClient returns a new LDAPClient.
func NewLDAPClient(creds nexus.Credentials) (LDAPClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &ldapClient{api: nc.SecurityManagementLDAPAPI}, nil
}

// Get returns the LDAP server with the given name.
func (c *ldapClient) Get(ctx context.Context, name string) (*v395.ReadLdapServerXo, error) {
	server, resp, err := c.api.GetSecurityLdap(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return server, nil
}

// Create creates an LDAP server.
func (c *ldapClient) Create(ctx context.Context, ldap v395.CreateLdapServerXo) error {
	return nexus.APIError(c.api.CreateSecurityLdap(ctx).CreateLdapServerXo(ldap).Execute())
}

// Update updates the LDAP server with the given name.
func (c *ldapClient) Update(ctx context.Context, name string, ldap v395.UpdateLdapServerXo) error {
	return nexus.APIError(c.api.UpdateSecurityLdap(ctx, name).UpdateLdapServerXo(ldap).Execute())
}

// Delete deletes the LDAP server with the given name.
func (c *ldapClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteSecurityLdap(ctx, name).Execute())
}

// GenerateLDAP converts an LDAP CR to the Nexus API creation request.
func GenerateLDAP(ldapCR *iamv1alpha1.LDAP, password string) v395.CreateLdapServerXo {
	spec := &ldapCR.Spec.ForProvider

	cfg := v395.CreateLdapServerXo{
		Name:       spec.Name,
		Protocol:   spec.Protocol,
		Host:       spec.Host,
		Port:       spec.Port,
		SearchBase: spec.SearchBase,
		AuthScheme: spec.AuthScheme,
		UserBaseDn: new(spec.UserBaseDN),
	}

	if password != "" {
		cfg.AuthPassword = new(password)
	}

	applyLDAPConnection(&cfg, spec)
	applyLDAPUserConfig(&cfg, spec)
	applyLDAPGroupConfig(&cfg, spec)

	return cfg
}

// GenerateLDAPUpdate converts an LDAP CR to the Nexus API update request.
func GenerateLDAPUpdate(ldapCR *iamv1alpha1.LDAP, password string) v395.UpdateLdapServerXo {
	cfg := GenerateLDAP(ldapCR, password)

	return v395.UpdateLdapServerXo{
		Name:                        cfg.Name,
		Protocol:                    cfg.Protocol,
		Host:                        cfg.Host,
		Port:                        cfg.Port,
		SearchBase:                  cfg.SearchBase,
		AuthScheme:                  cfg.AuthScheme,
		AuthUsername:                cfg.AuthUsername,
		AuthPassword:                cfg.AuthPassword,
		AuthRealm:                   cfg.AuthRealm,
		ConnectionTimeoutSeconds:    cfg.ConnectionTimeoutSeconds,
		ConnectionRetryDelaySeconds: cfg.ConnectionRetryDelaySeconds,
		MaxIncidentsCount:           cfg.MaxIncidentsCount,
		UseTrustStore:               cfg.UseTrustStore,
		UserBaseDn:                  cfg.UserBaseDn,
		UserSubtree:                 cfg.UserSubtree,
		UserObjectClass:             cfg.UserObjectClass,
		UserIdAttribute:             cfg.UserIdAttribute,
		UserRealNameAttribute:       cfg.UserRealNameAttribute,
		UserEmailAddressAttribute:   cfg.UserEmailAddressAttribute,
		UserPasswordAttribute:       cfg.UserPasswordAttribute,
		UserMemberOfAttribute:       cfg.UserMemberOfAttribute,
		UserLdapFilter:              cfg.UserLdapFilter,
		LdapGroupsAsRoles:           cfg.LdapGroupsAsRoles,
		GroupType:                   cfg.GroupType,
		GroupBaseDn:                 cfg.GroupBaseDn,
		GroupSubtree:                cfg.GroupSubtree,
		GroupObjectClass:            cfg.GroupObjectClass,
		GroupIdAttribute:            cfg.GroupIdAttribute,
		GroupMemberAttribute:        cfg.GroupMemberAttribute,
		GroupMemberFormat:           cfg.GroupMemberFormat,
	}
}

// GenerateLDAPObservation returns the observed LDAP server state.
//
// Every field Nexus reports is recorded, not only the ones that identify the
// server: drift detection compares the spec against this observation, so a
// field left out of it can never be seen to drift.
//
// The bind password is deliberately absent - Nexus never returns it, so it
// takes no part in the observation nor in drift detection.
func GenerateLDAPObservation(observed *v395.ReadLdapServerXo) iamv1alpha1.LDAPObservation {
	if observed == nil {
		return iamv1alpha1.LDAPObservation{}
	}

	obs := iamv1alpha1.LDAPObservation{
		Name:                        observed.Name,
		Protocol:                    observed.Protocol,
		Host:                        observed.Host,
		Port:                        observed.Port,
		SearchBase:                  observed.SearchBase,
		AuthScheme:                  observed.AuthScheme,
		AuthUsername:                observed.GetAuthUsername(),
		AuthRealm:                   observed.GetAuthRealm(),
		ConnectionTimeoutSeconds:    observed.ConnectionTimeoutSeconds,
		ConnectionRetryDelaySeconds: observed.ConnectionRetryDelaySeconds,
		MaxIncidentCount:            observed.MaxIncidentsCount,
		UseTrustStore:               observed.GetUseTrustStore(),
		UserBaseDN:                  observed.GetUserBaseDn(),
		UserSubtree:                 observed.GetUserSubtree(),
		UserObjectClass:             observed.UserObjectClass,
		UserIDAttribute:             observed.UserIdAttribute,
		UserRealNameAttribute:       observed.UserRealNameAttribute,
		UserEmailAddressAttribute:   observed.UserEmailAddressAttribute,
		UserPasswordAttribute:       observed.GetUserPasswordAttribute(),
		UserMemberOfAttribute:       observed.GetUserMemberOfAttribute(),
		UserLDAPFilter:              observed.GetUserLdapFilter(),
		LDAPGroupsAsRoles:           observed.GetLdapGroupsAsRoles(),
		GroupType:                   observed.GetGroupType(),
		GroupBaseDN:                 observed.GetGroupBaseDn(),
		GroupSubtree:                observed.GetGroupSubtree(),
		GroupObjectClass:            observed.GetGroupObjectClass(),
		GroupIDAttribute:            observed.GetGroupIdAttribute(),
		GroupMemberAttribute:        observed.GetGroupMemberAttribute(),
		GroupMemberFormat:           observed.GetGroupMemberFormat(),
	}

	if observed.GetId() != "" {
		obs.ID = observed.Id
	}

	return obs
}

// IsLDAPUpToDate reports whether the CR spec matches the observed LDAP config.
//
// Optional fields are only asserted when the spec sets them: an unset optional
// field carries no intent, and Nexus fills those in with its own defaults.
func IsLDAPUpToDate(ldapCR *iamv1alpha1.LDAP) bool {
	spec := &ldapCR.Spec.ForProvider
	obs := &ldapCR.Status.AtProvider

	return isLDAPConnectionUpToDate(spec, obs) &&
		isLDAPUserConfigUpToDate(spec, obs) &&
		isLDAPGroupConfigUpToDate(spec, obs)
}

// isLDAPConnectionUpToDate reports whether the connection settings in the spec
// match the observed LDAP config.
func isLDAPConnectionUpToDate(spec *iamv1alpha1.LDAPParameters, obs *iamv1alpha1.LDAPObservation) bool {
	return isLDAPEndpointUpToDate(spec, obs) && isLDAPRetryPolicyUpToDate(spec, obs)
}

// isLDAPEndpointUpToDate reports whether the address of the LDAP server and
// the credentials used to bind to it match the observed LDAP config.
func isLDAPEndpointUpToDate(spec *iamv1alpha1.LDAPParameters, obs *iamv1alpha1.LDAPObservation) bool {
	if spec.Protocol != obs.Protocol ||
		spec.Host != obs.Host ||
		spec.Port != obs.Port {
		return false
	}

	if spec.SearchBase != obs.SearchBase || spec.AuthScheme != obs.AuthScheme {
		return false
	}

	return helpers.IsComparablePtrEqualComparable(spec.AuthUsername, obs.AuthUsername) &&
		helpers.IsComparablePtrEqualComparable(spec.AuthRealm, obs.AuthRealm) &&
		helpers.IsComparablePtrEqualComparable(spec.UseTrustStore, obs.UseTrustStore)
}

// isLDAPRetryPolicyUpToDate reports whether the timeout and retry settings in
// the spec match the observed LDAP config.
func isLDAPRetryPolicyUpToDate(spec *iamv1alpha1.LDAPParameters, obs *iamv1alpha1.LDAPObservation) bool {
	return helpers.IsComparablePtrEqualComparable(spec.ConnectionTimeoutSeconds, obs.ConnectionTimeoutSeconds) &&
		helpers.IsComparablePtrEqualComparable(spec.ConnectionRetryDelaySeconds, obs.ConnectionRetryDelaySeconds) &&
		helpers.IsComparablePtrEqualComparable(spec.MaxIncidentCount, obs.MaxIncidentCount)
}

// isLDAPUserConfigUpToDate reports whether the user-mapping settings in the
// spec match the observed LDAP config.
func isLDAPUserConfigUpToDate(spec *iamv1alpha1.LDAPParameters, obs *iamv1alpha1.LDAPObservation) bool {
	if spec.UserBaseDN != obs.UserBaseDN {
		return false
	}

	return helpers.IsComparablePtrEqualComparable(spec.UserSubtree, obs.UserSubtree) &&
		helpers.IsComparablePtrEqualComparable(spec.UserObjectClass, obs.UserObjectClass) &&
		helpers.IsComparablePtrEqualComparable(spec.UserIDAttribute, obs.UserIDAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.UserRealNameAttribute, obs.UserRealNameAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.UserEmailAddressAttribute, obs.UserEmailAddressAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.UserPasswordAttribute, obs.UserPasswordAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.UserMemberOfAttribute, obs.UserMemberOfAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.UserLDAPFilter, obs.UserLDAPFilter)
}

// isLDAPGroupConfigUpToDate reports whether the group-mapping settings in the
// spec match the observed LDAP config.
//
// The group settings are only submitted when ldapGroupsAsRoles is set, so a
// spec that leaves it unset asks for nothing here - see applyLDAPGroupConfig.
func isLDAPGroupConfigUpToDate(spec *iamv1alpha1.LDAPParameters, obs *iamv1alpha1.LDAPObservation) bool {
	if spec.LDAPGroupsAsRoles == nil {
		return true
	}

	if *spec.LDAPGroupsAsRoles != obs.LDAPGroupsAsRoles {
		return false
	}

	return helpers.IsComparablePtrEqualComparable(spec.GroupType, obs.GroupType) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupBaseDN, obs.GroupBaseDN) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupSubtree, obs.GroupSubtree) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupObjectClass, obs.GroupObjectClass) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupIDAttribute, obs.GroupIDAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupMemberAttribute, obs.GroupMemberAttribute) &&
		helpers.IsComparablePtrEqualComparable(spec.GroupMemberFormat, obs.GroupMemberFormat)
}

// applyLDAPConnection applies connection-related fields from the spec to the
// LDAP config.
func applyLDAPConnection(cfg *v395.CreateLdapServerXo, spec *iamv1alpha1.LDAPParameters) {
	cfg.AuthUsername = spec.AuthUsername
	cfg.AuthRealm = spec.AuthRealm
	cfg.ConnectionTimeoutSeconds = ptr.Deref(spec.ConnectionTimeoutSeconds, defaultLDAPConnectionTimeoutSeconds)
	cfg.ConnectionRetryDelaySeconds = ptr.Deref(spec.ConnectionRetryDelaySeconds, defaultLDAPConnectionRetryDelaySeconds)
	cfg.MaxIncidentsCount = ptr.Deref(spec.MaxIncidentCount, defaultLDAPMaxIncidentsCount)
	cfg.UseTrustStore = spec.UseTrustStore
}

// applyLDAPUserConfig applies user-mapping fields from the spec to the LDAP
// config.
func applyLDAPUserConfig(cfg *v395.CreateLdapServerXo, spec *iamv1alpha1.LDAPParameters) {
	cfg.UserSubtree = spec.UserSubtree
	cfg.UserObjectClass = ptr.Deref(spec.UserObjectClass, defaultLDAPUserObjectClass)
	cfg.UserIdAttribute = ptr.Deref(spec.UserIDAttribute, defaultLDAPUserIDAttribute)
	cfg.UserRealNameAttribute = ptr.Deref(spec.UserRealNameAttribute, defaultLDAPUserRealNameAttribute)
	cfg.UserEmailAddressAttribute = ptr.Deref(spec.UserEmailAddressAttribute, defaultLDAPUserEmailAddressAttribute)
	cfg.UserPasswordAttribute = spec.UserPasswordAttribute
	cfg.UserMemberOfAttribute = spec.UserMemberOfAttribute
	cfg.UserLdapFilter = spec.UserLDAPFilter
}

// applyLDAPGroupConfig applies group-mapping fields from the spec to the
// LDAP config.
func applyLDAPGroupConfig(cfg *v395.CreateLdapServerXo, spec *iamv1alpha1.LDAPParameters) {
	if spec.LDAPGroupsAsRoles == nil {
		return
	}

	cfg.LdapGroupsAsRoles = spec.LDAPGroupsAsRoles
	cfg.GroupType = spec.GroupType
	cfg.GroupBaseDn = spec.GroupBaseDN
	cfg.GroupSubtree = spec.GroupSubtree
	cfg.GroupObjectClass = spec.GroupObjectClass
	cfg.GroupIdAttribute = spec.GroupIDAttribute
	cfg.GroupMemberAttribute = spec.GroupMemberAttribute
	cfg.GroupMemberFormat = spec.GroupMemberFormat
}
