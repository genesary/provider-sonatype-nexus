package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/repository/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// init registers all provider API group schemes.
func init() {
	AddToSchemes = append(AddToSchemes,
		v1alpha1.AddToScheme,
		contentv1alpha1.AddToScheme,
		iamv1alpha1.AddToScheme,
		instancev1alpha1.AddToScheme,
		repositoryv1alpha1.AddToScheme,
	)
}

// AddToSchemes collects all scheme registration functions for this provider.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme registers all provider API types to the given scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
