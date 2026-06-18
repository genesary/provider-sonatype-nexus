package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	APIGroup = "content.nexus.crossplane.io"
	Version  = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: APIGroup, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// CleanupPolicy type metadata.
var (
	CleanupPolicyKind             = reflect.TypeFor[CleanupPolicy]().Name()
	CleanupPolicyGroupKind        = schema.GroupKind{Group: APIGroup, Kind: CleanupPolicyKind}.String()
	CleanupPolicyKindAPIVersion   = CleanupPolicyKind + "." + SchemeGroupVersion.String()
	CleanupPolicyGroupVersionKind = SchemeGroupVersion.WithKind(CleanupPolicyKind)
)

// ContentSelector type metadata.
var (
	ContentSelectorKind             = reflect.TypeFor[ContentSelector]().Name()
	ContentSelectorGroupKind        = schema.GroupKind{Group: APIGroup, Kind: ContentSelectorKind}.String()
	ContentSelectorKindAPIVersion   = ContentSelectorKind + "." + SchemeGroupVersion.String()
	ContentSelectorGroupVersionKind = SchemeGroupVersion.WithKind(ContentSelectorKind)
)
