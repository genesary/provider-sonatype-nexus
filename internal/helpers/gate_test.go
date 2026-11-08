package helpers

import (
	"context"
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

	t.Run("WithTypedNilGate", func(t *testing.T) {
		t.Parallel()

		opts := testOptions()
		opts.Gate = (*recordingGate)(nil)
		calls := 0

		err := SetupGated(nil, opts, func(ctrl.Manager, Options) error {
			calls++

			return nil
		}, gvk)
		if err != nil {
			t.Fatalf("SetupGated() error = %v", err)
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

		err := SetupGated(&runnableManager{}, opts, func(ctrl.Manager, Options) error {
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

		mgr := &runnableManager{}
		recorder := &recordingGate{}
		opts := testOptions()
		opts.Gate = recorder

		err := SetupGated(mgr, opts, func(ctrl.Manager, Options) error { return errSetup }, gvk)
		if err != nil {
			t.Fatalf("SetupGated() error = %v", err)
		}

		if len(mgr.added) != 1 {
			t.Fatalf("SetupGated() added %d runnables, want 1", len(mgr.added))
		}

		// The gate runs released callbacks in their own goroutine.
		go recorder.fns[0]()

		err = mgr.added[0].Start(t.Context())
		if !errors.Is(err, errSetup) {
			t.Errorf("runnable error = %v, want %v", err, errSetup)
		}
	})

	t.Run("WithGateManagerStopped", func(t *testing.T) {
		t.Parallel()

		mgr := &runnableManager{}
		recorder := &recordingGate{}
		opts := testOptions()
		opts.Gate = recorder

		err := SetupGated(mgr, opts, func(ctrl.Manager, Options) error { return errSetup }, gvk)
		if err != nil {
			t.Fatalf("SetupGated() error = %v", err)
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err = mgr.added[0].Start(ctx)
		if err != nil {
			t.Errorf("runnable error = %v once stopped, want nil", err)
		}

		// A setup failing once nothing receives its error must not block.
		recorder.fns[0]()
	})
}
