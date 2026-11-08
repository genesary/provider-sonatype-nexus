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

	// A cancelled context fails the request with a url.Error at once, rather
	// than after the retry transport has backed off.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, resp, err := client.SecurityManagementUsersAPI.ListSecurityUsers(ctx).UserId("secret-user").Execute()
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

// TestDoRedactsTransportErrors checks that Do does not report the query of
// the URL of a request that fails at the transport layer.
func TestDoRedactsTransportErrors(t *testing.T) {
	t.Parallel()

	req, err := NewRequest(context.Background(), http.MethodGet, "http://127.0.0.1:1/license?token=secret", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	resp, err := Do(&http.Client{}, req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("Do() error = %v, want an error without the token", err)
	}

	if !strings.Contains(err.Error(), "http://127.0.0.1:1/license") {
		t.Errorf("Do() error = %q does not name the request path", err.Error())
	}
}

// TestParseAndNewRequestRedactUnparseableURLs checks that ParseURL and
// NewRequest do not report a URL that does not parse.
func TestParseAndNewRequestRedactUnparseableURLs(t *testing.T) {
	t.Parallel()

	const rawURL = "https://admin:p%ss@nexus.example.com/?token=secret"

	_, parseErr := ParseURL(rawURL)
	_, requestErr := NewRequest(context.Background(), http.MethodGet, rawURL, http.NoBody)

	for name, err := range map[string]error{"ParseURL": parseErr, "NewRequest": requestErr} {
		if err == nil {
			t.Errorf("%s() error = nil, want a parse error", name)

			continue
		}

		if strings.Contains(err.Error(), "admin") || strings.Contains(err.Error(), "secret") {
			t.Errorf("%s() error = %q reports the URL", name, err.Error())
		}
	}
}

// TestRedactURLErrorIsIdempotent checks that redacting an error twice leaves
// it as the first redaction did.
func TestRedactURLErrorIsIdempotent(t *testing.T) {
	t.Parallel()

	for _, rawURL := range []string{
		"https://bucket.example.com/license.lic?X-Amz-Signature=secret",
		"https://nexus.example.com/%zz?token=secret",
	} {
		once := RedactURLError(&url.Error{Op: "Get", URL: rawURL, Err: errors.New("failed")}).Error()

		twice := RedactURLError(RedactURLError(&url.Error{Op: "Get", URL: rawURL, Err: errors.New("failed")})).Error()
		if twice != once {
			t.Errorf("RedactURLError() twice = %q, want %q", twice, once)
		}
	}
}

// TestNewClientRedactsAnUnparseableURL checks that a Nexus URL that does not
// parse, for example because of a password with a stray %, is not reported.
func TestNewClientRedactsAnUnparseableURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Credentials{URL: "https://admin:p%ss@nexus.example.com"})
	if err == nil {
		t.Fatal("NewClient() error = nil, want a parse error")
	}

	if strings.Contains(err.Error(), "admin") || strings.Contains(err.Error(), "p%ss") {
		t.Errorf("NewClient() error = %q reports the URL userinfo", err.Error())
	}
}
