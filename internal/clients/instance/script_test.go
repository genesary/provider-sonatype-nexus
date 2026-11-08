package instance

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// scriptingDisabledBody is the body Nexus 3.96 answers a script write with
// when script creation is disabled, truncated message included.
const scriptingDisabledBody = `{"name":"x","result":"Creating and updating scripts is disable"}`

// recordedRequest is the method and path of the last request a test server
// received.
type recordedRequest struct {
	mu     sync.Mutex
	method string
	path   string
}

// get returns the recorded method and path.
func (r *recordedRequest) get() (method, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.method, r.path
}

// newTestScriptClient returns a ScriptClient talking to a test server that
// answers every request with status and body, and records the last request.
func newTestScriptClient(t *testing.T, status int, body string) (ScriptClient, *recordedRequest) {
	t.Helper()

	last := &recordedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last.mu.Lock()
		last.method, last.path = r.Method, r.URL.Path
		last.mu.Unlock()

		_, _ = io.Copy(io.Discard, r.Body)

		if body != "" {
			w.Header().Set("Content-Type", "application/json")
		}

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	c, err := NewScriptClient(nexus.Credentials{URL: server.URL, Username: "admin", Password: "admin123"})
	if err != nil {
		t.Fatalf("NewScriptClient() error = %v", err)
	}

	return c, last
}

// TestScriptClientWrites covers how Create and Update report the answers
// Nexus gives to a script write.
func TestScriptClientWrites(t *testing.T) {
	t.Parallel()

	script := v395.ScriptXO{Name: "x", Type: "groovy", Content: "return 1"}

	writes := map[string]func(ScriptClient) error{
		"Create": func(c ScriptClient) error { return c.Create(context.Background(), script) },
		"Update": func(c ScriptClient) error { return c.Update(context.Background(), "x", script) },
	}

	tests := []struct {
		name         string
		status       int
		body         string
		wantErr      bool
		wantDisabled bool
		wantNotFound bool
	}{
		{name: "Success", status: http.StatusNoContent},
		{name: "ScriptingDisabled", status: http.StatusGone, body: scriptingDisabledBody, wantErr: true, wantDisabled: true},
		{name: "NotFound", status: http.StatusNotFound, body: `{"status-code":404}`, wantErr: true, wantNotFound: true},
		{name: "ServerError", status: http.StatusInternalServerError, body: "DuplicateKeyException", wantErr: true},
	}

	for writeName, write := range writes {
		for _, tt := range tests {
			t.Run(writeName+"_"+tt.name, func(t *testing.T) {
				t.Parallel()

				c, _ := newTestScriptClient(t, tt.status, tt.body)

				err := write(c)
				if (err != nil) != tt.wantErr {
					t.Fatalf("%s() error = %v, wantErr %v", writeName, err, tt.wantErr)
				}

				if got := errors.Is(err, ErrScriptingDisabled); got != tt.wantDisabled {
					t.Errorf("%s() errors.Is(ErrScriptingDisabled) = %v, want %v (error: %v)", writeName, got, tt.wantDisabled, err)
				}

				if got := nexus.IsNotFound(err); got != tt.wantNotFound {
					t.Errorf("%s() IsNotFound = %v, want %v", writeName, got, tt.wantNotFound)
				}

				// The Nexus answer is kept next to the guidance.
				if tt.wantDisabled && !strings.Contains(err.Error(), "410") {
					t.Errorf("%s() error %q lost the Nexus answer", writeName, err)
				}
			})
		}
	}
}

// TestScriptClientGet covers reading a script, present or absent.
func TestScriptClientGet(t *testing.T) {
	t.Parallel()

	t.Run("Found", func(t *testing.T) {
		t.Parallel()

		c, last := newTestScriptClient(t, http.StatusOK, `{"name":"x","content":"return 1","type":"groovy"}`)

		got, err := c.Get(context.Background(), "x")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got.Name != "x" || got.Content != "return 1" || got.Type != "groovy" {
			t.Errorf("Get() = %+v", got)
		}

		if _, path := last.get(); path != "/service/rest/v1/script/x" {
			t.Errorf("Get() requested %q", path)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		c, _ := newTestScriptClient(t, http.StatusNotFound, `{"status-code":404}`)

		_, err := c.Get(context.Background(), "x")
		if !nexus.IsNotFound(err) {
			t.Errorf("Get() error = %v, want not found", err)
		}
	})
}

// TestScriptClientDelete covers deleting a script.
func TestScriptClientDelete(t *testing.T) {
	t.Parallel()

	c, last := newTestScriptClient(t, http.StatusNoContent, "")

	err := c.Delete(context.Background(), "x")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if method, path := last.get(); method != http.MethodDelete || path != "/service/rest/v1/script/x" {
		t.Errorf("Delete() sent %s %s", method, path)
	}
}

// TestGenerateScript covers the defaulting of the script type.
func TestGenerateScript(t *testing.T) {
	t.Parallel()

	got := GenerateScript(&instancev1alpha1.ScriptParameters{Name: "x", Content: "return 1"})
	if got.Name != "x" || got.Type != DefaultScriptType || got.Content != "return 1" {
		t.Errorf("GenerateScript() = %+v", got)
	}

	got = GenerateScript(&instancev1alpha1.ScriptParameters{Name: "x", Type: "javascript", Content: "1"})
	if got.Type != "javascript" {
		t.Errorf("GenerateScript() type = %q, want javascript", got.Type)
	}
}

// TestIsScriptUpToDate covers drift detection of a script.
func TestIsScriptUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   instancev1alpha1.ScriptParameters
		observed v395.ScriptXO
		want     bool
	}{
		{
			name:     "Matches",
			params:   instancev1alpha1.ScriptParameters{Name: "x", Type: "groovy", Content: "return 1"},
			observed: v395.ScriptXO{Name: "x", Type: "groovy", Content: "return 1"},
			want:     true,
		},
		{
			name:     "UnsetTypeMatchesGroovy",
			params:   instancev1alpha1.ScriptParameters{Name: "x", Content: "return 1"},
			observed: v395.ScriptXO{Name: "x", Type: "groovy", Content: "return 1"},
			want:     true,
		},
		{
			name:     "ContentDiffers",
			params:   instancev1alpha1.ScriptParameters{Name: "x", Content: "return 1"},
			observed: v395.ScriptXO{Name: "x", Type: "groovy", Content: "return 2"},
			want:     false,
		},
		{
			// Whitespace is part of the source: it is compared exactly.
			name:     "TrailingNewlineDiffers",
			params:   instancev1alpha1.ScriptParameters{Name: "x", Content: "return 1\n"},
			observed: v395.ScriptXO{Name: "x", Type: "groovy", Content: "return 1"},
			want:     false,
		},
		{
			name:     "TypeDiffers",
			params:   instancev1alpha1.ScriptParameters{Name: "x", Content: "return 1"},
			observed: v395.ScriptXO{Name: "x", Type: "javascript", Content: "return 1"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsScriptUpToDate(&tt.params, &tt.observed); got != tt.want {
				t.Errorf("IsScriptUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateScriptObservation checks that the source is only recorded as
// its digest.
func TestGenerateScriptObservation(t *testing.T) {
	t.Parallel()

	if got := GenerateScriptObservation(nil); got != (instancev1alpha1.ScriptObservation{}) {
		t.Errorf("GenerateScriptObservation(nil) = %+v, want empty", got)
	}

	got := GenerateScriptObservation(&v395.ScriptXO{Name: "x", Type: "groovy", Content: "abc"})

	// SHA-256 of "abc".
	const wantDigest = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got.Name != "x" || got.Type != "groovy" || got.ContentSHA256 != wantDigest {
		t.Errorf("GenerateScriptObservation() = %+v", got)
	}
}
