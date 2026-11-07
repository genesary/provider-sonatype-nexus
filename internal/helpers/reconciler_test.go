package helpers

import (
	"errors"
	"slices"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// sharedOptions is the number of options every controller gets.
const sharedOptions = 3

// recorderManager is a manager that only provides event recorders.
type recorderManager struct {
	ctrl.Manager
}

// GetEventRecorderFor returns a fake event recorder.
func (recorderManager) GetEventRecorderFor(string) record.EventRecorder {
	return record.NewFakeRecorder(1)
}

// testOptions returns the options of a provider with no feature enabled.
func testOptions() Options {
	return Options{Options: controller.Options{
		Logger:           logging.NewNopLogger(),
		Features:         &feature.Flags{},
		ChangeLogOptions: &controller.ChangeLogOptions{},
	}}
}

// TestReconcilerOptions checks which options are enabled by each setting, and
// that the options of the calling controller are kept, in order, after the
// shared ones so that they can override them.
func TestReconcilerOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		features []feature.Flag
		metrics  *controller.MetricOptions
		specific int
		want     int
	}{
		"NoFeatures": {
			want: sharedOptions,
		},
		"ManagementPolicies": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies},
			want:     sharedOptions + 1,
		},
		"ChangeLogs": {
			features: []feature.Flag{feature.EnableAlphaChangeLogs},
			want:     sharedOptions + 1,
		},
		"Metrics": {
			metrics: &controller.MetricOptions{MRMetrics: managed.NewMRMetricRecorder()},
			want:    sharedOptions + 1,
		},
		"MetricOptionsWithoutRecorder": {
			metrics: &controller.MetricOptions{},
			want:    sharedOptions,
		},
		"SpecificOptions": {
			specific: 2,
			want:     sharedOptions + 2,
		},
		"AllOptions": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies, feature.EnableAlphaChangeLogs},
			metrics:  &controller.MetricOptions{MRMetrics: managed.NewMRMetricRecorder()},
			specific: 2,
			want:     sharedOptions + 5,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var applied []int

			wantApplied := make([]int, test.specific)
			specific := make([]managed.ReconcilerOption, test.specific)

			for index := range specific {
				wantApplied[index] = index
				specific[index] = func(*managed.Reconciler) { applied = append(applied, index) }
			}

			opts := testOptions()
			opts.MetricOptions = test.metrics

			for _, flag := range test.features {
				opts.Features.Enable(flag)
			}

			got := ReconcilerOptions(recorderManager{}, "test", opts, specific...)
			if len(got) != test.want {
				t.Fatalf("ReconcilerOptions() returned %d options, want %d", len(got), test.want)
			}

			for _, option := range got[len(got)-test.specific:] {
				option(&managed.Reconciler{})
			}

			if !slices.Equal(applied, wantApplied) {
				t.Errorf("last options applied specific options %v, want %v", applied, wantApplied)
			}
		})
	}
}

// TestProviderConfigReconcilerOptions checks that the providerconfig
// reconciler options set a logger and an event recorder.
func TestProviderConfigReconcilerOptions(t *testing.T) {
	t.Parallel()

	const wantOptions = 2

	got := ProviderConfigReconcilerOptions(recorderManager{}, "test", testOptions())
	if len(got) != wantOptions {
		t.Fatalf("ProviderConfigReconcilerOptions() returned %d options, want %d", len(got), wantOptions)
	}

	for _, option := range got {
		option(&providerconfig.Reconciler{})
	}
}

// runnableManager is a manager that records the runnables added to it.
type runnableManager struct {
	ctrl.Manager

	added []manager.Runnable
	err   error
}

// GetClient returns a nil client, as the recorder is never started.
func (*runnableManager) GetClient() client.Client { return nil }

// Add records the runnable and returns the configured error.
func (m *runnableManager) Add(runnable manager.Runnable) error {
	m.added = append(m.added, runnable)

	return m.err
}

// TestAddMRStateRecorder checks that a recorder is only registered when state
// metrics are enabled, and that a registration failure is reported.
func TestAddMRStateRecorder(t *testing.T) {
	t.Parallel()

	errAdd := errors.New("add failed")

	tests := map[string]struct {
		metrics   *controller.MetricOptions
		addErr    error
		wantAdded int
		wantErr   error
	}{
		"NoMetricOptions": {},
		"NoStateMetrics": {
			metrics: &controller.MetricOptions{},
		},
		"StateMetrics": {
			metrics:   &controller.MetricOptions{MRStateMetrics: statemetrics.NewMRStateMetrics()},
			wantAdded: 1,
		},
		"AddFails": {
			metrics:   &controller.MetricOptions{MRStateMetrics: statemetrics.NewMRStateMetrics()},
			addErr:    errAdd,
			wantAdded: 1,
			wantErr:   errAdd,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mgr := &runnableManager{err: test.addErr}
			opts := testOptions()
			opts.MetricOptions = test.metrics

			err := AddMRStateRecorder(mgr, opts, &iamv1alpha1.RoleList{})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("AddMRStateRecorder() error = %v, want %v", err, test.wantErr)
			}

			if len(mgr.added) != test.wantAdded {
				t.Fatalf("AddMRStateRecorder() added %d runnables, want %d", len(mgr.added), test.wantAdded)
			}

			for _, runnable := range mgr.added {
				if _, isRecorder := runnable.(*statemetrics.MRStateRecorder); !isRecorder {
					t.Errorf("AddMRStateRecorder() added a %T, want a *statemetrics.MRStateRecorder", runnable)
				}
			}
		})
	}
}
