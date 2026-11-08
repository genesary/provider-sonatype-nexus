package helpers

import (
	"slices"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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
		Logger:   logging.NewNopLogger(),
		Features: &feature.Flags{},
	}}
}

// TestReconcilerOptions checks which options are enabled by each setting, and
// that the options of the calling controller are kept, in order, after the
// shared ones so that they can override them.
func TestReconcilerOptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		features []feature.Flag
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
		"SpecificOptions": {
			specific: 2,
			want:     sharedOptions + 2,
		},
		"AllOptions": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies},
			specific: 2,
			want:     sharedOptions + 3,
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
