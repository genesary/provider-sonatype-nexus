package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	// APIGroup is the API group name for the instance API group.
	APIGroup = "instance.nexus.crossplane.io"
	// Version is the API version for the instance API group.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is the group version for the instance API group.
	SchemeGroupVersion = schema.GroupVersion{Group: APIGroup, Version: Version}

	// SchemeBuilder is the scheme builder for the instance API group.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the instance API group types to the given scheme.
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
