package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CleanupPolicyParameters defines the desired state of a CleanupPolicy.
type CleanupPolicyParameters struct {
	// Name is the unique identifier for this cleanup policy.
	// Only letters, digits, underscores(_), hyphens(-), and dots(.) are allowed
	// and may not start with underscore or dot.
	// WARNING: This field is immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Format is the repository format this policy applies to.
	// Use * to apply to all formats.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="*";apt;bower;cargo;cocoapods;composer;conan;conda;docker;gitlfs;go;helm;huggingface;maven2;npm;nuget;p2;pypi;r;raw;rubygems;yum
	Format string `json:"format"`

	// Notes are details about this cleanup policy.
	// +kubebuilder:validation:Optional
	Notes *string `json:"notes,omitempty"`

	// CriteriaLastBlobUpdated is the age of the component in days.
	// +kubebuilder:validation:Optional
	CriteriaLastBlobUpdated *int `json:"criteriaLastBlobUpdated,omitempty"`

	// CriteriaLastDownloaded is the number of days since last download.
	// +kubebuilder:validation:Optional
	CriteriaLastDownloaded *int `json:"criteriaLastDownloaded,omitempty"`

	// CriteriaReleaseType filters by release type.
	// Only maven2, npm, and yum repositories support this field.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=RELEASES_AND_PRERELEASES;PRERELEASES;RELEASES
	CriteriaReleaseType *string `json:"criteriaReleaseType,omitempty"`

	// CriteriaAssetRegex is a regex to filter by asset path.
	// Not supported for gitlfs or * format.
	// +kubebuilder:validation:Optional
	CriteriaAssetRegex *string `json:"criteriaAssetRegex,omitempty"`

	// Retain is the number of versions to keep.
	// Only available for Docker and Maven release repositories on PostgreSQL.
	// +kubebuilder:validation:Optional
	Retain *int `json:"retain,omitempty"`
}

// CleanupPolicyObservation represents the observed state of a CleanupPolicy.
type CleanupPolicyObservation struct {
	// Name is the unique identifier for this cleanup policy.
	Name string `json:"name,omitempty"`
	// Format is the repository format this policy applies to.
	Format string `json:"format,omitempty"`
	// Notes are details about this cleanup policy.
	Notes string `json:"notes,omitempty"`
	// CriteriaLastBlobUpdated is the age of the component in days.
	CriteriaLastBlobUpdated int `json:"criteriaLastBlobUpdated,omitempty"`
	// CriteriaLastDownloaded is the number of days since last download.
	CriteriaLastDownloaded int `json:"criteriaLastDownloaded,omitempty"`
	// CriteriaReleaseType filters by release type.
	CriteriaReleaseType string `json:"criteriaReleaseType,omitempty"`
	// CriteriaAssetRegex is a regex to filter by asset path.
	CriteriaAssetRegex string `json:"criteriaAssetRegex,omitempty"`
	// Retain is the number of versions to keep.
	Retain int `json:"retain,omitempty"`
}

// CleanupPolicySpec defines the desired state of CleanupPolicy.
type CleanupPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider CleanupPolicyParameters `json:"forProvider"`
}

// CleanupPolicyStatus defines the observed state of CleanupPolicy.
type CleanupPolicyStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider CleanupPolicyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// CleanupPolicy is the Schema for the cleanuppolicies API.
type CleanupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CleanupPolicySpec   `json:"spec"`
	Status CleanupPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CleanupPolicyList contains a list of CleanupPolicy.
type CleanupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CleanupPolicy `json:"items"`
}

// CleanupPolicy type metadata.
var (
	CleanupPolicyKind             = reflect.TypeFor[CleanupPolicy]().Name()
	CleanupPolicyGroupKind        = schema.GroupKind{Group: APIGroup, Kind: CleanupPolicyKind}.String()
	CleanupPolicyKindAPIVersion   = CleanupPolicyKind + "." + SchemeGroupVersion.String()
	CleanupPolicyGroupVersionKind = SchemeGroupVersion.WithKind(CleanupPolicyKind)
)

// init registers the CleanupPolicy types with the scheme.
func init() {
	SchemeBuilder.Register(&CleanupPolicy{}, &CleanupPolicyList{})
}
