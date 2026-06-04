package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityRealmParameters defines the desired state of SecurityRealm.
type SecurityRealmParameters struct {
	// ActiveRealms is the ordered list of active realm IDs.
	// Common realms include:
	// - NexusAuthenticatingRealm: Local authentication
	// - NexusAuthorizingRealm: Local authorization
	// - LdapRealm: LDAP authentication
	// - DockerToken: Docker token authentication
	// - NpmToken: NPM token authentication
	// - NuGetApiKey: NuGet API key authentication
	// - rutauth-realm: Rut Auth realm
	// - SamlRealm: SAML authentication
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	ActiveRealms []string `json:"activeRealms"`
}

// RealmInfo contains information about a security realm.
type RealmInfo struct {
	// ID of the realm.
	ID string `json:"id"`

	// Name of the realm.
	Name string `json:"name"`
}

// SecurityRealmObservation represents the observed state of SecurityRealm.
type SecurityRealmObservation struct {
	// AvailableRealms lists all available realms in the system.
	AvailableRealms []RealmInfo `json:"availableRealms,omitempty"`
}

// SecurityRealmSpec defines the desired state of SecurityRealm.
type SecurityRealmSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	ForProvider SecurityRealmParameters `json:"forProvider"`
}

// SecurityRealmStatus defines the observed state of SecurityRealm.
type SecurityRealmStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider SecurityRealmObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nexus}

// SecurityRealm is the Schema for the securityrealms API.
// This is a singleton resource that configures which security realms are active.
type SecurityRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityRealmSpec   `json:"spec"`
	Status SecurityRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityRealmList contains a list of SecurityRealm.
type SecurityRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SecurityRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityRealm{}, &SecurityRealmList{})
}
