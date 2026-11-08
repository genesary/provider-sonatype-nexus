package instance

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// taskStart is the start date of the scheduled test tasks.
var taskStart = time.Date(2027, time.January, 3, 2, 30, 0, 0, time.UTC)

// newWeeklyTaskParams returns the spec of a weekly blob store compaction.
func newWeeklyTaskParams() instancev1alpha1.TaskParameters {
	return instancev1alpha1.TaskParameters{
		Name:    "weekly-compact",
		Type:    "blobstore.compact",
		Enabled: new(true),
		Schedule: instancev1alpha1.TaskSchedule{
			Type:          instancev1alpha1.TaskScheduleWeekly,
			StartDate:     new(metav1.NewTime(taskStart.Add(45 * time.Second))),
			RecurringDays: []int32{7, 1},
		},
		Properties: map[string]string{"blobstoreName": "default"},
	}
}

// newObservedWeeklyTask returns the task Nexus reports for
// newWeeklyTaskParams: the start date kept to the minute and the recurring
// days sorted.
func newObservedWeeklyTask() *v395.TaskXO {
	return &v395.TaskXO{
		Id:                    new("task-id"),
		Name:                  new("weekly-compact"),
		Type:                  new("blobstore.compact"),
		Enabled:               new(true),
		NotificationCondition: new("FAILURE"),
		Schedule:              new("weekly"),
		StartDate:             new(taskStart),
		RecurringDays:         []int32{1, 7},
		Properties:            new(map[string]string{"blobstoreName": "default"}),
	}
}

// newCronTaskParams returns the spec of a cron task evaluated in the given
// time zone offset.
func newCronTaskParams(offset *string) instancev1alpha1.TaskParameters {
	params := newWeeklyTaskParams()
	params.Schedule = instancev1alpha1.TaskSchedule{
		Type:           instancev1alpha1.TaskScheduleCron,
		CronExpression: new("0 0 2 * * ?"),
		TimeZoneOffset: offset,
	}

	return params
}

// newObservedCronTask returns a cron task as Nexus reports it: as an
// advanced schedule, stamped with the current time.
func newObservedCronTask(offset string) *v395.TaskXO {
	observed := newObservedWeeklyTask()
	observed.Schedule = new("advanced")
	observed.StartDate = new(time.Now())
	observed.RecurringDays = nil
	observed.CronExpression = new("0 0 2 * * ?")
	observed.TimeZoneOffset = new(offset)

	return observed
}

// TestGetTaskByName checks that a task is looked up among the tasks of its
// type first, and among all the tasks only when none of them has the name.
func TestGetTaskByName(t *testing.T) {
	t.Parallel()

	const (
		compact = `{"items":[{"id":"1","name":"nightly","type":"blobstore.compact"}]}`
		all     = `{"items":[{"id":"1","name":"nightly","type":"blobstore.compact"},{"id":"2","name":"cleanup","type":"repository.cleanup"}]}`
	)

	tests := []struct {
		name      string
		task      string
		taskType  string
		wantID    string
		wantQuery []string
	}{
		{
			name:      "FoundAmongItsType",
			task:      "nightly",
			taskType:  "blobstore.compact",
			wantID:    "1",
			wantQuery: []string{"type=blobstore.compact"},
		},
		{
			name:      "FoundUnderAnotherType",
			task:      "cleanup",
			taskType:  "blobstore.compact",
			wantID:    "2",
			wantQuery: []string{"type=blobstore.compact", ""},
		},
		{
			name:      "NotFound",
			task:      "missing",
			taskType:  "blobstore.compact",
			wantQuery: []string{"type=blobstore.compact", ""},
		},
		{
			name:      "NoType",
			task:      "cleanup",
			wantID:    "2",
			wantQuery: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				mu      sync.Mutex
				queries []string
			)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				queries = append(queries, r.URL.RawQuery)
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")

				body := all
				if r.URL.Query().Get("type") == "blobstore.compact" {
					body = compact
				}

				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(server.Close)

			client, err := NewTaskClient(nexus.Credentials{URL: server.URL, Username: "admin", Password: "password"})
			if err != nil {
				t.Fatalf("NewTaskClient() error = %v", err)
			}

			got, err := client.GetByName(context.Background(), tt.task, tt.taskType)

			if tt.wantID == "" {
				if !errors.Is(err, nexus.ErrNotFound) {
					t.Errorf("GetByName() error = %v, want nexus.ErrNotFound", err)
				}
			} else if err != nil || got.GetId() != tt.wantID {
				t.Errorf("GetByName() = %+v, %v, want task %s", got, err, tt.wantID)
			}

			if !slices.Equal(queries, tt.wantQuery) {
				t.Errorf("list queries = %q, want %q", queries, tt.wantQuery)
			}
		})
	}
}

// TestIsTaskUpToDate covers drift detection of scheduled tasks.
func TestIsTaskUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		params   instancev1alpha1.TaskParameters
		observed *v395.TaskXO
		want     bool
	}{
		{
			name:     "Matching",
			params:   newWeeklyTaskParams(),
			observed: newObservedWeeklyTask(),
			want:     true,
		},
		{
			name: "DefaultsMatchNexus",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.Enabled = nil
				params.NotificationCondition = nil

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     true,
		},
		{
			name:   "Disabled",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.Enabled = new(false)

				return observed
			}(),
			want: false,
		},
		{
			name: "AlertEmailMissing",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.AlertEmail = new("ops@example.com")

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     false,
		},
		{
			name: "EmptyAlertEmailMatchesNone",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.AlertEmail = new("")

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     true,
		},
		{
			name:   "UnwantedAlertEmail",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.AlertEmail = new("ops@example.com")

				return observed
			}(),
			want: false,
		},
		{
			name: "NotificationConditionChanged",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.NotificationCondition = new("SUCCESS_FAILURE")

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     false,
		},
		{
			name:   "ScheduleTypeChanged",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.Schedule = new("daily")
				observed.RecurringDays = nil

				return observed
			}(),
			want: false,
		},
		{
			name:   "StartDateChanged",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.StartDate = new(taskStart.Add(time.Hour))

				return observed
			}(),
			want: false,
		},
		{
			name:   "StartDateInAnotherZone",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.StartDate = new(taskStart.In(time.FixedZone("+05:00", 5*3600)))

				return observed
			}(),
			want: true,
		},
		{
			name:   "StartDateMissing",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.StartDate = nil

				return observed
			}(),
			want: false,
		},
		{
			name:   "RecurringDaysChanged",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.RecurringDays = []int32{1}

				return observed
			}(),
			want: false,
		},
		{
			name: "DuplicateRecurringDays",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.Schedule.RecurringDays = []int32{7, 1, 7}

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     true,
		},
		{
			name:     "CronReportedAsAdvanced",
			params:   newCronTaskParams(nil),
			observed: newObservedCronTask("Z"),
			want:     true,
		},
		{
			name:   "CronExpressionChanged",
			params: newCronTaskParams(nil),
			observed: func() *v395.TaskXO {
				observed := newObservedCronTask("Z")
				observed.CronExpression = new("0 0 3 * * ?")

				return observed
			}(),
			want: false,
		},
		{
			name:     "CronOffsetMatches",
			params:   newCronTaskParams(new("+05:00")),
			observed: newObservedCronTask("+05:00"),
			want:     true,
		},
		{
			name:     "CronOffsetReportedAsUTCAfterCreate",
			params:   newCronTaskParams(new("+05:00")),
			observed: newObservedCronTask("Z"),
			want:     false,
		},
		{
			name:     "CronUTCSpelledAsOffset",
			params:   newCronTaskParams(new("+00:00")),
			observed: newObservedCronTask("Z"),
			want:     true,
		},
		{
			name:     "CronUnwantedOffset",
			params:   newCronTaskParams(nil),
			observed: newObservedCronTask("-03:00"),
			want:     false,
		},
		{
			name: "PropertyChanged",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.Properties["blobsOlderThan"] = "3"

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     false,
		},
		{
			name:   "UnwantedProperty",
			params: newWeeklyTaskParams(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				(*observed.Properties)["blobsOlderThan"] = "3"

				return observed
			}(),
			want: false,
		},
		{
			name: "EmptyPropertyMatchesAbsent",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.Properties["blobsOlderThan"] = ""

				return params
			}(),
			observed: newObservedWeeklyTask(),
			want:     true,
		},
		{
			name: "NoPropertiesMatchNone",
			params: func() instancev1alpha1.TaskParameters {
				params := newWeeklyTaskParams()
				params.Properties = nil

				return params
			}(),
			observed: func() *v395.TaskXO {
				observed := newObservedWeeklyTask()
				observed.Properties = nil

				return observed
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsTaskUpToDate(&tt.params, tt.observed)
			if got != tt.want {
				t.Errorf("IsTaskUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateTaskCreate checks the creation request built from a spec.
func TestGenerateTaskCreate(t *testing.T) {
	t.Parallel()

	t.Run("Weekly", func(t *testing.T) {
		t.Parallel()

		params := newWeeklyTaskParams()
		params.Enabled = nil
		params.AlertEmail = new("")
		params.Properties["blobsOlderThan"] = ""

		got := GenerateTaskCreate(&params)

		if got.Name != "weekly-compact" || got.Type != "blobstore.compact" {
			t.Errorf("name/type = %q/%q", got.Name, got.Type)
		}

		if !got.Enabled {
			t.Error("Enabled should default to true")
		}

		if got.AlertEmail != nil {
			t.Errorf("AlertEmail = %q, want nil", *got.AlertEmail)
		}

		if got.NotificationCondition != "FAILURE" {
			t.Errorf("NotificationCondition = %q, want FAILURE", got.NotificationCondition)
		}

		if got.Frequency.Schedule != "weekly" {
			t.Errorf("Schedule = %q, want weekly", got.Frequency.Schedule)
		}

		// Nexus reads the start date as epoch milliseconds.
		wantStart := taskStart.Add(45 * time.Second).UnixMilli()
		if got.Frequency.StartDate == nil || *got.Frequency.StartDate != wantStart {
			t.Errorf("StartDate = %v, want %d", got.Frequency.StartDate, wantStart)
		}

		if !slices.Equal(got.Frequency.RecurringDays, []int32{7, 1}) {
			t.Errorf("RecurringDays = %v", got.Frequency.RecurringDays)
		}

		if got.Frequency.TimeZoneOffset != nil {
			t.Errorf("TimeZoneOffset = %q, want nil outside cron schedules", *got.Frequency.TimeZoneOffset)
		}

		if got.Properties == nil || !maps.Equal(*got.Properties, map[string]string{"blobstoreName": "default"}) {
			t.Errorf("Properties = %v, want only the non-empty ones", got.Properties)
		}
	})

	t.Run("Cron", func(t *testing.T) {
		t.Parallel()

		params := newCronTaskParams(new("+05:00"))
		params.Properties = nil

		got := GenerateTaskCreate(&params)

		if got.Frequency.Schedule != "cron" || got.Frequency.GetCronExpression() != "0 0 2 * * ?" {
			t.Errorf("Frequency = %+v", got.Frequency)
		}

		if got.Frequency.GetTimeZoneOffset() != "+05:00" {
			t.Errorf("TimeZoneOffset = %q, want +05:00", got.Frequency.GetTimeZoneOffset())
		}

		if got.Frequency.StartDate != nil {
			t.Errorf("StartDate = %d, want nil", *got.Frequency.StartDate)
		}

		if got.Properties != nil {
			t.Errorf("Properties = %v, want nil", *got.Properties)
		}
	})
}

// TestGenerateTaskCreateCronDefaultsToUTC checks that a cron schedule without
// an offset is sent as UTC rather than left for Nexus to default.
func TestGenerateTaskCreateCronDefaultsToUTC(t *testing.T) {
	t.Parallel()

	params := newCronTaskParams(nil)

	got := GenerateTaskCreate(&params)

	if got.Frequency.GetTimeZoneOffset() != "Z" {
		t.Errorf("TimeZoneOffset = %q, want Z when unset", got.Frequency.GetTimeZoneOffset())
	}
}

// TestGenerateTaskUpdate checks the update request built from a spec and the
// observed properties.
func TestGenerateTaskUpdate(t *testing.T) {
	t.Parallel()

	params := newWeeklyTaskParams()
	params.AlertEmail = new("ops@example.com")
	params.NotificationCondition = new("SUCCESS_FAILURE")

	got := GenerateTaskUpdate(&params, map[string]string{"blobstoreName": "old", "blobsOlderThan": "3"})

	if got.Type != nil {
		t.Errorf("Type = %q, want nil: Nexus rejects an update carrying the type", *got.Type)
	}

	if got.Name != "weekly-compact" || !got.Enabled || got.GetAlertEmail() != "ops@example.com" || got.NotificationCondition != "SUCCESS_FAILURE" {
		t.Errorf("update = %+v", got)
	}

	if got.Frequency.Schedule != "weekly" {
		t.Errorf("Schedule = %q, want weekly", got.Frequency.Schedule)
	}

	// The property left out of the spec is sent empty, which removes it.
	want := map[string]string{"blobstoreName": "default", "blobsOlderThan": ""}
	if got.Properties == nil || !maps.Equal(*got.Properties, want) {
		t.Errorf("Properties = %v, want %v", got.Properties, want)
	}
}

// TestGenerateTaskUpdateClearsAlertEmail checks that an update for a spec
// without an alert email omits it, which Nexus treats as clearing it.
func TestGenerateTaskUpdateClearsAlertEmail(t *testing.T) {
	t.Parallel()

	params := newWeeklyTaskParams()

	got := GenerateTaskUpdate(&params, nil)

	if got.AlertEmail != nil {
		t.Errorf("AlertEmail = %q, want nil so that Nexus clears it", *got.AlertEmail)
	}
}

// TestGenerateTaskObservation checks the observation built from a task.
func TestGenerateTaskObservation(t *testing.T) {
	t.Parallel()

	if got := GenerateTaskObservation(nil); got.ID != "" || got.Schedule != nil {
		t.Errorf("GenerateTaskObservation(nil) = %+v, want empty", got)
	}

	observed := newObservedCronTask("+05:00")
	observed.TypeName = new("Admin - Compact blob store")
	observed.CurrentState = new("WAITING")
	observed.LastRunResult = new("OK")
	observed.NextRun = new(taskStart)
	observed.Message = new("compacting")

	got := GenerateTaskObservation(observed)

	if got.ID != "task-id" || got.Type != "blobstore.compact" || got.TypeName != "Admin - Compact blob store" {
		t.Errorf("identity = %+v", got)
	}

	if got.CurrentState != "WAITING" || got.LastRunResult != "OK" || got.Message != "compacting" {
		t.Errorf("state = %+v", got)
	}

	if got.NextRun == nil || !got.NextRun.Time.Equal(taskStart) || got.LastRun != nil {
		t.Errorf("runs = %v / %v", got.NextRun, got.LastRun)
	}

	if got.Schedule == nil || got.Schedule.Type != "cron" || got.Schedule.TimeZoneOffset != "+05:00" || got.Schedule.CronExpression != "0 0 2 * * ?" {
		t.Errorf("Schedule = %+v, want cron at +05:00", got.Schedule)
	}

	if !maps.Equal(got.Properties, map[string]string{"blobstoreName": "default"}) {
		t.Errorf("Properties = %v", got.Properties)
	}

	// The observation must not alias the properties Nexus reported.
	got.Properties["blobstoreName"] = "changed"
	if (*observed.Properties)["blobstoreName"] != "default" {
		t.Error("observation aliases the observed properties")
	}
}
