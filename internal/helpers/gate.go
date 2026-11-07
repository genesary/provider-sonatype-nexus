package helpers

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupFn adds a controller to a manager.
type SetupFn func(mgr ctrl.Manager, opts Options) error

// SetupGated runs setup once the CustomResourceDefinition of gvk is
// established, so that the provider starts even when only some of its kinds
// are activated, as Crossplane's safe-start capability allows. Without a gate
// in opts, setup runs immediately.
//
// The gate runs setup in its own goroutine once the manager is started, so a
// setup error cannot be returned: it panics instead, which restarts the
// provider rather than leaving it running without the controller.
func SetupGated(mgr ctrl.Manager, opts Options, setup SetupFn, gvk schema.GroupVersionKind) error {
	if opts.Gate == nil {
		return setup(mgr, opts)
	}

	opts.Gate.Register(func() {
		err := setup(mgr, opts)
		if err != nil {
			panic(errors.Wrapf(err, "cannot set up the controller of %s", gvk))
		}
	}, gvk)

	return nil
}
