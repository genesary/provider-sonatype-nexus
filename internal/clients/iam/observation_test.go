package iam

import (
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// newUserCR returns a User CR whose observation matches the given Nexus user.
func newUserCR(observed *security.User) *iamv1alpha1.User {
	cr := &iamv1alpha1.User{}
	cr.Spec.ForProvider = iamv1alpha1.UserParameters{
		UserID:       observed.UserID,
		FirstName:    observed.FirstName,
		LastName:     observed.LastName,
		EmailAddress: observed.EmailAddress,
		Roles:        observed.Roles,
	}
	cr.Status.AtProvider = GenerateUserObservation(observed)

	return cr
}

// TestGenerateUserObservation_IdentityFields tests that the user ID and the
// authentication source reach the observation.
func TestGenerateUserObservation_IdentityFields(t *testing.T) {
	t.Parallel()

	obs := GenerateUserObservation(&security.User{
		UserID: "alice",
		Source: "LDAP",
		Status: "active",
	})

	if obs.UserID != "alice" || obs.Source != "LDAP" {
		t.Errorf("observation = %+v, want userID alice from source LDAP", obs)
	}
}

// TestIsUserUpToDate_UnsetStatusUsesDefault tests that an unset status is
// compared against the default GenerateUser submits, rather than ignored.
func TestIsUserUpToDate_UnsetStatusUsesDefault(t *testing.T) {
	t.Parallel()

	cr := newUserCR(&security.User{UserID: "alice", Status: "active", Source: "default"})
	if !IsUserUpToDate(cr) {
		t.Fatal("IsUserUpToDate() = false, want true for a user in the default state")
	}

	cr.Status.AtProvider.Status = "disabled"

	if IsUserUpToDate(cr) {
		t.Error("IsUserUpToDate() = true, want false for a user disabled out of band")
	}
}

// TestIsUserUpToDate_SourceDrift tests that a user whose authentication source
// no longer matches the spec is reported as drift.
func TestIsUserUpToDate_SourceDrift(t *testing.T) {
	t.Parallel()

	cr := newUserCR(&security.User{UserID: "alice", Status: "active", Source: "default"})
	cr.Spec.ForProvider.Source = new("LDAP")

	if IsUserUpToDate(cr) {
		t.Error("IsUserUpToDate() = true, want false for a differing source")
	}
}

// TestGenerateRoleObservation_ID tests that the role ID reaches the
// observation.
func TestGenerateRoleObservation_ID(t *testing.T) {
	t.Parallel()

	obs := GenerateRoleObservation(&security.Role{ID: "nx-admin", Name: "admin"})
	if obs.ID != "nx-admin" {
		t.Errorf("ID = %q, want nx-admin", obs.ID)
	}
}

// TestIsRoleUpToDate_DescriptionDropped tests that removing the description
// from the spec is reported as drift: GenerateRole submits an empty one, which
// is how Nexus is told to clear it.
func TestIsRoleUpToDate_DescriptionDropped(t *testing.T) {
	t.Parallel()

	cr := &iamv1alpha1.Role{}
	cr.Spec.ForProvider = iamv1alpha1.RoleParameters{ID: "nx-admin", Name: "admin"}
	cr.Status.AtProvider = GenerateRoleObservation(&security.Role{
		ID:          "nx-admin",
		Name:        "admin",
		Description: "leftover",
	})

	if IsRoleUpToDate(cr) {
		t.Error("IsRoleUpToDate() = true, want false when the description was dropped")
	}
}

// TestIsUserTokenConfigUpToDate_UnsetOptionalsAreFalse tests that an optional
// setting removed from the spec is reported as drift: the Nexus payload
// carries every setting on every write, so an unset one submits false.
func TestIsUserTokenConfigUpToDate_UnsetOptionalsAreFalse(t *testing.T) {
	t.Parallel()

	cr := &iamv1alpha1.UserTokenConfiguration{}
	cr.Spec.ForProvider = iamv1alpha1.UserTokenConfigurationParameters{Enabled: true}
	cr.Status.AtProvider = GenerateUserTokenConfigObservation(&security.UserTokenConfiguration{
		Enabled:        true,
		ProtectContent: true,
	})

	if IsUserTokenConfigUpToDate(cr) {
		t.Error("IsUserTokenConfigUpToDate() = true, want false when protectContent was dropped")
	}

	cr.Status.AtProvider.ProtectContent = false

	if !IsUserTokenConfigUpToDate(cr) {
		t.Error("IsUserTokenConfigUpToDate() = false, want true once the setting is off")
	}
}

// TestIsUserTokenConfigUpToDate_ExpirationDaysDropped tests that an expiration
// removed from the spec is reported as drift.
func TestIsUserTokenConfigUpToDate_ExpirationDaysDropped(t *testing.T) {
	t.Parallel()

	cr := &iamv1alpha1.UserTokenConfiguration{}
	cr.Spec.ForProvider = iamv1alpha1.UserTokenConfigurationParameters{
		Enabled:           true,
		ExpirationEnabled: new(true),
	}
	cr.Status.AtProvider = GenerateUserTokenConfigObservation(&security.UserTokenConfiguration{
		Enabled:           true,
		ExpirationEnabled: true,
		ExpirationDays:    30,
	})

	if IsUserTokenConfigUpToDate(cr) {
		t.Error("IsUserTokenConfigUpToDate() = true, want false when expirationDays was dropped")
	}
}
