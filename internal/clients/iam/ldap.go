package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// LDAPClient manages Nexus LDAP server configurations.
type LDAPClient interface {
	GetLDAP(ctx context.Context, name string) (*security.LDAP, error)
	CreateLDAP(ctx context.Context, ldap security.LDAP) error
	UpdateLDAP(ctx context.Context, name string, ldap security.LDAP) error
	DeleteLDAP(ctx context.Context, name string) error
}

// NewLDAPClient returns a new LDAPClient.
func NewLDAPClient(creds nexus.Credentials) (LDAPClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
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

	obs := iamv1alpha1.LDAPObservation{}

	if observed.ID != "" {
		obs.ID = &observed.ID
	}

	return obs
}

// IsLDAPUpToDate reports whether the CR matches the observed LDAP config.
func IsLDAPUpToDate(ldapCR *iamv1alpha1.LDAP, observed *security.LDAP) bool {
	if ldapCR.Spec.ForProvider.Protocol != observed.Protocol {
		return false
	}

	if ldapCR.Spec.ForProvider.Host != observed.Host {
		return false
	}

	if ldapCR.Spec.ForProvider.Port != observed.Port {
		return false
	}

	if ldapCR.Spec.ForProvider.SearchBase != observed.SearchBase {
		return false
	}

	if ldapCR.Spec.ForProvider.AuthScheme != observed.AuthSchema {
		return false
	}

	if ldapCR.Spec.ForProvider.UserBaseDN != observed.UserBaseDN {
		return false
	}

	return true
}

// applyLDAPConnection applies connection-related fields from the spec to the
// LDAP config.
func applyLDAPConnection(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
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
func applyLDAPUserConfig(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
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

// applyLDAPGroupConfig applies group-mapping fields from the spec to the
// LDAP config.
func applyLDAPGroupConfig(cfg *security.LDAP, spec *iamv1alpha1.LDAPParameters) {
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
