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

// AnonymousAccess type metadata.
var (
	AnonymousAccessKind             = reflect.TypeFor[AnonymousAccess]().Name()
	AnonymousAccessGroupKind        = schema.GroupKind{Group: APIGroup, Kind: AnonymousAccessKind}.String()
	AnonymousAccessKindAPIVersion   = AnonymousAccessKind + "." + SchemeGroupVersion.String()
	AnonymousAccessGroupVersionKind = SchemeGroupVersion.WithKind(AnonymousAccessKind)
)

// UserTokenConfiguration type metadata.
var (
	UserTokenConfigurationKind             = reflect.TypeFor[UserTokenConfiguration]().Name()
	UserTokenConfigurationGroupKind        = schema.GroupKind{Group: APIGroup, Kind: UserTokenConfigurationKind}.String()
	UserTokenConfigurationKindAPIVersion   = UserTokenConfigurationKind + "." + SchemeGroupVersion.String()
	UserTokenConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(UserTokenConfigurationKind)
)

// SecurityRealm type metadata.
var (
	SecurityRealmKind             = reflect.TypeFor[SecurityRealm]().Name()
	SecurityRealmGroupKind        = schema.GroupKind{Group: APIGroup, Kind: SecurityRealmKind}.String()
	SecurityRealmKindAPIVersion   = SecurityRealmKind + "." + SchemeGroupVersion.String()
	SecurityRealmGroupVersionKind = SchemeGroupVersion.WithKind(SecurityRealmKind)
)
