package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	APIGroup = "iam.nexus.crossplane.io"
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

// SecurityRealm type metadata.
var (
	SecurityRealmKind             = reflect.TypeFor[SecurityRealm]().Name()
	SecurityRealmGroupKind        = schema.GroupKind{Group: APIGroup, Kind: SecurityRealmKind}.String()
	SecurityRealmKindAPIVersion   = SecurityRealmKind + "." + SchemeGroupVersion.String()
	SecurityRealmGroupVersionKind = SchemeGroupVersion.WithKind(SecurityRealmKind)
)
