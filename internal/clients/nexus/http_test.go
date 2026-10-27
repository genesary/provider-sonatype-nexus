package nexus

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestReadLimited verifies that a body is read in full up to the limit and
// rejected beyond it.
func TestReadLimited(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body    string
		wantErr bool
	}{
		"BelowLimit": {body: "abc"},
		"AtLimit":    {body: "abcd"},
		"AboveLimit": {body: "abcde", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadLimited(strings.NewReader(tc.body), 4)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), ErrResponseTooLarge) {
					t.Fatalf("ReadLimited() error = %v, want %q", err, ErrResponseTooLarge)
				}

				return
			}

			if err != nil || string(got) != tc.body {
				t.Fatalf("ReadLimited() = %q, %v, want %q", got, err, tc.body)
			}
		})
	}
}

// TestReadLimited_GzipAmplification verifies that the cap applies to the
// decompressed body, so a small gzip response cannot expand past it.
func TestReadLimited_GzipAmplification(t *testing.T) {
	t.Parallel()

	var compressed bytes.Buffer

	zw := gzip.NewWriter(&compressed)

	_, err := zw.Write(bytes.Repeat([]byte("A"), 4*MaxResponseBytes))
	if err != nil {
		t.Fatalf("gzip write: %v", err)
	}

	err = zw.Close()
	if err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed.Bytes())
	}))
	defer server.Close()

	client := &http.Client{Transport: Transport(false), Timeout: RequestTimeout}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	defer resp.Body.Close()

	_, err = ReadLimited(resp.Body, MaxResponseBytes)
	if err == nil || !strings.Contains(err.Error(), ErrResponseTooLarge) {
		t.Fatalf("ReadLimited() error = %v, want %q", err, ErrResponseTooLarge)
	}
}

// TestErrorBody verifies that quoted error bodies are truncated.
func TestErrorBody(t *testing.T) {
	t.Parallel()

	got := ErrorBody(strings.NewReader(strings.Repeat("x", 4*MaxErrorBodyBytes)))
	if want := strings.Repeat("x", MaxErrorBodyBytes) + "..."; got != want {
		t.Fatalf("ErrorBody() length = %d, want %d", len(got), len(want))
	}
}

// TestErrorBody_ValidUTF8 verifies that truncating inside a multi-byte
// character does not leave invalid UTF-8 behind.
func TestErrorBody_ValidUTF8(t *testing.T) {
	t.Parallel()

	got := ErrorBody(strings.NewReader(strings.Repeat("é", MaxErrorBodyBytes)))
	if !utf8.ValidString(got) {
		t.Fatalf("ErrorBody() returned invalid UTF-8")
	}
}

// TestTransportShared verifies that clients share one transport per TLS mode
// instead of each building its own.
func TestTransportShared(t *testing.T) {
	t.Parallel()

	if Transport(false) != Transport(false) || Transport(true) != Transport(true) {
		t.Fatal("Transport() returned a new transport for the same mode")
	}

	if Transport(false) == Transport(true) {
		t.Fatal("Transport() returned the same transport for both TLS modes")
	}

	verifying, _ := Transport(false).(*http.Transport)
	insecure, _ := Transport(true).(*http.Transport)

	if verifying.IdleConnTimeout == 0 || insecure.IdleConnTimeout == 0 {
		t.Fatal("shared transports must time out idle connections")
	}

	if verifying.Proxy == nil || insecure.Proxy == nil {
		t.Fatal("shared transports must honour the proxy environment")
	}

	if insecure.TLSClientConfig == nil || !insecure.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("insecure transport must skip TLS verification")
	}

	if verifying.TLSClientConfig != nil && verifying.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("verifying transport must verify TLS")
	}
}
