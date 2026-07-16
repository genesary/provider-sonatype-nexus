// Package instance provides clients for Nexus instance API group.
package instance

import (
	"context"

	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// TaskClient defines operations for managing Nexus scheduled tasks.
type TaskClient interface {
	GetTaskByName(ctx context.Context, name string) (*nxtask.Task, error)
	CreateTask(ctx context.Context, t *nxtask.TaskCreateStruct) (*nxtask.Task, error)
	UpdateTask(ctx context.Context, id string, t *nxtask.TaskCreateStruct) error
	DeleteTask(ctx context.Context, id string) error
}

// NewTaskClient creates a TaskClient from Nexus credentials.
func NewTaskClient(creds nexus.Credentials) (TaskClient, error) {
	c, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return c.Task(), nil
}

// GenerateTaskCreateStruct builds a TaskCreateStruct from a Task CR spec.
func GenerateTaskCreateStruct(cr *instancev1alpha1.Task) *nxtask.TaskCreateStruct {
	params := cr.Spec.ForProvider

	taskCreate := &nxtask.TaskCreateStruct{
		Name:    params.Name,
		Type:    params.TypeID,
		Enabled: params.Enabled,
	}

	if params.AlertEmail != nil {
		taskCreate.AlertEmail = *params.AlertEmail
	}

	if params.NotificationCondition != nil {
		taskCreate.NotificationCondition = *params.NotificationCondition
	}

	if params.Message != nil {
		taskCreate.Message = *params.Message
	}

	if params.Schedule != nil {
		taskCreate.Frequency = generateFrequency(params.Schedule)
	}

	if len(params.TaskProperties) > 0 {
		props := make(map[string]any, len(params.TaskProperties))
		for k, v := range params.TaskProperties {
			props[k] = v
		}

		taskCreate.Properties = props
	}

	return taskCreate
}

// GenerateTaskObservation builds a TaskObservation from a Nexus Task.
func GenerateTaskObservation(observed *nxtask.Task) instancev1alpha1.TaskObservation {
	if observed == nil {
		return instancev1alpha1.TaskObservation{}
	}

	return instancev1alpha1.TaskObservation{
		ID:           observed.ID,
		CurrentState: observed.CurrentState,
		NextRun:      observed.NextRun,
		LastRun:      observed.LastRun,
	}
}

// IsTaskUpToDate reports whether the CR spec matches the observed Nexus task.
func IsTaskUpToDate(cr *instancev1alpha1.Task, observed *nxtask.Task) bool {
	params := cr.Spec.ForProvider

	if observed.Type != params.TypeID {
		return false
	}

	if observed.Name != params.Name {
		return false
	}

	// message is informational; check it if set
	if params.Message != nil && observed.Message != *params.Message {
		return false
	}

	// schedule comparison
	if !isScheduleUpToDate(params.Schedule, observed.Frequency) {
		return false
	}

	return true
}

// generateFrequency converts a TaskSchedule CR spec into a FrequencyXO.
func generateFrequency(sched *instancev1alpha1.TaskSchedule) *nxtask.FrequencyXO {
	if sched == nil {
		return nil
	}

	freq := &nxtask.FrequencyXO{
		Schedule: sched.Type,
	}

	if sched.StartDate != nil {
		freq.StartDate = int(*sched.StartDate)
	}

	if sched.Timezone != nil {
		freq.TimeZoneOffset = *sched.Timezone
	}

	if sched.CronExpression != nil {
		freq.CronExpression = *sched.CronExpression
	}

	if len(sched.RecurringDays) > 0 {
		days := make([]any, len(sched.RecurringDays))
		for i, d := range sched.RecurringDays {
			days[i] = d
		}

		freq.RecurringDays = days
	}

	return freq
}

// isScheduleUpToDate compares the desired schedule with the observed frequency.
func isScheduleUpToDate(desired *instancev1alpha1.TaskSchedule, observed *nxtask.FrequencyXO) bool {
	if desired == nil && observed == nil {
		return true
	}

	if desired == nil || observed == nil {
		return false
	}

	if desired.Type != observed.Schedule {
		return false
	}

	return isFrequencyFieldsUpToDate(desired, observed)
}

// isFrequencyFieldsUpToDate checks optional frequency fields for drift.
func isFrequencyFieldsUpToDate(desired *instancev1alpha1.TaskSchedule, observed *nxtask.FrequencyXO) bool {
	if desired.StartDate != nil && int(*desired.StartDate) != observed.StartDate {
		return false
	}

	if desired.Timezone != nil && *desired.Timezone != observed.TimeZoneOffset {
		return false
	}

	if desired.CronExpression != nil && *desired.CronExpression != observed.CronExpression {
		return false
	}

	return true
}
