package nexus

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	pkgerrors "github.com/pkg/errors"
)

// TestRedactURLError checks what is kept of a URL error's request URL, and
// that the original error stays in the chain.
func TestRedactURLError(t *testing.T) {
	t.Parallel()

	errDial := errors.New("dial tcp: connection refused")

	tests := map[string]struct {
		err      error
		want     string
		redacted bool
	}{
		"NotAURLError": {
			err:  errDial,
			want: errDial.Error(),
		},
		"URLWithNothingToRedact": {
			err:  &url.Error{Op: "Get", URL: "https://nexus.example.com/service/rest/v1/status", Err: errDial},
			want: `Get "https://nexus.example.com/service/rest/v1/status": dial tcp: connection refused`,
		},
		"QueryAndFragmentAreRemoved": {
			err:      &url.Error{Op: "Get", URL: "https://bucket.example.com/license.lic?X-Amz-Signature=secret#part", Err: errDial},
			want:     `Get "https://bucket.example.com/license.lic": dial tcp: connection refused`,
			redacted: true,
		},
		"UserinfoIsRemoved": {
			err:      &url.Error{Op: "Get", URL: "https://admin:secret@nexus.example.com/", Err: errDial},
			want:     `Get "https://nexus.example.com/": dial tcp: connection refused`,
			redacted: true,
		},
		"WrappedMessageIsRedacted": {
			err:      pkgerrors.Wrap(&url.Error{Op: "Get", URL: "https://bucket.example.com/license.lic?token=secret", Err: errDial}, "endpoint request failed"),
			want:     `endpoint request failed: Get "https://bucket.example.com/license.lic": dial tcp: connection refused`,
			redacted: true,
		},
		"UnparseableURLIsReplaced": {
			err:      &url.Error{Op: "parse", URL: "https://nexus.example.com/%zz?token=secret", Err: errDial},
			want:     `parse "<redacted>": dial tcp: connection refused`,
			redacted: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := RedactURLError(test.err)
			if got.Error() != test.want {
				t.Errorf("RedactURLError().Error() = %q, want %q", got.Error(), test.want)
			}

			if strings.Contains(got.Error(), "secret") {
				t.Errorf("RedactURLError().Error() = %q still holds a secret", got.Error())
			}

			if !errors.Is(got, errDial) {
				t.Errorf("RedactURLError() = %v lost the original error from its chain", got)
			}

			var urlErr *url.Error
			if test.redacted && (!errors.As(got, &urlErr) || strings.Contains(urlErr.URL, "secret")) {
				t.Errorf("RedactURLError() left the url.Error URL unredacted")
			}
		})
	}
}

// TestAPIErrorRedactsTransportErrors checks that a failed request of the
// generated client does not report the query of its URL.
func TestAPIErrorRedactsTransportErrors(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Credentials{URL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, resp, err := client.SecurityManagementUsersAPI.ListSecurityUsers(context.Background()).UserId("secret-user").Execute()
	if resp != nil {
		_ = resp.Body.Close()
	}

	got := APIError(resp, err)
	if got == nil {
		t.Fatal("APIError() = nil, want the transport error")
	}

	if strings.Contains(got.Error(), "secret-user") {
		t.Errorf("APIError() = %q reports the request query", got.Error())
	}

	if !strings.Contains(got.Error(), "http://127.0.0.1:1/service/rest/v1/security/users") {
		t.Errorf("APIError() = %q does not name the request path", got.Error())
	}
}

// TestRedactURLErrorOfARealRequest checks the redaction of an error returned
// by net/http itself.
func TestRedactURLErrorOfARealRequest(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1/license?token=secret", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := (&http.Client{}).Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	got := RedactURLError(pkgerrors.Wrap(err, "endpoint request failed"))
	if got == nil || strings.Contains(got.Error(), "secret") {
		t.Errorf("RedactURLError() = %v, want the error without the token", got)
	}
}
