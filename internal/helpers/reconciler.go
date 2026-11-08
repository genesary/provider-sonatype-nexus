package helpers

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
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
		managed.WithLogger(controllerLogger(opts, name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(newEventRecorder(mgr, name, opts)),
	)

	return append(options, specific...)
}

// ProviderConfigReconcilerOptions returns the providerconfig reconciler
// options of the named controller. They give it the same logger and event
// recorder as the managed reconcilers built from ReconcilerOptions.
func ProviderConfigReconcilerOptions(mgr ctrl.Manager, name string, opts Options) []providerconfig.ReconcilerOption {
	return []providerconfig.ReconcilerOption{
		providerconfig.WithLogger(controllerLogger(opts, name)),
		providerconfig.WithRecorder(newEventRecorder(mgr, name, opts)),
	}
}

// controllerLogger returns the logger of the named controller.
func controllerLogger(opts Options, name string) logging.Logger {
	return opts.Logger.WithValues("controller", name)
}

// newEventRecorder returns a Crossplane event recorder for the named
// controller. crossplane-runtime's APIRecorder still requires the legacy
// record.EventRecorder, which controller-runtime only exposes through the
// deprecated GetEventRecorderFor. This is the single place where that
// deprecated call is made. Events matching any of the configured filters are
// not recorded.
func newEventRecorder(mgr ctrl.Manager, name string, opts Options) event.Recorder {
	return event.NewAPIRecorder(mgr.GetEventRecorderFor(name), opts.EventFilterFunctions...)
}
