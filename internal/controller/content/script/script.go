// Package script contains the controller for Script resources.
package script

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	contentclient "github.com/genesary/provider-sonatype-nexus/internal/clients/content"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// errNotScript means managed resource is not Script.
	errNotScript = "managed resource is not Script custom resource"
	// errTrackPCUsage returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient returned when creating Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetScript is returned when retrieving the script fails.
	errGetScript = "cannot get script from Nexus"
	// errCreateScript is returned when creating the script fails.
	errCreateScript = "cannot create script in Nexus"
	// errUpdateScript is returned when updating the script fails.
	errUpdateScript = "cannot update script in Nexus"
	// errDeleteScript is returned when deleting the script fails.
	errDeleteScript = "cannot delete script from Nexus"

	// errScriptingDisabled is used when Nexus returns 403 because the Scripting
	// API has not been enabled (nexus.scripts.allowCreation=true is required).
	errScriptingDisabled = "Nexus Scripting API is disabled (HTTP 403): set " +
		"nexus.scripts.allowCreation=true in nexus.properties and restart Nexus. " +
		"Note: the Groovy scripting API is deprecated in Nexus 3.21+ and removed in 3.70+."
)

// Setup adds a controller that reconciles Script managed resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(contentv1alpha1.ScriptGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(contentv1alpha1.ScriptGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&contentv1alpha1.Script{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isScript := managedRes.(*contentv1alpha1.Script)
	if !isScript {
		return nil, errors.New(errNotScript)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	sc, err := contentclient.NewScriptClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: sc}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client contentclient.ScriptClient
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	script, isScript := managedRes.(*contentv1alpha1.Script)
	if !isScript {
		return managed.ExternalObservation{}, errors.New(errNotScript)
	}

	name := meta.GetExternalName(script)
	if name == "" {
		name = script.Spec.ForProvider.Name
	}

	observed, err := e.client.GetScript(ctx, name)
	if err != nil {
		if contentclient.IsForbidden(err) {
			return managed.ExternalObservation{}, errors.New(fmt.Sprintf("%s: %s", errGetScript, errScriptingDisabled))
		}

		if helpers.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetScript)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	script.Status.AtProvider = contentclient.GenerateScriptObservation(observed)
	script.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: contentclient.IsScriptUpToDate(script, observed),
	}, nil
}

// Create creates the external resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	script, isScript := managedRes.(*contentv1alpha1.Script)
	if !isScript {
		return managed.ExternalCreation{}, errors.New(errNotScript)
	}

	err := e.client.CreateScript(ctx, contentclient.GenerateScript(script))
	if err != nil {
		if contentclient.IsForbidden(err) {
			return managed.ExternalCreation{}, errors.New(fmt.Sprintf("%s: %s", errCreateScript, errScriptingDisabled))
		}

		return managed.ExternalCreation{}, errors.Wrap(err, errCreateScript)
	}

	meta.SetExternalName(script, script.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update updates the external resource to match the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	script, isScript := managedRes.(*contentv1alpha1.Script)
	if !isScript {
		return managed.ExternalUpdate{}, errors.New(errNotScript)
	}

	err := e.client.UpdateScript(ctx, contentclient.GenerateScript(script))
	if err != nil {
		if contentclient.IsForbidden(err) {
			return managed.ExternalUpdate{}, errors.New(fmt.Sprintf("%s: %s", errUpdateScript, errScriptingDisabled))
		}

		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateScript)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the external resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	script, isScript := managedRes.(*contentv1alpha1.Script)
	if !isScript {
		return managed.ExternalDelete{}, errors.New(errNotScript)
	}

	name := meta.GetExternalName(script)
	if name == "" {
		name = script.Spec.ForProvider.Name
	}

	err := e.client.DeleteScript(ctx, name)
	if err != nil {
		if helpers.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		if contentclient.IsForbidden(err) {
			return managed.ExternalDelete{}, errors.New(fmt.Sprintf("%s: %s", errDeleteScript, errScriptingDisabled))
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteScript)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}
