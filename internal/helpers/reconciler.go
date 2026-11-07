package helpers

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	ctrl "sigs.k8s.io/controller-runtime"
)

// maxSharedReconcilerOptions is the most options ReconcilerOptions shares
// between controllers, to size the slice it returns.
const maxSharedReconcilerOptions = 3

// Options are the crossplane-runtime controller options, extended with the
// settings of this provider.
type Options struct {
	controller.Options
}

// ReconcilerOptions returns the managed reconciler options shared by every
// controller, followed by the options specific to the calling controller.
func ReconcilerOptions(mgr ctrl.Manager, name string, opts Options, specific ...managed.ReconcilerOption) []managed.ReconcilerOption {
	options := make([]managed.ReconcilerOption, 0, maxSharedReconcilerOptions+len(specific))
	options = append(options,
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(NewEventRecorder(mgr, name, opts.EventFilterFunctions...)),
	)

	return append(options, specific...)
}

// NewEventRecorder returns a Crossplane event recorder for the named
// controller. crossplane-runtime's APIRecorder still requires the legacy
// record.EventRecorder, which controller-runtime only exposes through the
// deprecated GetEventRecorderFor. This is the single place where that
// deprecated call is made. Events matching any of the filters are not recorded.
func NewEventRecorder(mgr ctrl.Manager, name string, filters ...event.FilterFn) event.Recorder {
	return event.NewAPIRecorder(mgr.GetEventRecorderFor(name), filters...)
}
