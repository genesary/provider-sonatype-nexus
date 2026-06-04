// Package repository contains the controller for Repository resources.
// This controller manages Sonatype Nexus repositories of various formats.
//
// Architecture:
//   - repository.go: Main controller logic (this file)
//   - handler.go: FormatHandler interface and registry
//   - formats/: Format-specific handlers (maven, docker, npm, etc.)
//   - shared.go: Shared configuration generators (kept for compatibility, will be removed)
//
// To add a new repository format:
//  1. Create a new handler in formats/<format>.go
//  2. Implement the FormatHandler interface
//  3. Register the handler in formats/register.go
package repository

import (
	"context"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotRepository    = "managed resource is not a Repository custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create new Nexus client"
	errCreateRepository = "cannot create repository in Nexus"
	errUpdateRepository = "cannot update repository in Nexus"
	errDeleteRepository = "cannot delete repository from Nexus"
	errResolvePassword  = "cannot resolve password from secret"
)

// contextKey is used for passing resolved values through context.
type contextKey string

const resolvedPasswordKey contextKey = "resolvedHTTPClientPassword"

// withResolvedPassword stores a resolved password in the context.
func withResolvedPassword(ctx context.Context, password string) context.Context {
	return context.WithValue(ctx, resolvedPasswordKey, password)
}

// getResolvedPassword retrieves the resolved password from the context.
func getResolvedPassword(ctx context.Context) string {
	if v, ok := ctx.Value(resolvedPasswordKey).(string); ok {
		return v
	}

	return ""
}

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryGroupVersionKind),
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
		For(&v1alpha1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnecter.
type connector struct {
	kube  client.Client
	usage resource.Tracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
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

	return &external{client: nc, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
	kube   client.Client
}

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	format := cr.Spec.ForProvider.Format
	repoType := cr.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalObservation{}, errors.Errorf("unsupported format: %s", format)
	}

	exists, upToDate := handler.Observe(ctx, e.client, name, repoType, cr)
	if !exists {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	format := cr.Spec.ForProvider.Format
	repoType := cr.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalCreation{}, errors.Errorf("unsupported format: %s", format)
	}

	ctx, err := e.resolveHTTPClientPassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolvePassword)
	}

	if err := handler.Create(ctx, e.client, cr, repoType); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	format := cr.Spec.ForProvider.Format
	repoType := cr.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalUpdate{}, errors.Errorf("unsupported format: %s", format)
	}

	ctx, err := e.resolveHTTPClientPassword(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errResolvePassword)
	}

	if err := handler.Update(ctx, e.client, name, cr, repoType); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Repository)
	if !ok {
		return errors.New(errNotRepository)
	}

	name := meta.GetExternalName(cr)
	if name == "" {
		name = cr.Spec.ForProvider.Name
	}

	format := cr.Spec.ForProvider.Format
	repoType := cr.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return errors.Errorf("unsupported format: %s", format)
	}

	err := handler.Delete(ctx, e.client, name, repoType)
	if err != nil && !isNotFound(err) {
		return errors.Wrap(err, errDeleteRepository)
	}

	return nil
}

// resolveHTTPClientPassword resolves the password from a Kubernetes secret if
// httpClient.authentication.passwordSecretRef is configured. The resolved password
// is stored in the context for use by shared HTTP client generation functions.
func (e *external) resolveHTTPClientPassword(ctx context.Context, cr *v1alpha1.Repository) (context.Context, error) {
	if cr.Spec.ForProvider.HTTPClient == nil ||
		cr.Spec.ForProvider.HTTPClient.Authentication == nil ||
		cr.Spec.ForProvider.HTTPClient.Authentication.PasswordSecretRef == nil {
		return ctx, nil
	}

	ref := cr.Spec.ForProvider.HTTPClient.Authentication.PasswordSecretRef

	data, err := resource.ExtractSecret(ctx, e.kube, xpv1.CommonCredentialSelectors{
		SecretRef: ref,
	})
	if err != nil {
		return ctx, err
	}

	return withResolvedPassword(ctx, string(data)), nil
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}
