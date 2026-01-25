// Package main is the entry point for the Sonatype Nexus provider.
package main

import (
	"os"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/blobstore"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/repository"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/role"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/controller/user"
)

func main() {
	var (
		debug            = getEnvBool("DEBUG", false)
		pollInterval     = getEnvDuration("POLL_INTERVAL", time.Minute)
		leaderElection   = getEnvBool("LEADER_ELECTION", false)
		maxReconcileRate = getEnvInt("MAX_RECONCILE_RATE", 10)
	)

	zl := zap.New(zap.UseDevMode(debug))
	log := logging.NewLogrLogger(zl.WithName("provider-sonatype-nexus"))
	ctrl.SetLogger(zl)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Info("Cannot get config", "error", err)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:             leaderElection,
		LeaderElectionID:           "crossplane-leader-election-provider-sonatype-nexus",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
	})
	if err != nil {
		log.Info("Cannot create controller manager", "error", err)
		os.Exit(1)
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Info("Cannot add APIs to scheme", "error", err)
		os.Exit(1)
	}

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: maxReconcileRate,
		PollInterval:            pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(maxReconcileRate),
		Features:                &feature.Flags{},
	}

	if err := setupControllers(mgr, o); err != nil {
		log.Info("Cannot setup controllers", "error", err)
		os.Exit(1)
	}

	log.Info("Starting controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Info("Cannot start controller manager", "error", err)
		os.Exit(1)
	}
}

func setupControllers(mgr ctrl.Manager, o controller.Options) error {
	if err := blobstore.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "cannot setup BlobStore controller")
	}

	if err := repository.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "cannot setup Repository controller")
	}

	if err := user.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "cannot setup User controller")
	}

	if err := role.Setup(mgr, o); err != nil {
		return errors.Wrap(err, "cannot setup Role controller")
	}

	return nil
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := parseIntFromString(val, &i); err == nil {
			return i
		}
	}
	return defaultVal
}

func parseIntFromString(s string, i *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	*i = n
	return n, nil
}
