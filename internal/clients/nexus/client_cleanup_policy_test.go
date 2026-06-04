package nexus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// newTestClient creates a new nexus.Client that points
// to the given test server URL.
func newTestClient(t *testing.T, serverURL string) nexus.Client {
	t.Helper()

	c, err := nexus.NewClient(nexus.Credentials{
		URL:      serverURL,
		Username: "test",
		Password: "test",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	return c
}

// cleanupPoliciesPath is the base path for cleanup policy
// endpoints in the Nexus API.
const cleanupPoliciesPath = "/service/rest/v1/cleanup-policies"

// TestCleanupPolicyService_GetCleanupPolicy_Success tests that GetCleanupPolicy
// successfully retrieves a cleanup policy when the server responds with a 200
// status code and a valid JSON body.
func TestCleanupPolicyService_GetCleanupPolicy_Success(t *testing.T) {
	t.Parallel()

	want := &cleanuppolicies.CleanupPolicy{
		Name:   "test-policy",
		Format: cleanuppolicies.RepositoryFormatMaven2,
		Notes:  new("a test policy"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == cleanupPoliciesPath+"/test-policy" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.CleanupPolicy().GetCleanupPolicy(context.Background(), "test-policy")
	if err != nil {
		t.Fatalf("GetCleanupPolicy() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetCleanupPolicy() returned nil policy")
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	if got.Format != want.Format {
		t.Errorf("Format = %q, want %q", got.Format, want.Format)
	}
}

// TestCleanupPolicyService_GetCleanupPolicy_Error tests that GetCleanupPolicy
// returns an error when the server responds with a non-200 status code.
func TestCleanupPolicyService_GetCleanupPolicy_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.CleanupPolicy().GetCleanupPolicy(context.Background(), "missing-policy")
	if err == nil {
		t.Fatal("GetCleanupPolicy() expected error for non-200 status, got nil")
	}
}

// TestCleanupPolicyService_CreateCleanupPolicy_Success tests that
// CreateCleanupPolicy successfully creates a cleanup policy when the server
// responds with a 201 status code.
func TestCleanupPolicyService_CreateCleanupPolicy_Success(t *testing.T) {
	t.Parallel()

	policy := &cleanuppolicies.CleanupPolicy{
		Name:   "new-policy",
		Format: cleanuppolicies.RepositoryFormatNpm,
	}

	var received *cleanuppolicies.CleanupPolicy

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == cleanupPoliciesPath {
			received = &cleanuppolicies.CleanupPolicy{}
			_ = json.NewDecoder(r.Body).Decode(received)

			w.WriteHeader(http.StatusCreated)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().CreateCleanupPolicy(context.Background(), policy)
	if err != nil {
		t.Fatalf("CreateCleanupPolicy() unexpected error: %v", err)
	}

	if received == nil {
		t.Fatal("CreateCleanupPolicy() server did not receive request body")
	}

	if received.Name != policy.Name {
		t.Errorf("received Name = %q, want %q", received.Name, policy.Name)
	}
}

// TestCleanupPolicyService_CreateCleanupPolicy_Error tests that
// CreateCleanupPolicy returns an error when the server responds with a non-201
// status code.
func TestCleanupPolicyService_CreateCleanupPolicy_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid policy")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().CreateCleanupPolicy(context.Background(), &cleanuppolicies.CleanupPolicy{
		Name:   "bad",
		Format: cleanuppolicies.RepositoryFormatNpm,
	})
	if err == nil {
		t.Fatal("CreateCleanupPolicy() expected error for non-2xx status, got nil")
	}
}

// TestCleanupPolicyService_UpdateCleanupPolicy_Success tests that
// UpdateCleanupPolicy successfully updates a cleanup policy when the server
// responds with a 204 status code.
func TestCleanupPolicyService_UpdateCleanupPolicy_Success(t *testing.T) {
	t.Parallel()

	policy := &cleanuppolicies.CleanupPolicy{
		Name:   "existing-policy",
		Format: cleanuppolicies.RepositoryFormatDocker,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == cleanupPoliciesPath+"/"+policy.Name {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().UpdateCleanupPolicy(context.Background(), policy)
	if err != nil {
		t.Fatalf("UpdateCleanupPolicy() unexpected error: %v", err)
	}
}

// TestCleanupPolicyService_UpdateCleanupPolicy_Error tests that
// UpdateCleanupPolicy returns an error when the server
// responds with a non-204 status code.
func TestCleanupPolicyService_UpdateCleanupPolicy_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "failed to update")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().UpdateCleanupPolicy(context.Background(), &cleanuppolicies.CleanupPolicy{
		Name:   "policy",
		Format: cleanuppolicies.RepositoryFormatDocker,
	})
	if err == nil {
		t.Fatal("UpdateCleanupPolicy() expected error for non-204 status, got nil")
	}
}

// TestCleanupPolicyService_DeleteCleanupPolicy_Success tests that
// DeleteCleanupPolicy successfully deletes a cleanup policy when the server
// responds with a 204 status code.
func TestCleanupPolicyService_DeleteCleanupPolicy_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == cleanupPoliciesPath+"/old-policy" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().DeleteCleanupPolicy(context.Background(), "old-policy")
	if err != nil {
		t.Fatalf("DeleteCleanupPolicy() unexpected error: %v", err)
	}
}

// TestCleanupPolicyService_DeleteCleanupPolicy_Error tests that
// DeleteCleanupPolicy returns an error when the server
// responds with a non-204 status code.
func TestCleanupPolicyService_DeleteCleanupPolicy_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "failed to delete")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.CleanupPolicy().DeleteCleanupPolicy(context.Background(), "policy")
	if err == nil {
		t.Fatal("DeleteCleanupPolicy() expected error for non-204 status, got nil")
	}
}
