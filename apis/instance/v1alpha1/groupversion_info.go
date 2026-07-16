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

var (
	// TaskKind is the Kind string for the Task resource.
	TaskKind = reflect.TypeFor[Task]().Name()
	// TaskGroupKind is the GroupKind string for the Task resource.
	TaskGroupKind = schema.GroupKind{Group: APIGroup, Kind: TaskKind}.String()
	// TaskKindAPIVersion is the Kind and APIVersion string for Task.
	TaskKindAPIVersion = TaskKind + "." + SchemeGroupVersion.String()
	// TaskGroupVersionKind is the GroupVersionKind for the Task resource.
	TaskGroupVersionKind = SchemeGroupVersion.WithKind(TaskKind)
)
