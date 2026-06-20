package nexus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"
)

// capabilitiesPath is the Nexus REST API path for capabilities.
const capabilitiesPath = "/service/rest/v1/capabilities"

// TestCapabilityService_GetCapability_Success tests GetCapability for a
// successful retrieval.
func TestCapabilityService_GetCapability_Success(t *testing.T) {
	t.Parallel()

	want := &nexussdk.Capability{
		ID:         "abc123",
		Type:       "DockerBearerTokenRealm",
		Enabled:    true,
		Properties: map[string]string{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == capabilitiesPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]nexussdk.Capability{*want})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.Capability().GetCapability(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetCapability() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetCapability() returned nil")
	}

	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}

	if got.Type != want.Type {
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
}

// TestCapabilityService_GetCapability_NotFound tests that GetCapability returns
// nil for missing IDs.
func TestCapabilityService_GetCapability_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == capabilitiesPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]nexussdk.Capability{})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	// Get returns (nil, nil) when the ID is not in the list.
	got, err := c.Capability().GetCapability(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetCapability() unexpected error: %v", err)
	}

	if got != nil {
		t.Fatalf("GetCapability() = %v, want nil for nonexistent ID", got)
	}
}

// TestCapabilityService_GetCapability_Error tests GetCapability when the
// server returns an error.
func TestCapabilityService_GetCapability_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.Capability().GetCapability(context.Background(), "abc123")
	if err == nil {
		t.Fatal("GetCapability() expected error for non-200 status, got nil")
	}
}

// TestCapabilityService_CreateCapability_Success tests CreateCapability for
// a successful creation.
func TestCapabilityService_CreateCapability_Success(t *testing.T) {
	t.Parallel()

	created := &nexussdk.Capability{
		ID:         "new-id-123",
		Type:       "DockerBearerTokenRealm",
		Enabled:    true,
		Properties: map[string]string{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == capabilitiesPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(created)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	create := nexussdk.CapabilityCreate{
		Type:       "DockerBearerTokenRealm",
		Enabled:    true,
		Properties: map[string]string{},
	}

	got, err := c.Capability().CreateCapability(context.Background(), create)
	if err != nil {
		t.Fatalf("CreateCapability() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("CreateCapability() returned nil")
	}

	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

// TestCapabilityService_CreateCapability_Error tests CreateCapability when
// the server errors.
func TestCapabilityService_CreateCapability_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.Capability().CreateCapability(context.Background(), nexussdk.CapabilityCreate{
		Type:       "DockerBearerTokenRealm",
		Properties: map[string]string{},
	})
	if err == nil {
		t.Fatal("CreateCapability() expected error for non-200 status, got nil")
	}
}

// TestCapabilityService_UpdateCapability_Success tests UpdateCapability for
// a successful update.
func TestCapabilityService_UpdateCapability_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == capabilitiesPath+"/abc123" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Capability().UpdateCapability(context.Background(), "abc123", nexussdk.CapabilityUpdate{
		ID:         "abc123",
		Type:       "DockerBearerTokenRealm",
		Enabled:    true,
		Properties: map[string]string{},
	})
	if err != nil {
		t.Fatalf("UpdateCapability() unexpected error: %v", err)
	}
}

// TestCapabilityService_UpdateCapability_Error tests UpdateCapability when
// the server errors.
func TestCapabilityService_UpdateCapability_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "update failed")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Capability().UpdateCapability(context.Background(), "abc123", nexussdk.CapabilityUpdate{
		ID:   "abc123",
		Type: "DockerBearerTokenRealm",
	})
	if err == nil {
		t.Fatal("UpdateCapability() expected error for non-204 status, got nil")
	}
}

// TestCapabilityService_DeleteCapability_Success tests DeleteCapability for
// a successful deletion.
func TestCapabilityService_DeleteCapability_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == capabilitiesPath+"/abc123" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Capability().DeleteCapability(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("DeleteCapability() unexpected error: %v", err)
	}
}

// TestCapabilityService_DeleteCapability_Error tests DeleteCapability when
// the server errors.
func TestCapabilityService_DeleteCapability_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "delete failed")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Capability().DeleteCapability(context.Background(), "abc123")
	if err == nil {
		t.Fatal("DeleteCapability() expected error for non-204 status, got nil")
	}
}
