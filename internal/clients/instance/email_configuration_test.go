package instance_test

import (
	"testing"

	mailschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// newBasicCR returns a minimal EmailConfiguration CR for testing.
func newBasicCR() *instancev1alpha1.EmailConfiguration {
	return &instancev1alpha1.EmailConfiguration{
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Enabled:     true,
				Host:        "smtp.example.com",
				Port:        587,
				FromAddress: "nexus@example.com",
			},
		},
	}
}

// TestNewEmailConfigurationClient verifies a client can be created with valid
// credentials.
func TestNewEmailConfigurationClient(t *testing.T) {
	t.Parallel()

	creds := nexus.Credentials{
		URL:      "http://localhost:8081",
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	emailClient, err := instanceclient.NewEmailConfigurationClient(creds)
	if err != nil {
		t.Fatalf("NewEmailConfigurationClient() unexpected error: %v", err)
	}

	if emailClient == nil {
		t.Error("NewEmailConfigurationClient() returned nil client")
	}
}

// TestGenerateEmailConfiguration_Basic tests basic field mapping.
func TestGenerateEmailConfiguration_Basic(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cfg := instanceclient.GenerateEmailConfiguration(cr, "")

	if cfg.Enabled == nil || !*cfg.Enabled {
		t.Error("Enabled should be true")
	}

	if cfg.Host != "smtp.example.com" {
		t.Errorf("Host = %q, want smtp.example.com", cfg.Host)
	}

	if cfg.Port != 587 {
		t.Errorf("Port = %d, want 587", cfg.Port)
	}

	if cfg.FromAddress != "nexus@example.com" {
		t.Errorf("FromAddress = %q, want nexus@example.com", cfg.FromAddress)
	}
}

// TestGenerateEmailConfiguration_WithPassword tests password field mapping.
func TestGenerateEmailConfiguration_WithPassword(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cfg := instanceclient.GenerateEmailConfiguration(cr, "secret123")

	if cfg.Password == nil || *cfg.Password != "secret123" {
		t.Errorf("Password = %v, want %q", cfg.Password, "secret123")
	}
}

// TestGenerateEmailConfiguration_WithUsername verifies username is set.
func TestGenerateEmailConfiguration_WithUsername(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	user := "smtp-user"
	cr.Spec.ForProvider.Username = &user
	cfg := instanceclient.GenerateEmailConfiguration(cr, "")

	if cfg.Username == nil || *cfg.Username != "smtp-user" {
		t.Errorf("Username = %v, want %q", cfg.Username, "smtp-user")
	}
}

// TestGenerateEmailConfiguration_OptionalFields verifies optional bool/string
// fields are passed through correctly.
func TestGenerateEmailConfiguration_OptionalFields(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	pfx := "[Nexus]"
	tls := true
	tlsReq := false
	sslConn := false
	sslID := true
	trust := false
	cr.Spec.ForProvider.SubjectPrefix = &pfx
	cr.Spec.ForProvider.StartTlsEnabled = &tls
	cr.Spec.ForProvider.StartTlsRequired = &tlsReq
	cr.Spec.ForProvider.SslOnConnectEnabled = &sslConn
	cr.Spec.ForProvider.SslServerIdentityCheckEnabled = &sslID
	cr.Spec.ForProvider.NexusTrustStoreEnabled = &trust

	cfg := instanceclient.GenerateEmailConfiguration(cr, "")

	if cfg.SubjectPrefix == nil || *cfg.SubjectPrefix != "[Nexus]" {
		t.Errorf("SubjectPrefix = %v, want %q", cfg.SubjectPrefix, "[Nexus]")
	}

	if cfg.StartTlsEnabled == nil || !*cfg.StartTlsEnabled {
		t.Error("StartTlsEnabled should be true")
	}

	if cfg.StartTlsRequired == nil || *cfg.StartTlsRequired {
		t.Error("StartTlsRequired should be false")
	}

	if cfg.SslOnConnectEnabled == nil || *cfg.SslOnConnectEnabled {
		t.Error("SslOnConnectEnabled should be false")
	}

	if cfg.SslServerIdentityCheckEnabled == nil || !*cfg.SslServerIdentityCheckEnabled {
		t.Error("SslServerIdentityCheckEnabled should be true")
	}

	if cfg.NexusTrustStoreEnabled == nil || *cfg.NexusTrustStoreEnabled {
		t.Error("NexusTrustStoreEnabled should be false")
	}
}

// TestGenerateEmailConfigurationObservation_AllFields verifies all fields are
// mapped from the API response to the observation.
func TestGenerateEmailConfigurationObservation_AllFields(t *testing.T) {
	t.Parallel()

	enabled := true
	user := "smtp-user"
	pfx := "[Nexus]"
	tls := true
	tlsReq := false
	sslConnect := false
	sslID := true
	trustStore := false

	config := &mailschema.MailConfig{
		Enabled:                       &enabled,
		Host:                          "smtp.example.com",
		Port:                          587,
		FromAddress:                   "nexus@example.com",
		Username:                      &user,
		SubjectPrefix:                 &pfx,
		StartTlsEnabled:               &tls,
		StartTlsRequired:              &tlsReq,
		SslOnConnectEnabled:           &sslConnect,
		SslServerIdentityCheckEnabled: &sslID,
		NexusTrustStoreEnabled:        &trustStore,
	}

	obs := instanceclient.GenerateEmailConfigurationObservation(config)

	if !obs.Enabled {
		t.Error("Enabled should be true")
	}

	if obs.Host != "smtp.example.com" {
		t.Errorf("Host = %q, want smtp.example.com", obs.Host)
	}

	if obs.Port != 587 {
		t.Errorf("Port = %d, want 587", obs.Port)
	}

	if obs.FromAddress != "nexus@example.com" {
		t.Errorf("FromAddress = %q, want nexus@example.com", obs.FromAddress)
	}

	if obs.Username != "smtp-user" {
		t.Errorf("Username = %q, want smtp-user", obs.Username)
	}

	if obs.SubjectPrefix != "[Nexus]" {
		t.Errorf("SubjectPrefix = %q, want [Nexus]", obs.SubjectPrefix)
	}

	if !obs.StartTlsEnabled {
		t.Error("StartTlsEnabled should be true")
	}

	if obs.StartTlsRequired {
		t.Error("StartTlsRequired should be false")
	}

	if obs.SslOnConnectEnabled {
		t.Error("SslOnConnectEnabled should be false")
	}

	if !obs.SslServerIdentityCheckEnabled {
		t.Error("SslServerIdentityCheckEnabled should be true")
	}

	if obs.NexusTrustStoreEnabled {
		t.Error("NexusTrustStoreEnabled should be false")
	}
}

// TestGenerateEmailConfigurationObservation_NilPointers verifies nil pointer
// fields in MailConfig produce zero-value observations.
func TestGenerateEmailConfigurationObservation_NilPointers(t *testing.T) {
	t.Parallel()

	enabled := true
	config := &mailschema.MailConfig{
		Enabled:     &enabled,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
		// all optional pointer fields nil
	}

	obs := instanceclient.GenerateEmailConfigurationObservation(config)

	if obs.Username != "" {
		t.Errorf("Username = %q, want empty", obs.Username)
	}

	if obs.SubjectPrefix != "" {
		t.Errorf("SubjectPrefix = %q, want empty", obs.SubjectPrefix)
	}

	if obs.StartTlsEnabled {
		t.Error("StartTlsEnabled should be false (zero)")
	}
}

// TestGenerateEmailConfigurationObservation_Nil verifies nil input returns
// an empty observation.
func TestGenerateEmailConfigurationObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := instanceclient.GenerateEmailConfigurationObservation(nil)

	if obs.Host != "" || obs.Port != 0 || obs.Enabled {
		t.Errorf("expected zero-value observation, got %+v", obs)
	}
}

// TestIsEmailConfigurationUpToDate_UpToDate verifies matching spec/status
// returns true.
func TestIsEmailConfigurationUpToDate_UpToDate(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	if !instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = false, want true")
	}
}

// TestIsEmailConfigurationUpToDate_HostDiffers verifies host diff returns
// false.
func TestIsEmailConfigurationUpToDate_HostDiffers(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     true,
		Host:        "other.smtp.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	if instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = true, want false (host differs)")
	}
}

// TestIsEmailConfigurationUpToDate_PortDiffers verifies port diff returns
// false.
func TestIsEmailConfigurationUpToDate_PortDiffers(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        25,
		FromAddress: "nexus@example.com",
	}

	if instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = true, want false (port differs)")
	}
}

// TestIsEmailConfigurationUpToDate_EnabledDiffers verifies enabled diff
// returns false.
func TestIsEmailConfigurationUpToDate_EnabledDiffers(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     false,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	if instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = true, want false (enabled differs)")
	}
}

// TestIsEmailConfigurationUpToDate_UsernameDiffers verifies username diff
// returns false.
func TestIsEmailConfigurationUpToDate_UsernameDiffers(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	user := "admin"
	cr.Spec.ForProvider.Username = &user
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
		Username:    "other",
	}

	if instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = true, want false (username differs)")
	}
}

// TestIsEmailConfigurationUpToDate_SubjectPrefixDiffers verifies subject
// prefix diff returns false.
func TestIsEmailConfigurationUpToDate_SubjectPrefixDiffers(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	pfx := "[Nexus]"
	cr.Spec.ForProvider.SubjectPrefix = &pfx
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:       true,
		Host:          "smtp.example.com",
		Port:          587,
		FromAddress:   "nexus@example.com",
		SubjectPrefix: "[Old]",
	}

	if instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = true, want false")
	}
}

// TestIsEmailConfigurationUpToDate_TlsFlagsDiffer verifies TLS flag drift.
func TestIsEmailConfigurationUpToDate_TlsFlagsDiffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*instancev1alpha1.EmailConfiguration)
	}{
		{
			name: "StartTlsEnabledDiffers",
			mutate: func(cr *instancev1alpha1.EmailConfiguration) {
				v := true
				cr.Spec.ForProvider.StartTlsEnabled = &v
				cr.Status.AtProvider.StartTlsEnabled = false
			},
		},
		{
			name: "StartTlsRequiredDiffers",
			mutate: func(cr *instancev1alpha1.EmailConfiguration) {
				v := true
				cr.Spec.ForProvider.StartTlsRequired = &v
				cr.Status.AtProvider.StartTlsRequired = false
			},
		},
		{
			name: "SslOnConnectEnabledDiffers",
			mutate: func(cr *instancev1alpha1.EmailConfiguration) {
				v := true
				cr.Spec.ForProvider.SslOnConnectEnabled = &v
				cr.Status.AtProvider.SslOnConnectEnabled = false
			},
		},
		{
			name: "SslServerIdentityCheckEnabledDiffers",
			mutate: func(cr *instancev1alpha1.EmailConfiguration) {
				v := true
				cr.Spec.ForProvider.SslServerIdentityCheckEnabled = &v
				cr.Status.AtProvider.SslServerIdentityCheckEnabled = false
			},
		},
		{
			name: "NexusTrustStoreEnabledDiffers",
			mutate: func(cr *instancev1alpha1.EmailConfiguration) {
				v := true
				cr.Spec.ForProvider.NexusTrustStoreEnabled = &v
				cr.Status.AtProvider.NexusTrustStoreEnabled = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := newBasicCR()
			cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
				Enabled:     true,
				Host:        "smtp.example.com",
				Port:        587,
				FromAddress: "nexus@example.com",
			}
			tt.mutate(cr)

			if instanceclient.IsEmailConfigurationUpToDate(cr) {
				t.Errorf("IsEmailConfigurationUpToDate() = true, want false (%s)", tt.name)
			}
		})
	}
}

// TestIsEmailConfigurationUpToDate_NilPtrMatchesZero verifies nil spec pointer
// matches the zero-value observation.
func TestIsEmailConfigurationUpToDate_NilPtrMatchesZero(t *testing.T) {
	t.Parallel()

	cr := newBasicCR()
	// spec has nil StartTlsEnabled, status has false (zero) — should match
	cr.Status.AtProvider = instancev1alpha1.EmailConfigurationObservation{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	if !instanceclient.IsEmailConfigurationUpToDate(cr) {
		t.Error("IsEmailConfigurationUpToDate() = false, want true (nil ptr matches zero)")
	}
}
