package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HTTPClientParameters are the configurable Nexus HTTP client settings
// (Administration, System, HTTP), which Nexus applies to its outbound
// requests. The proxies and non-proxy hosts replace the whole Nexus
// configuration: a proxy the spec leaves out is disabled. The user agent
// suffix, timeout and retries are only managed when set.
type HTTPClientParameters struct {
	// UserAgentSuffix is appended to the User-Agent header of outbound
	// requests. If unset, the value in Nexus is kept.
	// +kubebuilder:validation:Optional
	UserAgentSuffix *string `json:"userAgentSuffix,omitempty"`

	// TimeoutSeconds is the connection and socket timeout of outbound
	// requests, in seconds. If unset, the value in Nexus is kept.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`

	// Retries is the number of connection and socket retry attempts of
	// outbound requests. If unset, the value in Nexus is kept.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	Retries *int `json:"retries,omitempty"`

	// HTTPProxy is the proxy for outbound HTTP requests. If unset, no HTTP
	// proxy is used.
	// +kubebuilder:validation:Optional
	HTTPProxy *HTTPClientProxy `json:"httpProxy,omitempty"`

	// HTTPSProxy is the proxy for outbound HTTPS requests. If unset, no HTTPS
	// proxy is used.
	// +kubebuilder:validation:Optional
	HTTPSProxy *HTTPClientProxy `json:"httpsProxy,omitempty"`

	// NonProxyHosts are the hosts reached without a proxy. Nexus accepts
	// wildcards such as *.example.com.
	// +kubebuilder:validation:Optional
	// +listType=set
	// +kubebuilder:validation:items:Pattern=`^\S+$`
	NonProxyHosts []string `json:"nonProxyHosts,omitempty"`
}

// HTTPClientProxy is a proxy server for outbound requests.
type HTTPClientProxy struct {
	// Host is the proxy host name or IP address.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`

	// Port is the proxy port.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port"`

	// Authentication holds the proxy credentials. If unset, the proxy is used
	// without authentication.
	// +kubebuilder:validation:Optional
	Authentication *HTTPClientProxyAuthentication `json:"authentication,omitempty"`
}

// HTTPClientProxyAuthentication holds the credentials of a proxy server.
// Setting the NTLM host and domain selects Windows NTLM authentication.
type HTTPClientProxyAuthentication struct {
	// Username is the proxy user name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`

	// PasswordSecretRef references the Kubernetes Secret key holding the
	// proxy password. The secret must be in the same namespace as the
	// HTTPClient.
	// +kubebuilder:validation:Required
	PasswordSecretRef xpv2.LocalSecretKeySelector `json:"passwordSecretRef"`

	// NTLMHost is the Windows NTLM host name.
	// +kubebuilder:validation:Optional
	NTLMHost *string `json:"ntlmHost,omitempty"`

	// NTLMDomain is the Windows NTLM domain.
	// +kubebuilder:validation:Optional
	NTLMDomain *string `json:"ntlmDomain,omitempty"`
}

// HTTPClientObservation is the observed Nexus HTTP client state.
type HTTPClientObservation struct {
	// UserAgentSuffix is the observed User-Agent suffix.
	UserAgentSuffix string `json:"userAgentSuffix,omitempty"`

	// TimeoutSeconds is the observed connection and socket timeout.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Retries is the observed number of connection and socket retries.
	Retries int `json:"retries,omitempty"`

	// HTTPProxy is the observed HTTP proxy, absent when disabled.
	HTTPProxy *HTTPClientProxyObservation `json:"httpProxy,omitempty"`

	// HTTPSProxy is the observed HTTPS proxy, absent when disabled.
	HTTPSProxy *HTTPClientProxyObservation `json:"httpsProxy,omitempty"`

	// NonProxyHosts are the observed hosts reached without a proxy.
	NonProxyHosts []string `json:"nonProxyHosts,omitempty"`

	// CredentialsFingerprint identifies the proxy credentials last applied
	// to Nexus, which Nexus never reports back. It changes when a referenced
	// Secret changes or is repointed, which triggers an update.
	CredentialsFingerprint string `json:"credentialsFingerprint,omitempty"`

	// ResetToDefaults is true once the provider has reset the Nexus HTTP
	// client settings to their defaults while deleting this resource.
	ResetToDefaults bool `json:"resetToDefaults,omitempty"`
}

// HTTPClientProxyObservation is an observed proxy server.
type HTTPClientProxyObservation struct {
	// Host is the observed proxy host.
	Host string `json:"host"`

	// Port is the observed proxy port.
	Port int `json:"port"`

	// Username is the observed proxy user name, absent without
	// authentication.
	Username string `json:"username,omitempty"`

	// NTLMHost is the observed Windows NTLM host name.
	NTLMHost string `json:"ntlmHost,omitempty"`

	// NTLMDomain is the observed Windows NTLM domain.
	NTLMDomain string `json:"ntlmDomain,omitempty"`
}

// HTTPClientSpec defines the desired Nexus HTTP client state.
type HTTPClientSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider HTTPClientParameters `json:"forProvider"`
}

// HTTPClientStatus holds the observed Nexus HTTP client state.
type HTTPClientStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider HTTPClientObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// HTTPClient manages the Nexus HTTP client settings: the User-Agent suffix,
// timeout, retries and outbound HTTP and HTTPS proxies. It is a singleton:
// declare at most one per Nexus instance. Deleting it resets every HTTP
// client setting to the Nexus default. Requires Nexus Repository Pro.
type HTTPClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPClientSpec   `json:"spec"`
	Status HTTPClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HTTPClientList contains a list of HTTPClient.
type HTTPClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []HTTPClient `json:"items"`
}

// HTTPClient type metadata.
var (
	HTTPClientKind             = reflect.TypeFor[HTTPClient]().Name()
	HTTPClientGroupKind        = schema.GroupKind{Group: APIGroup, Kind: HTTPClientKind}.String()
	HTTPClientKindAPIVersion   = HTTPClientKind + "." + SchemeGroupVersion.String()
	HTTPClientGroupVersionKind = SchemeGroupVersion.WithKind(HTTPClientKind)
)

// init registers HTTPClient types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&HTTPClient{}, &HTTPClientList{})
}
