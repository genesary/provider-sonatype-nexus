// Package saml contains the controller for SAML resources.
package saml

import (
	"context"
	"reflect"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
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
	errNotSAML      = "managed resource is not a SAML custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Nexus client"
	errGetSAML      = "cannot get SAML configuration from Nexus"
	errApplySAML    = "cannot apply SAML configuration in Nexus"
	errDeleteSAML   = "cannot delete SAML configuration from Nexus"
)

// Setup adds a controller that reconciles SAML managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.SAMLGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SAMLGroupVersionKind),
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
		For(&v1alpha1.SAML{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.SAML)
	if !ok {
		return nil, errors.New(errNotSAML)
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
	cr, ok := mg.(*v1alpha1.SAML)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSAML)
	}

	saml, err := e.client.Security().GetSAML(ctx)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetSAML)
	}

	if saml == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	upToDate := isSAMLUpToDate(cr, saml)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.SAML)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotSAML)
	}

	saml := generateSAML(cr)
	if err := e.client.Security().ApplySAML(ctx, saml); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.SAML)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotSAML)
	}

	saml := generateSAML(cr)
	if err := e.client.Security().ApplySAML(ctx, saml); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	_, ok := mg.(*v1alpha1.SAML)
	if !ok {
		return errors.New(errNotSAML)
	}

	if err := e.client.Security().DeleteSAML(ctx); err != nil {
		if isNotFound(err) {
			return nil
		}
		return errors.Wrap(err, errDeleteSAML)
	}

	return nil
}

// generateSAML generates SAML configuration from the CR spec.
func generateSAML(cr *v1alpha1.SAML) security.SAML {
	saml := security.SAML{
		IdpMetadata:                cr.Spec.ForProvider.IdpMetadata,
		EntityId:                   cr.Spec.ForProvider.EntityId,
		UsernameAttribute:          cr.Spec.ForProvider.UsernameAttribute,
		FirstNameAttribute:         cr.Spec.ForProvider.FirstNameAttribute,
		LastNameAttribute:          cr.Spec.ForProvider.LastNameAttribute,
		EmailAttribute:             cr.Spec.ForProvider.EmailAttribute,
		GroupsAttribute:            cr.Spec.ForProvider.GroupsAttribute,
		ValidateResponseSignature:  cr.Spec.ForProvider.ValidateResponseSignature,
		ValidateAssertionSignature: cr.Spec.ForProvider.ValidateAssertionSignature,
	}

	return saml
}

// isSAMLUpToDate checks if SAML configuration is up to date.
func isSAMLUpToDate(cr *v1alpha1.SAML, saml *security.SAML) bool {
	desired := generateSAML(cr)
	return reflect.DeepEqual(desired, *saml)
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
