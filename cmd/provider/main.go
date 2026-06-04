// Package main is the entry point for the Sonatype Nexus provider.
package main

import (
	"os"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/genesary/provider-sonatype-nexus/apis"
	nexuscontroller "github.com/genesary/provider-sonatype-nexus/internal/controller"
)

const (
	// defaultMaxReconcileRate is the default maximum reconcile rate.
	defaultMaxReconcileRate = 10
	// decimalBase is the base for decimal integer parsing.
	decimalBase = 10
)

// main is the entry point for the provider.
func main() {
	var (
		debug            = getEnvBool("DEBUG", false)
		pollInterval     = getEnvDuration("POLL_INTERVAL", time.Minute)
		leaderElection   = getEnvBool("LEADER_ELECTION", false)
		maxReconcileRate = getEnvInt("MAX_RECONCILE_RATE", defaultMaxReconcileRate)
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

	err = apis.AddToScheme(mgr.GetScheme())
	if err != nil {
		log.Info("Cannot add APIs to scheme", "error", err)
		os.Exit(1)
	}

	opts := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: maxReconcileRate,
		PollInterval:            pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(maxReconcileRate),
		Features:                &feature.Flags{},
	}

	err = nexuscontroller.Setup(mgr, opts)
	if err != nil {
		log.Info("Cannot setup controllers", "error", err)
		os.Exit(1)
	}

	log.Info("Starting controller manager")

	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		log.Info("Cannot start controller manager", "error", err)
		os.Exit(1)
	}
}

// getEnvBool returns a boolean environment variable value or a default.
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}

	return defaultVal
}

// getEnvDuration returns a duration environment variable value or a default.
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		d, err := time.ParseDuration(val)
		if err == nil {
			return d
		}
	}

	return defaultVal
}

// getEnvInt returns an integer environment variable value or a default.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int

		for _, c := range val {
			if c < '0' || c > '9' {
				return defaultVal
			}

			result = result*decimalBase + int(c-'0')
		}

		return result
	}

	return defaultVal
}
