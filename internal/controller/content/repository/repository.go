// Package repository contains the controller for Repository resources, which
// manage Sonatype Nexus repositories of every supported format.
//
// Layout:
//   - repository.go: the Crossplane controller (this file)
//   - handler.go:    the format registry and the generic per-type operations
//   - convert.go:    the spec to Nexus payload conversion shared by all formats
//   - diff.go:       drift detection between the spec and what Nexus reports
//   - format_*.go:   one registration table and payload builder set per format
//
// To add a repository format, add a format_<name>.go that calls
// registerFormat with one binding per repository type the format supports.
// Nothing else has to change: observation, drift detection, error handling and
// the controller itself are format agnostic.
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
	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
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
	// errGetRepository is returned when reading the repository from Nexus fails.
	errGetRepository = "cannot get repository from Nexus"
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

// Setup creates a controller for Repository resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(repositoryv1alpha1.RepositoryGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(repositoryv1alpha1.RepositoryGroupVersionKind),
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
		For(&repositoryv1alpha1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the Repository controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isRepo := managedRes.(*repositoryv1alpha1.Repository)
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

	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		client:     nexusClient.Repository,
		kube:       c.kube,
		baseURL:    creds.URL,
		getHandler: getHandler,
	}, nil
}

// external implements managed.ExternalClient.
type external struct {
	// client is the Nexus repository API.
	client *pkgrepository.RepositoryService
	// kube resolves the secrets the spec references.
	kube client.Client
	// baseURL is the Nexus base URL, used to report the repository URL.
	baseURL string
	// getHandler resolves a repository format to its handler.
	getHandler func(format string) formatHandler
}

// Observe checks whether the Repository exists in Nexus and is up to date.
//
// The desired payload it diffs against is built without the HTTP client
// password: Nexus never returns credentials, so they take no part in drift
// detection and Observe does not need to read the secret on every poll.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	repoCR, isRepo := managedRes.(*repositoryv1alpha1.Repository)
	if !isRepo {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	name := repositoryName(repoCR)

	state, err := handler.Observe(e.client, name, repoCR.Spec.ForProvider.Type, desiredRepo{repo: repoCR})
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}

	if !state.exists {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	meta.SetExternalName(repoCR, name)

	repoCR.Status.AtProvider = generateRepositoryObservation(state.fields, e.repositoryURL(name))
	repoCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: state.upToDate,
	}, nil
}

// Create creates the Repository in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	repoCR, isRepo := managedRes.(*repositoryv1alpha1.Repository)
	if !isRepo {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	desired, err := e.desiredState(ctx, repoCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = handler.Create(e.client, repoCR.Spec.ForProvider.Type, desired)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	meta.SetExternalName(repoCR, repositoryName(repoCR))

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Repository in Nexus with the spec.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	repoCR, isRepo := managedRes.(*repositoryv1alpha1.Repository)
	if !isRepo {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	desired, err := e.desiredState(ctx, repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = handler.Update(e.client, repositoryName(repoCR), repoCR.Spec.ForProvider.Type, desired)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Repository from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	repoCR, isRepo := managedRes.(*repositoryv1alpha1.Repository)
	if !isRepo {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	err = handler.Delete(e.client, repositoryName(repoCR), repoCR.Spec.ForProvider.Type)
	if err != nil && !helpers.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteRepository)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// handlerFor returns the handler registered for the repository's format.
func (e *external) handlerFor(repoCR *repositoryv1alpha1.Repository) (formatHandler, error) {
	handler := e.getHandler(repoCR.Spec.ForProvider.Format)
	if handler == nil {
		return nil, errors.Errorf("unsupported repository format %q, supported formats are %v", repoCR.Spec.ForProvider.Format, supportedFormats())
	}

	return handler, nil
}

// desiredState assembles the desired repository, resolving the secrets the
// spec references.
func (e *external) desiredState(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (desiredRepo, error) {
	password, err := e.resolveHTTPClientPassword(ctx, repoCR)
	if err != nil {
		return desiredRepo{}, errors.Wrap(err, errResolvePassword)
	}

	return desiredRepo{repo: repoCR, httpPassword: password}, nil
}

// repositoryURL returns the URL Nexus serves the repository under.
func (e *external) repositoryURL(name string) string {
	return strings.TrimSuffix(e.baseURL, "/") + "/repository/" + name
}

// resolveHTTPClientPassword resolves the password from a Kubernetes secret if
// httpClient.authentication.passwordSecretRef is configured. It returns an
// empty password when the spec references no secret.
func (e *external) resolveHTTPClientPassword(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (string, error) {
	httpClient := repoCR.Spec.ForProvider.HTTPClient
	if httpClient == nil || httpClient.Authentication == nil || httpClient.Authentication.PasswordSecretRef == nil {
		return "", nil
	}

	data, err := resource.ExtractSecret(ctx, e.kube, xpv2.CommonCredentialSelectors{
		SecretRef: httpClient.Authentication.PasswordSecretRef,
	})
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// repositoryName returns the name that identifies the repository in Nexus.
//
// spec.forProvider.name is authoritative. crossplane-runtime seeds the
// crossplane.io/external-name annotation from metadata.name, which is not
// necessarily the repository name, and Nexus cannot rename a repository, so
// trusting the annotation first would make the controller look up - and then
// try to create - the wrong repository whenever the two names differ. The
// annotation is only a fallback for a spec that carries no name at all.
func repositoryName(repoCR *repositoryv1alpha1.Repository) string {
	if repoCR.Spec.ForProvider.Name != "" {
		return repoCR.Spec.ForProvider.Name
	}

	return meta.GetExternalName(repoCR)
}
