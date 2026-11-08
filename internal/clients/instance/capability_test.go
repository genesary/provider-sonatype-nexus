package instance_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"

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

// TestCapabilityCreate_OnlyOne checks that Create reports Nexus refusing a
// second capability of a single-instance type as ErrCapabilityExists, and
// other failures as plain errors. The bodies are what Nexus 3.96 answers.
func TestCapabilityCreate_OnlyOne(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		wantExists bool
	}{
		{
			name:       "OnlyOne",
			status:     http.StatusInternalServerError,
			body:       `"jakarta.validation.ValidationException: Only one capability of type 'Audit' can be created"`,
			wantExists: true,
		},
		{
			name:       "OtherServerError",
			status:     http.StatusInternalServerError,
			body:       `"java.lang.NullPointerException"`,
			wantExists: false,
		},
		{
			name:       "ValidationError",
			status:     http.StatusBadRequest,
			body:       `[{"id":"DefaultRoleCapabilityConfiguration.role","message":"Missing role"}]`,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := instanceclient.NewCapabilityClient(nexus.Credentials{URL: server.URL, Username: "admin", Password: "password"})
			if err != nil {
				t.Fatalf("NewCapabilityClient() error = %v", err)
			}

			_, err = client.Create(context.Background(), instanceclient.GenerateCapabilityCreate(&newCR("audit", true, "", nil).Spec.ForProvider))
			if err == nil {
				t.Fatal("Create() error = nil, want an error")
			}

			if got := errors.Is(err, instanceclient.ErrCapabilityExists); got != tt.wantExists {
				t.Errorf("errors.Is(err, ErrCapabilityExists) = %v, want %v (err: %v)", got, tt.wantExists, err)
			}
		})
	}
}

// TestIsSameCapabilityInstance checks which existing capability a new one
// would collide with.
func TestIsSameCapabilityInstance(t *testing.T) {
	t.Parallel()

	webhook := func(repository, url string) map[string]string {
		return map[string]string{"repository": repository, "url": url, "names": "component"}
	}

	tests := []struct {
		name     string
		params   *instancev1alpha1.CapabilityParameters
		observed instanceclient.Capability
		want     bool
	}{
		{
			name:     "SingletonSameType",
			params:   &newCR("audit", true, "", nil).Spec.ForProvider,
			observed: instanceclient.Capability{Type: new("audit"), Properties: &map[string]string{"enabled": "x"}},
			want:     true,
		},
		{
			name:     "OtherType",
			params:   &newCR("audit", true, "", nil).Spec.ForProvider,
			observed: instanceclient.Capability{Type: new("baseurl")},
			want:     false,
		},
		{
			name:     "MultiInstanceSameIdentity",
			params:   &newCR("webhook.repository", true, "", webhook("maven", "https://a")).Spec.ForProvider,
			observed: instanceclient.Capability{Type: new("webhook.repository"), Properties: new(webhook("maven", "https://a"))},
			want:     true,
		},
		{
			name:     "MultiInstanceOtherRepository",
			params:   &newCR("webhook.repository", true, "", webhook("maven", "https://a")).Spec.ForProvider,
			observed: instanceclient.Capability{Type: new("webhook.repository"), Properties: new(webhook("npm", "https://a"))},
			want:     false,
		},
		{
			name:     "MultiInstanceOtherURL",
			params:   &newCR("webhook.repository", true, "", webhook("maven", "https://a")).Spec.ForProvider,
			observed: instanceclient.Capability{Type: new("webhook.repository"), Properties: new(webhook("maven", "https://b"))},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := instanceclient.IsSameCapabilityInstance(tt.params, &tt.observed); got != tt.want {
				t.Errorf("IsSameCapabilityInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}
