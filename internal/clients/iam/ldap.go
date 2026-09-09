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
//
// Every field Nexus reports is recorded, not only the ones that identify the
// server: drift detection compares the spec against this observation, so a
// field left out of it can never be seen to drift.
//
// The bind password is deliberately absent - Nexus never returns it, so it
// takes no part in the observation nor in drift detection.
func GenerateLDAPObservation(observed *security.LDAP) iamv1alpha1.LDAPObservation {
	if observed == nil {
		return iamv1alpha1.LDAPObservation{}
	}

	obs := iamv1alpha1.LDAPObservation{
		Name:                        observed.Name,
		Protocol:                    observed.Protocol,
		Host:                        observed.Host,
		Port:                        observed.Port,
		SearchBase:                  observed.SearchBase,
		AuthScheme:                  observed.AuthSchema,
		AuthUsername:                observed.AuthUserName,
		AuthRealm:                   observed.AuthRealm,
		ConnectionTimeoutSeconds:    observed.ConnectionTimeoutSeconds,
		ConnectionRetryDelaySeconds: observed.ConnectionRetryDelaySeconds,
		MaxIncidentCount:            observed.MaxIncidentCount,
		UseTrustStore:               observed.UseTrustStore,
		UserBaseDN:                  observed.UserBaseDN,
		UserSubtree:                 observed.UserSubtree,
		UserObjectClass:             observed.UserObjectClass,
		UserIDAttribute:             observed.UserIDAttribute,
		UserRealNameAttribute:       observed.UserRealNameAttribute,
		UserEmailAddressAttribute:   observed.UserEmailAddressAttribute,
		UserPasswordAttribute:       observed.UserPasswordAttribute,
		UserMemberOfAttribute:       observed.UserMemberOfAttribute,
		UserLDAPFilter:              observed.UserLDAPFilter,
		LDAPGroupsAsRoles:           observed.LDAPGroupsAsRoles,
		GroupType:                   observed.GroupType,
		GroupBaseDN:                 observed.GroupBaseDn,
		GroupSubtree:                observed.GroupSubtree,
		GroupObjectClass:            observed.GroupObjectClass,
		GroupIDAttribute:            observed.GroupIDAttribute,
		GroupMemberAttribute:        observed.GroupMemberAttribute,
		GroupMemberFormat:           observed.GroupMemberFormat,
	}

	if observed.ID != "" {
		obs.ID = &observed.ID
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
