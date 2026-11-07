// Package main is the entry point for the Sonatype Nexus provider.
package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin/v2"
	changelogsv1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/changelogs/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/genesary/provider-sonatype-nexus/apis"
	nexuscontroller "github.com/genesary/provider-sonatype-nexus/internal/controller"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
	"github.com/genesary/provider-sonatype-nexus/internal/version"
)

const (
	// leaseDuration is how long a leader election lease is valid.
	leaseDuration = 60 * time.Second
	// renewDeadline is how long the leader keeps trying to renew its lease
	// before giving it up.
	renewDeadline = 50 * time.Second
)

// main is the entry point for the provider.
func main() {
	var parsed flags

	kingpin.MustParse(newApp(filepath.Base(os.Args[0]), &parsed).Parse(os.Args[1:]))

	log := newLogger(parsed.Debug)

	// A rate below 1 gives the global rate limiter a zero or negative burst,
	// so no reconcile would ever be admitted.
	if parsed.MaxReconcileRate < 1 {
		kingpin.Fatalf("--max-reconcile-rate must be at least 1, got %d", parsed.MaxReconcileRate)
	}

	// The state metrics recorders tick at this interval, and a ticker panics
	// on a zero or negative one once the manager starts them.
	if parsed.PollStateMetricInterval <= 0 {
		kingpin.Fatalf("--poll-state-metric must be positive, got %s", parsed.PollStateMetricInterval)
	}

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := newManager(cfg, parsed)
	kingpin.FatalIfError(err, "Cannot create controller manager")

	opts := controllerOptions(log, parsed)

	if parsed.EnableManagementPolicies {
		opts.Features.Enable(feature.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", feature.EnableBetaManagementPolicies)
	}

	if parsed.EnableChangeLogs {
		changeLogOptions, err := newChangeLogOptions(parsed.ChangeLogsSocketPath)
		kingpin.FatalIfError(err, "Cannot set up change logs")

		opts.Features.Enable(feature.EnableAlphaChangeLogs)
		opts.ChangeLogOptions = changeLogOptions
		log.Info("Alpha feature enabled", "flag", feature.EnableAlphaChangeLogs, "socket", parsed.ChangeLogsSocketPath)
	}

	kingpin.FatalIfError(nexuscontroller.Setup(mgr, opts), "Cannot setup Nexus controllers")

	log.Info("Starting controller manager", "version", version.Version)
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

// newLogger returns the provider logger, and sets the controller-runtime one.
func newLogger(debug bool) logging.Logger {
	zapLogger := zap.New(zap.UseDevMode(debug))

	if debug {
		// controller-runtime is very verbose even at info level, so it only
		// gets a real logger in debug mode.
		ctrl.SetLogger(zapLogger)
	} else {
		// A no-op logger keeps controller-runtime from warning that none was
		// set.
		ctrl.SetLogger(logr.Discard())
	}

	return logging.NewLogrLogger(zapLogger.WithName("provider-sonatype-nexus"))
}

// newManager returns the controller manager of the provider.
func newManager(cfg *rest.Config, parsed flags) (ctrl.Manager, error) {
	scheme, err := newScheme()
	if err != nil {
		return nil, errors.Wrap(err, "cannot build the manager scheme")
	}

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, parsed.MaxReconcileRate), ctrl.Options{
		Scheme: scheme,
		Cache:  cache.Options{SyncPeriod: &parsed.SyncInterval},

		// controller-runtime defaults to 15s leases with a 10s renewal
		// deadline, which a busy provider can miss and lose leadership.
		// Crossplane providers use Leases only, with longer leases.
		LeaderElection:             parsed.LeaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-sonatype-nexus",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              new(leaseDuration),
		RenewDeadline:              new(renewDeadline),
		// Release the lease on shutdown so a replacement replica takes over
		// at once instead of waiting out the 60s lease. main exits as soon
		// as the manager stops, so nothing runs after the release.
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create the manager")
	}

	return mgr, nil
}

// newScheme returns the manager scheme: the Kubernetes built-in types, which
// the provider reads Secrets through, and the provider's own APIs.
func newScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	builder := runtime.NewSchemeBuilder(clientgoscheme.AddToScheme, apis.AddToScheme)

	err := builder.AddToScheme(scheme)
	if err != nil {
		return nil, errors.Wrap(err, "cannot add types to scheme")
	}

	return scheme, nil
}

// controllerOptions returns the options shared by the provider controllers.
// It registers the managed resource metrics with the controller-runtime
// registry, which the manager serves.
func controllerOptions(log logging.Logger, parsed flags) helpers.Options {
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	metrics.Registry.MustRegister(metricRecorder)
	metrics.Registry.MustRegister(stateMetrics)

	return helpers.Options{
		Options: controller.Options{
			Logger:                  log,
			MaxConcurrentReconciles: parsed.MaxReconcileRate,
			PollInterval:            parsed.PollInterval,
			GlobalRateLimiter:       ratelimiter.NewGlobal(parsed.MaxReconcileRate),
			Features:                &feature.Flags{},
			MetricOptions: &controller.MetricOptions{
				PollStateMetricInterval: parsed.PollStateMetricInterval,
				MRMetrics:               metricRecorder,
				MRStateMetrics:          stateMetrics,
			},
		},
	}
}

// newChangeLogOptions returns the options that send change logs to the
// change logs service listening on the socket at socketPath. The connection
// is only established when the first change log is sent.
func newChangeLogOptions(socketPath string) (*controller.ChangeLogOptions, error) {
	conn, err := grpc.NewClient("unix://"+socketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create change logs client connection at %s", socketPath)
	}

	return &controller.ChangeLogOptions{
		ChangeLogger: managed.NewGRPCChangeLogger(
			changelogsv1alpha1.NewChangeLogServiceClient(conn),
			managed.WithProviderVersion("provider-sonatype-nexus:"+version.Version),
		),
	}, nil
}
