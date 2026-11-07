package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

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

	for _, gvk := range []struct {
		name  string
		known bool
	}{
		{name: "Secret", known: scheme.Recognizes(corev1.SchemeGroupVersion.WithKind("Secret"))},
		{name: "ProviderConfig", known: scheme.Recognizes(nexusv1alpha1.ProviderConfigGroupVersionKind)},
		{name: "ClusterProviderConfig", known: scheme.Recognizes(nexusv1alpha1.ClusterProviderConfigGroupVersionKind)},
		{name: "Role", known: scheme.Recognizes(iamv1alpha1.RoleGroupVersionKind)},
	} {
		if !gvk.known {
			t.Errorf("newScheme() does not recognize %s", gvk.name)
		}
	}
}
