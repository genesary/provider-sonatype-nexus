package instance

import (
	"context"
	"strconv"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// HTTPSettingsClient provides methods for the Nexus HTTP client settings
// (/v1/http, Nexus Repository Pro only).
type HTTPSettingsClient interface {
	Get(ctx context.Context) (*v395.HttpSettingsXo, error)
	Update(ctx context.Context, settings v395.HttpSettingsXo) error
	Reset(ctx context.Context) error
}

// HTTPProxyPasswords are the proxy passwords to apply with the HTTP client
// settings, read from the Secrets the spec references.
type HTTPProxyPasswords struct {
	HTTP  string
	HTTPS string
}

// httpSettingsClient implements HTTPSettingsClient over the Nexus API.
type httpSettingsClient struct {
	api *v395.ManageSonatypeHTTPSystemSettingsAPIService
}

// NewHTTPSettingsClient creates an HTTPSettingsClient for a live Nexus
// connection.
func NewHTTPSettingsClient(creds nexus.Credentials) (HTTPSettingsClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &httpSettingsClient{api: nc.ManageSonatypeHTTPSystemSettingsAPI}, nil
}

// Get returns the HTTP client settings.
func (c *httpSettingsClient) Get(ctx context.Context) (*v395.HttpSettingsXo, error) {
	settings, resp, err := c.api.ListHttp(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return settings, nil
}

// Update replaces the HTTP client settings.
func (c *httpSettingsClient) Update(ctx context.Context, settings v395.HttpSettingsXo) error {
	return nexus.APIError(c.api.UpdateHttp(ctx).HttpSettingsXo(settings).Execute())
}

// Reset restores the default HTTP client settings.
func (c *httpSettingsClient) Reset(ctx context.Context) error {
	return nexus.APIError(c.api.DeleteHttp(ctx).Execute())
}

// GenerateHTTPSettingsUpdate builds the HTTP client settings from the CR
// spec. Nexus replaces all settings on update and requires the timeout and
// retries, so the user agent suffix, timeout and retries the spec leaves
// unset are copied from current, the settings Nexus holds.
func GenerateHTTPSettingsUpdate(params *instancev1alpha1.HTTPClientParameters, current *v395.HttpSettingsXo, passwords HTTPProxyPasswords) v395.HttpSettingsXo {
	settings := v395.HttpSettingsXo{
		HttpProxy:  *v395.NewNullableProxySettingsXo(generateProxySettings(params.HTTPProxy, passwords.HTTP)),
		HttpsProxy: *v395.NewNullableProxySettingsXo(generateProxySettings(params.HTTPSProxy, passwords.HTTPS)),
		// Never nil: a nil slice is omitted from the request, which would not
		// clear the non-proxy hosts the spec no longer lists.
		NonProxyHosts: append([]string{}, params.NonProxyHosts...),
	}

	if current != nil {
		settings.UserAgent = current.UserAgent
		settings.Timeout = current.Timeout
		settings.Retries = current.Retries
	}

	if params.UserAgentSuffix != nil {
		settings.UserAgent = *v395.NewNullableString(params.UserAgentSuffix)
	}

	if params.TimeoutSeconds != nil {
		settings.Timeout = helpers.ClampInt32(*params.TimeoutSeconds)
	}

	if params.Retries != nil {
		settings.Retries = helpers.ClampInt32(*params.Retries)
	}

	return settings
}

// generateProxySettings builds the settings of one proxy, or nil to disable
// it.
func generateProxySettings(proxy *instancev1alpha1.HTTPClientProxy, password string) *v395.ProxySettingsXo {
	if proxy == nil {
		return nil
	}

	settings := &v395.ProxySettingsXo{
		Enabled: true,
		Host:    proxy.Host,
		Port:    strconv.Itoa(proxy.Port),
	}

	if auth := proxy.Authentication; auth != nil {
		settings.AuthInfo = v395.AuthSettingsXo{
			Enabled:    true,
			Username:   auth.Username,
			Password:   password,
			NtlmHost:   ptr.Deref(auth.NTLMHost, ""),
			NtlmDomain: ptr.Deref(auth.NTLMDomain, ""),
		}
	}

	return settings
}

// IsHTTPSettingsUpToDate returns true when the Nexus HTTP client settings
// match the CR spec. The user agent suffix, timeout and retries are only
// compared when the spec sets them, and proxy passwords never are: Nexus does
// not report them, so the controller tracks them by fingerprint instead.
func IsHTTPSettingsUpToDate(cr *instancev1alpha1.HTTPClient, observed *v395.HttpSettingsXo) bool {
	params := cr.Spec.ForProvider

	if params.UserAgentSuffix != nil && *params.UserAgentSuffix != ptr.Deref(observed.UserAgent.Get(), "") {
		return false
	}

	if params.TimeoutSeconds != nil && helpers.ClampInt32(*params.TimeoutSeconds) != observed.Timeout {
		return false
	}

	if params.Retries != nil && helpers.ClampInt32(*params.Retries) != observed.Retries {
		return false
	}

	if !isProxyUpToDate(params.HTTPProxy, observed.HttpProxy.Get()) {
		return false
	}

	if !isProxyUpToDate(params.HTTPSProxy, observed.HttpsProxy.Get()) {
		return false
	}

	return helpers.AreSlicesEqualDeDuped(params.NonProxyHosts, observed.NonProxyHosts)
}

// isProxyUpToDate reports whether an observed proxy matches the spec. A
// proxy the spec leaves out must be absent or disabled.
func isProxyUpToDate(proxy *instancev1alpha1.HTTPClientProxy, observed *v395.ProxySettingsXo) bool {
	if observed == nil || !observed.Enabled {
		return proxy == nil
	}

	if proxy == nil || proxy.Host != observed.Host || proxy.Port != parsePort(observed.Port) {
		return false
	}

	return isProxyAuthenticationUpToDate(proxy.Authentication, &observed.AuthInfo)
}

// isProxyAuthenticationUpToDate reports whether observed proxy credentials
// match the spec. The password is never compared: Nexus does not report it.
func isProxyAuthenticationUpToDate(auth *instancev1alpha1.HTTPClientProxyAuthentication, observed *v395.AuthSettingsXo) bool {
	if auth == nil || !observed.Enabled {
		return auth == nil && !observed.Enabled
	}

	return auth.Username == observed.Username &&
		ptr.Deref(auth.NTLMHost, "") == observed.NtlmHost &&
		ptr.Deref(auth.NTLMDomain, "") == observed.NtlmDomain
}

// GenerateHTTPSettingsObservation converts the HTTP client settings to an
// observation.
func GenerateHTTPSettingsObservation(observed *v395.HttpSettingsXo) instancev1alpha1.HTTPClientObservation {
	if observed == nil {
		return instancev1alpha1.HTTPClientObservation{}
	}

	return instancev1alpha1.HTTPClientObservation{
		UserAgentSuffix: ptr.Deref(observed.UserAgent.Get(), ""),
		TimeoutSeconds:  int(observed.Timeout),
		Retries:         int(observed.Retries),
		HTTPProxy:       generateProxyObservation(observed.HttpProxy.Get()),
		HTTPSProxy:      generateProxyObservation(observed.HttpsProxy.Get()),
		NonProxyHosts:   observed.NonProxyHosts,
	}
}

// generateProxyObservation converts one proxy to an observation, or nil when
// it is disabled.
func generateProxyObservation(observed *v395.ProxySettingsXo) *instancev1alpha1.HTTPClientProxyObservation {
	if observed == nil || !observed.Enabled {
		return nil
	}

	proxy := &instancev1alpha1.HTTPClientProxyObservation{
		Host: observed.Host,
		Port: parsePort(observed.Port),
	}

	if observed.AuthInfo.Enabled {
		proxy.Username = observed.AuthInfo.Username
		proxy.NTLMHost = observed.AuthInfo.NtlmHost
		proxy.NTLMDomain = observed.AuthInfo.NtlmDomain
	}

	return proxy
}

// parsePort converts the proxy port Nexus reports as a string, returning 0
// when it is not a number.
func parsePort(port string) int {
	value, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}

	return value
}
