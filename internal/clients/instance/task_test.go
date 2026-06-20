package instance_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// strPtr returns a pointer to the given string value.
func strPtr(s string) *string { return &s } //nolint:modernize // intentional inlinable wrapper for concise *string creation in test literals

// int64Ptr returns a pointer to the given int64 value.
func int64Ptr(i int64) *int64 { return &i } //nolint:modernize // intentional inlinable wrapper for concise *int64 creation in test literals

// newTaskCR returns a minimal Task CR for tests.
func newTaskCR(name, typeID string) *instancev1alpha1.Task {
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

// TestGenerateTaskCreateStruct_Minimal tests generation with only required
// fields.
func TestGenerateTaskCreateStruct_Minimal(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	got := instance.GenerateTaskCreateStruct(cr)

	if got.Name != "my-task" {
		t.Errorf("Name = %q, want %q", got.Name, "my-task")
	}

	if got.Type != "repository.cleanup" {
		t.Errorf("Type = %q, want %q", got.Type, "repository.cleanup")
	}

	if !got.Enabled {
		t.Error("Enabled = false, want true")
	}

	if got.AlertEmail != "" {
		t.Errorf("AlertEmail = %q, want empty", got.AlertEmail)
	}

	if got.Message != "" {
		t.Errorf("Message = %q, want empty", got.Message)
	}

	if got.Frequency != nil {
		t.Error("Frequency should be nil for task without schedule")
	}
}

// TestGenerateTaskCreateStruct_AllFields tests generation with all optional
// fields set.
func TestGenerateTaskCreateStruct_AllFields(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("full-task", "db.backup")
	cr.Spec.ForProvider.AlertEmail = strPtr("admin@example.com")  //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	cr.Spec.ForProvider.NotificationCondition = strPtr("FAILURE") //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	cr.Spec.ForProvider.Message = strPtr("Backup task")           //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	cr.Spec.ForProvider.TaskProperties = map[string]string{"key": "value"}
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:          "weekly",
		Timezone:      strPtr("UTC"), //nolint:modernize // strPtr sets specific value; new(T) gives zero value
		RecurringDays: []string{"sunday"},
	}

	got := instance.GenerateTaskCreateStruct(cr)

	if got.AlertEmail != "admin@example.com" {
		t.Errorf("AlertEmail = %q, want %q", got.AlertEmail, "admin@example.com")
	}

	if got.NotificationCondition != "FAILURE" {
		t.Errorf("NotificationCondition = %q, want %q", got.NotificationCondition, "FAILURE")
	}

	if got.Message != "Backup task" {
		t.Errorf("Message = %q, want %q", got.Message, "Backup task")
	}

	if got.Properties["key"] != "value" {
		t.Errorf("Properties[key] = %v, want %q", got.Properties["key"], "value")
	}

	if got.Frequency == nil {
		t.Fatal("Frequency should not be nil")
	}

	if got.Frequency.Schedule != "weekly" {
		t.Errorf("Frequency.Schedule = %q, want %q", got.Frequency.Schedule, "weekly")
	}

	if got.Frequency.TimeZoneOffset != "UTC" {
		t.Errorf("Frequency.TimeZoneOffset = %q, want %q", got.Frequency.TimeZoneOffset, "UTC")
	}

	if len(got.Frequency.RecurringDays) != 1 || got.Frequency.RecurringDays[0] != "sunday" {
		t.Errorf("Frequency.RecurringDays = %v, want [sunday]", got.Frequency.RecurringDays)
	}
}

// TestGenerateTaskCreateStruct_CronSchedule tests cron schedule generation.
func TestGenerateTaskCreateStruct_CronSchedule(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("cron-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:           "cron",
		CronExpression: strPtr("0 0 * * 0"), //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	}

	got := instance.GenerateTaskCreateStruct(cr)

	if got.Frequency == nil {
		t.Fatal("Frequency should not be nil")
	}

	if got.Frequency.CronExpression != "0 0 * * 0" {
		t.Errorf("CronExpression = %q, want %q", got.Frequency.CronExpression, "0 0 * * 0")
	}
}

// TestGenerateTaskCreateStruct_StartDate tests start date generation.
func TestGenerateTaskCreateStruct_StartDate(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("once-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:      "once",
		StartDate: int64Ptr(1700000000000),
	}

	got := instance.GenerateTaskCreateStruct(cr)

	if got.Frequency == nil {
		t.Fatal("Frequency should not be nil")
	}

	if got.Frequency.StartDate != 1700000000000 {
		t.Errorf("StartDate = %d, want %d", got.Frequency.StartDate, 1700000000000)
	}
}

// TestGenerateTaskObservation_Nil tests that nil task returns empty
// observation.
func TestGenerateTaskObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := instance.GenerateTaskObservation(nil)

	if obs.ID != "" || obs.CurrentState != "" || obs.NextRun != "" || obs.LastRun != "" {
		t.Errorf("GenerateTaskObservation(nil) = %v, want zero value", obs)
	}
}

// TestGenerateTaskObservation_Full tests observation generation from a task.
func TestGenerateTaskObservation_Full(t *testing.T) {
	t.Parallel()

	task := &nxtask.Task{
		ID:           "task-id",
		CurrentState: "WAITING",
		NextRun:      "2025-01-01T00:00:00Z",
		LastRun:      "2024-12-31T00:00:00Z",
	}

	obs := instance.GenerateTaskObservation(task)

	if obs.ID != "task-id" {
		t.Errorf("ID = %q, want %q", obs.ID, "task-id")
	}

	if obs.CurrentState != "WAITING" {
		t.Errorf("CurrentState = %q, want %q", obs.CurrentState, "WAITING")
	}

	if obs.NextRun != "2025-01-01T00:00:00Z" {
		t.Errorf("NextRun = %q, want %q", obs.NextRun, "2025-01-01T00:00:00Z")
	}

	if obs.LastRun != "2024-12-31T00:00:00Z" {
		t.Errorf("LastRun = %q, want %q", obs.LastRun, "2024-12-31T00:00:00Z")
	}
}

// TestIsTaskUpToDate_Match tests that identical spec/observed returns true.
func TestIsTaskUpToDate_Match(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Message = strPtr("cleanup") //nolint:modernize // strPtr sets specific value; new(T) gives zero value

	observed := &nxtask.Task{
		Name:    "my-task",
		Type:    "repository.cleanup",
		Message: "cleanup",
	}

	if !instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = false, want true for matching spec")
	}
}

// TestIsTaskUpToDate_TypeMismatch tests that a changed typeId is detected.
func TestIsTaskUpToDate_TypeMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	observed := &nxtask.Task{Name: "my-task", Type: "db.backup"}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when typeId differs")
	}
}

// TestIsTaskUpToDate_MessageMismatch tests that a changed message is detected.
func TestIsTaskUpToDate_MessageMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Message = strPtr("new message") //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	observed := &nxtask.Task{Name: "my-task", Type: "repository.cleanup", Message: "old message"}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when message differs")
	}
}

// TestIsTaskUpToDate_ScheduleTypeMismatch tests schedule type change detection.
func TestIsTaskUpToDate_ScheduleTypeMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{Type: "daily"}

	observed := &nxtask.Task{
		Name:      "my-task",
		Type:      "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{Schedule: "weekly"},
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when schedule type differs")
	}
}

// TestIsTaskUpToDate_ScheduleNilVsSet tests nil desired vs set observed
// schedule.
func TestIsTaskUpToDate_ScheduleNilVsSet(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")

	observed := &nxtask.Task{
		Name:      "my-task",
		Type:      "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{Schedule: "daily"},
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when desired schedule is nil but observed is not")
	}
}

// TestIsTaskUpToDate_NameMismatch tests that a changed name is detected.
func TestIsTaskUpToDate_NameMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("new-name", "repository.cleanup")
	observed := &nxtask.Task{Name: "old-name", Type: "repository.cleanup"}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when name differs")
	}
}

// TestIsTaskUpToDate_NoSchedule tests that nil/nil schedule is considered
// up-to-date.
func TestIsTaskUpToDate_NoSchedule(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	observed := &nxtask.Task{Name: "my-task", Type: "repository.cleanup"}

	if !instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = false, want true for matching task with no schedule")
	}
}

// TestIsScheduleUpToDate_TimezoneMismatch tests timezone change detection.
func TestIsScheduleUpToDate_TimezoneMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:     "daily",
		Timezone: strPtr("UTC"), //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	}

	observed := &nxtask.Task{
		Name: "my-task",
		Type: "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{
			Schedule:       "daily",
			TimeZoneOffset: "America/New_York",
		},
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when timezone differs")
	}
}

// TestIsScheduleUpToDate_CronMismatch tests cron expression change detection.
func TestIsScheduleUpToDate_CronMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:           "cron",
		CronExpression: strPtr("0 0 * * 1"), //nolint:modernize // strPtr sets specific value; new(T) gives zero value
	}

	observed := &nxtask.Task{
		Name: "my-task",
		Type: "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{
			Schedule:       "cron",
			CronExpression: "0 0 * * 0",
		},
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when cron expression differs")
	}
}

// TestIsScheduleUpToDate_StartDateMismatch tests start date change detection.
func TestIsScheduleUpToDate_StartDateMismatch(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{
		Type:      "once",
		StartDate: int64Ptr(1000000),
	}

	observed := &nxtask.Task{
		Name: "my-task",
		Type: "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{
			Schedule:  "once",
			StartDate: 2000000,
		},
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when start date differs")
	}
}

// TestIsScheduleUpToDate_MatchingNonNil tests that matching non-nil schedules
// are considered up-to-date.
func TestIsScheduleUpToDate_MatchingNonNil(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{Type: "daily"}

	observed := &nxtask.Task{
		Name: "my-task",
		Type: "repository.cleanup",
		Frequency: &nxtask.FrequencyXO{
			Schedule: "daily",
		},
	}

	if !instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = false, want true for matching daily schedule")
	}
}

// TestIsScheduleUpToDate_ObservedNilDesiredSet tests nil observed vs set
// desired schedule.
func TestIsScheduleUpToDate_ObservedNilDesiredSet(t *testing.T) {
	t.Parallel()

	cr := newTaskCR("my-task", "repository.cleanup")
	cr.Spec.ForProvider.Schedule = &instancev1alpha1.TaskSchedule{Type: "daily"}

	observed := &nxtask.Task{
		Name:      "my-task",
		Type:      "repository.cleanup",
		Frequency: nil,
	}

	if instance.IsTaskUpToDate(cr, observed) {
		t.Error("IsTaskUpToDate() = true, want false when desired schedule is set but observed is nil")
	}
}

// TestNewTaskClient_Success tests that NewTaskClient creates a valid client.
func TestNewTaskClient_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := instance.NewTaskClient(nexus.Credentials{
		URL:      server.URL,
		Username: "admin",
		Password: "secret",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("NewTaskClient() unexpected error: %v", err)
	}

	if c == nil {
		t.Error("NewTaskClient() returned nil client")
	}
}

// TestNewTaskClient_EmptyURL tests that NewTaskClient works even with empty
// credentials (the nexus3 client does not eagerly validate the URL).
func TestNewTaskClient_EmptyURL(t *testing.T) {
	t.Parallel()

	c, err := instance.NewTaskClient(nexus.Credentials{})
	if err != nil {
		t.Fatalf("NewTaskClient() unexpected error: %v", err)
	}

	if c == nil {
		t.Error("NewTaskClient() returned nil client")
	}
}
