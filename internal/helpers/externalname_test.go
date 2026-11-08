package helpers

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// cleanupPolicyName is the spec identity of a CleanupPolicy.
func cleanupPolicyName(cr *contentv1alpha1.CleanupPolicy) string {
	return cr.Spec.ForProvider.Name
}

// newCleanupPolicy returns a CleanupPolicy whose metadata.name differs from
// the Nexus name its spec gives.
func newCleanupPolicy(specName string, annotations map[string]string) *contentv1alpha1.CleanupPolicy {
	cr := &contentv1alpha1.CleanupPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "k8s-name", Namespace: "default", Annotations: annotations},
	}
	cr.Spec.ForProvider.Name = specName

	return cr
}

// newExternalNameClient returns a fake client holding cr.
func newExternalNameClient(t *testing.T, cr client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()

	err := contentv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()
}

// TestSpecExternalName checks which external name the initializer settles on
// and that it persists it.
func TestSpecExternalName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		specName    string
		annotations map[string]string
		want        string
	}{
		{
			name:     "FromSpecNotMetadataName",
			specName: "nexus-name",
			want:     "nexus-name",
		},
		{
			name:        "ExplicitAnnotationKept",
			specName:    "nexus-name",
			annotations: map[string]string{meta.AnnotationKeyExternalName: "explicit"},
			want:        "explicit",
		},
		{
			name: "EmptySpecLeavesItUnset",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := newCleanupPolicy(tt.specName, tt.annotations)
			kube := newExternalNameClient(t, cr)

			err := SpecExternalName(kube, cleanupPolicyName).Initialize(context.Background(), cr)
			if err != nil {
				t.Fatalf("Initialize() error = %v", err)
			}

			if got := meta.GetExternalName(cr); got != tt.want {
				t.Errorf("external name = %q, want %q", got, tt.want)
			}

			stored := &contentv1alpha1.CleanupPolicy{}

			err = kube.Get(context.Background(), client.ObjectKeyFromObject(cr), stored)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if got := meta.GetExternalName(stored); got != tt.want {
				t.Errorf("stored external name = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSpecExternalName_WrongKind checks that the initializer rejects a
// managed resource of another kind instead of panicking.
func TestSpecExternalName_WrongKind(t *testing.T) {
	t.Parallel()

	cr := newCleanupPolicy("nexus-name", nil)
	kube := newExternalNameClient(t, cr)

	err := SpecExternalName(kube, func(r *iamv1alpha1.Role) string { return r.Spec.ForProvider.ID }).
		Initialize(context.Background(), cr)
	if err == nil {
		t.Fatal("Initialize() error = nil, want an unexpected kind error")
	}
}

// TestExternalNameOr checks that an external name wins over the fallback.
func TestExternalNameOr(t *testing.T) {
	t.Parallel()

	if got := ExternalNameOr(newCleanupPolicy("spec-name", nil), "spec-name"); got != "spec-name" {
		t.Errorf("ExternalNameOr() without external name = %q, want %q", got, "spec-name")
	}

	cr := newCleanupPolicy("spec-name", map[string]string{meta.AnnotationKeyExternalName: "imported"})
	if got := ExternalNameOr(cr, "spec-name"); got != "imported" {
		t.Errorf("ExternalNameOr() with external name = %q, want %q", got, "imported")
	}
}
