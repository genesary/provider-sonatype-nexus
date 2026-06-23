package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// LDAPClient manages Nexus LDAP server configurations.
type LDAPClient interface {
	Get(name string) (*security.LDAP, error)
	Create(ldap security.LDAP) error
	Update(name string, ldap security.LDAP) error
	Delete(name string) error
}

// NewLDAPClient returns a new LDAPClient.
func NewLDAPClient(creds nexus.Credentials) (LDAPClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Security.LDAP, nil
}

// GenerateLDAP converts an LDAP CR to the Nexus API type.
func GenerateLDAP(ldapCR *iamv1alpha1.LDAP, password string) security.LDAP {
	cfg := security.LDAP{
		Name:         ldapCR.Spec.ForProvider.Name,
		Protocol:     ldapCR.Spec.ForProvider.Protocol,
		Host:         ldapCR.Spec.ForProvider.Host,
		Port:         ldapCR.Spec.ForProvider.Port,
		SearchBase:   ldapCR.Spec.ForProvider.SearchBase,
		AuthSchema:   ldapCR.Spec.ForProvider.AuthScheme,
		UserBaseDN:   ldapCR.Spec.ForProvider.UserBaseDN,
		AuthPassword: password,
	}

	applyLDAPConnection(&cfg, &ldapCR.Spec.ForProvider)
	applyLDAPUserConfig(&cfg, &ldapCR.Spec.ForProvider)
	applyLDAPGroupConfig(&cfg, &ldapCR.Spec.ForProvider)

	return cfg
}

// GenerateLDAPObservation returns the observed LDAP server state.
func GenerateLDAPObservation(observed *security.LDAP) iamv1alpha1.LDAPObservation {
	if observed == nil {
		return iamv1alpha1.LDAPObservation{}
	}

	obs := iamv1alpha1.LDAPObservation{
		Protocol:   observed.Protocol,
		Host:       observed.Host,
		Port:       observed.Port,
		SearchBase: observed.SearchBase,
		AuthScheme: observed.AuthSchema,
		UserBaseDN: observed.UserBaseDN,
	}

	if observed.ID != "" {
		obs.ID = &observed.ID
	}

	return obs
}

// IsLDAPUpToDate reports whether the CR spec matches the observed LDAP config.
func IsLDAPUpToDate(ldapCR *iamv1alpha1.LDAP) bool {
	obs := ldapCR.Status.AtProvider

	if ldapCR.Spec.ForProvider.Protocol != obs.Protocol {
		return false
	}

	if ldapCR.Spec.ForProvider.Host != obs.Host {
		return false
	}

	if ldapCR.Spec.ForProvider.Port != obs.Port {
		return false
	}

	if ldapCR.Spec.ForProvider.SearchBase != obs.SearchBase {
		return false
	}

	if ldapCR.Spec.ForProvider.AuthScheme != obs.AuthScheme {
		return false
	}

	if ldapCR.Spec.ForProvider.UserBaseDN != obs.UserBaseDN {
		return false
	}

	return true
}

// applyLDAPConnection applies connection-related fields from the spec to the
// LDAP config.
func applyLDAPConnection(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
	helpers.AssignIfNonNil(&cfg.AuthUserName, spec.AuthUsername)
	helpers.AssignIfNonNil(&cfg.AuthRealm, spec.AuthRealm)
	helpers.AssignIfNonNil(&cfg.ConnectionTimeoutSeconds, spec.ConnectionTimeoutSeconds)
	helpers.AssignIfNonNil(&cfg.ConnectionRetryDelaySeconds, spec.ConnectionRetryDelaySeconds)
	helpers.AssignIfNonNil(&cfg.MaxIncidentCount, spec.MaxIncidentCount)
	helpers.AssignIfNonNil(&cfg.UseTrustStore, spec.UseTrustStore)
}

// applyLDAPUserConfig applies user-mapping fields from the spec to the LDAP
// config.
func applyLDAPUserConfig(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
	helpers.AssignIfNonNil(&cfg.UserSubtree, spec.UserSubtree)
	helpers.AssignIfNonNil(&cfg.UserObjectClass, spec.UserObjectClass)
	helpers.AssignIfNonNil(&cfg.UserIDAttribute, spec.UserIDAttribute)
	helpers.AssignIfNonNil(&cfg.UserRealNameAttribute, spec.UserRealNameAttribute)
	helpers.AssignIfNonNil(&cfg.UserEmailAddressAttribute, spec.UserEmailAddressAttribute)
	helpers.AssignIfNonNil(&cfg.UserPasswordAttribute, spec.UserPasswordAttribute)
	helpers.AssignIfNonNil(&cfg.UserMemberOfAttribute, spec.UserMemberOfAttribute)
	helpers.AssignIfNonNil(&cfg.UserLDAPFilter, spec.UserLDAPFilter)
}

// applyLDAPGroupConfig applies group-mapping fields from the spec to the
// LDAP config.
func applyLDAPGroupConfig(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
	if spec.LDAPGroupsAsRoles == nil {
		return
	}

	cfg.LDAPGroupsAsRoles = *spec.LDAPGroupsAsRoles

	helpers.AssignIfNonNil(&cfg.GroupType, spec.GroupType)
	helpers.AssignIfNonNil(&cfg.GroupBaseDn, spec.GroupBaseDN)
	helpers.AssignIfNonNil(&cfg.GroupSubtree, spec.GroupSubtree)
	helpers.AssignIfNonNil(&cfg.GroupObjectClass, spec.GroupObjectClass)
	helpers.AssignIfNonNil(&cfg.GroupIDAttribute, spec.GroupIDAttribute)
	helpers.AssignIfNonNil(&cfg.GroupMemberAttribute, spec.GroupMemberAttribute)
	helpers.AssignIfNonNil(&cfg.GroupMemberFormat, spec.GroupMemberFormat)
}
