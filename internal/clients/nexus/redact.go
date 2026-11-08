package nexus

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// redactedURL replaces a request URL that cannot be parsed for redaction.
const redactedURL = "<redacted>"

// redactedError overrides the message of an error whose wrappers captured
// the unredacted URL, while keeping the original chain for [errors.Is] and
// [errors.As].
type redactedError struct {
	msg string
	err error
}

// Error returns the redacted message.
func (e *redactedError) Error() string { return e.msg }

// Unwrap returns the original error.
func (e *redactedError) Unwrap() error { return e.err }

// RedactURLError strips the query string, fragment and userinfo from the URL
// of the first [url.Error] in err's chain, and from err's message.
//
// When a request fails at the transport layer, or its URL does not parse,
// net/http returns a [url.Error] whose message quotes the request URL. That
// message ends up in status conditions, events, change logs and logs, which
// more people can read than the managed resource's spec, and a URL can carry
// secrets: the signature of a pre-signed license download URL, an access
// token in its query, or a username in its userinfo.
func RedactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) || urlErr == nil {
		return err
	}

	originalURL := urlErr.URL

	redacted := redactURL(originalURL)
	if redacted == originalURL {
		return err
	}

	// The url.Error is redacted in place, so that errors.As on any error
	// sharing it, including err itself, finds the redacted URL.
	urlErr.URL = redacted

	// Wrappers such as errors.Wrap compute their message eagerly, so it may
	// still hold the original URL even though the url.Error is redacted.
	// url.Error formats its URL with %q, so replace the quoted form first in
	// case quoting escaped any of its characters.
	msg := err.Error()

	redactedMsg := strings.ReplaceAll(msg, strconv.Quote(originalURL), strconv.Quote(redacted))
	redactedMsg = strings.ReplaceAll(redactedMsg, originalURL, redacted)

	if redactedMsg != msg {
		return &redactedError{msg: redactedMsg, err: err}
	}

	return err
}

// ParseURL is [url.Parse] with any error passed through RedactURLError. Code
// outside this package parses URLs through it, which the forbidigo linter
// enforces, so that a URL holding secrets never reaches an error message.
func ParseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, RedactURLError(err)
	}

	return parsed, nil
}

// NewRequest is [http.NewRequestWithContext] with any error passed through
// RedactURLError. Code outside this package builds requests through it, which
// the forbidigo linter enforces.
func NewRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, RedactURLError(err)
	}

	return req, nil
}

// Do sends req with client, passing any error through RedactURLError. Code
// outside this package sends requests through it, which the forbidigo linter
// enforces.
func Do(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req) //nolint:gosec // G704: requesting a user-configured URL is the point: the ProviderConfig Nexus URL or a License endpointUrl
	if err != nil {
		return nil, RedactURLError(err)
	}

	return resp, nil
}

// redactURL returns rawURL without its query string, fragment and userinfo.
// It is idempotent: redacting an already redacted URL returns it unchanged.
func redactURL(rawURL string) string {
	if rawURL == redactedURL {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return redactedURL
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""

	return parsed.String()
}
