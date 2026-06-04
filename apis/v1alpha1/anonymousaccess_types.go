package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnonymousAccessParameters defines the desired state of AnonymousAccess.
type AnonymousAccessParameters struct {
	// Enabled determines if anonymous access is allowed.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// UserID is the username of the anonymous user account.
	// +kubebuilder:validation:Required
	UserID string `json:"userId"`

	// RealmName is the authentication realm for the anonymous user.
	// +kubebuilder:validation:Required
	RealmName string `json:"realmName"`
}

// AnonymousAccessObservation represents the observed state of AnonymousAccess.
type AnonymousAccessObservation struct {
}

// AnonymousAccessSpec defines the desired state of AnonymousAccess.
type AnonymousAccessSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	ForProvider AnonymousAccessParameters `json:"forProvider"`
}

// AnonymousAccessStatus defines the observed state of AnonymousAccess.
type AnonymousAccessStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider AnonymousAccessObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".spec.forProvider.enabled"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// AnonymousAccess is the Schema for the anonymousaccesses API.
// This is a singleton resource that configures anonymous access settings.
type AnonymousAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnonymousAccessSpec   `json:"spec"`
	Status AnonymousAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AnonymousAccessList contains a list of AnonymousAccess.
type AnonymousAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AnonymousAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AnonymousAccess{}, &AnonymousAccessList{})
}
