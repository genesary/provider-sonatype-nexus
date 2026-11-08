package instance

import (
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// newProxyParams returns an authenticated proxy spec.
func newProxyParams() *instancev1alpha1.HTTPClientProxy {
	return &instancev1alpha1.HTTPClientProxy{
		Host: "proxy.internal",
		Port: 3128,
		Authentication: &instancev1alpha1.HTTPClientProxyAuthentication{
			Username: "proxy-user",
			PasswordSecretRef: xpv2.LocalSecretKeySelector{
				LocalSecretReference: xpv2.LocalSecretReference{Name: "proxy-credentials"},
				Key:                  "password",
			},
		},
	}
}

// newObservedProxy returns the Nexus proxy matching newProxyParams. Nexus
// reports the port as a string and never reports the password.
func newObservedProxy() *v395.ProxySettingsXo {
	return &v395.ProxySettingsXo{
		Enabled: true,
		Host:    "proxy.internal",
		Port:    "3128",
		AuthInfo: v395.AuthSettingsXo{
			Enabled:  true,
			Username: "proxy-user",
		},
	}
}

// newObservedSettings returns Nexus HTTP settings with the given HTTP proxy
// and the defaults of a fresh instance.
func newObservedSettings(httpProxy *v395.ProxySettingsXo) *v395.HttpSettingsXo {
	return &v395.HttpSettingsXo{
		HttpProxy: *v395.NewNullableProxySettingsXo(httpProxy),
		Timeout:   20,
		Retries:   2,
	}
}

// TestIsHTTPSettingsUpToDate covers drift detection of the HTTP client
// settings.
func TestIsHTTPSettingsUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   instancev1alpha1.HTTPClientParameters
		observed *v395.HttpSettingsXo
		want     bool
	}{
		{
			// Nexus reports its own user agent, timeout and retries; an
			// empty spec must not fight them.
			name:     "EmptySpecMatchesDefaults",
			params:   instancev1alpha1.HTTPClientParameters{},
			observed: newObservedSettings(nil),
			want:     true,
		},
		{
			name:     "TimeoutDiffers",
			params:   instancev1alpha1.HTTPClientParameters{TimeoutSeconds: new(60)},
			observed: newObservedSettings(nil),
			want:     false,
		},
		{
			name:     "RetriesMatch",
			params:   instancev1alpha1.HTTPClientParameters{Retries: new(2)},
			observed: newObservedSettings(nil),
			want:     true,
		},
		{
			name:     "UserAgentSuffixUnreported",
			params:   instancev1alpha1.HTTPClientParameters{UserAgentSuffix: new("my-org/1.0")},
			observed: newObservedSettings(nil),
			want:     false,
		},
		{
			name:     "ProxyMatches",
			params:   instancev1alpha1.HTTPClientParameters{HTTPProxy: newProxyParams()},
			observed: newObservedSettings(newObservedProxy()),
			want:     true,
		},
		{
			name:     "ProxyMissing",
			params:   instancev1alpha1.HTTPClientParameters{HTTPProxy: newProxyParams()},
			observed: newObservedSettings(nil),
			want:     false,
		},
		{
			name:   "ProxyDisabledInNexus",
			params: instancev1alpha1.HTTPClientParameters{HTTPProxy: newProxyParams()},
			observed: func() *v395.HttpSettingsXo {
				proxy := newObservedProxy()
				proxy.Enabled = false

				return newObservedSettings(proxy)
			}(),
			want: false,
		},
		{
			name:     "UnwantedProxyInNexus",
			params:   instancev1alpha1.HTTPClientParameters{},
			observed: newObservedSettings(newObservedProxy()),
			want:     false,
		},
		{
			// A disabled proxy Nexus still reports is as good as none.
			name:   "DisabledProxyIgnoredWhenUnwanted",
			params: instancev1alpha1.HTTPClientParameters{},
			observed: func() *v395.HttpSettingsXo {
				proxy := newObservedProxy()
				proxy.Enabled = false

				return newObservedSettings(proxy)
			}(),
			want: true,
		},
		{
			name: "PortDiffers",
			params: func() instancev1alpha1.HTTPClientParameters {
				proxy := newProxyParams()
				proxy.Port = 8080

				return instancev1alpha1.HTTPClientParameters{HTTPProxy: proxy}
			}(),
			observed: newObservedSettings(newObservedProxy()),
			want:     false,
		},
		{
			name: "AuthenticationRemoved",
			params: func() instancev1alpha1.HTTPClientParameters {
				proxy := newProxyParams()
				proxy.Authentication = nil

				return instancev1alpha1.HTTPClientParameters{HTTPProxy: proxy}
			}(),
			observed: newObservedSettings(newObservedProxy()),
			want:     false,
		},
		{
			name: "UsernameDiffers",
			params: func() instancev1alpha1.HTTPClientParameters {
				proxy := newProxyParams()
				proxy.Authentication.Username = "other-user"

				return instancev1alpha1.HTTPClientParameters{HTTPProxy: proxy}
			}(),
			observed: newObservedSettings(newObservedProxy()),
			want:     false,
		},
		{
			name: "NTLMDomainDiffers",
			params: func() instancev1alpha1.HTTPClientParameters {
				proxy := newProxyParams()
				proxy.Authentication.NTLMDomain = new("CORP")

				return instancev1alpha1.HTTPClientParameters{HTTPProxy: proxy}
			}(),
			observed: newObservedSettings(newObservedProxy()),
			want:     false,
		},
		{
			name:     "HTTPSProxyMissing",
			params:   instancev1alpha1.HTTPClientParameters{HTTPSProxy: newProxyParams()},
			observed: newObservedSettings(nil),
			want:     false,
		},
		{
			name:   "NonProxyHostsMatchAsSets",
			params: instancev1alpha1.HTTPClientParameters{NonProxyHosts: []string{"localhost", "*.internal"}},
			observed: func() *v395.HttpSettingsXo {
				settings := newObservedSettings(nil)
				settings.NonProxyHosts = []string{"*.internal", "localhost"}

				return settings
			}(),
			want: true,
		},
		{
			name:   "NonProxyHostsDiffer",
			params: instancev1alpha1.HTTPClientParameters{NonProxyHosts: []string{"localhost"}},
			observed: func() *v395.HttpSettingsXo {
				settings := newObservedSettings(nil)
				settings.NonProxyHosts = []string{"localhost", "*.internal"}

				return settings
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := &instancev1alpha1.HTTPClient{}
			cr.Spec.ForProvider = tt.params

			got := IsHTTPSettingsUpToDate(cr, tt.observed)
			if got != tt.want {
				t.Errorf("IsHTTPSettingsUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateHTTPSettingsUpdate_CarriesUnsetScalars verifies that the user
// agent, timeout and retries the spec leaves unset keep their Nexus values,
// since the update replaces every setting.
func TestGenerateHTTPSettingsUpdate_CarriesUnsetScalars(t *testing.T) {
	t.Parallel()

	current := newObservedSettings(newObservedProxy())
	current.UserAgent = *v395.NewNullableString(new("from-nexus"))

	got := GenerateHTTPSettingsUpdate(&instancev1alpha1.HTTPClientParameters{}, current, HTTPProxyPasswords{})

	if got.Timeout != 20 || got.Retries != 2 {
		t.Errorf("Timeout, Retries = %d, %d, want the Nexus values 20, 2", got.Timeout, got.Retries)
	}

	if ua := got.UserAgent.Get(); ua == nil || *ua != "from-nexus" {
		t.Errorf("UserAgent = %v, want the Nexus value from-nexus", ua)
	}

	if got.HttpProxy.Get() != nil || got.HttpsProxy.Get() != nil {
		t.Error("proxies left out of the spec must be disabled")
	}
}

// TestGenerateHTTPSettingsUpdate_AppliesSpec verifies the spec values and
// proxy passwords reach the update.
func TestGenerateHTTPSettingsUpdate_AppliesSpec(t *testing.T) {
	t.Parallel()

	params := &instancev1alpha1.HTTPClientParameters{
		UserAgentSuffix: new("my-org/1.0"),
		TimeoutSeconds:  new(60),
		Retries:         new(5),
		HTTPSProxy:      newProxyParams(),
		NonProxyHosts:   []string{"localhost"},
	}
	params.HTTPSProxy.Authentication.NTLMHost = new("workstation")
	params.HTTPSProxy.Authentication.NTLMDomain = new("CORP")

	got := GenerateHTTPSettingsUpdate(params, newObservedSettings(nil), HTTPProxyPasswords{HTTPS: "s3cret"})

	if ua := got.UserAgent.Get(); ua == nil || *ua != "my-org/1.0" {
		t.Errorf("UserAgent = %v, want my-org/1.0", ua)
	}

	if got.Timeout != 60 || got.Retries != 5 {
		t.Errorf("Timeout, Retries = %d, %d, want 60, 5", got.Timeout, got.Retries)
	}

	if got.HttpProxy.Get() != nil {
		t.Error("HttpProxy is set, want it disabled")
	}

	proxy := got.HttpsProxy.Get()
	if proxy == nil {
		t.Fatal("HttpsProxy is nil, want the spec proxy")
	}

	want := v395.ProxySettingsXo{
		Enabled: true,
		Host:    "proxy.internal",
		Port:    "3128",
		AuthInfo: v395.AuthSettingsXo{
			Enabled:    true,
			Username:   "proxy-user",
			Password:   "s3cret",
			NtlmHost:   "workstation",
			NtlmDomain: "CORP",
		},
	}
	if *proxy != want {
		t.Errorf("HttpsProxy = %+v, want %+v", *proxy, want)
	}

	if len(got.NonProxyHosts) != 1 || got.NonProxyHosts[0] != "localhost" {
		t.Errorf("NonProxyHosts = %v, want [localhost]", got.NonProxyHosts)
	}
}

// TestGenerateHTTPSettingsObservation verifies disabled proxies are not
// observed and enabled ones are converted.
func TestGenerateHTTPSettingsObservation(t *testing.T) {
	t.Parallel()

	disabled := newObservedProxy()
	disabled.Enabled = false

	observed := newObservedSettings(newObservedProxy())
	observed.HttpsProxy = *v395.NewNullableProxySettingsXo(disabled)

	got := GenerateHTTPSettingsObservation(observed)

	if got.TimeoutSeconds != 20 || got.Retries != 2 {
		t.Errorf("TimeoutSeconds, Retries = %d, %d, want 20, 2", got.TimeoutSeconds, got.Retries)
	}

	if got.HTTPProxy == nil || got.HTTPProxy.Port != 3128 || got.HTTPProxy.Username != "proxy-user" {
		t.Errorf("HTTPProxy = %+v, want proxy.internal:3128 as proxy-user", got.HTTPProxy)
	}

	if got.HTTPSProxy != nil {
		t.Errorf("HTTPSProxy = %+v, want nil for a disabled proxy", got.HTTPSProxy)
	}
}
