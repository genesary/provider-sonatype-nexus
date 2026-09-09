package iam

import (
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// newLDAPServer returns the LDAP server Nexus reports for a fully configured
// server, so that a test can change one field and assert it is compared.
func newLDAPServer() *security.LDAP {
	return &security.LDAP{
		ID:                          "ldap-id",
		Name:                        "corp",
		Protocol:                    "ldaps",
		Host:                        "ldap.example.com",
		Port:                        636,
		SearchBase:                  "dc=example,dc=com",
		AuthSchema:                  "simple",
		AuthUserName:                "cn=admin,dc=example,dc=com",
		AuthRealm:                   "EXAMPLE",
		ConnectionTimeoutSeconds:    30,
		ConnectionRetryDelaySeconds: 300,
		MaxIncidentCount:            3,
		UseTrustStore:               true,
		UserBaseDN:                  "ou=people",
		UserSubtree:                 true,
		UserObjectClass:             "inetOrgPerson",
		UserIDAttribute:             "uid",
		UserRealNameAttribute:       "cn",
		UserEmailAddressAttribute:   "mail",
		UserPasswordAttribute:       "userPassword",
		UserMemberOfAttribute:       "memberOf",
		UserLDAPFilter:              "(uid=dom*)",
		LDAPGroupsAsRoles:           true,
		GroupType:                   "static",
		GroupBaseDn:                 "ou=groups",
		GroupSubtree:                true,
		GroupObjectClass:            "groupOfUniqueNames",
		GroupIDAttribute:            "cn",
		GroupMemberAttribute:        "uniqueMember",
		GroupMemberFormat:           "uid=${username},ou=people,dc=example,dc=com",
	}
}

// newLDAPCR returns an LDAP CR whose spec asks for exactly the server
// newLDAPServer reports.
func newLDAPCR() *iamv1alpha1.LDAP {
	server := newLDAPServer()

	cr := &iamv1alpha1.LDAP{}
	cr.Spec.ForProvider = iamv1alpha1.LDAPParameters{
		Name:                        server.Name,
		Protocol:                    server.Protocol,
		Host:                        server.Host,
		Port:                        server.Port,
		SearchBase:                  server.SearchBase,
		AuthScheme:                  server.AuthSchema,
		AuthUsername:                &server.AuthUserName,
		AuthRealm:                   &server.AuthRealm,
		ConnectionTimeoutSeconds:    &server.ConnectionTimeoutSeconds,
		ConnectionRetryDelaySeconds: &server.ConnectionRetryDelaySeconds,
		MaxIncidentCount:            &server.MaxIncidentCount,
		UseTrustStore:               &server.UseTrustStore,
		UserBaseDN:                  server.UserBaseDN,
		UserSubtree:                 &server.UserSubtree,
		UserObjectClass:             &server.UserObjectClass,
		UserIDAttribute:             &server.UserIDAttribute,
		UserRealNameAttribute:       &server.UserRealNameAttribute,
		UserEmailAddressAttribute:   &server.UserEmailAddressAttribute,
		UserPasswordAttribute:       &server.UserPasswordAttribute,
		UserMemberOfAttribute:       &server.UserMemberOfAttribute,
		UserLDAPFilter:              &server.UserLDAPFilter,
		LDAPGroupsAsRoles:           &server.LDAPGroupsAsRoles,
		GroupType:                   &server.GroupType,
		GroupBaseDN:                 &server.GroupBaseDn,
		GroupSubtree:                &server.GroupSubtree,
		GroupObjectClass:            &server.GroupObjectClass,
		GroupIDAttribute:            &server.GroupIDAttribute,
		GroupMemberAttribute:        &server.GroupMemberAttribute,
		GroupMemberFormat:           &server.GroupMemberFormat,
	}
	cr.Status.AtProvider = GenerateLDAPObservation(newLDAPServer())

	return cr
}

// TestGenerateLDAPObservation_Nil tests that a nil LDAP server produces an
// empty observation.
func TestGenerateLDAPObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := GenerateLDAPObservation(nil)
	if obs != (iamv1alpha1.LDAPObservation{}) {
		t.Errorf("GenerateLDAPObservation(nil) = %+v, want the zero observation", obs)
	}
}

// TestGenerateLDAPObservation_AllFields tests that every field Nexus reports
// reaches the observation: a field left out of it can never be seen to drift.
func TestGenerateLDAPObservation_AllFields(t *testing.T) {
	t.Parallel()

	server := newLDAPServer()
	obs := GenerateLDAPObservation(server)

	if obs.ID == nil || *obs.ID != server.ID {
		t.Errorf("ID = %v, want %q", obs.ID, server.ID)
	}

	checks := map[string][2]any{
		"Name":                        {obs.Name, server.Name},
		"AuthScheme":                  {obs.AuthScheme, server.AuthSchema},
		"AuthUsername":                {obs.AuthUsername, server.AuthUserName},
		"AuthRealm":                   {obs.AuthRealm, server.AuthRealm},
		"ConnectionTimeoutSeconds":    {obs.ConnectionTimeoutSeconds, server.ConnectionTimeoutSeconds},
		"ConnectionRetryDelaySeconds": {obs.ConnectionRetryDelaySeconds, server.ConnectionRetryDelaySeconds},
		"MaxIncidentCount":            {obs.MaxIncidentCount, server.MaxIncidentCount},
		"UseTrustStore":               {obs.UseTrustStore, server.UseTrustStore},
		"UserSubtree":                 {obs.UserSubtree, server.UserSubtree},
		"UserObjectClass":             {obs.UserObjectClass, server.UserObjectClass},
		"UserIDAttribute":             {obs.UserIDAttribute, server.UserIDAttribute},
		"UserRealNameAttribute":       {obs.UserRealNameAttribute, server.UserRealNameAttribute},
		"UserEmailAddressAttribute":   {obs.UserEmailAddressAttribute, server.UserEmailAddressAttribute},
		"UserPasswordAttribute":       {obs.UserPasswordAttribute, server.UserPasswordAttribute},
		"UserMemberOfAttribute":       {obs.UserMemberOfAttribute, server.UserMemberOfAttribute},
		"UserLDAPFilter":              {obs.UserLDAPFilter, server.UserLDAPFilter},
		"LDAPGroupsAsRoles":           {obs.LDAPGroupsAsRoles, server.LDAPGroupsAsRoles},
		"GroupType":                   {obs.GroupType, server.GroupType},
		"GroupBaseDN":                 {obs.GroupBaseDN, server.GroupBaseDn},
		"GroupSubtree":                {obs.GroupSubtree, server.GroupSubtree},
		"GroupObjectClass":            {obs.GroupObjectClass, server.GroupObjectClass},
		"GroupIDAttribute":            {obs.GroupIDAttribute, server.GroupIDAttribute},
		"GroupMemberAttribute":        {obs.GroupMemberAttribute, server.GroupMemberAttribute},
		"GroupMemberFormat":           {obs.GroupMemberFormat, server.GroupMemberFormat},
	}

	for field, values := range checks {
		if values[0] != values[1] {
			t.Errorf("%s = %v, want %v", field, values[0], values[1])
		}
	}
}

// TestIsLDAPUpToDate_AllMatch tests that a spec asking for exactly what Nexus
// reports is up to date.
func TestIsLDAPUpToDate_AllMatch(t *testing.T) {
	t.Parallel()

	if !IsLDAPUpToDate(newLDAPCR()) {
		t.Error("IsLDAPUpToDate() = false, want true when the spec matches the server")
	}
}

// TestIsLDAPUpToDate_DriftPerField tests that a change to any managed field is
// reported as drift. Before the observation carried these fields, only the
// connection settings were compared and every other change went unnoticed.
func TestIsLDAPUpToDate_DriftPerField(t *testing.T) {
	t.Parallel()

	drifts := map[string]func(*iamv1alpha1.LDAPObservation){
		"Host":                      func(o *iamv1alpha1.LDAPObservation) { o.Host = "other.example.com" },
		"Port":                      func(o *iamv1alpha1.LDAPObservation) { o.Port = 389 },
		"AuthUsername":              func(o *iamv1alpha1.LDAPObservation) { o.AuthUsername = "cn=other" },
		"ConnectionTimeoutSeconds":  func(o *iamv1alpha1.LDAPObservation) { o.ConnectionTimeoutSeconds = 60 },
		"MaxIncidentCount":          func(o *iamv1alpha1.LDAPObservation) { o.MaxIncidentCount = 9 },
		"UseTrustStore":             func(o *iamv1alpha1.LDAPObservation) { o.UseTrustStore = false },
		"UserBaseDN":                func(o *iamv1alpha1.LDAPObservation) { o.UserBaseDN = "ou=other" },
		"UserObjectClass":           func(o *iamv1alpha1.LDAPObservation) { o.UserObjectClass = "person" },
		"UserIDAttribute":           func(o *iamv1alpha1.LDAPObservation) { o.UserIDAttribute = "sAMAccountName" },
		"UserEmailAddressAttribute": func(o *iamv1alpha1.LDAPObservation) { o.UserEmailAddressAttribute = "email" },
		"UserLDAPFilter":            func(o *iamv1alpha1.LDAPObservation) { o.UserLDAPFilter = "" },
		"LDAPGroupsAsRoles":         func(o *iamv1alpha1.LDAPObservation) { o.LDAPGroupsAsRoles = false },
		"GroupType":                 func(o *iamv1alpha1.LDAPObservation) { o.GroupType = "dynamic" },
		"GroupBaseDN":               func(o *iamv1alpha1.LDAPObservation) { o.GroupBaseDN = "ou=other" },
		"GroupMemberFormat":         func(o *iamv1alpha1.LDAPObservation) { o.GroupMemberFormat = "cn=${username}" },
	}

	for field, drift := range drifts {
		t.Run(field, func(t *testing.T) {
			t.Parallel()

			cr := newLDAPCR()
			drift(&cr.Status.AtProvider)

			if IsLDAPUpToDate(cr) {
				t.Errorf("IsLDAPUpToDate() = true, want false after %s drifted", field)
			}
		})
	}
}

// TestIsLDAPUpToDate_UnsetOptionalsIgnored tests that an optional field the
// spec leaves unset is not compared: Nexus fills those in with its own
// defaults, which would otherwise read as permanent drift.
func TestIsLDAPUpToDate_UnsetOptionalsIgnored(t *testing.T) {
	t.Parallel()

	cr := newLDAPCR()
	cr.Spec.ForProvider.UserObjectClass = nil
	cr.Spec.ForProvider.UserLDAPFilter = nil
	cr.Status.AtProvider.UserObjectClass = "somethingElse"
	cr.Status.AtProvider.UserLDAPFilter = "(objectClass=*)"

	if !IsLDAPUpToDate(cr) {
		t.Error("IsLDAPUpToDate() = false, want true when the spec sets no value")
	}
}

// TestIsLDAPUpToDate_GroupConfigIgnoredWhenGroupsAsRolesUnset tests that the
// group settings are not compared when the spec does not ask for LDAP groups
// as roles: applyLDAPGroupConfig submits none of them in that case.
func TestIsLDAPUpToDate_GroupConfigIgnoredWhenGroupsAsRolesUnset(t *testing.T) {
	t.Parallel()

	cr := newLDAPCR()
	cr.Spec.ForProvider.LDAPGroupsAsRoles = nil
	cr.Spec.ForProvider.GroupType = new("dynamic")
	cr.Status.AtProvider.GroupType = "static"

	if !IsLDAPUpToDate(cr) {
		t.Error("IsLDAPUpToDate() = false, want true when ldapGroupsAsRoles is unset")
	}
}
