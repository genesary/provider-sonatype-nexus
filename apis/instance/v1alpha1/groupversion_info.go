package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	// APIGroup is the API group for the instance API.
	APIGroup = "instance.nexus.crossplane.io"
	// Version is the API version.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: APIGroup, Version: Version}

	// SchemeBuilder is used to add functions to this group's scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// EmailConfigurationKind is the kind for EmailConfiguration.
	EmailConfigurationKind = reflect.TypeFor[EmailConfiguration]().Name()
	// EmailConfigurationGroupKind is the group+kind for EmailConfiguration.
	EmailConfigurationGroupKind = schema.GroupKind{Group: APIGroup, Kind: EmailConfigurationKind}.String()
	// EmailConfigurationKindAPIVersion is the kind+apiversion for
	// EmailConfiguration.
	EmailConfigurationKindAPIVersion = EmailConfigurationKind + "." + SchemeGroupVersion.String()
	// EmailConfigurationGroupVersionKind is the group+version+kind for
	// EmailConfiguration.
	EmailConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(EmailConfigurationKind)
)
