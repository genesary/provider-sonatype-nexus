package helpers

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// TestFinalizerKeepsTheObservedStatus checks, through a real first reconcile
// of a Role whose Nexus role already exists, that the status recorded by
// Observe is persisted, rather than discarded when the finalizer is added.
func TestFinalizerKeepsTheObservedStatus(t *testing.T) {
	t.Parallel()

	observing := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(_ context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
				role, _ := mg.(*iamv1alpha1.Role)
				role.Status.AtProvider.ID = "nexus-role"
				role.SetConditions(xpv2.Available())

				return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	tests := map[string]struct {
		finalizer func(client.Client) resource.Finalizer
		wantReady bool
	}{
		"StatusPreservingFinalizer": {
			wantReady: true,
		},
		// Documents the crossplane-runtime behavior worked around: when this
		// case fails, the workaround is no longer needed.
		"UpstreamFinalizerLosesTheStatus": {
			finalizer: func(kube client.Client) resource.Finalizer {
				return resource.NewAPIFinalizer(kube, managed.FinalizerName)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mgr := newTestManager(t, testRole(""))

			specific := []managed.ReconcilerOption{managed.WithExternalConnector(observing)}
			if test.finalizer != nil {
				specific = append(specific, managed.WithFinalizer(test.finalizer(mgr.GetClient())))
			}

			reconcileWithManager(t, mgr, testOptions(), testRole(""), specific)

			stored := &iamv1alpha1.Role{}

			err := mgr.GetClient().Get(context.Background(), client.ObjectKey{Name: "role", Namespace: "team"}, stored)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if len(stored.GetFinalizers()) != 1 {
				t.Fatalf("finalizers = %v, want the managed resource finalizer", stored.GetFinalizers())
			}

			ready := stored.GetCondition(xpv2.TypeReady).Status == corev1.ConditionTrue
			if ready != test.wantReady {
				t.Errorf("stored Ready = %t, want %t", ready, test.wantReady)
			}

			if (stored.Status.AtProvider.ID == "nexus-role") != test.wantReady {
				t.Errorf("stored status.atProvider.id = %q, want it kept %t", stored.Status.AtProvider.ID, test.wantReady)
			}
		})
	}
}

// TestRestoreStatusIgnoresMismatchedObjects checks that a status is only
// copied between objects of the same type.
func TestRestoreStatusIgnoresMismatchedObjects(t *testing.T) {
	t.Parallel()

	role := testRole("")
	role.Status.AtProvider.ID = "kept"

	restoreStatus(role, &iamv1alpha1.User{})
	restoreStatus(role, nil)

	if role.Status.AtProvider.ID != "kept" {
		t.Errorf("status.atProvider.id = %q, want it untouched", role.Status.AtProvider.ID)
	}
}
