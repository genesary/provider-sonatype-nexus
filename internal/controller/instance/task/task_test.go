package task

import (
	"context"
	"errors"
	"testing"

	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// newTestTask returns a minimal Task CR for tests.
func newTestTask(name, typeID string) *instancev1alpha1.Task {
	return &instancev1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: instancev1alpha1.TaskSpec{
			ForProvider: instancev1alpha1.TaskParameters{
				Name:    name,
				TypeID:  typeID,
				Enabled: true,
			},
		},
	}
}

// newTestScheme builds a scheme with instance and nexus v1alpha1 types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := instancev1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(instance) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           resource.Managed
		mockSetup    func(*instancemocks.MockTaskClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_EmptyList",
			cr:   newTestTask("my-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.GetTaskByNameFn = func(_ context.Context, _ string) (*nxtask.Task, error) {
					//nolint:nilnil // intentionally simulating not-found
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NotFound_404Error",
			cr:   newTestTask("my-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.GetTaskByNameFn = func(_ context.Context, _ string) (*nxtask.Task, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "Found_UpToDate",
			cr:   newTestTask("my-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.GetTaskByNameFn = func(_ context.Context, _ string) (*nxtask.Task, error) {
					return &nxtask.Task{
						ID:   "task-id",
						Name: "my-task",
						Type: "repository.cleanup",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "Found_OutOfDate",
			cr:   newTestTask("my-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.GetTaskByNameFn = func(_ context.Context, _ string) (*nxtask.Task, error) {
					return &nxtask.Task{
						ID:   "task-id",
						Name: "my-task",
						Type: "db.backup", // different type
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestTask("my-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.GetTaskByNameFn = func(_ context.Context, _ string) (*nxtask.Task, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name:      "NilManaged",
			cr:        nil,
			mockSetup: func(*instancemocks.MockTaskClient) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockTaskClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			obs, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        resource.Managed
		mockSetup func(*instancemocks.MockTaskClient)
		wantErr   bool
	}{
		{
			name: "Success",
			cr:   newTestTask("new-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.CreateTaskFn = func(_ context.Context, _ *nxtask.TaskCreateStruct) (*nxtask.Task, error) {
					return &nxtask.Task{ID: "new-id", Name: "new-task", Type: "repository.cleanup"}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr:   newTestTask("fail-task", "repository.cleanup"),
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.CreateTaskFn = func(_ context.Context, _ *nxtask.TaskCreateStruct) (*nxtask.Task, error) {
					return nil, errors.New("create failed")
				}
			},
			wantErr: true,
		},
		{
			name:      "NilManaged",
			cr:        nil,
			mockSetup: func(*instancemocks.MockTaskClient) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockTaskClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cr         resource.Managed
		atProvider instancev1alpha1.TaskObservation
		mockSetup  func(*instancemocks.MockTaskClient)
		wantErr    bool
	}{
		{
			name:       "Success_WithObservedID",
			cr:         newTestTask("my-task", "repository.cleanup"),
			atProvider: instancev1alpha1.TaskObservation{ID: "observed-id"},
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.UpdateTaskFn = func(_ context.Context, id string, _ *nxtask.TaskCreateStruct) error {
					if id != "observed-id" {
						return errors.New("wrong id: " + id)
					}

					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "UpdateError",
			cr:         newTestTask("my-task", "repository.cleanup"),
			atProvider: instancev1alpha1.TaskObservation{ID: "task-id"},
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.UpdateTaskFn = func(_ context.Context, _ string, _ *nxtask.TaskCreateStruct) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
		{
			name:      "NilManaged",
			cr:        nil,
			mockSetup: func(*instancemocks.MockTaskClient) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockTaskClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			if taskCR, ok := tt.cr.(*instancev1alpha1.Task); ok {
				taskCR.Status.AtProvider = tt.atProvider
			}

			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cr         resource.Managed
		atProvider instancev1alpha1.TaskObservation
		mockSetup  func(*instancemocks.MockTaskClient)
		wantErr    bool
	}{
		{
			name:       "Success",
			cr:         newTestTask("my-task", "repository.cleanup"),
			atProvider: instancev1alpha1.TaskObservation{ID: "del-id"},
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.DeleteTaskFn = func(_ context.Context, id string) error {
					if id != "del-id" {
						return errors.New("wrong id: " + id)
					}

					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "AlreadyDeleted_NotFound",
			cr:         newTestTask("my-task", "repository.cleanup"),
			atProvider: instancev1alpha1.TaskObservation{ID: "gone-id"},
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.DeleteTaskFn = func(_ context.Context, _ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name:       "DeleteError",
			cr:         newTestTask("my-task", "repository.cleanup"),
			atProvider: instancev1alpha1.TaskObservation{ID: "err-id"},
			mockSetup: func(mc *instancemocks.MockTaskClient) {
				mc.DeleteTaskFn = func(_ context.Context, _ string) error {
					return errors.New("connection refused")
				}
			},
			wantErr: true,
		},
		{
			name:      "NilManaged",
			cr:        nil,
			mockSetup: func(*instancemocks.MockTaskClient) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockTaskClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			if taskCR, ok := tt.cr.(*instancev1alpha1.Task); ok {
				taskCR.Status.AtProvider = tt.atProvider
			}

			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDisconnect tests that Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockTaskClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}

// TestConnect_NilManaged tests connector.Connect rejects nil managed resources.
func TestConnect_NilManaged(t *testing.T) {
	t.Parallel()

	s := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	c := &connector{
		kube:  fakeClient,
		usage: resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ClusterProviderConfigUsage{}),
	}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() expected error for nil managed resource, got nil")
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	s := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestTask("track-fail", "repository.cleanup")
	// Setting ref with empty Kind causes Track to fail early.
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests Connect when ProviderConfig does
// not exist.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	s := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestTask("get-pc-fail", "repository.cleanup")
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig is not in store")
	}
}

// TestUpdate_FallbackToExternalName tests that Update uses external-name
// annotation when AtProvider.ID is empty.
func TestUpdate_FallbackToExternalName(t *testing.T) {
	t.Parallel()

	cr := newTestTask("my-task", "repository.cleanup")
	cr.Annotations = map[string]string{"crossplane.io/external-name": "external-id"}

	usedID := ""
	mc := instancemocks.NewMockTaskClient()
	mc.UpdateTaskFn = func(_ context.Context, id string, _ *nxtask.TaskCreateStruct) error {
		usedID = id

		return nil
	}

	e := &external{client: mc}

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if usedID != "external-id" {
		t.Errorf("Update() used id = %q, want %q", usedID, "external-id")
	}
}

// TestDelete_FallbackToExternalName tests that Delete uses external-name
// annotation when AtProvider.ID is empty.
func TestDelete_FallbackToExternalName(t *testing.T) {
	t.Parallel()

	cr := newTestTask("my-task", "repository.cleanup")
	cr.Annotations = map[string]string{"crossplane.io/external-name": "ext-del-id"}

	usedID := ""
	mc := instancemocks.NewMockTaskClient()
	mc.DeleteTaskFn = func(_ context.Context, id string) error {
		usedID = id

		return nil
	}

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if usedID != "ext-del-id" {
		t.Errorf("Delete() used id = %q, want %q", usedID, "ext-del-id")
	}
}

// TestObserve_ExternalNameUsed tests that the external-name annotation is used
// as the lookup name when present.
func TestObserve_ExternalNameUsed(t *testing.T) {
	t.Parallel()

	cr := newTestTask("cr-name", "repository.cleanup")
	cr.Annotations = map[string]string{
		"crossplane.io/external-name": "external-task-name",
	}

	lookupName := ""
	mc := instancemocks.NewMockTaskClient()
	mc.GetTaskByNameFn = func(_ context.Context, name string) (*nxtask.Task, error) {
		lookupName = name

		return &nxtask.Task{ID: "id", Name: name, Type: "repository.cleanup"}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if lookupName != "external-task-name" {
		t.Errorf("looked up name = %q, want %q", lookupName, "external-task-name")
	}
}

// TestCreate_SetsExternalName tests that Create sets the external-name
// annotation to the server-assigned task ID.
func TestCreate_SetsExternalName(t *testing.T) {
	t.Parallel()

	cr := newTestTask("my-task", "repository.cleanup")

	mc := instancemocks.NewMockTaskClient()
	mc.CreateTaskFn = func(_ context.Context, _ *nxtask.TaskCreateStruct) (*nxtask.Task, error) {
		return &nxtask.Task{ID: "server-id", Name: "my-task", Type: "repository.cleanup"}, nil
	}

	e := &external{client: mc}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if cr.Annotations["crossplane.io/external-name"] != "server-id" {
		t.Errorf("external-name = %q, want %q", cr.Annotations["crossplane.io/external-name"], "server-id")
	}
}
