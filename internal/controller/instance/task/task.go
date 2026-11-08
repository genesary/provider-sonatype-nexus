// Package task contains the controller for Task resources.
package task

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetTask is returned when retrieving a task fails.
	errGetTask = "cannot get task from Nexus"
	// errCreateTask is returned when creating a task fails.
	errCreateTask = "cannot create task in Nexus"
	// errUpdateTask is returned when updating a task fails.
	errUpdateTask = "cannot update task in Nexus"
	// errDeleteTask is returned when deleting a task fails.
	errDeleteTask = "cannot delete task from Nexus"
	// errNoTaskID is returned when the ID of an observed task is unknown.
	errNoTaskID = "the Nexus task ID is unknown"
	// errTaskTypeMismatch is returned when the task found under the name has
	// another type than the spec, which an update cannot change.
	errTaskTypeMismatch = "a Nexus task with this name already exists with another type"
)

// Setup adds a controller that reconciles Task managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.TaskGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.Task](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *instancev1alpha1.Task) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.TaskGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.TaskList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.Task{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, task *instancev1alpha1.Task) (managed.TypedExternalClient[*instancev1alpha1.Task], error) {
	err := c.usage.Track(ctx, task)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, task)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	taskClient, err := instanceclient.NewTaskClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: taskClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.TaskClient
}

// Observe looks the task up by its name, the external name, and compares it
// with the spec. It records the server-assigned ID, which Update and Delete
// address the task by.
func (e *external) Observe(ctx context.Context, task *instancev1alpha1.Task) (managed.ExternalObservation, error) {
	name := taskName(task)

	observed, err := e.client.GetByName(ctx, name, task.Spec.ForProvider.Type)
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetTask)
	}

	task.Status.AtProvider = instanceclient.GenerateTaskObservation(observed)

	// Nexus cannot change the type of a task, so a task of another type found
	// under the name is not this one: it is neither updated nor, once the
	// Task is deleted, deleted.
	if observed.GetType() != task.Spec.ForProvider.Type {
		if meta.WasDeleted(task) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Errorf("%s: %q has type %q, want %q", errTaskTypeMismatch, name, observed.GetType(), task.Spec.ForProvider.Type)
	}

	task.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsTaskUpToDate(&task.Spec.ForProvider, observed),
	}, nil
}

// Create creates the task in Nexus.
func (e *external) Create(ctx context.Context, task *instancev1alpha1.Task) (managed.ExternalCreation, error) {
	created, err := e.client.Create(ctx, instanceclient.GenerateTaskCreate(&task.Spec.ForProvider))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTask)
	}

	meta.SetExternalName(task, task.Spec.ForProvider.Name)
	task.Status.AtProvider = instanceclient.GenerateTaskObservation(created)

	return managed.ExternalCreation{}, nil
}

// Update replaces the task in Nexus with the spec. Observe has recorded the
// task ID and properties during the same reconcile.
func (e *external) Update(ctx context.Context, task *instancev1alpha1.Task) (managed.ExternalUpdate, error) {
	taskID := task.Status.AtProvider.ID
	if taskID == "" {
		return managed.ExternalUpdate{}, errors.New(errNoTaskID)
	}

	update := instanceclient.GenerateTaskUpdate(&task.Spec.ForProvider, task.Status.AtProvider.Properties)
	// Nexus would rename the task to whatever name the update carries, and
	// the task is found by name, so keep the name it was observed under.
	update.Name = taskName(task)

	err := e.client.Update(ctx, taskID, update)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateTask)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the task from Nexus. A task already gone is not an error.
func (e *external) Delete(ctx context.Context, task *instancev1alpha1.Task) (managed.ExternalDelete, error) {
	taskID := task.Status.AtProvider.ID
	if taskID == "" {
		return managed.ExternalDelete{}, errors.New(errNoTaskID)
	}

	err := nexus.IgnoreNotFound(e.client.Delete(ctx, taskID))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTask)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// taskName returns the Nexus name of the task: its external name, or the
// spec name until the external name is set.
func taskName(task *instancev1alpha1.Task) string {
	return helpers.ExternalNameOr(task, task.Spec.ForProvider.Name)
}
