package crdgate

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// crdName is the name of the CustomResourceDefinition of the tests.
const crdName = "roles.iam.nexus.crossplane.io"

// recordingGate is a gate that records the conditions set on it.
type recordingGate struct {
	set map[schema.GroupVersionKind]bool
}

// Register is not used by the reconciler.
func (*recordingGate) Register(func(), ...schema.GroupVersionKind) {}

// Set records the condition.
func (g *recordingGate) Set(gvk schema.GroupVersionKind, value bool) bool {
	g.set[gvk] = value

	return true
}

// roleCRD returns the Role CustomResourceDefinition, serving v1alpha1 but not
// v1alpha0, with the given Established status.
func roleCRD(established apiextensionsv1.ConditionStatus) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: crdName},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "iam.nexus.crossplane.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: "Role"},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: "v1alpha0", Served: false},
				{Name: "v1alpha1", Served: true},
			},
		},
		Status: apiextensionsv1.CustomResourceDefinitionStatus{
			Conditions: []apiextensionsv1.CustomResourceDefinitionCondition{{Type: apiextensionsv1.Established, Status: established}},
		},
	}
}

// TestCRDName checks the CustomResourceDefinition names of a few kinds.
func TestCRDName(t *testing.T) {
	t.Parallel()

	for kind, want := range map[string]string{
		"Role":            "roles.iam.nexus.crossplane.io",
		"AnonymousAccess": "anonymousaccesses.iam.nexus.crossplane.io",
		"Capability":      "capabilities.iam.nexus.crossplane.io",
	} {
		got := CRDName(schema.GroupVersionKind{Group: "iam.nexus.crossplane.io", Version: "v1alpha1", Kind: kind})
		if got != want {
			t.Errorf("CRDName(%s) = %s, want %s", kind, got, want)
		}
	}
}

// TestReconcile checks the gate conditions set for each state of a watched
// CustomResourceDefinition, and that other definitions are ignored.
func TestReconcile(t *testing.T) {
	t.Parallel()

	served := schema.GroupVersionKind{Group: "iam.nexus.crossplane.io", Version: "v1alpha1", Kind: "Role"}
	unserved := schema.GroupVersionKind{Group: "iam.nexus.crossplane.io", Version: "v1alpha0", Kind: "Role"}

	deleting := roleCRD(apiextensionsv1.ConditionTrue)
	deleting.DeletionTimestamp = new(metav1.Now())
	deleting.Finalizers = []string{"test"}

	cases := map[string]struct {
		crd     *apiextensionsv1.CustomResourceDefinition
		request string
		want    map[schema.GroupVersionKind]bool
	}{
		"Established":    {crd: roleCRD(apiextensionsv1.ConditionTrue), request: crdName, want: map[schema.GroupVersionKind]bool{served: true}},
		"NotEstablished": {crd: roleCRD(apiextensionsv1.ConditionFalse), request: crdName, want: map[schema.GroupVersionKind]bool{served: false, unserved: false}},
		"Deleting":       {crd: deleting, request: crdName, want: map[schema.GroupVersionKind]bool{served: false, unserved: false}},
		"NotFound":       {request: crdName, want: map[schema.GroupVersionKind]bool{}},
		"NotWatched":     {crd: roleCRD(apiextensionsv1.ConditionTrue), request: "other.example.com", want: map[schema.GroupVersionKind]bool{}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			err := apiextensionsv1.AddToScheme(scheme)
			if err != nil {
				t.Fatal(err)
			}

			builder := fake.NewClientBuilder().WithScheme(scheme)
			if tc.crd != nil {
				builder = builder.WithObjects(tc.crd)
			}

			gate := &recordingGate{set: map[schema.GroupVersionKind]bool{}}
			reconciler := NewReconciler(logging.NewNopLogger(), gate, map[string]client.Reader{crdName: builder.Build()})

			_, err = reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.request}})
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			if len(gate.set) != len(tc.want) {
				t.Fatalf("Reconcile() set %v, want %v", gate.set, tc.want)
			}

			for gvk, want := range tc.want {
				if got, set := gate.set[gvk]; !set || got != want {
					t.Errorf("Reconcile() set %s to %v (set: %v), want %v", gvk, got, set, want)
				}
			}
		})
	}
}
