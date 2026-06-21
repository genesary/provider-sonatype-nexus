package nexus_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mailschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// mailConfigAPIPath is the Nexus REST API path for email configuration.
const mailConfigAPIPath = "/service/rest/v1/email"

// TestMailConfigService_GetEmailConfiguration_Success verifies
// GetEmailConfiguration returns config on HTTP 200.
func TestMailConfigService_GetEmailConfiguration_Success(t *testing.T) {
	t.Parallel()

	enabled := true
	want := &mailschema.MailConfig{
		Enabled:     &enabled,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mailConfigAPIPath || r.Method != http.MethodGet {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.MailConfig().GetEmailConfiguration(context.Background())
	if err != nil {
		t.Fatalf("GetEmailConfiguration() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetEmailConfiguration() returned nil")
	}

	if got.Host != want.Host {
		t.Errorf("Host = %q, want %q", got.Host, want.Host)
	}

	if got.Port != want.Port {
		t.Errorf("Port = %d, want %d", got.Port, want.Port)
	}
}

// TestMailConfigService_GetEmailConfiguration_Error verifies
// GetEmailConfiguration returns an error on HTTP 500.
func TestMailConfigService_GetEmailConfiguration_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.MailConfig().GetEmailConfiguration(context.Background())
	if err == nil {
		t.Error("GetEmailConfiguration() should fail on server error")
	}
}

// TestMailConfigService_UpdateEmailConfiguration_Success verifies
// UpdateEmailConfiguration succeeds on HTTP 204.
func TestMailConfigService_UpdateEmailConfiguration_Success(t *testing.T) {
	t.Parallel()

	enabled := true
	config := mailschema.MailConfig{
		Enabled:     &enabled,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != mailConfigAPIPath || r.Method != http.MethodPut {
			http.NotFound(w, r)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.MailConfig().UpdateEmailConfiguration(context.Background(), config)
	if err != nil {
		t.Fatalf("UpdateEmailConfiguration() unexpected error: %v", err)
	}
}

// TestMailConfigService_UpdateEmailConfiguration_Error verifies
// UpdateEmailConfiguration returns an error on HTTP 500.
func TestMailConfigService_UpdateEmailConfiguration_Error(t *testing.T) {
	t.Parallel()

	enabled := true
	config := mailschema.MailConfig{
		Enabled:     &enabled,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "nexus@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.MailConfig().UpdateEmailConfiguration(context.Background(), config)
	if err == nil {
		t.Error("UpdateEmailConfiguration() should fail on server error")
	}
}

// TestNewEmailConfigurationClient_ViaRealClient verifies that the
// MailConfig() method on nexus.Client satisfies nexus.MailConfigService.
func TestNewEmailConfigurationClient_ViaRealClient(t *testing.T) {
	t.Parallel()

	c, err := nexus.NewClient(nexus.Credentials{
		URL:      "http://localhost:8081",
		Username: "admin",
		Password: "admin",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	_ = c.MailConfig()
}
