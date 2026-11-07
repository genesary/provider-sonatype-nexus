package helpers

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// recordingGate is a gate that records the callbacks registered with it,
// instead of running them once their conditions are met.
type recordingGate struct {
	fns     []func()
	depends [][]schema.GroupVersionKind
}

// Register records fn and its dependencies.
func (g *recordingGate) Register(fn func(), depends ...schema.GroupVersionKind) {
	g.fns = append(g.fns, fn)
	g.depends = append(g.depends, depends)
}

// Set is not used by SetupGated.
func (*recordingGate) Set(schema.GroupVersionKind, bool) bool { return false }

// TestSetupGated checks that a controller is set up immediately without a
// gate, and otherwise only once the gate releases it.
func TestSetupGated(t *testing.T) {
	t.Parallel()

	gvk := schema.GroupVersionKind{Group: "iam.nexus.crossplane.io", Version: "v1alpha1", Kind: "Role"}
	errSetup := errors.New("setup failed")

	t.Run("WithoutGate", func(t *testing.T) {
		t.Parallel()

		calls := 0

		err := SetupGated(nil, testOptions(), func(ctrl.Manager, Options) error {
			calls++

			return errSetup
		}, gvk)
		if !errors.Is(err, errSetup) {
			t.Errorf("SetupGated() error = %v, want %v", err, errSetup)
		}

		if calls != 1 {
			t.Errorf("setup ran %d times, want 1", calls)
		}
	})

	t.Run("WithGate", func(t *testing.T) {
		t.Parallel()

		recorder := &recordingGate{}
		opts := testOptions()
		opts.Gate = recorder
		calls := 0

		err := SetupGated(nil, opts, func(ctrl.Manager, Options) error {
			calls++

			return nil
		}, gvk)
		if err != nil {
			t.Fatalf("SetupGated() error = %v", err)
		}

		if calls != 0 {
			t.Fatalf("setup ran %d times before the gate released it, want 0", calls)
		}

		if len(recorder.fns) != 1 || len(recorder.depends[0]) != 1 || recorder.depends[0][0] != gvk {
			t.Fatalf("SetupGated() registered %v, want one callback depending on %v", recorder.depends, gvk)
		}

		recorder.fns[0]()

		if calls != 1 {
			t.Errorf("setup ran %d times once released, want 1", calls)
		}
	})

	t.Run("WithGateSetupFails", func(t *testing.T) {
		t.Parallel()

		recorder := &recordingGate{}
		opts := testOptions()
		opts.Gate = recorder

		err := SetupGated(nil, opts, func(ctrl.Manager, Options) error { return errSetup }, gvk)
		if err != nil {
			t.Fatalf("SetupGated() error = %v", err)
		}

		defer func() {
			recovered := recover()

			recoveredErr, isErr := recovered.(error)
			if !isErr || !errors.Is(recoveredErr, errSetup) {
				t.Errorf("released setup panicked with %v, want %v", recovered, errSetup)
			}
		}()

		recorder.fns[0]()
	})
}
