package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// AnonymousAccessObservation is the observed state of an AnonymousAccess.
type AnonymousAccessObservation struct {
	// Enabled is the observed enabled state.
	Enabled bool `json:"enabled,omitempty"`
	// UserID is the observed anonymous user ID.
	UserID string `json:"userId,omitempty"`
	// RealmName is the observed authentication realm.
	RealmName string `json:"realmName,omitempty"`
}

// AnonymousAccessSpec defines the desired state of AnonymousAccess.
type AnonymousAccessSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider AnonymousAccessParameters `json:"forProvider"`
}

// AnonymousAccessStatus defines the observed state of AnonymousAccess.
type AnonymousAccessStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider AnonymousAccessObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ENABLED",type="boolean",JSONPath=".spec.forProvider.enabled"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// AnonymousAccess configures anonymous access settings.
// This is a singleton resource managing Nexus anonymous access.
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

// AnonymousAccess type metadata.
var (
	AnonymousAccessKind             = reflect.TypeFor[AnonymousAccess]().Name()
	AnonymousAccessGroupKind        = schema.GroupKind{Group: APIGroup, Kind: AnonymousAccessKind}.String()
	AnonymousAccessKindAPIVersion   = AnonymousAccessKind + "." + SchemeGroupVersion.String()
	AnonymousAccessGroupVersionKind = SchemeGroupVersion.WithKind(AnonymousAccessKind)
)

// init registers the AnonymousAccess types with the scheme.
func init() {
	SchemeBuilder.Register(&AnonymousAccess{}, &AnonymousAccessList{})
}
