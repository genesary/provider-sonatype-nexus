// Package main is the entry point for the Sonatype Nexus provider.
package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
//
// Every flag can also be set through the environment variable named in its
// help, which is how earlier releases were configured.
func main() {
	var (
		app = kingpin.New(filepath.Base(os.Args[0]), "Sonatype Nexus support for Crossplane.").
			Version(version.Version)
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Envar("DEBUG").Bool()
		leaderElection   = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Envar("LEADER_ELECTION").Bool()
		syncInterval     = app.Flag("sync", "How often all resources will be double-checked for drift from the desired state.").Short('s').Default("1h").Envar("SYNC_INTERVAL").Duration()
		pollInterval     = app.Flag("poll", "How often individual resources will be checked for drift from the desired state.").Default("1m").Envar("POLL_INTERVAL").Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may be checked for drift from the desired state.").Default("10").Envar("MAX_RECONCILE_RATE").Int()

		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for Management Policies.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").Bool()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	// A rate below 1 gives the global rate limiter a zero or negative burst,
	// so no reconcile would ever be admitted.
	if *maxReconcileRate < 1 {
		app.Fatalf("--max-reconcile-rate must be at least 1, got %d", *maxReconcileRate)
	}

	zapLogger := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zapLogger.WithName("provider-sonatype-nexus"))

	if *debug {
		// controller-runtime is very verbose even at info level, so it only
		// gets a real logger in debug mode.
		ctrl.SetLogger(zapLogger)
	} else {
		// A no-op logger keeps controller-runtime from warning that none was
		// set.
		ctrl.SetLogger(logr.Discard())
	}

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	scheme, err := newScheme()
	kingpin.FatalIfError(err, "Cannot build the manager scheme")

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		Scheme: scheme,
		Cache:  cache.Options{SyncPeriod: syncInterval},

		// controller-runtime defaults to 15s leases with a 10s renewal
		// deadline, which a busy provider can miss and lose leadership.
		// Crossplane providers use Leases only, with longer leases.
		LeaderElection:             *leaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-sonatype-nexus",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              new(leaseDuration),
		RenewDeadline:              new(renewDeadline),
		// Release the lease on shutdown so a replacement replica takes over
		// at once instead of waiting out the 60s lease. main exits as soon
		// as the manager stops, so nothing runs after the release.
		LeaderElectionReleaseOnCancel: true,
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	opts := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
	}

	if *enableManagementPolicies {
		opts.Features.Enable(feature.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", feature.EnableBetaManagementPolicies)
	}

	kingpin.FatalIfError(nexuscontroller.Setup(mgr, helpers.Options{Options: opts}), "Cannot setup Nexus controllers")

	log.Info("Starting controller manager", "version", version.Version)
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
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
