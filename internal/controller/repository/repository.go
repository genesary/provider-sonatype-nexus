// Package repository contains the controller for Repository resources.
// This controller manages Sonatype Nexus repositories of various formats.
//
// Architecture:
//   - repository.go: Main controller logic (this file)
//   - handler.go: FormatHandler interface and registry
//   - formats/: Format-specific handlers (maven, docker, npm, etc.)
//   - shared.go: Shared configuration generators (will be removed)
//
// To add a new repository format:
//  1. Create a new handler in formats/<format>.go
//  2. Implement the FormatHandler interface
//  3. Register the handler in formats/register.go
package repository

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotRepository is returned when the managed resource is not a Repository.
	errNotRepository = "managed resource is not a Repository custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when getting the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errCreateRepository is returned when creating the repository in Nexus fails.
	errCreateRepository = "cannot create repository in Nexus"
	// errUpdateRepository is returned when updating the repository in Nexus fails.
	errUpdateRepository = "cannot update repository in Nexus"
	// errDeleteRepository is returned when deleting the repository from
	// Nexus fails.
	errDeleteRepository = "cannot delete repository from Nexus"
	// errResolvePassword is returned when resolving the password from a
	// secret fails.
	errResolvePassword = "cannot resolve password from secret"
)

// contextKey is used for passing resolved values through context.
type contextKey string

// resolvedPasswordKey is the context key used to store the resolved HTTP
// client password.
const resolvedPasswordKey contextKey = "resolvedHTTPClientPassword"

// withResolvedPassword stores a resolved password in the context.
func withResolvedPassword(ctx context.Context, password string) context.Context {
	return context.WithValue(ctx, resolvedPasswordKey, password)
}

// getResolvedPassword retrieves the resolved password from the context.
func getResolvedPassword(ctx context.Context) string {
	if val, isString := ctx.Value(resolvedPasswordKey).(string); isString {
		return val
	}

	return ""
}

// Setup creates a controller for Repository resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.RepositoryGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RepositoryGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the Repository controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isRepo := managedRes.(*v1alpha1.Repository)
	if !isRepo {
		return nil, errors.New(errNotRepository)
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

// Observe checks if the Repository resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	repoCR, isRepo := managedRes.(*v1alpha1.Repository)
	if !isRepo {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	name := meta.GetExternalName(repoCR)
	if name == "" {
		name = repoCR.Spec.ForProvider.Name
	}

	format := repoCR.Spec.ForProvider.Format
	repoType := repoCR.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalObservation{}, errors.Errorf("unsupported format: %s", format)
	}

	exists, upToDate := handler.Observe(ctx, e.client, name, repoType, repoCR)
	if !exists {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	repoCR.SetConditions(v1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new Repository resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	repoCR, isRepo := managedRes.(*v1alpha1.Repository)
	if !isRepo {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	format := repoCR.Spec.ForProvider.Format
	repoType := repoCR.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalCreation{}, errors.Errorf("unsupported format: %s", format)
	}

	password, err := e.resolveHTTPClientPassword(ctx, repoCR)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errResolvePassword)
	}

	enrichedCtx := withResolvedPassword(ctx, password)

	err = handler.Create(enrichedCtx, e.client, repoCR, repoType)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	meta.SetExternalName(repoCR, repoCR.Spec.ForProvider.Name)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing Repository resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	repoCR, isRepo := managedRes.(*v1alpha1.Repository)
	if !isRepo {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	name := meta.GetExternalName(repoCR)
	if name == "" {
		name = repoCR.Spec.ForProvider.Name
	}

	format := repoCR.Spec.ForProvider.Format
	repoType := repoCR.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalUpdate{}, errors.Errorf("unsupported format: %s", format)
	}

	password, err := e.resolveHTTPClientPassword(ctx, repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errResolvePassword)
	}

	enrichedCtx := withResolvedPassword(ctx, password)

	err = handler.Update(enrichedCtx, e.client, name, repoCR, repoType)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing Repository resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	repoCR, isRepo := managedRes.(*v1alpha1.Repository)
	if !isRepo {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	name := meta.GetExternalName(repoCR)
	if name == "" {
		name = repoCR.Spec.ForProvider.Name
	}

	format := repoCR.Spec.ForProvider.Format
	repoType := repoCR.Spec.ForProvider.Type

	handler := GetHandler(format)
	if handler == nil {
		return managed.ExternalDelete{}, errors.Errorf("unsupported format: %s", format)
	}

	err := handler.Delete(ctx, e.client, name, repoType)
	if err != nil && !isNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

// resolveHTTPClientPassword resolves the password from a Kubernetes secret if
// httpClient.authentication.passwordSecretRef is configured. The resolved
// password is returned for use by shared HTTP client generation functions.
func (e *external) resolveHTTPClientPassword(ctx context.Context, repoCR *v1alpha1.Repository) (string, error) {
	if repoCR.Spec.ForProvider.HTTPClient == nil ||
		repoCR.Spec.ForProvider.HTTPClient.Authentication == nil ||
		repoCR.Spec.ForProvider.HTTPClient.Authentication.PasswordSecretRef == nil {
		return "", nil
	}

	ref := repoCR.Spec.ForProvider.HTTPClient.Authentication.PasswordSecretRef

	data, err := resource.ExtractSecret(ctx, e.kube, xpv2.CommonCredentialSelectors{
		SecretRef: ref,
	})
	if err != nil {
		return "", err
	}

	return string(data), nil
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
