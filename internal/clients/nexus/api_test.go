package nexus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAPIError_NotFound verifies that only a 404 answer counts as not found,
// whatever the body of any other error says.
func TestAPIError_NotFound(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status       int
		body         string
		wantNotFound bool
	}{
		"404":                    {status: http.StatusNotFound, body: "", wantNotFound: true},
		"400MentioningNotFound":  {status: http.StatusBadRequest, body: "blob store not found", wantNotFound: false},
		"500MentioningNotExist":  {status: http.StatusInternalServerError, body: "format does not exist", wantNotFound: false},
		"500MentioningStatus404": {status: http.StatusInternalServerError, body: "upstream answered 404", wantNotFound: false},
		"403MentioningNotFound":  {status: http.StatusForbidden, body: "Not Found", wantNotFound: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)

			client, err := NewClient(Credentials{URL: srv.URL, Username: "admin", Password: "secret"})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			err = APIError(client.SecurityManagementRolesAPI.DeleteSecurityRoles(context.Background(), "role").Execute())
			if err == nil {
				t.Fatal("APIError() = nil, want an error")
			}

			if got := IsNotFound(err); got != tc.wantNotFound {
				t.Errorf("IsNotFound(%v) = %v, want %v", err, got, tc.wantNotFound)
			}

			if got := IgnoreNotFound(err) == nil; got != tc.wantNotFound {
				t.Errorf("IgnoreNotFound(%v) == nil is %v, want %v", err, got, tc.wantNotFound)
			}

			if tc.body != "" && !strings.Contains(err.Error(), tc.body) {
				t.Errorf("APIError() = %q, want it to quote the body %q", err, tc.body)
			}
		})
	}
}

// TestAPIError_Success verifies that a successful call yields no error.
func TestAPIError_Success(t *testing.T) {
	t.Parallel()

	err := APIError(&http.Response{StatusCode: http.StatusNoContent}, nil)
	if err != nil {
		t.Fatalf("APIError() = %v, want nil", err)
	}
}
