package iam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// newTestLicenseClient creates a LicenseClient pointing at the test server.
func newTestLicenseClient(t *testing.T, serverURL string) iamclient.LicenseClient {
	t.Helper()

	return iamclient.NewLicenseClient(nexus.Credentials{
		URL:      serverURL,
		Username: "admin",
		Password: "password",
		Insecure: true,
	})
}

// isErrNoLicense checks whether err wraps iamclient.ErrNoLicense.
func isErrNoLicense(err error) bool {
	return strings.Contains(err.Error(), iamclient.ErrNoLicense.Error())
}

// newMinimalLicenseCR returns an empty License CR for testing.
func newMinimalLicenseCR() *iamv1alpha1.License {
	return &iamv1alpha1.License{}
}

// TestGetLicense_Success verifies GetLicense returns info on HTTP 200.
func TestGetLicense_Success(t *testing.T) {
	t.Parallel()

	want := &iamclient.LicenseInfo{
		Fingerprint:    "abc123",
		ContactEmail:   "admin@example.com",
		ContactCompany: "ACME Corp",
		ContactName:    "Jane Doe",
		EffectiveDate:  "2024-01-01",
		ExpirationDate: "2025-01-01",
		LicenseType:    "Enterprise",
		LicensedUsers:  "100",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/system/license" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(want)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	got, err := c.GetLicense(context.Background())
	if err != nil {
		t.Fatalf("GetLicense() unexpected error: %v", err)
	}

	if got.Fingerprint != want.Fingerprint {
		t.Errorf("Fingerprint = %q, want %q", got.Fingerprint, want.Fingerprint)
	}

	if got.ContactEmail != want.ContactEmail {
		t.Errorf("ContactEmail = %q, want %q", got.ContactEmail, want.ContactEmail)
	}

	if got.ExpirationDate != want.ExpirationDate {
		t.Errorf("ExpirationDate = %q, want %q", got.ExpirationDate, want.ExpirationDate)
	}
}

// TestGetLicense_NoLicense verifies GetLicense returns ErrNoLicense on 404.
func TestGetLicense_NoLicense(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	_, err := c.GetLicense(context.Background())
	if err == nil {
		t.Fatal("GetLicense() expected ErrNoLicense, got nil")
	}

	if !isErrNoLicense(err) {
		t.Errorf("GetLicense() error = %v, want ErrNoLicense", err)
	}
}

// TestGetLicense_ServerError verifies GetLicense returns error on 500.
func TestGetLicense_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	_, err := c.GetLicense(context.Background())
	if err == nil {
		t.Fatal("GetLicense() expected error on 500, got nil")
	}
}

// TestGetLicense_InvalidJSON tests GetLicense with invalid JSON body.
func TestGetLicense_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "not-json{{{")
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	_, err := c.GetLicense(context.Background())
	if err == nil {
		t.Fatal("GetLicense() expected JSON decode error, got nil")
	}
}

// TestGetLicense_RequestError tests GetLicense with unreachable server.
func TestGetLicense_RequestError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()

	c := newTestLicenseClient(t, server.URL)

	_, err := c.GetLicense(context.Background())
	if err == nil {
		t.Fatal("GetLicense() expected error for unreachable server, got nil")
	}
}

// TestInstallLicense_Success200 tests InstallLicense on HTTP 200.
func TestInstallLicense_Success200(t *testing.T) {
	t.Parallel()

	licenseData := []byte("fake-license-binary-content")

	var received []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/service/rest/v1/system/license" {
			received = make([]byte, r.ContentLength)
			_, _ = r.Body.Read(received)

			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.InstallLicense(context.Background(), licenseData)
	if err != nil {
		t.Fatalf("InstallLicense() unexpected error: %v", err)
	}

	if len(received) == 0 {
		t.Error("InstallLicense() server received no body")
	}
}

// TestInstallLicense_Success204 tests InstallLicense on HTTP 204.
func TestInstallLicense_Success204(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/service/rest/v1/system/license" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.InstallLicense(context.Background(), []byte("license-data"))
	if err != nil {
		t.Fatalf("InstallLicense() with 204 returned unexpected error: %v", err)
	}
}

// TestInstallLicense_Error verifies InstallLicense returns error on 400.
func TestInstallLicense_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid license file")
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.InstallLicense(context.Background(), []byte("bad-data"))
	if err == nil {
		t.Fatal("InstallLicense() expected error on 400, got nil")
	}
}

// TestInstallLicense_RequestError tests InstallLicense when unreachable.
func TestInstallLicense_RequestError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.InstallLicense(context.Background(), []byte("data"))
	if err == nil {
		t.Fatal("InstallLicense() expected error for unreachable server, got nil")
	}
}

// TestDeleteLicense_Success200 tests DeleteLicense on HTTP 200.
func TestDeleteLicense_Success200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/service/rest/v1/system/license" {
			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.DeleteLicense(context.Background())
	if err != nil {
		t.Fatalf("DeleteLicense() unexpected error: %v", err)
	}
}

// TestDeleteLicense_Success204 tests DeleteLicense on HTTP 204.
func TestDeleteLicense_Success204(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/service/rest/v1/system/license" {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.DeleteLicense(context.Background())
	if err != nil {
		t.Fatalf("DeleteLicense() with 204 returned unexpected error: %v", err)
	}
}

// TestDeleteLicense_NotFound verifies DeleteLicense treats 404 as success.
func TestDeleteLicense_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.DeleteLicense(context.Background())
	if err != nil {
		t.Fatalf("DeleteLicense() with 404 returned unexpected error: %v", err)
	}
}

// TestDeleteLicense_Error tests DeleteLicense on server failure.
func TestDeleteLicense_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.DeleteLicense(context.Background())
	if err == nil {
		t.Fatal("DeleteLicense() expected error on 500, got nil")
	}
}

// TestDeleteLicense_RequestError tests DeleteLicense when unreachable.
func TestDeleteLicense_RequestError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Close()

	c := newTestLicenseClient(t, server.URL)

	err := c.DeleteLicense(context.Background())
	if err == nil {
		t.Fatal("DeleteLicense() expected error for unreachable server, got nil")
	}
}

// TestHashLicense tests HashLicense produces a deterministic SHA-256 digest.
func TestHashLicense(t *testing.T) {
	t.Parallel()

	data := []byte("test-license-content")
	hash := iamclient.HashLicense(data)

	if len(hash) != 64 {
		t.Errorf("HashLicense() length = %d, want 64", len(hash))
	}

	hash2 := iamclient.HashLicense(data)
	if hash != hash2 {
		t.Error("HashLicense() is not deterministic")
	}

	hashOther := iamclient.HashLicense([]byte("other-data"))
	if hash == hashOther {
		t.Error("HashLicense() produced same hash for different inputs")
	}
}

// TestHashLicense_Empty verifies HashLicense handles empty input.
func TestHashLicense_Empty(t *testing.T) {
	t.Parallel()

	hash := iamclient.HashLicense([]byte{})
	if len(hash) != 64 {
		t.Errorf("HashLicense() empty input length = %d, want 64", len(hash))
	}
}

// TestGenerateLicenseObservation_NilInfo tests nil info preserves hash.
func TestGenerateLicenseObservation_NilInfo(t *testing.T) {
	t.Parallel()

	prevHash := "previous-hash"
	obs := iamclient.GenerateLicenseObservation(nil, prevHash)

	if obs.InstalledHash != prevHash {
		t.Errorf("InstalledHash = %q, want %q", obs.InstalledHash, prevHash)
	}

	if obs.Fingerprint != "" {
		t.Errorf("Fingerprint = %q, want empty", obs.Fingerprint)
	}
}

// TestGenerateLicenseObservation_WithInfo tests all observation fields.
func TestGenerateLicenseObservation_WithInfo(t *testing.T) {
	t.Parallel()

	info := &iamclient.LicenseInfo{
		Fingerprint:    "fp-abc",
		ContactEmail:   "user@example.com",
		ContactCompany: "Test Corp",
		ContactName:    "John",
		EffectiveDate:  "2024-06-01",
		ExpirationDate: "2025-06-01",
		LicenseType:    "Pro",
		LicensedUsers:  "50",
	}

	hash := "installed-hash"
	obs := iamclient.GenerateLicenseObservation(info, hash)

	if obs.Fingerprint != info.Fingerprint {
		t.Errorf("Fingerprint = %q, want %q", obs.Fingerprint, info.Fingerprint)
	}

	if obs.ContactEmail != info.ContactEmail {
		t.Errorf("ContactEmail = %q, want %q", obs.ContactEmail, info.ContactEmail)
	}

	if obs.ExpirationDate != info.ExpirationDate {
		t.Errorf("ExpirationDate = %q, want %q", obs.ExpirationDate, info.ExpirationDate)
	}

	if obs.InstalledHash != hash {
		t.Errorf("InstalledHash = %q, want %q", obs.InstalledHash, hash)
	}
}

// TestIsLicenseUpToDate_NoFingerprint tests false with empty fingerprint.
func TestIsLicenseUpToDate_NoFingerprint(t *testing.T) {
	t.Parallel()

	cr := newMinimalLicenseCR()
	cr.Status.AtProvider.InstalledHash = iamclient.HashLicense([]byte("data"))
	cr.Status.AtProvider.Fingerprint = ""

	if iamclient.IsLicenseUpToDate(cr, cr.Status.AtProvider.InstalledHash) {
		t.Error("IsLicenseUpToDate() = true with empty fingerprint, want false")
	}
}

// TestIsLicenseUpToDate_HashMismatch verifies false when hashes differ.
func TestIsLicenseUpToDate_HashMismatch(t *testing.T) {
	t.Parallel()

	cr := newMinimalLicenseCR()
	cr.Status.AtProvider.Fingerprint = "fp-123"
	cr.Status.AtProvider.InstalledHash = iamclient.HashLicense([]byte("old-data"))

	if iamclient.IsLicenseUpToDate(cr, iamclient.HashLicense([]byte("new-data"))) {
		t.Error("IsLicenseUpToDate() = true with mismatched hash, want false")
	}
}

// TestIsLicenseUpToDate_UpToDate tests true when hashes match.
func TestIsLicenseUpToDate_UpToDate(t *testing.T) {
	t.Parallel()

	data := []byte("current-license-data")
	hash := iamclient.HashLicense(data)

	cr := newMinimalLicenseCR()
	cr.Status.AtProvider.Fingerprint = "fp-abc"
	cr.Status.AtProvider.InstalledHash = hash

	if !iamclient.IsLicenseUpToDate(cr, hash) {
		t.Error("IsLicenseUpToDate() = false when fingerprint set and hashes match")
	}
}
