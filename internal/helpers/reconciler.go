package helpers

import (
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

// maxSharedReconcilerOptions is the most options ReconcilerOptions shares
// between controllers, to size the slice it returns.
const maxSharedReconcilerOptions = 7

// Options are the crossplane-runtime controller options, extended with the
// settings of this provider.
type Options struct {
	controller.Options

	// MinPollInterval is the shortest poll interval a resource may request
	// through the crossplane.io/poll-interval annotation. Shorter requests are
	// raised to it, which keeps the annotation from letting one resource use
	// the reconcile budget shared by all tenants, or flood Nexus with
	// requests. Zero removes the limit. It never exceeds PollInterval, see
	// minPollInterval.
	MinPollInterval time.Duration
}

// minPollInterval returns the floor applied to the poll interval annotation.
// It is capped at the poll interval, so that an annotation can always ask for
// the polling every resource without one already gets: with --poll=10s and the
// default 30s minimum, an annotation of 20s would otherwise be raised to 30s,
// polling slower than unannotated resources.
func minPollInterval(opts Options) time.Duration {
	if opts.PollInterval > 0 && opts.PollInterval < opts.MinPollInterval {
		return opts.PollInterval
	}

	return opts.MinPollInterval
}

// ReconcilerOptions returns the managed reconciler options shared by every
// controller, followed by the options specific to the calling controller.
func ReconcilerOptions(mgr ctrl.Manager, name string, opts Options, specific ...managed.ReconcilerOption) []managed.ReconcilerOption {
	options := make([]managed.ReconcilerOption, 0, maxSharedReconcilerOptions+len(specific))
	options = append(options,
		managed.WithLogger(controllerLogger(opts, name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithMinPollInterval(minPollInterval(opts)),
		managed.WithRecorder(newEventRecorder(mgr, name, opts)),
	)

	if opts.Features.Enabled(feature.EnableBetaManagementPolicies) {
		options = append(options, managed.WithManagementPolicies())
	}

	// A nil change logger would replace the reconciler's no-op one, and the
	// reconciler would panic on its first create, update or delete.
	if opts.Features.Enabled(feature.EnableAlphaChangeLogs) && opts.ChangeLogOptions != nil && opts.ChangeLogOptions.ChangeLogger != nil {
		options = append(options, managed.WithChangeLogger(opts.ChangeLogOptions.ChangeLogger))
	}

	if opts.MetricOptions != nil && opts.MetricOptions.MRMetrics != nil {
		options = append(options, managed.WithMetricRecorder(opts.MetricOptions.MRMetrics))
	}

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

// AddMRStateRecorder registers with mgr a recorder of the state metrics of the
// managed resources in list. It does nothing when state metrics are disabled.
func AddMRStateRecorder(mgr ctrl.Manager, opts Options, list resource.ManagedList) error {
	if opts.MetricOptions == nil || opts.MetricOptions.MRStateMetrics == nil {
		return nil
	}

	recorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), opts.Logger, opts.MetricOptions.MRStateMetrics, list, opts.MetricOptions.PollStateMetricInterval,
	)

	err := mgr.Add(recorder)
	if err != nil {
		return errors.Wrapf(err, "cannot register MR state metrics recorder for kind %T", list)
	}

	return nil
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
