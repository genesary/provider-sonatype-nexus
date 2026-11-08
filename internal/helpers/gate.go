package helpers

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupFn adds a controller to a manager.
type SetupFn func(mgr ctrl.Manager, opts Options) error

// HasGate reports whether opts carries a gate. A typed nil gate, such as a
// nil *gate.Gate, is not a nil interface but is no gate either.
func (o Options) HasGate() bool {
	return o.Gate != nil && !reflect.ValueOf(o.Gate).IsNil()
}

// SetupGated runs setup once the CustomResourceDefinition of gvk is
// established, so that the provider starts even when only some of its kinds
// are activated, as Crossplane's safe-start capability allows. Without a gate
// in opts, setup runs immediately.
//
// The gate runs setup in its own goroutine once the manager is started, so a
// setup error cannot be returned to the caller. It is returned by a runnable
// added to mgr instead, which stops the manager: mgr.Start then shuts down
// cleanly, releasing the leader election lease, and returns the error.
func SetupGated(mgr ctrl.Manager, opts Options, setup SetupFn, gvk schema.GroupVersionKind) error {
	if !opts.HasGate() {
		return setup(mgr, opts)
	}

	// The gate releases setup at most once, so one slot is enough, and the
	// send never blocks even once the manager has stopped.
	errs := make(chan error, 1)

	err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return err
		}
	}))
	if err != nil {
		return errors.Wrapf(err, "cannot add the setup error reporter of %s", gvk)
	}

	opts.Gate.Register(func() {
		err := setup(mgr, opts)
		if err != nil {
			errs <- errors.Wrapf(err, "cannot set up the controller of %s", gvk)
		}
	}, gvk)

	return nil
}
