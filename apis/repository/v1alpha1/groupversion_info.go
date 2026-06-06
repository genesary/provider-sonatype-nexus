package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	APIGroup = "repository.nexus.crossplane.io"
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

// BlobStore type metadata.
var (
	BlobStoreKind             = reflect.TypeFor[BlobStore]().Name()
	BlobStoreGroupKind        = schema.GroupKind{Group: APIGroup, Kind: BlobStoreKind}.String()
	BlobStoreKindAPIVersion   = BlobStoreKind + "." + SchemeGroupVersion.String()
	BlobStoreGroupVersionKind = SchemeGroupVersion.WithKind(BlobStoreKind)
)

// Repository type metadata.
var (
	RepositoryKind             = reflect.TypeFor[Repository]().Name()
	RepositoryGroupKind        = schema.GroupKind{Group: APIGroup, Kind: RepositoryKind}.String()
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)
