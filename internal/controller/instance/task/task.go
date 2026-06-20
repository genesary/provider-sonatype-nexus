// Package task contains the controller for Task resources.
package task

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
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
	// errNotTask is returned when the managed resource is not a Task CR.
	errNotTask = "managed resource is not a Task custom resource"
	// errTrackPCUsage is returned when ProviderConfig usage tracking fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating a new Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetTask is returned when retrieving the Task from Nexus fails.
	errGetTask = "cannot get Task"
	// errCreateTask is returned when creating the Task in Nexus fails.
	errCreateTask = "cannot create Task"
	// errUpdateTask is returned when updating the Task in Nexus fails.
	errUpdateTask = "cannot update Task"
	// errDeleteTask is returned when deleting the Task from Nexus fails.
	errDeleteTask = "cannot delete Task"
)

// Setup adds a controller that reconciles Task resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(instancev1alpha1.TaskGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(instancev1alpha1.TaskGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ClusterProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.Task{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isTask := managedRes.(*instancev1alpha1.Task)
	if !isTask {
		return nil, errors.New(errNotTask)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	taskClient, err := instanceclient.NewTaskClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: taskClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client instanceclient.TaskClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	taskCR, isTask := managedRes.(*instancev1alpha1.Task)
	if !isTask {
		return managed.ExternalObservation{}, errors.New(errNotTask)
	}

	name := meta.GetExternalName(taskCR)
	if name == "" {
		name = taskCR.Spec.ForProvider.Name
	}

	observed, err := e.client.GetTaskByName(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetTask)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	taskCR.Status.AtProvider = instanceclient.GenerateTaskObservation(observed)
	taskCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsTaskUpToDate(taskCR, observed),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	taskCR, isTask := managedRes.(*instancev1alpha1.Task)
	if !isTask {
		return managed.ExternalCreation{}, errors.New(errNotTask)
	}

	created, err := e.client.CreateTask(ctx, instanceclient.GenerateTaskCreateStruct(taskCR))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTask)
	}

	meta.SetExternalName(taskCR, created.ID)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	taskCR, isTask := managedRes.(*instancev1alpha1.Task)
	if !isTask {
		return managed.ExternalUpdate{}, errors.New(errNotTask)
	}

	id := taskCR.Status.AtProvider.ID
	if id == "" {
		id = meta.GetExternalName(taskCR)
	}

	err := e.client.UpdateTask(ctx, id, instanceclient.GenerateTaskCreateStruct(taskCR))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateTask)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	taskCR, isTask := managedRes.(*instancev1alpha1.Task)
	if !isTask {
		return managed.ExternalDelete{}, errors.New(errNotTask)
	}

	id := taskCR.Status.AtProvider.ID
	if id == "" {
		id = meta.GetExternalName(taskCR)
	}

	err := e.client.DeleteTask(ctx, id)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTask)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
