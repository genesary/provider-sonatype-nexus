package main

import (
	"time"

	"github.com/alecthomas/kingpin/v2"

	"github.com/genesary/provider-sonatype-nexus/internal/version"
)

// flags are the settings of the provider, read from its command line or from
// the environment variable named in each flag's help, which is how earlier
// releases were configured.
type flags struct {
	Debug                    bool
	LeaderElection           bool
	SyncInterval             time.Duration
	PollInterval             time.Duration
	PollStateMetricInterval  time.Duration
	MaxReconcileRate         int
	EnableManagementPolicies bool
	EnableSecretCache        bool
	EnableChangeLogs         bool
	ChangeLogsSocketPath     string
}

// newApp returns the command line of the provider, which parses its flags
// into parsed.
func newApp(name string, parsed *flags) *kingpin.Application {
	app := kingpin.New(name, "Sonatype Nexus support for Crossplane.").Version(version.Version)

	app.Flag("debug", "Run with debug logging.").Short('d').Envar("DEBUG").
		BoolVar(&parsed.Debug)
	app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Envar("LEADER_ELECTION").
		BoolVar(&parsed.LeaderElection)
	app.Flag("sync", "How often all resources will be double-checked for drift from the desired state.").Short('s').Default("1h").Envar("SYNC_INTERVAL").
		DurationVar(&parsed.SyncInterval)
	app.Flag("poll", "How often individual resources will be checked for drift from the desired state.").Default("1m").Envar("POLL_INTERVAL").
		DurationVar(&parsed.PollInterval)
	app.Flag("poll-state-metric", "How often the state metrics of the managed resources are recorded.").Default("5s").Envar("POLL_STATE_METRIC_INTERVAL").
		DurationVar(&parsed.PollStateMetricInterval)
	app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may be checked for drift from the desired state.").Default("10").Envar("MAX_RECONCILE_RATE").
		IntVar(&parsed.MaxReconcileRate)
	app.Flag("enable-management-policies", "Enable support for Management Policies. Disable with --no-enable-management-policies.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").
		BoolVar(&parsed.EnableManagementPolicies)
	app.Flag("enable-secret-cache", "Serve Kubernetes Secret reads from the informer cache, which requires cluster-wide list and watch on Secrets. Disable to read each Secret directly from the API server.").Default("true").Envar("ENABLE_SECRET_CACHE").
		BoolVar(&parsed.EnableSecretCache)
	app.Flag("enable-changelogs", "Enable support for capturing change logs during reconciliation.").Default("false").Envar("ENABLE_CHANGE_LOGS").
		BoolVar(&parsed.EnableChangeLogs)
	app.Flag("changelogs-socket-path", "Path for changelogs socket (if enabled).").Default("/var/run/changelogs/changelogs.sock").Envar("CHANGELOGS_SOCKET_PATH").
		StringVar(&parsed.ChangeLogsSocketPath)

	return app
}
