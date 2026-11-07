package instance_test

import (
	"testing"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// newCR returns a minimal Capability CR for tests.
func newCR(typeID string, enabled bool, notes string, props map[string]string) *instancev1alpha1.Capability {
	return &instancev1alpha1.Capability{
		ObjectMeta: metav1.ObjectMeta{Name: "cap"},
		Spec: instancev1alpha1.CapabilitySpec{
			ForProvider: instancev1alpha1.CapabilityParameters{
				TypeId:     typeID,
				Enabled:    enabled,
				Notes:      notes,
				Properties: props,
			},
		},
	}
}

// TestGenerateCapabilityCreate tests GenerateCapabilityCreate.
func TestGenerateCapabilityCreate(t *testing.T) {
	t.Parallel()

	cr := newCR("DockerBearerTokenRealm", true, "notes", map[string]string{"key": "val"})

	got := instanceclient.GenerateCapabilityCreate(&cr.Spec.ForProvider)

	if got.GetType() != "DockerBearerTokenRealm" {
		t.Errorf("Type = %q, want DockerBearerTokenRealm", got.GetType())
	}

	if !got.GetEnabled() {
		t.Error("Enabled = false, want true")
	}

	if got.GetNotes() != "notes" {
		t.Errorf("Notes = %q, want notes", got.GetNotes())
	}

	if got.GetProperties()["key"] != "val" {
		t.Errorf("Properties[key] = %q, want val", got.GetProperties()["key"])
	}
}

// TestGenerateCapabilityCreate_NilProperties tests GenerateCapabilityCreate
// with nil properties.
func TestGenerateCapabilityCreate_NilProperties(t *testing.T) {
	t.Parallel()

	cr := newCR("DockerBearerTokenRealm", true, "", nil)

	got := instanceclient.GenerateCapabilityCreate(&cr.Spec.ForProvider)

	if got.GetProperties() == nil {
		t.Error("Properties should not be nil")
	}

	if len(got.GetProperties()) != 0 {
		t.Errorf("Properties len = %d, want 0", len(got.GetProperties()))
	}
}

// TestGenerateCapabilityUpdate tests GenerateCapabilityUpdate.
func TestGenerateCapabilityUpdate(t *testing.T) {
	t.Parallel()

	cr := newCR("DockerBearerTokenRealm", false, "updated", map[string]string{"a": "b"})

	got := instanceclient.GenerateCapabilityUpdate(&cr.Spec.ForProvider, "id-xyz")

	if got.GetId() != "id-xyz" {
		t.Errorf("ID = %q, want id-xyz", got.GetId())
	}

	if got.GetType() != "DockerBearerTokenRealm" {
		t.Errorf("Type = %q, want DockerBearerTokenRealm", got.GetType())
	}

	if got.GetEnabled() {
		t.Error("Enabled = true, want false")
	}

	if got.GetNotes() != "updated" {
		t.Errorf("Notes = %q, want updated", got.GetNotes())
	}

	if got.GetProperties()["a"] != "b" {
		t.Errorf("Properties[a] = %q, want b", got.GetProperties()["a"])
	}
}

// TestGenerateCapabilityUpdate_NilProperties tests GenerateCapabilityUpdate
// with nil properties.
func TestGenerateCapabilityUpdate_NilProperties(t *testing.T) {
	t.Parallel()

	cr := newCR("DockerBearerTokenRealm", true, "", nil)

	got := instanceclient.GenerateCapabilityUpdate(&cr.Spec.ForProvider, "id-abc")

	if got.GetProperties() == nil {
		t.Error("Properties should not be nil")
	}
}

// TestIsCapabilityUpToDate tests IsCapabilityUpToDate across all comparison
// branches.
func TestIsCapabilityUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cr       *instancev1alpha1.Capability
		observed *instanceclient.Capability
		want     bool
	}{
		{
			name: "UpToDate",
			cr:   newCR("DockerBearerTokenRealm", true, "notes", map[string]string{"k": "v"}),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(true),
				Notes:      new("notes"),
				Properties: new(map[string]string{"k": "v"}),
			},
			want: true,
		},
		{
			name: "DifferentType",
			cr:   newCR("DockerBearerTokenRealm", true, "", nil),
			observed: &instanceclient.Capability{
				Type:       new("NuGetApiKey"),
				Enabled:    new(true),
				Properties: new(map[string]string{}),
			},
			want: false,
		},
		{
			name: "DifferentEnabled",
			cr:   newCR("DockerBearerTokenRealm", true, "", nil),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(false),
				Properties: new(map[string]string{}),
			},
			want: false,
		},
		{
			name: "DifferentNotes",
			cr:   newCR("DockerBearerTokenRealm", true, "new", nil),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(true),
				Notes:      new("old"),
				Properties: new(map[string]string{}),
			},
			want: false,
		},
		{
			name: "DifferentProperties",
			cr:   newCR("DockerBearerTokenRealm", true, "", map[string]string{"k": "v"}),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(true),
				Properties: new(map[string]string{}),
			},
			want: false,
		},
		{
			name: "NilDesiredPropertiesMatchesEmpty",
			cr:   newCR("DockerBearerTokenRealm", true, "", nil),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(true),
				Properties: new(map[string]string{}),
			},
			want: true,
		},
		{
			name: "NilObservedPropertiesMatchesEmpty",
			cr:   newCR("DockerBearerTokenRealm", true, "", map[string]string{}),
			observed: &instanceclient.Capability{
				Type:       new("DockerBearerTokenRealm"),
				Enabled:    new(true),
				Properties: nil,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := instanceclient.IsCapabilityUpToDate(tt.cr, tt.observed)
			if got != tt.want {
				t.Errorf("IsCapabilityUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateCapabilityObservation tests GenerateCapabilityObservation.
func TestGenerateCapabilityObservation(t *testing.T) {
	t.Parallel()

	obs := instanceclient.GenerateCapabilityObservation(&instanceclient.Capability{
		Id:   new("abc123"),
		Type: new("DockerBearerTokenRealm"),
	})

	if obs.ID != "abc123" {
		t.Errorf("ID = %q, want abc123", obs.ID)
	}
}

// TestGenerateCapabilityObservation_Nil tests GenerateCapabilityObservation
// with nil input.
func TestGenerateCapabilityObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := instanceclient.GenerateCapabilityObservation(nil)

	if obs.ID != "" {
		t.Errorf("ID = %q, want empty for nil input", obs.ID)
	}
}
