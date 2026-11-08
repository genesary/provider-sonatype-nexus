package helpers

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	changelogsv1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/changelogs/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	resourcefake "github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
)

// sharedOptions is the number of options every controller gets.
const sharedOptions = 5

// recorderManager is a manager that only provides event recorders.
type recorderManager struct {
	ctrl.Manager
}

// GetEventRecorderFor returns a fake event recorder.
func (recorderManager) GetEventRecorderFor(string) record.EventRecorder {
	return record.NewFakeRecorder(1)
}

// nopChangeLogger is a change logger that records nothing.
type nopChangeLogger struct{}

// Log records nothing.
func (nopChangeLogger) Log(context.Context, resource.Managed, changelogsv1alpha1.OperationType, error, managed.AdditionalDetails) error {
	return nil
}

// testOptions returns the options of a provider with no feature enabled.
func testOptions() Options {
	return Options{Options: controller.Options{
		Logger:           logging.NewNopLogger(),
		Features:         &feature.Flags{},
		ChangeLogOptions: &controller.ChangeLogOptions{ChangeLogger: nopChangeLogger{}},
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
		// noChangeLogger leaves the change log options without a logger.
		noChangeLogger bool
		jitter         int
		specific       int
		want           int
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
		"ChangeLogOptionsWithoutLogger": {
			features:       []feature.Flag{feature.EnableAlphaChangeLogs},
			noChangeLogger: true,
			want:           sharedOptions,
		},
		"Metrics": {
			metrics: &controller.MetricOptions{MRMetrics: managed.NewMRMetricRecorder()},
			want:    sharedOptions + 1,
		},
		"MetricOptionsWithoutRecorder": {
			metrics: &controller.MetricOptions{},
			want:    sharedOptions,
		},
		"PollJitter": {
			jitter: MaxPollJitterPercentage,
			want:   sharedOptions + 1,
		},
		"SpecificOptions": {
			specific: 2,
			want:     sharedOptions + 2,
		},
		"AllOptions": {
			features: []feature.Flag{feature.EnableBetaManagementPolicies, feature.EnableAlphaChangeLogs},
			metrics:  &controller.MetricOptions{MRMetrics: managed.NewMRMetricRecorder()},
			jitter:   10,
			specific: 2,
			want:     sharedOptions + 6,
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
			opts.PollJitterPercentage = test.jitter

			if test.noChangeLogger {
				opts.ChangeLogOptions = &controller.ChangeLogOptions{}
			}

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

// newTestManager returns a manager whose client holds role.
func newTestManager(t *testing.T, role *iamv1alpha1.Role) recorderManager {
	t.Helper()

	scheme := runtime.NewScheme()

	err := iamv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kube := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(role).WithStatusSubresource(role).Build()

	return recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: scheme}}
}

// upToDateConnector is a connector to a Nexus where every resource exists
// and is up to date.
var upToDateConnector = managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
	return &managed.ExternalClientFns{
		ObserveFn: func(context.Context, resource.Managed) (managed.ExternalObservation, error) {
			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
		},
		DisconnectFn: func(context.Context) error { return nil },
	}, nil
})

// reconcileRoleWith runs one reconcile of role, by a reconciler built with
// ReconcilerOptions and the options returned by specific for the manager's
// client, and returns its requeue delay.
func reconcileRoleWith(t *testing.T, opts Options, role *iamv1alpha1.Role, specific func(client.Client) []managed.ReconcilerOption) time.Duration {
	t.Helper()

	mgr := newTestManager(t, role)
	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.RoleGroupVersionKind),
		ReconcilerOptions(mgr, "test", opts, specific(mgr.GetClient())...)...)

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: role.GetName(), Namespace: role.GetNamespace()}}

	got, err := reconciler.Reconcile(t.Context(), request)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	return got.RequeueAfter
}

// reconcileRole runs one reconcile of a role that is up to date with Nexus,
// and returns its requeue delay.
func reconcileRole(t *testing.T, opts Options, role *iamv1alpha1.Role) time.Duration {
	t.Helper()

	return reconcileRoleWith(t, opts, role, func(client.Client) []managed.ReconcilerOption {
		return []managed.ReconcilerOption{managed.WithExternalConnector(upToDateConnector)}
	})
}

// TestReconcilerOptionsRecoverInterruptedCreate checks that a Role whose create
// was interrupted before its outcome was recorded is observed again by
// default, including when its external name comes from the spec, and blocked
// when its external name is assigned by Nexus.
func TestReconcilerOptionsRecoverInterruptedCreate(t *testing.T) {
	t.Parallel()

	const pollInterval = time.Minute

	tests := map[string]struct {
		externalName func(client.Client) managed.ReconcilerOption
		want         time.Duration
	}{
		"DefaultObservesAgain": {
			want: pollInterval,
		},
		"SpecExternalNameObservesAgain": {
			externalName: func(kube client.Client) managed.ReconcilerOption {
				return WithSpecExternalName(kube, func(cr *iamv1alpha1.Role) string {
					return cr.Spec.ForProvider.ID
				})
			},
			want: pollInterval,
		},
		"ServerAssignedExternalNameBlocks": {
			externalName: func(client.Client) managed.ReconcilerOption {
				return WithServerAssignedExternalName()
			},
			want: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := testOptions()
			opts.PollInterval = pollInterval

			role := testRole("")
			role.Spec.ForProvider.ID = "nexus-role"
			meta.SetExternalCreatePending(role, time.Now())

			got := reconcileRoleWith(t, opts, role, func(kube client.Client) []managed.ReconcilerOption {
				specific := []managed.ReconcilerOption{managed.WithExternalConnector(upToDateConnector)}
				if test.externalName != nil {
					specific = append(specific, test.externalName(kube))
				}

				return specific
			})
			if got != test.want {
				t.Errorf("Reconcile() RequeueAfter = %v, want %v", got, test.want)
			}
		})
	}
}

// testRole returns a Role whose poll interval annotation is annotation, or
// none when it is empty.
func testRole(annotation string) *iamv1alpha1.Role {
	role := &iamv1alpha1.Role{ObjectMeta: metav1.ObjectMeta{Name: "role", Namespace: "team"}}
	if annotation != "" {
		meta.AddAnnotations(role, map[string]string{meta.AnnotationKeyPollInterval: annotation})
	}

	return role
}

// TestReconcilerOptionsPollInterval checks the requeue delay of a real
// reconcile for each value of the crossplane.io/poll-interval annotation.
func TestReconcilerOptionsPollInterval(t *testing.T) {
	t.Parallel()

	const (
		pollInterval    = time.Minute
		minPollInterval = 30 * time.Second
	)

	tests := map[string]struct {
		annotation   string
		noMinimum    bool
		pollInterval time.Duration
		want         time.Duration
	}{
		"NoAnnotationUsesThePollInterval": {
			want: pollInterval,
		},
		"InvalidAnnotationUsesThePollInterval": {
			annotation: "-1s",
			want:       pollInterval,
		},
		"AnnotationAboveTheMinimumIsHonored": {
			annotation: "2h",
			want:       2 * time.Hour,
		},
		"AnnotationBelowThePollIntervalIsHonored": {
			annotation: "45s",
			want:       45 * time.Second,
		},
		"AnnotationBelowTheMinimumIsRaised": {
			annotation: "1ms",
			want:       minPollInterval,
		},
		"AnnotationAtTheMinimumIsHonored": {
			annotation: "30s",
			want:       minPollInterval,
		},
		"ZeroMinimumRemovesTheLimit": {
			annotation: "1ms",
			noMinimum:  true,
			want:       time.Millisecond,
		},
		"MinimumIsCappedAtAShorterPollInterval": {
			annotation:   "1ms",
			pollInterval: 10 * time.Second,
			want:         10 * time.Second,
		},
		"AnnotationBetweenAShorterPollIntervalAndTheMinimumIsHonored": {
			annotation:   "20s",
			pollInterval: 10 * time.Second,
			want:         20 * time.Second,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := testOptions()
			opts.PollInterval = pollInterval
			opts.MinPollInterval = minPollInterval

			if test.noMinimum {
				opts.MinPollInterval = 0
			}

			if test.pollInterval != 0 {
				opts.PollInterval = test.pollInterval
			}

			got := reconcileRole(t, opts, testRole(test.annotation))
			if got != test.want {
				t.Errorf("Reconcile() RequeueAfter = %v, want %v", got, test.want)
			}
		})
	}
}

// TestValidatePollJitterPercentage checks the accepted range of the percentage.
func TestValidatePollJitterPercentage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		percentage int
		wantErr    bool
	}{
		"Zero":         {},
		"Default":      {percentage: 10},
		"Maximum":      {percentage: MaxPollJitterPercentage},
		"Negative":     {percentage: -1, wantErr: true},
		"AboveMaximum": {percentage: MaxPollJitterPercentage + 1, wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePollJitterPercentage(test.percentage)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidatePollJitterPercentage(%d) error = %v, want error %t", test.percentage, err, test.wantErr)
			}
		})
	}
}

// TestPollJitterHook checks the bounds of the jittered interval, and that
// samples do not pile up on a single value, the minimum in particular.
func TestPollJitterHook(t *testing.T) {
	t.Parallel()

	const (
		samples     = 2000
		minDistinct = samples * 9 / 10
	)

	tests := map[string]struct {
		percentage      int
		minPollInterval time.Duration
		pollInterval    time.Duration
		wantMin         time.Duration
		wantMax         time.Duration
	}{
		"SpreadsInBothDirections": {
			percentage:   10,
			pollInterval: time.Minute,
			wantMin:      54 * time.Second,
			wantMax:      66 * time.Second,
		},
		"ScalesWithTheInterval": {
			percentage:   10,
			pollInterval: time.Hour,
			wantMin:      54 * time.Minute,
			wantMax:      66 * time.Minute,
		},
		"NeverGoesBelowTheMinimum": {
			percentage:      10,
			minPollInterval: 30 * time.Second,
			pollInterval:    30 * time.Second,
			wantMin:         30 * time.Second,
			wantMax:         33 * time.Second,
		},
		"MaximumPercentageStaysPositive": {
			percentage:   MaxPollJitterPercentage,
			pollInterval: time.Minute,
			wantMin:      30 * time.Second,
			wantMax:      90 * time.Second,
		},
		"ClampsAnOutOfRangePercentage": {
			percentage:   4 * MaxPollJitterPercentage,
			pollInterval: time.Minute,
			wantMin:      30 * time.Second,
			wantMax:      90 * time.Second,
		},
		"SaturatesInsteadOfOverflowing": {
			percentage:      10,
			minPollInterval: 30 * time.Second,
			pollInterval:    math.MaxInt64 - time.Hour,
			wantMin:         math.MaxInt64 / 2,
			wantMax:         math.MaxInt64,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			hook := pollJitterHook(test.percentage, test.minPollInterval)
			seen := make(map[time.Duration]struct{}, samples)

			for range samples {
				got := hook(nil, test.pollInterval)
				if got < test.wantMin || got > test.wantMax {
					t.Fatalf("hook() = %v, want between %v and %v", got, test.wantMin, test.wantMax)
				}

				seen[got] = struct{}{}
			}

			if len(seen) < minDistinct {
				t.Errorf("hook() returned %d distinct intervals out of %d samples, want at least %d", len(seen), samples, minDistinct)
			}
		})
	}
}

// TestPollJitterHookKeepsUnspreadableIntervals checks that intervals with
// nothing to spread are returned as they are.
func TestPollJitterHookKeepsUnspreadableIntervals(t *testing.T) {
	t.Parallel()

	hook := pollJitterHook(10, 30*time.Second)

	for _, pollInterval := range []time.Duration{-time.Minute, 0, time.Nanosecond} {
		got := hook(nil, pollInterval)
		if got != pollInterval {
			t.Errorf("hook(%v) = %v, want it unchanged", pollInterval, got)
		}
	}
}

// TestReconcilerOptionsApplyPollJitter checks the requeue delay of a real
// reconcile when jitter is enabled: an exact interval means the hook is not
// wired, and a minimum that is not passed on lets jitter go below it.
func TestReconcilerOptionsApplyPollJitter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		annotation string
		exact      time.Duration
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		"SpreadsThePollInterval": {
			exact:   time.Minute,
			wantMin: 54 * time.Second,
			wantMax: 66 * time.Second,
		},
		"SpreadsTheAnnotation": {
			annotation: "1h",
			exact:      time.Hour,
			wantMin:    54 * time.Minute,
			wantMax:    66 * time.Minute,
		},
		"StaysAboveTheMinimum": {
			annotation: "1s",
			exact:      30 * time.Second,
			wantMin:    30 * time.Second,
			wantMax:    33 * time.Second,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			opts := testOptions()
			opts.PollInterval = time.Minute
			opts.MinPollInterval = 30 * time.Second
			opts.PollJitterPercentage = 10

			got := reconcileRole(t, opts, testRole(test.annotation))
			if got < test.wantMin || got > test.wantMax {
				t.Errorf("Reconcile() RequeueAfter = %v, want between %v and %v", got, test.wantMin, test.wantMax)
			}

			if got == test.exact {
				t.Errorf("Reconcile() RequeueAfter = %v, want it spread", got)
			}
		})
	}
}
