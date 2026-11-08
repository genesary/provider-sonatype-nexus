package instance

import (
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// newTestEmailParameters returns a complete email server spec.
func newTestEmailParameters() instancev1alpha1.EmailConfigurationParameters {
	return instancev1alpha1.EmailConfigurationParameters{
		Enabled:                       new(true),
		Host:                          "smtp.example.com",
		Port:                          587,
		Username:                      new("nexus"),
		FromAddress:                   "nexus@example.com",
		SubjectPrefix:                 new("[Nexus]"),
		StartTLSEnabled:               new(true),
		StartTLSRequired:              new(true),
		SSLServerIdentityCheckEnabled: new(true),
	}
}

// newTestEmailConfiguration returns the configuration Nexus reports for
// newTestEmailParameters: never with the password.
func newTestEmailConfiguration() v395.ApiEmailConfiguration {
	return v395.ApiEmailConfiguration{
		Enabled:                       new(true),
		Host:                          "smtp.example.com",
		Port:                          587,
		Username:                      new("nexus"),
		FromAddress:                   "nexus@example.com",
		SubjectPrefix:                 new("[Nexus]"),
		StartTlsEnabled:               new(true),
		StartTlsRequired:              new(true),
		SslOnConnectEnabled:           new(false),
		SslServerIdentityCheckEnabled: new(true),
		NexusTrustStoreEnabled:        new(false),
	}
}

// TestIsEmailConfigurationUpToDate covers drift detection of every email
// server setting, and the equivalence of the unset forms Nexus reports.
func TestIsEmailConfigurationUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   func(*instancev1alpha1.EmailConfigurationParameters)
		observed func(*v395.ApiEmailConfiguration)
		want     bool
	}{
		{name: "UpToDate", want: true},
		{
			name:     "UnsetEnabledMatchesEnabled",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.Enabled = nil },
			observed: func(*v395.ApiEmailConfiguration) {},
			want:     true,
		},
		{
			name:     "EnabledDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.Enabled = new(false) },
		},
		{
			name:     "HostDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.Host = "smtp.other.com" },
		},
		{
			name:     "PortDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.Port = 25 },
		},
		{
			name:     "UsernameDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.Username = new("other") },
		},
		{
			// Nexus reports an omitted user name as an empty string.
			name:     "UnsetUsernameMatchesEmpty",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.Username = nil },
			observed: func(o *v395.ApiEmailConfiguration) { o.Username = new("") },
			want:     true,
		},
		{
			name:     "UnsetUsernameMatchesNull",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.Username = nil },
			observed: func(o *v395.ApiEmailConfiguration) { o.Username = nil },
			want:     true,
		},
		{
			name:     "UnwantedUsername",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.Username = nil },
			observed: func(*v395.ApiEmailConfiguration) {},
		},
		{
			name:     "FromAddressDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.FromAddress = "other@example.com" },
		},
		{
			// Nexus reports an omitted subject prefix as null.
			name:     "UnsetSubjectPrefixMatchesNull",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.SubjectPrefix = nil },
			observed: func(o *v395.ApiEmailConfiguration) { o.SubjectPrefix = nil },
			want:     true,
		},
		{
			name:     "EmptySubjectPrefixMatchesNull",
			params:   func(p *instancev1alpha1.EmailConfigurationParameters) { p.SubjectPrefix = new("") },
			observed: func(o *v395.ApiEmailConfiguration) { o.SubjectPrefix = nil },
			want:     true,
		},
		{
			name:     "SubjectPrefixDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.SubjectPrefix = new("[Other]") },
		},
		{
			name:     "StartTLSEnabledDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.StartTlsEnabled = new(false) },
		},
		{
			name:     "StartTLSRequiredDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.StartTlsRequired = nil },
		},
		{
			// An unset boolean asks for false.
			name:     "UnsetSSLOnConnectDetectsEnabled",
			observed: func(o *v395.ApiEmailConfiguration) { o.SslOnConnectEnabled = new(true) },
		},
		{
			name:     "SSLServerIdentityCheckDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.SslServerIdentityCheckEnabled = new(false) },
		},
		{
			name:     "NexusTrustStoreDiffers",
			observed: func(o *v395.ApiEmailConfiguration) { o.NexusTrustStoreEnabled = new(true) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := &instancev1alpha1.EmailConfiguration{}
			cr.Spec.ForProvider = newTestEmailParameters()

			if tt.params != nil {
				tt.params(&cr.Spec.ForProvider)
			}

			observed := newTestEmailConfiguration()
			if tt.observed != nil {
				tt.observed(&observed)
			}

			got := IsEmailConfigurationUpToDate(cr, &observed)
			if got != tt.want {
				t.Errorf("IsEmailConfigurationUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateEmailConfigurationUpdate verifies the spec is applied in full,
// with unset booleans as false, and the password only when resolved.
func TestGenerateEmailConfigurationUpdate(t *testing.T) {
	t.Parallel()

	t.Run("WithPassword", func(t *testing.T) {
		t.Parallel()

		params := newTestEmailParameters()

		got := GenerateEmailConfigurationUpdate(&params, "secret")

		cr := &instancev1alpha1.EmailConfiguration{}
		cr.Spec.ForProvider = params

		if !IsEmailConfigurationUpToDate(cr, &got) {
			t.Errorf("GenerateEmailConfigurationUpdate() = %+v, which does not match its own spec", got)
		}

		if got.GetPassword() != "secret" {
			t.Errorf("Password = %q, want %q", got.GetPassword(), "secret")
		}
	})

	t.Run("MinimalSpec", func(t *testing.T) {
		t.Parallel()

		params := instancev1alpha1.EmailConfigurationParameters{
			Host:        "smtp.example.com",
			Port:        25,
			FromAddress: "nexus@example.com",
		}

		got := GenerateEmailConfigurationUpdate(&params, "")

		if !got.GetEnabled() {
			t.Error("Enabled = false, want the default true")
		}

		if got.Password != nil || got.Username != nil || got.SubjectPrefix != nil {
			t.Errorf("Password, Username, SubjectPrefix = %v, %v, %v, want them unset", got.Password, got.Username, got.SubjectPrefix)
		}

		for name, value := range map[string]*bool{
			"StartTlsEnabled":        got.StartTlsEnabled,
			"StartTlsRequired":       got.StartTlsRequired,
			"SslOnConnectEnabled":    got.SslOnConnectEnabled,
			"NexusTrustStoreEnabled": got.NexusTrustStoreEnabled,
		} {
			if value == nil || *value {
				t.Errorf("%s = %v, want an explicit false", name, value)
			}
		}

		// Certificate verification is on unless the spec turns it off, so
		// enabling TLS never silently accepts any certificate.
		if value := got.SslServerIdentityCheckEnabled; value == nil || !*value {
			t.Errorf("SslServerIdentityCheckEnabled = %v, want the default true", value)
		}
	})
}

// TestIsResetEmailConfiguration verifies only the configuration Nexus holds
// after a reset is recognised, not the placeholder of a fresh instance.
func TestIsResetEmailConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		observed v395.ApiEmailConfiguration
		want     bool
	}{
		{
			name:     "Reset",
			observed: v395.ApiEmailConfiguration{Enabled: new(false), StartTlsEnabled: new(false)},
			want:     true,
		},
		{
			name:     "ResetWithoutEnabled",
			observed: v395.ApiEmailConfiguration{},
			want:     true,
		},
		{
			name:     "FreshInstancePlaceholder",
			observed: v395.ApiEmailConfiguration{Enabled: new(false), Host: "localhost", Port: 25, FromAddress: "nexus@example.org"},
		},
		{
			name:     "Configured",
			observed: newTestEmailConfiguration(),
		},
		{
			name:     "EnabledWithoutHost",
			observed: v395.ApiEmailConfiguration{Enabled: new(true)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsResetEmailConfiguration(&tt.observed)
			if got != tt.want {
				t.Errorf("IsResetEmailConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateEmailConfigurationObservation verifies the observation maps
// every reported setting, and nil as an empty observation.
func TestGenerateEmailConfigurationObservation(t *testing.T) {
	t.Parallel()

	empty := GenerateEmailConfigurationObservation(nil)
	if empty != (instancev1alpha1.EmailConfigurationObservation{}) {
		t.Errorf("GenerateEmailConfigurationObservation(nil) = %+v, want empty", empty)
	}

	observed := newTestEmailConfiguration()

	got := GenerateEmailConfigurationObservation(&observed)

	want := instancev1alpha1.EmailConfigurationObservation{
		Enabled:                       true,
		Host:                          "smtp.example.com",
		Port:                          587,
		Username:                      "nexus",
		FromAddress:                   "nexus@example.com",
		SubjectPrefix:                 "[Nexus]",
		StartTLSEnabled:               true,
		StartTLSRequired:              true,
		SSLServerIdentityCheckEnabled: true,
	}
	if got != want {
		t.Errorf("GenerateEmailConfigurationObservation() = %+v, want %+v", got, want)
	}
}
