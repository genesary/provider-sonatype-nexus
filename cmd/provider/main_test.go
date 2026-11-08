package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
		apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"),
	} {
		if !scheme.Recognizes(gvk) {
			t.Errorf("newScheme() does not recognize %s", gvk)
		}
	}
}

// TestNewChangeLogOptions checks that change logs are set up without the
// change logs service running: the connection is only made when the first
// change log is sent.
func TestNewChangeLogOptions(t *testing.T) {
	t.Parallel()

	opts, err := newChangeLogOptions(t.TempDir() + "/changelogs.sock")
	if err != nil {
		t.Fatalf("newChangeLogOptions() error = %v", err)
	}

	if opts == nil || opts.ChangeLogger == nil {
		t.Fatal("newChangeLogOptions() returned no change logger")
	}
}

// TestNewChangeLogOptionsRelativePath checks that a socket path the gRPC unix
// resolver cannot handle is rejected at startup.
func TestNewChangeLogOptionsRelativePath(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"", "changelogs.sock", "run/changelogs.sock"} {
		_, err := newChangeLogOptions(path)
		if err == nil {
			t.Errorf("newChangeLogOptions(%q) error = nil, want an error", path)
		}
	}
}
