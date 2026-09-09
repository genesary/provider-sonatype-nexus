package iam

import (
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// newPrivilegeCR returns a Privilege CR of the given type whose observation
// already matches its spec.
func newPrivilegeCR(privType string, mutate func(*iamv1alpha1.PrivilegeParameters)) *iamv1alpha1.Privilege {
	cr := &iamv1alpha1.Privilege{}
	cr.Spec.ForProvider = iamv1alpha1.PrivilegeParameters{
		Name:        "priv",
		Type:        privType,
		Description: new("a privilege"),
		Actions:     []string{"READ", "BROWSE"},
	}

	mutate(&cr.Spec.ForProvider)

	spec := cr.Spec.ForProvider
	cr.Status.AtProvider = GeneratePrivilegeObservation(&security.Privilege{
		Name:            spec.Name,
		Type:            spec.Type,
		Description:     ptr.Deref(spec.Description, ""),
		Actions:         spec.Actions,
		Domain:          ptr.Deref(spec.Domain, ""),
		Format:          ptr.Deref(spec.Format, ""),
		Repository:      ptr.Deref(spec.Repository, ""),
		ContentSelector: ptr.Deref(spec.ContentSelector, ""),
		ScriptName:      ptr.Deref(spec.ScriptName, ""),
		Pattern:         ptr.Deref(spec.Pattern, ""),
	})

	return cr
}

// TestGeneratePrivilegeObservation_Nil tests that a nil privilege produces an
// empty observation.
func TestGeneratePrivilegeObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := GeneratePrivilegeObservation(nil)
	if obs.ReadOnly != nil || obs.Name != "" || obs.Actions != nil {
		t.Errorf("GeneratePrivilegeObservation(nil) = %+v, want the zero observation", obs)
	}
}

// TestGeneratePrivilegeObservation_TypeSpecificFields tests that the type
// specific fields Nexus reports reach the observation.
func TestGeneratePrivilegeObservation_TypeSpecificFields(t *testing.T) {
	t.Parallel()

	obs := GeneratePrivilegeObservation(&security.Privilege{
		Name:            "priv",
		Type:            "repository-content-selector",
		Description:     "a privilege",
		Actions:         []string{"READ"},
		Format:          "maven2",
		Repository:      "maven-central",
		ContentSelector: "selector",
		ReadOnly:        true,
	})

	if obs.Name != "priv" || obs.Type != "repository-content-selector" {
		t.Errorf("identity = %q/%q, want priv/repository-content-selector", obs.Name, obs.Type)
	}

	if obs.Format != "maven2" || obs.Repository != "maven-central" || obs.ContentSelector != "selector" {
		t.Errorf("type specific fields = %+v, want maven2/maven-central/selector", obs)
	}

	if obs.ReadOnly == nil || !*obs.ReadOnly {
		t.Errorf("ReadOnly = %v, want true", obs.ReadOnly)
	}
}

// TestIsPrivilegeUpToDate_TypeSpecificDrift tests that a change to a field the
// privilege type carries is reported as drift. Only the description and the
// actions were compared before the observation carried these fields.
func TestIsPrivilegeUpToDate_TypeSpecificDrift(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		privType string
		mutate   func(*iamv1alpha1.PrivilegeParameters)
		drift    func(*iamv1alpha1.PrivilegeObservation)
	}{
		"ApplicationDomain": {
			privType: privilegeTypeApplication,
			mutate:   func(p *iamv1alpha1.PrivilegeParameters) { p.Domain = new("users") },
			drift:    func(o *iamv1alpha1.PrivilegeObservation) { o.Domain = "roles" },
		},
		"RepositoryViewRepository": {
			privType: privilegeTypeRepoView,
			mutate: func(p *iamv1alpha1.PrivilegeParameters) {
				p.Format = new("maven2")
				p.Repository = new("maven-central")
			},
			drift: func(o *iamv1alpha1.PrivilegeObservation) { o.Repository = "maven-releases" },
		},
		"RepositoryAdminFormat": {
			privType: privilegeTypeRepoAdmin,
			mutate: func(p *iamv1alpha1.PrivilegeParameters) {
				p.Format = new("maven2")
				p.Repository = new("maven-central")
			},
			drift: func(o *iamv1alpha1.PrivilegeObservation) { o.Format = "npm" },
		},
		"ContentSelector": {
			privType: privilegeTypeRepoContentSelector,
			mutate: func(p *iamv1alpha1.PrivilegeParameters) {
				p.Format = new("maven2")
				p.Repository = new("maven-central")
				p.ContentSelector = new("selector")
			},
			drift: func(o *iamv1alpha1.PrivilegeObservation) { o.ContentSelector = "other" },
		},
		"ScriptName": {
			privType: privilegeTypeScript,
			mutate:   func(p *iamv1alpha1.PrivilegeParameters) { p.ScriptName = new("script") },
			drift:    func(o *iamv1alpha1.PrivilegeObservation) { o.ScriptName = "other" },
		},
		"WildcardPattern": {
			privType: privilegeTypeWildcard,
			mutate:   func(p *iamv1alpha1.PrivilegeParameters) { p.Pattern = new("nexus:*") },
			drift:    func(o *iamv1alpha1.PrivilegeObservation) { o.Pattern = "nexus:repository-view:*" },
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cr := newPrivilegeCR(tt.privType, tt.mutate)
			if !IsPrivilegeUpToDate(cr) {
				t.Fatal("IsPrivilegeUpToDate() = false, want true before the drift")
			}

			tt.drift(&cr.Status.AtProvider)

			if IsPrivilegeUpToDate(cr) {
				t.Error("IsPrivilegeUpToDate() = true, want false after the drift")
			}
		})
	}
}

// TestIsPrivilegeUpToDate_OtherTypesFieldsIgnored tests that a field belonging
// to another privilege type is not compared: the payload builders never submit
// it, so Nexus reports it empty.
func TestIsPrivilegeUpToDate_OtherTypesFieldsIgnored(t *testing.T) {
	t.Parallel()

	cr := newPrivilegeCR(privilegeTypeWildcard, func(p *iamv1alpha1.PrivilegeParameters) {
		p.Pattern = new("nexus:*")
		p.Repository = new("maven-central")
	})

	if !IsPrivilegeUpToDate(cr) {
		t.Error("IsPrivilegeUpToDate() = false, want true for a field the type does not carry")
	}
}

// TestIsPrivilegeUpToDate_TypeDrift tests that a privilege whose type no
// longer matches the spec is reported as drift.
func TestIsPrivilegeUpToDate_TypeDrift(t *testing.T) {
	t.Parallel()

	cr := newPrivilegeCR(privilegeTypeScript, func(p *iamv1alpha1.PrivilegeParameters) {
		p.ScriptName = new("script")
	})
	cr.Status.AtProvider.Type = privilegeTypeWildcard

	if IsPrivilegeUpToDate(cr) {
		t.Error("IsPrivilegeUpToDate() = true, want false when the privilege type differs")
	}
}

// TestIsPrivilegeUpToDate_DescriptionDropped tests that removing the
// description from the spec is reported as drift.
func TestIsPrivilegeUpToDate_DescriptionDropped(t *testing.T) {
	t.Parallel()

	cr := newPrivilegeCR(privilegeTypeWildcard, func(p *iamv1alpha1.PrivilegeParameters) {
		p.Pattern = new("nexus:*")
	})
	cr.Spec.ForProvider.Description = nil

	if IsPrivilegeUpToDate(cr) {
		t.Error("IsPrivilegeUpToDate() = true, want false when the description was dropped")
	}
}
