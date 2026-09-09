package instance

import (
	"testing"

	nexuscapability "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"
	nexusiq "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// TestGenerateCapabilityObservation_AllFields tests that the capability
// settings Nexus reports reach the observation, not only the assigned ID.
func TestGenerateCapabilityObservation_AllFields(t *testing.T) {
	t.Parallel()

	obs := GenerateCapabilityObservation(&nexuscapability.Capability{
		ID:         "cap-1",
		Type:       "baseurl",
		Notes:      "managed by crossplane",
		Enabled:    true,
		Properties: map[string]string{"url": "https://nexus.example.com"},
	})

	if obs.ID != "cap-1" || obs.TypeId != "baseurl" {
		t.Errorf("identity = %q/%q, want cap-1/baseurl", obs.ID, obs.TypeId)
	}

	if !obs.Enabled || obs.Notes != "managed by crossplane" {
		t.Errorf("observation = %+v, want an enabled capability with notes", obs)
	}

	if obs.Properties["url"] != "https://nexus.example.com" {
		t.Errorf("Properties = %v, want the reported url", obs.Properties)
	}
}

// TestGenerateCapabilityObservation_PropertiesAreCopied tests that the
// observation does not alias the map the client returned.
func TestGenerateCapabilityObservation_PropertiesAreCopied(t *testing.T) {
	t.Parallel()

	properties := map[string]string{"url": "https://nexus.example.com"}

	obs := GenerateCapabilityObservation(&nexuscapability.Capability{Properties: properties})
	properties["url"] = "https://other.example.com"

	if obs.Properties["url"] != "https://nexus.example.com" {
		t.Errorf("Properties = %v, want the value observed at the time", obs.Properties)
	}
}

// TestIsIQServerUpToDate_URLDrift tests that an IQ Server URL that no longer
// matches the spec is reported as drift, including when Nexus reports none.
func TestIsIQServerUpToDate_URLDrift(t *testing.T) {
	t.Parallel()

	cr := &instancev1alpha1.IQServerConfiguration{}
	cr.Spec.ForProvider = instancev1alpha1.IQServerConfigurationParameters{
		URL:     "https://iq.example.com",
		Enabled: new(true),
	}

	matching := &nexusiq.IQServerConfiguration{
		Enabled: true,
		URL:     new("https://iq.example.com"),
	}
	if !IsIQServerUpToDate(cr, matching) {
		t.Fatal("IsIQServerUpToDate() = false, want true for a matching URL")
	}

	drifted := &nexusiq.IQServerConfiguration{
		Enabled: true,
		URL:     new("https://other.example.com"),
	}
	if IsIQServerUpToDate(cr, drifted) {
		t.Error("IsIQServerUpToDate() = true, want false for a differing URL")
	}

	unreported := &nexusiq.IQServerConfiguration{Enabled: true}
	if IsIQServerUpToDate(cr, unreported) {
		t.Error("IsIQServerUpToDate() = true, want false when Nexus reports no URL")
	}
}
