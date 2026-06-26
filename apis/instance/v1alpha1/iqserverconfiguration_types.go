package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type IQServerConfigurationParameters struct {
	Enabled bool `json:"enabled"`

	ShowLink bool `json:"showLink"`

	// +optional
	URL *string `json:"url,omitempty"`

	// AuthenticationMethod is either "USER" or "PKI".
	// +optional
	AuthenticationMethod *string `json:"authenticationMethod,omitempty"`

	// +optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef references the Kubernetes Secret key holding the IQ Server password.
	// +optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	UseTrustStoreForURL bool `json:"useTrustStoreForUrl,omitempty"`

	// +optional
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`

	// +optional
	Properties *string `json:"properties,omitempty"`

	FailOpenModeEnabled bool `json:"failOpenModeEnabled,omitempty"`
}

type IQServerConfigurationObservation struct {
	Enabled bool `json:"enabled,omitempty"`

	ShowLink bool `json:"showLink,omitempty"`

	URL *string `json:"url,omitempty"`

	AuthenticationMethod *string `json:"authenticationMethod,omitempty"`

	Username *string `json:"username,omitempty"`

	UseTrustStoreForURL bool `json:"useTrustStoreForUrl,omitempty"`

	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`

	Properties *string `json:"properties,omitempty"`

	FailOpenModeEnabled bool `json:"failOpenModeEnabled,omitempty"`
}

type IQServerConfigurationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider IQServerConfigurationParameters `json:"forProvider"`
}

type IQServerConfigurationStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider IQServerConfigurationObservation `json:"atProvider,omitempty"`
}

type IQServerConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IQServerConfigurationSpec   `json:"spec"`
	Status IQServerConfigurationStatus `json:"status,omitempty"`
}

type IQServerConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IQServerConfiguration `json:"items"`
}

var (
	IQServerConfigurationKind             = reflect.TypeFor[IQServerConfiguration]().Name()
	IQServerConfigurationGroupKind        = schema.GroupKind{Group: APIGroup, Kind: IQServerConfigurationKind}.String()
	IQServerConfigurationKindAPIVersion   = IQServerConfigurationKind + "." + SchemeGroupVersion.String()
	IQServerConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(IQServerConfigurationKind)
)

func init() {
	SchemeBuilder.Register(&IQServerConfiguration{}, &IQServerConfigurationList{})
}
