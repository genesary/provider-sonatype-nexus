package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Task schedule types, as the Task spec spells them.
const (
	// TaskScheduleManual runs the task only when started by hand.
	TaskScheduleManual = "manual"
	// TaskScheduleOnce runs the task once, at the start date.
	TaskScheduleOnce = "once"
	// TaskScheduleHourly runs the task every hour from the start date.
	TaskScheduleHourly = "hourly"
	// TaskScheduleDaily runs the task every day from the start date.
	TaskScheduleDaily = "daily"
	// TaskScheduleWeekly runs the task on the recurring days of each week.
	TaskScheduleWeekly = "weekly"
	// TaskScheduleMonthly runs the task on the recurring days of each month.
	TaskScheduleMonthly = "monthly"
	// TaskScheduleCron runs the task on a cron expression.
	TaskScheduleCron = "cron"
)

// TaskParameters are the configurable fields of a Nexus scheduled task
// (Administration, System, Tasks).
type TaskParameters struct {
	// Name of the task. Nexus requires task names to be unique, and the task
	// is looked up by this name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Type is the task type identifier, such as repository.cleanup,
	// blobstore.compact or repository.docker.gc. GET
	// /service/rest/v1/tasks/templates lists the types a Nexus instance
	// offers. Tasks of type script can only be created when Nexus scripting
	// is enabled (nexus.scripts.allowCreation=true).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
	Type string `json:"type"`

	// Enabled specifies whether the task runs on its schedule.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// AlertEmail is the address Nexus notifies about task runs. If unset, no
	// notification is sent.
	// +kubebuilder:validation:Optional
	AlertEmail *string `json:"alertEmail,omitempty"`

	// NotificationCondition selects the runs Nexus notifies the alert email
	// about: failed runs only, or every run.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=FAILURE;SUCCESS_FAILURE
	// +kubebuilder:default=FAILURE
	NotificationCondition *string `json:"notificationCondition,omitempty"`

	// Schedule defines when the task runs.
	// +kubebuilder:validation:Required
	Schedule TaskSchedule `json:"schedule"`

	// Properties are the type-specific settings of the task, such as
	// blobstoreName or repositoryName. They are the complete set: a property
	// the spec leaves out is removed from the task. Nexus rejects unknown
	// properties and requires the ones the type marks as required. A property
	// set to an empty string is treated as unset.
	// +kubebuilder:validation:Optional
	Properties map[string]string `json:"properties,omitempty"`
}

// TaskSchedule defines when a task runs.
//
// +kubebuilder:validation:XValidation:rule="(self.type in ['manual', 'cron']) != has(self.startDate)",message="startDate is required for once, hourly, daily, weekly and monthly schedules, and not allowed for manual and cron ones."
// +kubebuilder:validation:XValidation:rule="(self.type in ['weekly', 'monthly']) == (has(self.recurringDays) && size(self.recurringDays) > 0)",message="recurringDays is required for weekly and monthly schedules, and only allowed for them."
// +kubebuilder:validation:XValidation:rule="self.type != 'weekly' || !has(self.recurringDays) || self.recurringDays.all(d, d >= 1 && d <= 7)",message="Weekly recurringDays must be between 1 (Sunday) and 7 (Saturday)."
// +kubebuilder:validation:XValidation:rule="self.type != 'monthly' || !has(self.recurringDays) || self.recurringDays.all(d, (d >= 1 && d <= 31) || d == 999)",message="Monthly recurringDays must be between 1 and 31, or 999 for the last day of the month."
// +kubebuilder:validation:XValidation:rule="(self.type == 'cron') == has(self.cronExpression)",message="cronExpression is required for a cron schedule, and only allowed for it."
// +kubebuilder:validation:XValidation:rule="self.type == 'cron' || !has(self.timeZoneOffset)",message="timeZoneOffset is only allowed for a cron schedule."
type TaskSchedule struct {
	// Type of the schedule.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=manual;once;hourly;daily;weekly;monthly;cron
	Type string `json:"type"`

	// StartDate is when the schedule starts: the single run of a once
	// schedule, and the first run and time of day of the hourly, daily,
	// weekly and monthly ones. Nexus keeps it to the minute. Required for
	// those schedules, and not allowed for manual and cron ones: Nexus
	// ignores it on a cron schedule.
	// +kubebuilder:validation:Optional
	StartDate *metav1.Time `json:"startDate,omitempty"`

	// RecurringDays are the days the task runs on: 1 (Sunday) to 7
	// (Saturday) for a weekly schedule, 1 to 31 or 999 (the last day of the
	// month) for a monthly one.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	RecurringDays []int32 `json:"recurringDays,omitempty"`

	// CronExpression is the Quartz cron expression of a cron schedule, with
	// a seconds field, such as "0 0 2 * * ?".
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	CronExpression *string `json:"cronExpression,omitempty"`

	// TimeZoneOffset is the UTC offset the cron expression is evaluated in,
	// such as "Z" or "+02:00". If unset, the expression is evaluated in UTC.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^(Z|[+-](0[0-9]|1[0-4]):[0-5][0-9])$`
	TimeZoneOffset *string `json:"timeZoneOffset,omitempty"`
}

// TaskObservation is the observed state of a Nexus scheduled task.
type TaskObservation struct {
	// ID is the identifier Nexus assigned to the task.
	ID string `json:"id,omitempty"`

	// Type is the observed task type.
	Type string `json:"type,omitempty"`

	// TypeName is the human-readable name of the task type.
	TypeName string `json:"typeName,omitempty"`

	// Enabled is the observed enabled state.
	Enabled bool `json:"enabled,omitempty"`

	// AlertEmail is the observed alert email.
	AlertEmail string `json:"alertEmail,omitempty"`

	// NotificationCondition is the observed notification condition.
	NotificationCondition string `json:"notificationCondition,omitempty"`

	// Schedule is the observed schedule.
	Schedule *TaskScheduleObservation `json:"schedule,omitempty"`

	// Properties are the observed type-specific settings.
	Properties map[string]string `json:"properties,omitempty"`

	// CurrentState is the state of the task, such as WAITING or RUNNING.
	CurrentState string `json:"currentState,omitempty"`

	// Message is the message Nexus reports for the task.
	Message string `json:"message,omitempty"`

	// LastRun is when the last run started.
	LastRun *metav1.Time `json:"lastRun,omitempty"`

	// LastRunResult is the result of the last run, such as OK or FAILED.
	LastRunResult string `json:"lastRunResult,omitempty"`

	// NextRun is when the task runs next.
	NextRun *metav1.Time `json:"nextRun,omitempty"`
}

// TaskScheduleObservation is the observed schedule of a task.
type TaskScheduleObservation struct {
	// Type is the observed schedule type, spelled as in the spec.
	Type string `json:"type,omitempty"`

	// StartDate is the observed start date.
	StartDate *metav1.Time `json:"startDate,omitempty"`

	// RecurringDays are the observed recurring days.
	RecurringDays []int32 `json:"recurringDays,omitempty"`

	// CronExpression is the observed cron expression.
	CronExpression string `json:"cronExpression,omitempty"`

	// TimeZoneOffset is the observed time zone offset of a cron schedule.
	TimeZoneOffset string `json:"timeZoneOffset,omitempty"`
}

// TaskSpec defines the desired state of a Task.
type TaskSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	// ForProvider holds the provider-specific configuration for this resource.
	ForProvider TaskParameters `json:"forProvider"`
}

// TaskStatus represents the observed state of a Task.
type TaskStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	// AtProvider holds the provider-specific observed state.
	AtProvider TaskObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.forProvider.type"
// +kubebuilder:printcolumn:name="NEXT-RUN",type="string",JSONPath=".status.atProvider.nextRun"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Task manages a Nexus scheduled task, such as a repository cleanup, a blob
// store compaction or an index rebuild. Its external name is the task name.
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

// Task type metadata.
var (
	TaskKind             = reflect.TypeFor[Task]().Name()
	TaskGroupKind        = schema.GroupKind{Group: APIGroup, Kind: TaskKind}.String()
	TaskKindAPIVersion   = TaskKind + "." + SchemeGroupVersion.String()
	TaskGroupVersionKind = SchemeGroupVersion.WithKind(TaskKind)
)

// init registers Task types with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
