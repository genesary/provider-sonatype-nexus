package task

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	taskmocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// errBoom is a generic Nexus failure.
var errBoom = errors.New("boom")

// newTestTask returns a manual blob store compaction task.
func newTestTask() *instancev1alpha1.Task {
	return &instancev1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "compact", Namespace: "default"},
		Spec: instancev1alpha1.TaskSpec{
			ForProvider: instancev1alpha1.TaskParameters{
				Name:       "compact-default",
				Type:       "blobstore.compact",
				Schedule:   instancev1alpha1.TaskSchedule{Type: instancev1alpha1.TaskScheduleManual},
				Properties: map[string]string{"blobstoreName": "default"},
			},
		},
	}
}

// newObservedTask returns the Nexus task matching newTestTask.
func newObservedTask() *v395.TaskXO {
	return &v395.TaskXO{
		Id:                    new("task-id"),
		Name:                  new("compact-default"),
		Type:                  new("blobstore.compact"),
		Enabled:               new(true),
		NotificationCondition: new("FAILURE"),
		Schedule:              new("manual"),
		CurrentState:          new("WAITING"),
		Properties:            new(map[string]string{"blobstoreName": "default"}),
	}
}

// TestObserve covers the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		externalName string
		getByName    func(name string) (*v395.TaskXO, error)
		wantLookup   string
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:       "NotFound",
			getByName:  func(string) (*v395.TaskXO, error) { return nil, nexus.ErrNotFound },
			wantLookup: "compact-default",
		},
		{
			name:       "Error",
			getByName:  func(string) (*v395.TaskXO, error) { return nil, errBoom },
			wantLookup: "compact-default",
			wantErr:    true,
		},
		{
			name:         "UpToDate",
			getByName:    func(string) (*v395.TaskXO, error) { return newObservedTask(), nil },
			wantLookup:   "compact-default",
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name: "Drifted",
			getByName: func(string) (*v395.TaskXO, error) {
				observed := newObservedTask()
				observed.Enabled = new(false)

				return observed, nil
			},
			wantLookup: "compact-default",
			wantExists: true,
		},
		{
			name: "TypeMismatch",
			getByName: func(string) (*v395.TaskXO, error) {
				observed := newObservedTask()
				observed.Type = new("repository.cleanup")

				return observed, nil
			},
			wantLookup: "compact-default",
			wantErr:    true,
		},
		{
			name:         "LooksUpExternalName",
			externalName: "imported-task",
			getByName:    func(string) (*v395.TaskXO, error) { return newObservedTask(), nil },
			wantLookup:   "imported-task",
			wantExists:   true,
			wantUpToDate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := taskmocks.NewMockTaskClient()
			mock.GetByNameFn = tt.getByName

			cr := newTestTask()
			if tt.externalName != "" {
				meta.SetExternalName(cr, tt.externalName)
			}

			obs, err := (&external{client: mock}).Observe(context.Background(), cr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if obs.ResourceExists != tt.wantExists || obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() = exists %v upToDate %v, want %v %v", obs.ResourceExists, obs.ResourceUpToDate, tt.wantExists, tt.wantUpToDate)
			}

			if len(mock.GetByNameCalls) != 1 || mock.GetByNameCalls[0] != tt.wantLookup {
				t.Errorf("GetByName calls = %v, want [%s]", mock.GetByNameCalls, tt.wantLookup)
			}

			if len(mock.GetByNameTypes) != 1 || mock.GetByNameTypes[0] != cr.Spec.ForProvider.Type {
				t.Errorf("GetByName types = %v, want [%s]", mock.GetByNameTypes, cr.Spec.ForProvider.Type)
			}

			if !tt.wantExists {
				return
			}

			if cr.Status.AtProvider.ID != "task-id" || cr.Status.AtProvider.CurrentState != "WAITING" {
				t.Errorf("AtProvider = %+v, want the observed task", cr.Status.AtProvider)
			}

			if cr.GetCondition(xpv2.TypeReady).Status != corev1.ConditionTrue {
				t.Error("an observed task should be Available")
			}
		})
	}
}

// TestObserveDeletedTypeMismatch checks that a deleted Task does not claim,
// and so does not delete, a Nexus task of another type with its name.
func TestObserveDeletedTypeMismatch(t *testing.T) {
	t.Parallel()

	mock := taskmocks.NewMockTaskClient()
	mock.GetByNameFn = func(string) (*v395.TaskXO, error) {
		observed := newObservedTask()
		observed.Type = new("repository.cleanup")

		return observed, nil
	}

	cr := newTestTask()
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)

	obs, err := (&external{client: mock}).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if obs.ResourceExists {
		t.Error("a task of another type should not be reported as existing")
	}
}

// TestCreate covers the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mock := taskmocks.NewMockTaskClient()
		mock.CreateFn = func(v395.TaskTemplateXO) (*v395.TaskXO, error) { return newObservedTask(), nil }

		cr := newTestTask()

		_, err := (&external{client: mock}).Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if len(mock.CreateCalls) != 1 || mock.CreateCalls[0].Name != "compact-default" || mock.CreateCalls[0].Type != "blobstore.compact" {
			t.Errorf("Create calls = %+v", mock.CreateCalls)
		}

		if meta.GetExternalName(cr) != "compact-default" {
			t.Errorf("external name = %q, want compact-default", meta.GetExternalName(cr))
		}

		if cr.Status.AtProvider.ID != "task-id" {
			t.Errorf("AtProvider.ID = %q, want task-id", cr.Status.AtProvider.ID)
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mock := taskmocks.NewMockTaskClient()
		mock.CreateFn = func(v395.TaskTemplateXO) (*v395.TaskXO, error) { return nil, errBoom }

		_, err := (&external{client: mock}).Create(context.Background(), newTestTask())
		if !errors.Is(err, errBoom) {
			t.Errorf("Create() error = %v, want %v", err, errBoom)
		}
	})
}

// TestUpdate covers the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("SendsSpecUnderObservedName", func(t *testing.T) {
		t.Parallel()

		mock := taskmocks.NewMockTaskClient()
		mock.UpdateFn = func(string, v395.UpdateTasksRequest) error { return nil }

		cr := newTestTask()
		meta.SetExternalName(cr, "imported-task")
		cr.Status.AtProvider.ID = "task-id"
		cr.Status.AtProvider.Properties = map[string]string{"blobstoreName": "default", "blobsOlderThan": "3"}

		_, err := (&external{client: mock}).Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if len(mock.UpdateIDs) != 1 || mock.UpdateIDs[0] != "task-id" {
			t.Fatalf("Update IDs = %v, want [task-id]", mock.UpdateIDs)
		}

		update := mock.UpdateCalls[0]
		if update.Name != "imported-task" {
			t.Errorf("Name = %q, want the name the task was observed under", update.Name)
		}

		if update.Properties == nil || (*update.Properties)["blobsOlderThan"] != "" || (*update.Properties)["blobstoreName"] != "default" {
			t.Errorf("Properties = %v, want the stale property cleared", update.Properties)
		}
	})

	t.Run("UnknownID", func(t *testing.T) {
		t.Parallel()

		mock := taskmocks.NewMockTaskClient()

		_, err := (&external{client: mock}).Update(context.Background(), newTestTask())
		if err == nil || len(mock.UpdateCalls) != 0 {
			t.Errorf("Update() error = %v, calls %d; want an error and no call", err, len(mock.UpdateCalls))
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mock := taskmocks.NewMockTaskClient()
		mock.UpdateFn = func(string, v395.UpdateTasksRequest) error { return errBoom }

		cr := newTestTask()
		cr.Status.AtProvider.ID = "task-id"

		_, err := (&external{client: mock}).Update(context.Background(), cr)
		if !errors.Is(err, errBoom) {
			t.Errorf("Update() error = %v, want %v", err, errBoom)
		}
	})
}

// TestDelete covers the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		id        string
		deleteErr error
		wantCalls int
		wantErr   bool
	}{
		{name: "Success", id: "task-id", wantCalls: 1},
		{name: "AlreadyGone", id: "task-id", deleteErr: nexus.ErrNotFound, wantCalls: 1},
		{name: "Error", id: "task-id", deleteErr: errBoom, wantCalls: 1, wantErr: true},
		{name: "UnknownID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := taskmocks.NewMockTaskClient()
			mock.DeleteFn = func(string) error { return tt.deleteErr }

			cr := newTestTask()
			cr.Status.AtProvider.ID = tt.id

			_, err := (&external{client: mock}).Delete(context.Background(), cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(mock.DeleteCalls) != tt.wantCalls {
				t.Errorf("Delete calls = %v, want %d", mock.DeleteCalls, tt.wantCalls)
			}
		})
	}
}
