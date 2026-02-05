// Package contentselector contains the controller for ContentSelector resources.
package contentselector

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotContentSelector    = "managed resource is not a ContentSelector custom resource"
	errTrackPCUsage          = "cannot track ProviderConfig usage"
	errGetPC                 = "cannot get ProviderConfig"
	errGetCreds              = "cannot get credentials"
	errNewClient             = "cannot create new Nexus client"
	errGetContentSelector    = "cannot get content selector from Nexus"
	errCreateContentSelector = "cannot create content selector in Nexus"
	errUpdateContentSelector = "cannot update content selector in Nexus"
	errDeleteContentSelector = "cannot delete content selector from Nexus"
)

// Setup adds a controller that reconciles ContentSelector managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ContentSelectorGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ContentSelectorGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ContentSelector{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.ContentSelector)
	if !ok {
		return nil, errors.New(errNotContentSelector)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nc}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
}

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ContentSelector)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	cs, err := e.client.Security().GetContentSelector(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetContentSelector)
	}

	if cs == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	upToDate := isContentSelectorUpToDate(cr, cs)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ContentSelector)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotContentSelector)
	}

	cs := generateContentSelector(cr)
	if err := e.client.Security().CreateContentSelector(ctx, cs); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateContentSelector)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)
	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ContentSelector)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	cs := generateContentSelector(cr)
	if err := e.client.Security().UpdateContentSelector(ctx, name, cs); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateContentSelector)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ContentSelector)
	if !ok {
		return errors.New(errNotContentSelector)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	if err := e.client.Security().DeleteContentSelector(ctx, name); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeleteContentSelector)
	}

	return nil
}

// generateContentSelector generates a ContentSelector from the CR spec.
func generateContentSelector(cr *v1alpha1.ContentSelector) security.ContentSelector {
	cs := security.ContentSelector{
		Name:       cr.Spec.ForProvider.Name,
		Expression: cr.Spec.ForProvider.Expression,
	}

	if cr.Spec.ForProvider.Description != nil {
		cs.Description = *cr.Spec.ForProvider.Description
	}

	return cs
}

// isContentSelectorUpToDate checks if a ContentSelector is up to date.
func isContentSelectorUpToDate(cr *v1alpha1.ContentSelector, cs *security.ContentSelector) bool {
	if cr.Spec.ForProvider.Expression != cs.Expression {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && *cr.Spec.ForProvider.Description != cs.Description {
		return false
	}
	return true
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
