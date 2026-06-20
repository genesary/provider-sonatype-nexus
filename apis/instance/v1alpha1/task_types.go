package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskSchedule defines the execution schedule for a Nexus task.
type TaskSchedule struct {
	// Type is the schedule type.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=manual;once;hourly;daily;weekly;monthly;cron
	Type string `json:"type"`

	// StartDate is the Unix timestamp in milliseconds for the schedule start date.
	// +kubebuilder:validation:Optional
	StartDate *int64 `json:"startDate,omitempty"`

	// Timezone is the timezone offset string (e.g., "UTC", "+05:00").
	// +kubebuilder:validation:Optional
	Timezone *string `json:"timezone,omitempty"`

	// RecurringDays lists the days for weekly (day names e.g. "sunday") or
	// monthly (day numbers e.g. "1") schedules.
	// +kubebuilder:validation:Optional
	RecurringDays []string `json:"recurringDays,omitempty"`

	// CronExpression is a cron expression used when Type is "cron".
	// +kubebuilder:validation:Optional
	CronExpression *string `json:"cronExpression,omitempty"`
}

// TaskParameters defines the desired state of a Nexus scheduled task.
type TaskParameters struct {
	// Name is the unique task name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// TypeID is the Nexus task type identifier (e.g., "repository.cleanup", "db.backup").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	TypeID string `json:"typeId"`

	// Enabled controls whether the task is active.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// AlertEmail is the email address for task alert notifications.
	// +kubebuilder:validation:Optional
	AlertEmail *string `json:"alertEmail,omitempty"`

	// NotificationCondition controls when email alerts are sent.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=FAILURE;SUCCESS_FAILURE;SUCCESS
	NotificationCondition *string `json:"notificationCondition,omitempty"`

	// Message is an optional description for the task.
	// +kubebuilder:validation:Optional
	Message *string `json:"message,omitempty"`

	// Schedule defines the task execution schedule.
	// If omitted the task is created with a manual schedule.
	// +kubebuilder:validation:Optional
	Schedule *TaskSchedule `json:"schedule,omitempty"`

	// TaskProperties are type-specific configuration key/value pairs.
	// +kubebuilder:validation:Optional
	TaskProperties map[string]string `json:"taskProperties,omitempty"`
}

// TaskObservation defines the observed state of a Nexus scheduled task.
type TaskObservation struct {
	// ID is the server-assigned task identifier.
	ID string `json:"id,omitempty"`

	// CurrentState is the current execution state reported by Nexus.
	CurrentState string `json:"currentState,omitempty"`

	// NextRun is the next scheduled run time.
	NextRun string `json:"nextRun,omitempty"`

	// LastRun is the most recent run time.
	LastRun string `json:"lastRun,omitempty"`
}

// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider TaskParameters `json:"forProvider"`
}

// TaskStatus defines the observed state of Task.
type TaskStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider TaskObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Task manages a Nexus Repository Manager scheduled task.
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task.
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Task `json:"items"`
}

// init registers Task types with the scheme.
func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
