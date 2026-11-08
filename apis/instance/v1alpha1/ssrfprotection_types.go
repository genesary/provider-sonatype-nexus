package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SSRFProtectionParameters are the configurable Nexus SSRF protection
// settings. They replace the whole Nexus configuration: an exception missing
// from the lists is removed from Nexus.
type SSRFProtectionParameters struct {
	// Enabled indicates whether SSRF protection blocks outbound requests to
	// private, loopback and link-local addresses.
	// If unset, will default to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`

	// AllowedIPs are the IP addresses allowed to bypass SSRF protection.
	// Nexus compares them as a set, so their order does not matter.
	// +kubebuilder:validation:Optional
	// +listType=set
	// +kubebuilder:validation:items:Pattern=`^\S+$`
	AllowedIPs []string `json:"allowedIPs,omitempty"`

	// AllowedDomains are the domain names allowed to bypass SSRF protection.
	// Nexus compares them as a set, so their order does not matter.
	// +kubebuilder:validation:Optional
	// +listType=set
	// +kubebuilder:validation:items:Pattern=`^\S+$`
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

// SSRFProtectionObservation is the observed SSRF protection state.
type SSRFProtectionObservation struct {
	// Enabled reflects whether SSRF protection is enabled.
	Enabled bool `json:"enabled"`

	// AllowedIPs are the observed IP address exceptions.
	AllowedIPs []string `json:"allowedIPs,omitempty"`

	// AllowedDomains are the observed domain name exceptions.
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

// SSRFProtectionSpec defines the desired SSRF protection state.
type SSRFProtectionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider SSRFProtectionParameters `json:"forProvider"`
}

// SSRFProtectionStatus holds the observed SSRF protection state.
type SSRFProtectionStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider SSRFProtectionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".status.atProvider.enabled"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// SSRFProtection manages the Nexus SSRF protection settings, which control
// whether repository connectors may reach private network addresses. It is a
// singleton: declare at most one per Nexus instance. Deleting it restores the
// Nexus default, protection enabled with no exceptions.
type SSRFProtection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSRFProtectionSpec   `json:"spec"`
	Status SSRFProtectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SSRFProtectionList contains a list of SSRFProtection.
type SSRFProtectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SSRFProtection `json:"items"`
}

// SSRFProtection type metadata.
var (
	SSRFProtectionKind             = reflect.TypeFor[SSRFProtection]().Name()
	SSRFProtectionGroupKind        = schema.GroupKind{Group: APIGroup, Kind: SSRFProtectionKind}.String()
	SSRFProtectionKindAPIVersion   = SSRFProtectionKind + "." + SchemeGroupVersion.String()
	SSRFProtectionGroupVersionKind = SchemeGroupVersion.WithKind(SSRFProtectionKind)
)

// init registers SSRFProtection types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&SSRFProtection{}, &SSRFProtectionList{})
}
