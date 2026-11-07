package main

import (
	"testing"
	"time"

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

// TestCacheOptions checks that the cache resyncs at the sync interval, and
// strips the schemas of the CustomResourceDefinitions it holds.
func TestCacheOptions(t *testing.T) {
	t.Parallel()

	opts := cacheOptions(time.Hour)

	if opts.SyncPeriod == nil || *opts.SyncPeriod != time.Hour {
		t.Errorf("cacheOptions().SyncPeriod = %v, want %v", opts.SyncPeriod, time.Hour)
	}

	var transform func(any) (any, error)

	for object, byObject := range opts.ByObject {
		if _, isCRD := object.(*apiextensionsv1.CustomResourceDefinition); isCRD {
			transform = byObject.Transform
		}
	}

	if transform == nil {
		t.Fatal("cacheOptions() sets no transform for CustomResourceDefinitions")
	}

	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "iam.nexus.crossplane.io",
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:   "v1alpha1",
				Schema: &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"}},
			}},
		},
	}

	got, err := transform(crd)
	if err != nil {
		t.Fatalf("transform() error = %v", err)
	}

	stripped, isCRD := got.(*apiextensionsv1.CustomResourceDefinition)
	if !isCRD {
		t.Fatalf("transform() returned a %T, want a CustomResourceDefinition", got)
	}

	if stripped.Spec.Group != crd.Spec.Group || len(stripped.Spec.Versions) != 1 || stripped.Spec.Versions[0].Name != "v1alpha1" {
		t.Errorf("transform() lost the group or versions: %+v", stripped.Spec)
	}

	if stripped.Spec.Versions[0].Schema != nil {
		t.Errorf("transform() kept the schema of version %s", stripped.Spec.Versions[0].Name)
	}
}
