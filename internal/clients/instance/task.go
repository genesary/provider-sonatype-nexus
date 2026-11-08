package instance

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// defaultTaskNotificationCondition is the notification condition
	// submitted when the spec sets none.
	defaultTaskNotificationCondition = "FAILURE"

	// taskScheduleAdvanced is how Nexus reports a cron schedule. It only
	// accepts "cron" when the schedule is written.
	taskScheduleAdvanced = "advanced"

	// utcOffset is the time zone offset Nexus reports for a cron schedule
	// evaluated in UTC.
	utcOffset = "Z"
)

// TaskClient provides methods for managing Nexus scheduled tasks.
type TaskClient interface {
	Create(ctx context.Context, task v395.TaskTemplateXO) (*v395.TaskXO, error)
	Delete(ctx context.Context, id string) error
	GetByName(ctx context.Context, name, taskType string) (*v395.TaskXO, error)
	Update(ctx context.Context, id string, task v395.UpdateTasksRequest) error
}

// taskClient implements TaskClient over the Nexus API.
type taskClient struct {
	api *v395.TasksAPIService
}

// NewTaskClient returns a TaskClient backed by a live Nexus connection.
func NewTaskClient(creds nexus.Credentials) (TaskClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &taskClient{api: nc.TasksAPI}, nil
}

// Create creates a task and returns it with its server-assigned ID.
func (c *taskClient) Create(ctx context.Context, task v395.TaskTemplateXO) (*v395.TaskXO, error) {
	created, resp, err := c.api.CreateTasks(ctx).TaskTemplateXO(task).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return created, nil
}

// Delete deletes the task with the given ID.
func (c *taskClient) Delete(ctx context.Context, id string) error {
	return nexus.APIError(c.api.DeleteTasks(ctx, id).Execute())
}

// GetByName returns the task with the given name, or an error wrapping
// nexus.ErrNotFound when there is none. Task names are unique in Nexus, but
// tasks are only addressed by ID, so the task is looked up in the listing,
// which Nexus returns in a single page.
//
// When taskType is set, only the tasks of that type are listed first, which
// keeps the steady-state lookup small. Only when none of them has the name
// are all the tasks listed, so that a task of another type holding the name
// is still returned for the caller to report.
func (c *taskClient) GetByName(ctx context.Context, name, taskType string) (*v395.TaskXO, error) {
	if taskType != "" {
		task, err := findTaskByName(c.api.ListTasks(ctx).Type_(taskType), name)
		if err == nil {
			return task, nil
		}

		if !errors.Is(err, nexus.ErrNotFound) {
			return nil, err
		}
	}

	return findTaskByName(c.api.ListTasks(ctx), name)
}

// findTaskByName returns the task with the given name in the listing, or an
// error wrapping nexus.ErrNotFound when the listing has none.
func findTaskByName(list v395.ApiListTasksRequest, name string) (*v395.TaskXO, error) {
	page, resp, err := list.Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	if page != nil {
		for i := range page.Items {
			if page.Items[i].GetName() == name {
				return &page.Items[i], nil
			}
		}
	}

	return nil, errors.Wrapf(nexus.ErrNotFound, "task %q", name)
}

// Update replaces the task with the given ID.
func (c *taskClient) Update(ctx context.Context, id string, task v395.UpdateTasksRequest) error {
	return nexus.APIError(c.api.UpdateTasks(ctx, id).UpdateTasksRequest(task).Execute())
}

// GenerateTaskCreate builds a task creation request from the CR spec.
func GenerateTaskCreate(params *instancev1alpha1.TaskParameters) v395.TaskTemplateXO {
	return v395.TaskTemplateXO{
		Type:                  params.Type,
		Name:                  params.Name,
		Enabled:               ptr.Deref(params.Enabled, true),
		AlertEmail:            nonEmpty(params.AlertEmail),
		NotificationCondition: ptr.Deref(params.NotificationCondition, defaultTaskNotificationCondition),
		Frequency:             generateTaskFrequency(&params.Schedule),
		Properties:            nonEmptyMap(desiredTaskProperties(params.Properties)),
	}
}

// GenerateTaskUpdate builds a task update request from the CR spec and the
// properties Nexus reports for the task. Nexus merges the properties of an
// update into the ones it holds, and removes a property sent with an empty
// value, so every observed property the spec leaves out is sent empty: the
// spec is the complete set. Unlike properties, an alert email left out of
// an update is cleared, so an unset one is simply omitted. The type cannot
// change, and Nexus rejects an update that carries it.
func GenerateTaskUpdate(params *instancev1alpha1.TaskParameters, observedProperties map[string]string) v395.UpdateTasksRequest {
	properties := desiredTaskProperties(params.Properties)

	for key := range observedProperties {
		if _, ok := properties[key]; !ok {
			properties[key] = ""
		}
	}

	return v395.UpdateTasksRequest{
		Name:                  params.Name,
		Enabled:               ptr.Deref(params.Enabled, true),
		AlertEmail:            nonEmpty(params.AlertEmail),
		NotificationCondition: ptr.Deref(params.NotificationCondition, defaultTaskNotificationCondition),
		Frequency:             generateTaskFrequency(&params.Schedule),
		Properties:            nonEmptyMap(properties),
	}
}

// generateTaskFrequency builds the Nexus frequency of a schedule. Nexus
// reads the start date as epoch milliseconds, and only honours a time zone
// offset for cron schedules.
func generateTaskFrequency(schedule *instancev1alpha1.TaskSchedule) v395.FrequencyXO {
	frequency := v395.FrequencyXO{
		Schedule:       schedule.Type,
		RecurringDays:  slices.Clone(schedule.RecurringDays),
		CronExpression: schedule.CronExpression,
	}

	if schedule.StartDate != nil {
		frequency.StartDate = new(schedule.StartDate.UnixMilli())
	}

	// An unset offset is sent as UTC rather than left for Nexus to default,
	// so the expression runs in UTC as documented and reads back as "Z".
	if schedule.Type == instancev1alpha1.TaskScheduleCron {
		frequency.TimeZoneOffset = new(ptr.Deref(schedule.TimeZoneOffset, utcOffset))
	}

	return frequency
}

// IsTaskUpToDate reports whether the Nexus task matches the CR spec.
func IsTaskUpToDate(params *instancev1alpha1.TaskParameters, observed *v395.TaskXO) bool {
	if ptr.Deref(params.Enabled, true) != observed.GetEnabled() {
		return false
	}

	if ptr.Deref(params.AlertEmail, "") != observed.GetAlertEmail() {
		return false
	}

	if ptr.Deref(params.NotificationCondition, defaultTaskNotificationCondition) != observed.GetNotificationCondition() {
		return false
	}

	if !isTaskScheduleUpToDate(&params.Schedule, observed) {
		return false
	}

	return maps.Equal(desiredTaskProperties(params.Properties), observed.GetProperties())
}

// isTaskScheduleUpToDate reports whether the schedule Nexus reports for a
// task matches the spec.
//
// The start date is compared to the minute, the precision Nexus keeps, and
// only when the spec sets it: Nexus stamps the current time on cron
// schedules, which take none. Recurring days are compared as sets, since
// Nexus sorts and de-duplicates them.
//
// The time zone offset of a cron schedule is compared too, but Nexus reports
// "Z" after creating a cron task whatever offset it applied, and only reports
// the applied offset after an update. A cron task with another offset is
// therefore updated once right after its creation.
func isTaskScheduleUpToDate(schedule *instancev1alpha1.TaskSchedule, observed *v395.TaskXO) bool {
	if schedule.Type != normalizeTaskSchedule(observed.GetSchedule()) {
		return false
	}

	if schedule.StartDate != nil {
		if observed.StartDate == nil || !schedule.StartDate.Truncate(time.Minute).Equal(observed.StartDate.Truncate(time.Minute)) {
			return false
		}
	}

	if !helpers.AreSlicesEqualDeDuped(schedule.RecurringDays, observed.RecurringDays) {
		return false
	}

	if ptr.Deref(schedule.CronExpression, "") != observed.GetCronExpression() {
		return false
	}

	if schedule.Type == instancev1alpha1.TaskScheduleCron {
		return normalizeTimeZoneOffset(ptr.Deref(schedule.TimeZoneOffset, "")) == normalizeTimeZoneOffset(observed.GetTimeZoneOffset())
	}

	return true
}

// GenerateTaskObservation converts a Nexus task to an observation.
func GenerateTaskObservation(observed *v395.TaskXO) instancev1alpha1.TaskObservation {
	if observed == nil {
		return instancev1alpha1.TaskObservation{}
	}

	obs := instancev1alpha1.TaskObservation{
		ID:                    observed.GetId(),
		Type:                  observed.GetType(),
		TypeName:              observed.GetTypeName(),
		Enabled:               observed.GetEnabled(),
		AlertEmail:            observed.GetAlertEmail(),
		NotificationCondition: observed.GetNotificationCondition(),
		Properties:            maps.Clone(observed.GetProperties()),
		CurrentState:          observed.GetCurrentState(),
		Message:               observed.GetMessage(),
		LastRun:               helpers.TimeToMetaTime(observed.LastRun),
		LastRunResult:         observed.GetLastRunResult(),
		NextRun:               helpers.TimeToMetaTime(observed.NextRun),
		Schedule: &instancev1alpha1.TaskScheduleObservation{
			Type:           normalizeTaskSchedule(observed.GetSchedule()),
			StartDate:      helpers.TimeToMetaTime(observed.StartDate),
			RecurringDays:  slices.Clone(observed.RecurringDays),
			CronExpression: observed.GetCronExpression(),
			TimeZoneOffset: observed.GetTimeZoneOffset(),
		},
	}

	return obs
}

// normalizeTaskSchedule returns the schedule type Nexus reports, spelled as
// in the spec: Nexus reports cron schedules as "advanced".
func normalizeTaskSchedule(schedule string) string {
	schedule = strings.ToLower(schedule)
	if schedule == taskScheduleAdvanced {
		return instancev1alpha1.TaskScheduleCron
	}

	return schedule
}

// normalizeTimeZoneOffset returns the offset Nexus reports for a cron
// schedule evaluated in the given offset: UTC, whether unset or spelled
// +00:00, is reported as "Z".
func normalizeTimeZoneOffset(offset string) string {
	switch offset {
	case "", "+00:00", "-00:00":
		return utcOffset
	default:
		return offset
	}
}

// desiredTaskProperties returns a copy of the spec properties without the
// empty ones, which Nexus drops.
func desiredTaskProperties(properties map[string]string) map[string]string {
	desired := make(map[string]string, len(properties))

	for key, value := range properties {
		if value != "" {
			desired[key] = value
		}
	}

	return desired
}

// nonEmptyMap returns a pointer to m, or nil when m is empty. Callers pass
// maps they own.
func nonEmptyMap(m map[string]string) *map[string]string {
	if len(m) == 0 {
		return nil
	}

	return &m
}
