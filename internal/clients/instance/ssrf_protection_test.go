package instance

import (
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// TestIsSSRFProtectionUpToDate covers drift detection of the SSRF protection
// settings, including the set semantics Nexus applies to the exceptions.
func TestIsSSRFProtectionUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   instancev1alpha1.SSRFProtectionParameters
		observed v395.SsrfProtectionConfigurationXO
		want     bool
	}{
		{
			name:     "UnsetEnabledMatchesDefault",
			params:   instancev1alpha1.SSRFProtectionParameters{},
			observed: v395.SsrfProtectionConfigurationXO{Enabled: true},
			want:     true,
		},
		{
			name:     "UnsetEnabledDetectsDisabled",
			params:   instancev1alpha1.SSRFProtectionParameters{},
			observed: v395.SsrfProtectionConfigurationXO{Enabled: false},
			want:     false,
		},
		{
			name:     "EnabledDiffers",
			params:   instancev1alpha1.SSRFProtectionParameters{Enabled: new(false)},
			observed: v395.SsrfProtectionConfigurationXO{Enabled: true},
			want:     false,
		},
		{
			// Nexus reorders and de-duplicates the exceptions it stores.
			name: "ExceptionsMatchAsSets",
			params: instancev1alpha1.SSRFProtectionParameters{
				AllowedIPs:     []string{"10.0.0.50", "192.168.1.100"},
				AllowedDomains: []string{"registry.local", "internal.corp.com"},
			},
			observed: v395.SsrfProtectionConfigurationXO{
				Enabled:        true,
				AllowedIPs:     []string{"192.168.1.100", "10.0.0.50"},
				AllowedDomains: []string{"internal.corp.com", "registry.local"},
			},
			want: true,
		},
		{
			name:   "ExtraObservedIP",
			params: instancev1alpha1.SSRFProtectionParameters{AllowedIPs: []string{"10.0.0.50"}},
			observed: v395.SsrfProtectionConfigurationXO{
				Enabled:    true,
				AllowedIPs: []string{"10.0.0.50", "10.0.0.51"},
			},
			want: false,
		},
		{
			name:     "MissingObservedDomain",
			params:   instancev1alpha1.SSRFProtectionParameters{AllowedDomains: []string{"registry.local"}},
			observed: v395.SsrfProtectionConfigurationXO{Enabled: true, AllowedDomains: []string{}},
			want:     false,
		},
		{
			// Nexus keeps the case of domains, so a case change is drift.
			name:     "DomainCaseDiffers",
			params:   instancev1alpha1.SSRFProtectionParameters{AllowedDomains: []string{"registry.local"}},
			observed: v395.SsrfProtectionConfigurationXO{Enabled: true, AllowedDomains: []string{"Registry.LOCAL"}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := &instancev1alpha1.SSRFProtection{}
			cr.Spec.ForProvider = tt.params

			got := IsSSRFProtectionUpToDate(cr, &tt.observed)
			if got != tt.want {
				t.Errorf("IsSSRFProtectionUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateSSRFProtectionUpdate verifies the spec defaults to enabled
// protection and passes the exceptions through.
func TestGenerateSSRFProtectionUpdate(t *testing.T) {
	t.Parallel()

	got := GenerateSSRFProtectionUpdate(&instancev1alpha1.SSRFProtectionParameters{
		AllowedIPs:     []string{"10.0.0.50"},
		AllowedDomains: []string{"registry.local"},
	})

	if !got.Enabled {
		t.Error("Enabled = false, want true when the spec leaves it unset")
	}

	if len(got.AllowedIPs) != 1 || got.AllowedIPs[0] != "10.0.0.50" {
		t.Errorf("AllowedIPs = %v, want [10.0.0.50]", got.AllowedIPs)
	}

	if len(got.AllowedDomains) != 1 || got.AllowedDomains[0] != "registry.local" {
		t.Errorf("AllowedDomains = %v, want [registry.local]", got.AllowedDomains)
	}
}

// TestIsDefaultSSRFProtection verifies only enabled protection without
// exceptions counts as the Nexus default.
func TestIsDefaultSSRFProtection(t *testing.T) {
	t.Parallel()

	defaults := DefaultSSRFProtection()
	if !IsDefaultSSRFProtection(&defaults) {
		t.Error("IsDefaultSSRFProtection(DefaultSSRFProtection()) = false, want true")
	}

	for name, observed := range map[string]v395.SsrfProtectionConfigurationXO{
		"Disabled":   {Enabled: false},
		"AllowedIP":  {Enabled: true, AllowedIPs: []string{"10.0.0.50"}},
		"AllowedDNS": {Enabled: true, AllowedDomains: []string{"registry.local"}},
	} {
		if IsDefaultSSRFProtection(&observed) {
			t.Errorf("IsDefaultSSRFProtection(%s) = true, want false", name)
		}
	}
}
