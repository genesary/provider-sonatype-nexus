// Package script contains the controller for Script resources.
package script

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
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
	// errGetScript is returned when retrieving a script fails.
	errGetScript = "cannot get script from Nexus"
	// errCreateScript is returned when creating a script fails.
	errCreateScript = "cannot create script in Nexus"
	// errUpdateScript is returned when updating a script fails.
	errUpdateScript = "cannot update script in Nexus"
	// errDeleteScript is returned when deleting a script fails.
	errDeleteScript = "cannot delete script from Nexus"
)

// Setup adds a controller that reconciles Script managed resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.ScriptGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.Script](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *instancev1alpha1.Script) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.ScriptGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.ScriptList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.Script{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, script *instancev1alpha1.Script) (managed.TypedExternalClient[*instancev1alpha1.Script], error) {
	err := c.usage.Track(ctx, script)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, script)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	scriptClient, err := instanceclient.NewScriptClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: scriptClient}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.ScriptClient
}

// Observe checks whether the script exists and matches the spec.
func (e *external) Observe(ctx context.Context, script *instancev1alpha1.Script) (managed.ExternalObservation, error) {
	observed, err := e.client.Get(ctx, scriptName(script))
	if err != nil {
		if nexus.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetScript)
	}

	script.Status.AtProvider = instanceclient.GenerateScriptObservation(observed)
	script.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: instanceclient.IsScriptUpToDate(&script.Spec.ForProvider, observed),
	}, nil
}

// Create stores the script in Nexus, under its external name.
func (e *external) Create(ctx context.Context, script *instancev1alpha1.Script) (managed.ExternalCreation, error) {
	name := scriptName(script)

	err := e.client.Create(ctx, desiredScript(script, name))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateScript)
	}

	meta.SetExternalName(script, name)

	return managed.ExternalCreation{}, nil
}

// Update replaces the script in Nexus with the spec. Nexus ignores the type
// of an update, so a script whose observed type differs from the spec, such
// as an adopted one, is reported instead of updated over and over.
func (e *external) Update(ctx context.Context, script *instancev1alpha1.Script) (managed.ExternalUpdate, error) {
	name := scriptName(script)

	observedType := script.Status.AtProvider.Type
	desiredType := instanceclient.ScriptType(&script.Spec.ForProvider)

	if observedType != "" && observedType != desiredType {
		return managed.ExternalUpdate{}, errors.Wrapf(instanceclient.ErrScriptTypeChanged,
			"%s: script %q is of type %q, not %q", errUpdateScript, name, observedType, desiredType)
	}

	err := e.client.Update(ctx, name, desiredScript(script, name))
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateScript)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete deletes the script from Nexus.
func (e *external) Delete(ctx context.Context, script *instancev1alpha1.Script) (managed.ExternalDelete, error) {
	err := nexus.IgnoreNotFound(e.client.Delete(ctx, scriptName(script)))
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteScript)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// desiredScript builds the script to store under name from the spec. Nexus
// refuses an update whose body names another script than its path, so the
// body carries the name the script is stored under, which an explicit
// external name may set apart from the spec name.
func desiredScript(script *instancev1alpha1.Script, name string) v395.ScriptXO {
	desired := instanceclient.GenerateScript(&script.Spec.ForProvider)
	desired.Name = name

	return desired
}

// scriptName returns the name of the script in Nexus: the external name, or
// the spec name before one is set.
func scriptName(script *instancev1alpha1.Script) string {
	name := meta.GetExternalName(script)
	if name == "" {
		return script.Spec.ForProvider.Name
	}

	return name
}
