package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	// APIGroup is the group name for the instance API.
	APIGroup = "instance.nexus.crossplane.io"
	// Version is the version for the instance API.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: APIGroup, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Capability type metadata.
var (
	CapabilityKind             = reflect.TypeFor[Capability]().Name()
	CapabilityGroupKind        = schema.GroupKind{Group: APIGroup, Kind: CapabilityKind}.String()
	CapabilityKindAPIVersion   = CapabilityKind + "." + SchemeGroupVersion.String()
	CapabilityGroupVersionKind = SchemeGroupVersion.WithKind(CapabilityKind)
)
