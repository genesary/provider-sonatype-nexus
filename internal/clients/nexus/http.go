package nexus

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// MaxResponseBytes caps the response body the provider buffers from
	// Nexus. Responses are JSON documents far below it; the cap bounds what a
	// misbehaving or hostile endpoint can make the shared provider process
	// allocate, including through a gzip-compressed body.
	MaxResponseBytes = 8 << 20

	// MaxErrorBodyBytes caps how much of an error response body is quoted in
	// an error, and therefore in conditions and events.
	MaxErrorBodyBytes = 1 << 10

	// RequestTimeout bounds every HTTP request the provider sends, including
	// reading the response body.
	RequestTimeout = 30 * time.Second

	// ErrResponseTooLarge is returned when a response body exceeds its cap.
	ErrResponseTooLarge = "response body is too large"
)

// Shared transports. Every client reuses one of them instead of building a
// Transport per Connect: a Transport cannot be closed once its client is
// dropped, so a per-Connect Transport leaks its keep-alive connections.
var (
	verifyingTransport = newTransport(false)
	insecureTransport  = newTransport(true)
)

// newTransport returns a clone of [http.DefaultTransport], which keeps its
// proxy settings, dial and TLS handshake timeouts, and finite idle connection
// timeout, with TLS verification disabled when insecure is set.
func newTransport(insecure bool) *http.Transport {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		defaultTransport = &http.Transport{}
	}

	transport := defaultTransport.Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user opt-in
	}

	return transport
}

// Transport returns the shared transport for the given TLS verification mode.
func Transport(insecure bool) http.RoundTripper {
	if insecure {
		return insecureTransport
	}

	return verifyingTransport
}

// ReadLimited reads all of r, failing once it yields more than limit bytes
// rather than buffering the excess.
func ReadLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, errors.Errorf("%s: more than %d bytes", ErrResponseTooLarge, limit)
	}

	return data, nil
}

// ErrorBody returns at most MaxErrorBodyBytes of r as valid UTF-8, for quoting
// a response body in an error. Truncated bodies end with an ellipsis.
func ErrorBody(r io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(r, MaxErrorBodyBytes+1))

	truncated := int64(len(data)) > MaxErrorBodyBytes
	if truncated {
		data = data[:MaxErrorBodyBytes]
	}

	body := strings.ToValidUTF8(string(data), "")
	if truncated {
		body += "..."
	}

	return body
}
