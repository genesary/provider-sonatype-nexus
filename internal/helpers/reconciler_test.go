package helpers

import (
	"context"
	"errors"
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
const sharedOptions = 4

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

// newTestReconciler returns a reconciler for a Role that is up to date with
// Nexus, built with the options of ReconcilerOptions.
func newTestReconciler(t *testing.T, opts Options, role *iamv1alpha1.Role) *managed.Reconciler {
	t.Helper()

	scheme := runtime.NewScheme()

	err := iamv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kube := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(role).WithStatusSubresource(role).Build()
	mgr := recorderManager{Manager: &resourcefake.Manager{Client: kube, Scheme: scheme}}

	connector := managed.ExternalConnectorFn(func(context.Context, resource.Managed) (managed.ExternalClient, error) {
		return &managed.ExternalClientFns{
			ObserveFn: func(context.Context, resource.Managed) (managed.ExternalObservation, error) {
				return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
			},
			DisconnectFn: func(context.Context) error { return nil },
		}, nil
	})

	return managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.RoleGroupVersionKind),
		ReconcilerOptions(mgr, "test", opts, managed.WithExternalConnector(connector))...)
}

// reconcileRole runs one reconcile of role with opts and returns its requeue
// delay.
func reconcileRole(t *testing.T, opts Options, role *iamv1alpha1.Role) time.Duration {
	t.Helper()

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: role.GetName(), Namespace: role.GetNamespace()}}

	got, err := newTestReconciler(t, opts, role).Reconcile(t.Context(), request)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	return got.RequeueAfter
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
