// Package apis contains Kubernetes API for the Nexus provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// init registers this type with the SchemeBuilder.
func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.AddToScheme,
		contentv1alpha1.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the
// project to a Scheme.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
