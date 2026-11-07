package main

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// defaultFlags are the settings of a provider started with no flag and no
// environment variable.
func defaultFlags() flags {
	return flags{
		SyncInterval:             time.Hour,
		PollInterval:             time.Minute,
		PollStateMetricInterval:  5 * time.Second,
		MaxReconcileRate:         10,
		EnableManagementPolicies: true,
		ChangeLogsSocketPath:     "/var/run/changelogs/changelogs.sock",
	}
}

// providerEnvars are the environment variables the provider flags read.
var providerEnvars = []string{
	"DEBUG",
	"LEADER_ELECTION",
	"SYNC_INTERVAL",
	"POLL_INTERVAL",
	"POLL_STATE_METRIC_INTERVAL",
	"MAX_RECONCILE_RATE",
	"ENABLE_MANAGEMENT_POLICIES",
}

// TestFlags checks the defaults of the provider flags, and that flags take
// precedence over their environment variables.
func TestFlags(t *testing.T) {
	tests := map[string]struct {
		args    []string
		env     map[string]string
		want    func(*flags)
		wantErr bool
	}{
		"Defaults": {
			want: func(*flags) {},
		},
		"Flags": {
			args: []string{"--debug", "--leader-election", "--sync=2h", "--poll=5m", "--poll-state-metric=30s", "--max-reconcile-rate=20", "--no-enable-management-policies", "--enable-changelogs", "--changelogs-socket-path=/tmp/changelogs.sock"},
			want: func(f *flags) {
				f.Debug = true
				f.LeaderElection = true
				f.SyncInterval = 2 * time.Hour
				f.PollInterval = 5 * time.Minute
				f.PollStateMetricInterval = 30 * time.Second
				f.MaxReconcileRate = 20
				f.EnableManagementPolicies = false
				f.EnableChangeLogs = true
				f.ChangeLogsSocketPath = "/tmp/changelogs.sock"
			},
		},
		"EnvironmentVariables": {
			env: map[string]string{
				"DEBUG":                      "1",
				"LEADER_ELECTION":            "true",
				"POLL_INTERVAL":              "30s",
				"POLL_STATE_METRIC_INTERVAL": "10s",
				"MAX_RECONCILE_RATE":         "5",
			},
			want: func(f *flags) {
				f.Debug = true
				f.LeaderElection = true
				f.PollInterval = 30 * time.Second
				f.PollStateMetricInterval = 10 * time.Second
				f.MaxReconcileRate = 5
			},
		},
		"FlagsOverrideEnvironmentVariables": {
			args: []string{"--poll=2m"},
			env:  map[string]string{"POLL_INTERVAL": "30s"},
			want: func(f *flags) { f.PollInterval = 2 * time.Minute },
		},
		"InvalidEnvironmentVariable": {
			env:     map[string]string{"POLL_INTERVAL": "5"},
			wantErr: true,
		},
		"UnknownFlag": {
			args:    []string{"--unknown"},
			wantErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Blank every provider variable, so one set in the environment
			// running the tests does not change the expected settings.
			// kingpin ignores an empty variable.
			for _, key := range providerEnvars {
				t.Setenv(key, "")
			}

			for key, value := range test.env {
				t.Setenv(key, value)
			}

			var got flags

			_, err := newApp("provider", &got).Parse(test.args)
			if test.wantErr {
				if err == nil {
					t.Fatal("Parse() error = nil, want an error")
				}

				return
			}

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			want := defaultFlags()
			test.want(&want)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Parse() -want +got:\n%s", diff)
			}
		})
	}
}
