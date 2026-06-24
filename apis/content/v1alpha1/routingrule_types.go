package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RoutingRuleParameters defines the desired state of a RoutingRule.
type RoutingRuleParameters struct {
	// Name of the routing rule.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the routing rule.
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// Mode controls whether matching requests are blocked or allowed.
	// +kubebuilder:validation:Enum=ALLOW;BLOCK
	// +kubebuilder:validation:Required
	Mode string `json:"mode"`

	// Matchers is a list of regular expressions used to match request paths.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Matchers []string `json:"matchers"`
}

// RoutingRuleObservation is the observed state of a RoutingRule.
type RoutingRuleObservation struct {
	// Name of the routing rule.
	Name string `json:"name,omitempty"`
	// Description of the routing rule.
	Description string `json:"description,omitempty"`
	// Mode controls whether matching requests are blocked or allowed.
	Mode string `json:"mode,omitempty"`
	// Matchers is the list of path-matching regular expressions.
	Matchers []string `json:"matchers,omitempty"`
}

// RoutingRuleSpec defines the desired state of RoutingRule.
type RoutingRuleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider RoutingRuleParameters `json:"forProvider"`
}

// RoutingRuleStatus defines the observed state of RoutingRule.
type RoutingRuleStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider RoutingRuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// RoutingRule is the Schema for the routingrules API.
type RoutingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoutingRuleSpec   `json:"spec"`
	Status RoutingRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RoutingRuleList contains a list of RoutingRule.
type RoutingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RoutingRule `json:"items"`
}

// RoutingRule type metadata.
var (
	RoutingRuleKind             = reflect.TypeFor[RoutingRule]().Name()
	RoutingRuleGroupKind        = schema.GroupKind{Group: APIGroup, Kind: RoutingRuleKind}.String()
	RoutingRuleKindAPIVersion   = RoutingRuleKind + "." + SchemeGroupVersion.String()
	RoutingRuleGroupVersionKind = SchemeGroupVersion.WithKind(RoutingRuleKind)
)

// init registers the RoutingRule types with the scheme.
func init() {
	SchemeBuilder.Register(&RoutingRule{}, &RoutingRuleList{})
}
