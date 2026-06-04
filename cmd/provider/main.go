// Package main is the entry point for the Sonatype Nexus provider.
package main

import (
	"os"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/genesary/provider-sonatype-nexus/apis"
	nexuscontroller "github.com/genesary/provider-sonatype-nexus/internal/controller"
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

	if err := nexuscontroller.Setup(mgr, o); err != nil {
		log.Info("Cannot setup controllers", "error", err)
		os.Exit(1)
	}

	log.Info("Starting controller manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Info("Cannot start controller manager", "error", err)
		os.Exit(1)
	}
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

		for _, c := range val {
			if c < '0' || c > '9' {
				return defaultVal
			}

			i = i*10 + int(c-'0')
		}

		return i
	}

	return defaultVal
}
