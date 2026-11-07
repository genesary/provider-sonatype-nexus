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
	}
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
			args: []string{"--debug", "--leader-election", "--sync=2h", "--poll=5m", "--max-reconcile-rate=20", "--no-enable-management-policies"},
			want: func(f *flags) {
				f.Debug = true
				f.LeaderElection = true
				f.SyncInterval = 2 * time.Hour
				f.PollInterval = 5 * time.Minute
				f.MaxReconcileRate = 20
				f.EnableManagementPolicies = false
			},
		},
		"EnvironmentVariables": {
			env: map[string]string{
				"DEBUG":              "1",
				"LEADER_ELECTION":    "true",
				"POLL_INTERVAL":      "30s",
				"MAX_RECONCILE_RATE": "5",
			},
			want: func(f *flags) {
				f.Debug = true
				f.LeaderElection = true
				f.PollInterval = 30 * time.Second
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
