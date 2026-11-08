package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// TestNewScheme checks that the manager scheme knows the Kubernetes types the
// provider reads and the provider's own APIs.
func TestNewScheme(t *testing.T) {
	t.Parallel()

	scheme, err := newScheme()
	if err != nil {
		t.Fatalf("newScheme() error = %v", err)
	}

	for _, gvk := range []schema.GroupVersionKind{
		corev1.SchemeGroupVersion.WithKind("Secret"),
		nexusv1alpha1.ProviderConfigGroupVersionKind,
		nexusv1alpha1.ClusterProviderConfigGroupVersionKind,
		iamv1alpha1.RoleGroupVersionKind,
	} {
		if !scheme.Recognizes(gvk) {
			t.Errorf("newScheme() does not recognize %s", gvk)
		}
	}
}
