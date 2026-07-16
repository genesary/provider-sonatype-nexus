package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// IQServerConfigurationParameters are configurable IQ Server settings.
type IQServerConfigurationParameters struct {
	// Enabled indicates whether the IQ Server configuration is enabled or disabled.
	// If unset, will default to true.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled"`

	// ShowLink indicates whether to show the IQ Server link in the Nexus UI.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ShowLink *bool `json:"showLink"`

	// URL is the URL of the IQ Server.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// AuthenticationType is either "USER" or "PKI".
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=USER;PKI
	AuthenticationType *string `json:"authenticationType"`

	// UsernameSecretRef references the Kubernetes Secret key holding the IQ Server username.
	// +kubebuilder:validation:Optional
	UsernameSecretRef *xpv2.SecretKeySelector `json:"usernameSecretRef"`

	// PasswordSecretRef references the Kubernetes Secret key holding the IQ Server password.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef"`

	// UseTrustStoreForURL indicates whether to use the trust store for the IQ Server URL.
	// +kubebuilder:validation:Optional
	UseTrustStoreForURL *bool `json:"useTrustStoreForUrl"`

	// TimeoutSeconds is the timeout in seconds for the IQ Server connection.
	// If unset, will default to 10 seconds.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=10
	TimeoutSeconds *int `json:"timeoutSeconds"`

	// Properties is a string of properties to set for the IQ Server configuration.
	// +kubebuilder:validation:Optional
	Properties *string `json:"properties"`

	// FailOpenModeEnabled indicates whether the fail open mode is enabled.
	// If unset, will default to false.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	FailOpenModeEnabled *bool `json:"failOpenModeEnabled"`
}

// IQServerConfigurationObservation is the observed IQ Server state.
type IQServerConfigurationObservation struct {
	// Enabled reflects whether the IQ Server integration is enabled.
	Enabled bool `json:"enabled"`

	// ShowLink reflects whether the IQ Server link is shown in the Nexus UI.
	ShowLink bool `json:"showLink"`

	// URL is the observed IQ Server URL.
	URL string `json:"url"`

	// AuthenticationType is the observed authentication type.
	AuthenticationType string `json:"authenticationType"`

	// UseTrustStoreForURL reflects whether the trust store is used for the IQ Server URL.
	UseTrustStoreForURL bool `json:"useTrustStoreForUrl"`

	// TimeoutSeconds is the observed connection timeout in seconds.
	TimeoutSeconds int `json:"timeoutSeconds"`

	// Properties is the observed properties string.
	Properties string `json:"properties"`

	// FailOpenModeEnabled reflects whether fail open mode is enabled.
	FailOpenModeEnabled bool `json:"failOpenModeEnabled"`
}

// IQServerConfigurationSpec defines the desired IQ Server state.
type IQServerConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider IQServerConfigurationParameters `json:"forProvider"`
}

// IQServerConfigurationStatus holds the observed IQ Server state.
type IQServerConfigurationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider IQServerConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IQServerConfiguration is the Schema for the IQ Server configuration API.
type IQServerConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IQServerConfigurationSpec   `json:"spec"`
	Status IQServerConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IQServerConfigurationList contains a list of IQServerConfiguration.
type IQServerConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IQServerConfiguration `json:"items"`
}

// IQServerConfiguration type metadata.
var (
	IQServerConfigurationKind             = reflect.TypeFor[IQServerConfiguration]().Name()
	IQServerConfigurationGroupKind        = schema.GroupKind{Group: APIGroup, Kind: IQServerConfigurationKind}.String()
	IQServerConfigurationKindAPIVersion   = IQServerConfigurationKind + "." + SchemeGroupVersion.String()
	IQServerConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(IQServerConfigurationKind)
)

// init registers IQServerConfiguration types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&IQServerConfiguration{}, &IQServerConfigurationList{})
}
