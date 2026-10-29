package nexus

import (
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
)

// apiBasePath is the path the Nexus REST API is served under.
const apiBasePath = "/service/rest"

// userAgent identifies the provider in the Nexus request log.
const userAgent = "provider-sonatype-nexus"

// ErrNotFound is wrapped by every error that stems from Nexus answering 404.
var ErrNotFound = errors.New("not found")

// NewClient creates a Nexus REST API client from the provided credentials.
//
// Every request it sends authenticates with creds, goes through the shared
// transport for creds.Insecure, is bounded by RequestTimeout, and fails once
// its response body exceeds MaxResponseBytes.
func NewClient(creds Credentials) (*v395.APIClient, error) {
	if creds.URL == "" {
		return nil, errors.New("nexus URL is empty")
	}

	cfg := v395.NewConfiguration()
	cfg.UserAgent = userAgent
	cfg.Servers = v395.ServerConfigurations{{URL: strings.TrimSuffix(creds.URL, "/") + apiBasePath}}
	cfg.HTTPClient = &http.Client{
		Transport: &apiTransport{
			base:     Transport(creds.Insecure),
			username: creds.Username,
			password: creds.Password,
		},
		Timeout: RequestTimeout,
	}

	return v395.NewAPIClient(cfg), nil
}

// apiTransport is an [http.RoundTripper] that authenticates every request with
// basic auth and bounds the response body, which the generated client reads in
// full.
type apiTransport struct {
	base     http.RoundTripper
	username string
	password string
}

// RoundTrip implements [http.RoundTripper].
func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	authenticated := req.Clone(req.Context())
	authenticated.SetBasicAuth(t.username, t.password)

	resp, err := t.base.RoundTrip(authenticated)
	if err != nil {
		return nil, err
	}

	resp.Body = &limitedBody{ReadCloser: resp.Body, remaining: MaxResponseBytes}

	return resp, nil
}

// limitedBody fails a read once more than its limit has been read, instead of
// silently truncating a body the generated client would then misdecode.
type limitedBody struct {
	io.ReadCloser

	remaining int64
}

// Read implements [io.Reader].
func (b *limitedBody) Read(buf []byte) (int, error) {
	if b.remaining < 0 {
		return 0, errors.Errorf("%s: more than %d bytes", ErrResponseTooLarge, MaxResponseBytes)
	}

	if int64(len(buf)) > b.remaining+1 {
		buf = buf[:b.remaining+1]
	}

	read, err := b.ReadCloser.Read(buf)
	b.remaining -= int64(read)

	if b.remaining < 0 {
		return read, errors.Errorf("%s: more than %d bytes", ErrResponseTooLarge, MaxResponseBytes)
	}

	return read, err
}

// APIError turns the outcome of a generated client call into a provider
// error: nil on success, an error wrapping ErrNotFound when Nexus answered
// 404, and otherwise an error naming the HTTP status and quoting the start of
// the response body, which is where Nexus explains a rejected request.
func APIError(resp *http.Response, err error) error {
	if err == nil {
		return nil
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return errors.Wrap(ErrNotFound, err.Error())
	}

	var apiErr *v395.GenericOpenAPIError
	if errors.As(err, &apiErr) && len(apiErr.Body()) > 0 {
		return errors.Errorf("%s: %s", apiErr.Error(), ErrorBody(strings.NewReader(string(apiErr.Body()))))
	}

	return err
}

// IsNotFound reports whether err stems from Nexus answering 404.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IgnoreNotFound returns nil when err stems from Nexus answering 404, and err
// otherwise.
func IgnoreNotFound(err error) error {
	if IsNotFound(err) {
		return nil
	}

	return err
}
