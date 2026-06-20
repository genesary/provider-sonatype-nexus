package nexus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// tasksPath is the Nexus REST API path for task operations.
const tasksPath = "/service/rest/v1/tasks"

// TestTaskService_GetTaskByName_Found tests that GetTaskByName returns the task
// when the Nexus list endpoint contains a task with the matching name.
func TestTaskService_GetTaskByName_Found(t *testing.T) {
	t.Parallel()

	want := nxtask.Task{ID: "abc123", Name: "my-task", Type: "repository.cleanup"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == tasksPath {
			type paginatedResp struct {
				Items             []nxtask.Task `json:"items"`
				ContinuationToken *string       `json:"continuationToken"`
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(paginatedResp{Items: []nxtask.Task{want}})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.Task().GetTaskByName(context.Background(), "my-task")
	if err != nil {
		t.Fatalf("GetTaskByName() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetTaskByName() returned nil, want task")
	}

	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
}

// TestTaskService_GetTaskByName_NotFound tests that GetTaskByName returns nil
// when no task with the given name exists.
func TestTaskService_GetTaskByName_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == tasksPath {
			type paginatedResp struct {
				Items             []nxtask.Task `json:"items"`
				ContinuationToken *string       `json:"continuationToken"`
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(paginatedResp{Items: []nxtask.Task{}})

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.Task().GetTaskByName(context.Background(), "missing")
	if err != nil {
		t.Fatalf("GetTaskByName() unexpected error: %v", err)
	}

	if got != nil {
		t.Errorf("GetTaskByName() = %v, want nil", got)
	}
}

// TestTaskService_GetTaskByName_Paginated tests that GetTaskByName follows
// continuation tokens across multiple pages.
func TestTaskService_GetTaskByName_Paginated(t *testing.T) {
	t.Parallel()

	page2Token := "page2"
	want := nxtask.Task{ID: "xyz", Name: "paginated-task", Type: "db.backup"}

	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != tasksPath {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		calls++

		type paginatedResp struct {
			Items             []nxtask.Task `json:"items"`
			ContinuationToken *string       `json:"continuationToken"`
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.URL.Query().Get("continuationToken") == "" {
			_ = json.NewEncoder(w).Encode(paginatedResp{
				Items:             []nxtask.Task{{ID: "other", Name: "other-task", Type: "db.backup"}},
				ContinuationToken: &page2Token,
			})
		} else {
			_ = json.NewEncoder(w).Encode(paginatedResp{Items: []nxtask.Task{want}})
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.Task().GetTaskByName(context.Background(), "paginated-task")
	if err != nil {
		t.Fatalf("GetTaskByName() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("GetTaskByName() returned nil, want task on page 2")
	}

	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}

	if calls != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
}

// TestTaskService_GetTaskByName_Error tests that GetTaskByName propagates HTTP
// errors.
func TestTaskService_GetTaskByName_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.Task().GetTaskByName(context.Background(), "any")
	if err == nil {
		t.Fatal("GetTaskByName() expected error for 500 status, got nil")
	}
}

// TestTaskService_CreateTask_Success tests that CreateTask sends a POST request
// and returns the created task.
func TestTaskService_CreateTask_Success(t *testing.T) {
	t.Parallel()

	input := &nxtask.TaskCreateStruct{
		Name:    "new-task",
		Type:    "repository.cleanup",
		Enabled: true,
	}
	returned := nxtask.Task{ID: "created-id", Name: "new-task", Type: "repository.cleanup"}

	var received *nxtask.TaskCreateStruct

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == tasksPath {
			received = &nxtask.TaskCreateStruct{}
			_ = json.NewDecoder(r.Body).Decode(received)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(returned)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	got, err := c.Task().CreateTask(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateTask() unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("CreateTask() returned nil task")
	}

	if got.ID != returned.ID {
		t.Errorf("ID = %q, want %q", got.ID, returned.ID)
	}

	if received == nil {
		t.Fatal("CreateTask() server did not receive request body")
	}

	if received.Name != input.Name {
		t.Errorf("received Name = %q, want %q", received.Name, input.Name)
	}
}

// TestTaskService_CreateTask_Error tests that CreateTask returns an error on
// non-201 status codes.
func TestTaskService_CreateTask_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad request")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.Task().CreateTask(context.Background(), &nxtask.TaskCreateStruct{Name: "bad"})
	if err == nil {
		t.Fatal("CreateTask() expected error for non-201 status, got nil")
	}
}

// TestTaskService_UpdateTask_Success tests that UpdateTask sends a PUT request.
func TestTaskService_UpdateTask_Success(t *testing.T) {
	t.Parallel()

	taskID := "task-id-1"
	input := &nxtask.TaskCreateStruct{Name: "updated-task", Type: "db.backup", Enabled: true}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == tasksPath+"/"+taskID {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Task().UpdateTask(context.Background(), taskID, input)
	if err != nil {
		t.Fatalf("UpdateTask() unexpected error: %v", err)
	}
}

// TestTaskService_UpdateTask_Error tests that UpdateTask returns an error on
// non-204 status codes.
func TestTaskService_UpdateTask_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Task().UpdateTask(context.Background(), "bad-id", &nxtask.TaskCreateStruct{})
	if err == nil {
		t.Fatal("UpdateTask() expected error for non-204 status, got nil")
	}
}

// TestTaskService_DeleteTask_Success tests that DeleteTask sends a DELETE
// request.
func TestTaskService_DeleteTask_Success(t *testing.T) {
	t.Parallel()

	taskID := "to-delete"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == tasksPath+"/"+taskID {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Task().DeleteTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("DeleteTask() unexpected error: %v", err)
	}
}

// TestTaskService_DeleteTask_Error tests that DeleteTask returns an error on
// non-204 status codes.
func TestTaskService_DeleteTask_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	err := c.Task().DeleteTask(context.Background(), "any-id")
	if err == nil {
		t.Fatal("DeleteTask() expected error for non-204 status, got nil")
	}
}

// TestTaskService_GetTaskByName_PaginationError tests that GetTaskByName
// propagates errors that occur while fetching subsequent pages.
func TestTaskService_GetTaskByName_PaginationError(t *testing.T) {
	t.Parallel()

	page2Token := "page2"
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != tasksPath {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		calls++

		if r.URL.Query().Get("continuationToken") == "" {
			type paginatedResp struct {
				Items             []nxtask.Task `json:"items"`
				ContinuationToken *string       `json:"continuationToken"`
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(paginatedResp{
				Items:             []nxtask.Task{{ID: "first", Name: "other", Type: "db.backup"}},
				ContinuationToken: &page2Token,
			})

			return
		}

		// second page: return error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error on page 2")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, err := c.Task().GetTaskByName(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetTaskByName() expected error when page 2 fails, got nil")
	}

	if calls != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
}

// TestNewTaskClient tests that NewTaskClient returns a valid client
// that wraps the nexus.Client TaskService.
func TestNewTaskClient(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	creds := nexus.Credentials{
		URL:      server.URL,
		Username: "admin",
		Password: "secret",
		Insecure: true,
	}

	c, err := nexus.NewClient(creds)
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	if c.Task() == nil {
		t.Error("Task() returned nil service")
	}
}
