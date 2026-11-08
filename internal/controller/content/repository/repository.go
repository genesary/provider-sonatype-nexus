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

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
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
	// errFingerprint is returned when the repository credentials cannot be
	// fingerprinted.
	errFingerprint = "cannot fingerprint repository credentials"
	// errEmptyBlobStoreName is returned when the spec declares storage
	// without a blob store, which happens when an Optional blobStoreNameRef or
	// blobStoreNameSelector resolves to nothing. Nexus would reject the
	// request, and a repository cannot move to another blob store later.
	errEmptyBlobStoreName = "spec.forProvider.storage.blobStoreName is empty: set it, or reference a BlobStore that is Ready"
)

// Setup creates a controller for Repository resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(repositoryv1alpha1.RepositoryGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*repositoryv1alpha1.Repository](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		helpers.WithSpecExternalName(mgr.GetClient(), func(cr *repositoryv1alpha1.Repository) string {
			return cr.Spec.ForProvider.Name
		}),
	)

	rec := managed.NewReconciler(mgr, resource.ManagedKind(repositoryv1alpha1.RepositoryGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &repositoryv1alpha1.RepositoryList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&repositoryv1alpha1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates a TypedExternalClient for the Repository controller.
func (c *connector) Connect(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (managed.TypedExternalClient[*repositoryv1alpha1.Repository], error) {
	err := c.usage.Track(ctx, repoCR)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, repoCR)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		client:     nexusClient.RepositoryManagementAPI,
		kube:       c.kube,
		baseURL:    creds.URL,
		getHandler: getHandler,
	}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	// client is the Nexus repository API.
	client *nexusAPI
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
// detection. Credentials are compared through the fingerprint recorded when
// they were last applied instead, which needs the Secret's metadata only.
func (e *external) Observe(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (managed.ExternalObservation, error) {
	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	name := repositoryName(repoCR)

	state, err := handler.Observe(ctx, e.client, name, repoCR.Spec.ForProvider.Type, desiredRepo{repo: repoCR})
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetRepository)
	}

	if !state.exists {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	credentialsUpToDate, err := e.credentialsUpToDate(ctx, repoCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	meta.SetExternalName(repoCR, name)

	appliedFingerprint := repoCR.Status.AtProvider.CredentialsFingerprint
	repoCR.Status.AtProvider = generateRepositoryObservation(state.fields, e.repositoryURL(name))
	repoCR.Status.AtProvider.CredentialsFingerprint = appliedFingerprint
	repoCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: credentialsUpToDate && state.upToDate,
	}, nil
}

// Create creates the Repository in Nexus.
func (e *external) Create(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (managed.ExternalCreation, error) {
	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	// One Secret read serves both the fingerprint and the password, so the
	// recorded fingerprint describes exactly the password applied.
	secrets := nexus.MemoizeSecrets(e.kube)

	fingerprint, err := e.credentialsFingerprint(ctx, secrets, repoCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	desired, err := e.desiredState(ctx, secrets, repoCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = handler.Create(ctx, e.client, repoCR.Spec.ForProvider.Type, desired)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateRepository)
	}

	if fingerprint != "" {
		// Non-fatal: without a recorded fingerprint the next Observe reports
		// drift and Update re-applies the credentials.
		_ = nexus.PatchStatus(ctx, e.kube, repoCR, func(r *repositoryv1alpha1.Repository) {
			r.Status.AtProvider.CredentialsFingerprint = fingerprint
		})
	}

	// Set after the status patch, which overwrites in-memory metadata with the
	// server response.
	meta.SetExternalName(repoCR, repositoryName(repoCR))

	return managed.ExternalCreation{}, nil
}

// Update reconciles the Repository in Nexus with the spec.
func (e *external) Update(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (managed.ExternalUpdate, error) {
	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	secrets := nexus.MemoizeSecrets(e.kube)

	fingerprint, err := e.credentialsFingerprint(ctx, secrets, repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	desired, err := e.desiredState(ctx, secrets, repoCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	err = handler.Update(ctx, e.client, repositoryName(repoCR), repoCR.Spec.ForProvider.Type, desired)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateRepository)
	}

	repoCR.Status.AtProvider.CredentialsFingerprint = fingerprint

	return managed.ExternalUpdate{}, nil
}

// Delete removes the Repository from Nexus.
func (e *external) Delete(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (managed.ExternalDelete, error) {
	handler, err := e.handlerFor(repoCR)
	if err != nil {
		return managed.ExternalDelete{}, err
	}

	err = handler.Delete(ctx, e.client, repositoryName(repoCR), repoCR.Spec.ForProvider.Type)
	if err != nil && !nexus.IsNotFound(err) {
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
// spec references through kube. It fails when the spec cannot be applied.
func (e *external) desiredState(ctx context.Context, kube client.Reader, repoCR *repositoryv1alpha1.Repository) (desiredRepo, error) {
	if storage := repoCR.Spec.ForProvider.Storage; storage != nil && storage.BlobStoreName == "" {
		return desiredRepo{}, errors.New(errEmptyBlobStoreName)
	}

	password, err := resolveHTTPClientPassword(ctx, kube, repoCR)
	if err != nil {
		return desiredRepo{}, errors.Wrap(err, errResolvePassword)
	}

	return desiredRepo{repo: repoCR, httpPassword: password}, nil
}

// repositoryURL returns the URL Nexus serves the repository under.
func (e *external) repositoryURL(name string) string {
	return strings.TrimSuffix(e.baseURL, "/") + "/repository/" + name
}

// credentialsFingerprint fingerprints the credentials the Repository applies
// that Nexus redacts on read: the HTTP client authentication, including its
// password Secret, and the APT, Yum, alpine and Terraform signing keys. It
// reads the Secret through kube.
//
// Every field is omitted when unset, so that declaring a new kind of
// credential leaves the fingerprint of a Repository not using it unchanged.
func (e *external) credentialsFingerprint(ctx context.Context, kube client.Reader, repoCR *repositoryv1alpha1.Repository) (string, error) {
	params := repoCR.Spec.ForProvider

	var authentication *repositoryv1alpha1.HTTPClientAuthentication
	if params.HTTPClient != nil {
		authentication = params.HTTPClient.Authentication
	}

	var spec any
	if authentication != nil || params.AptSigning != nil || params.YumSigning != nil ||
		params.AlpineSigning != nil || params.TerraformSigning != nil {
		spec = struct {
			Authentication   *repositoryv1alpha1.HTTPClientAuthentication `json:"authentication,omitempty"`
			AptSigning       *repositoryv1alpha1.AptSigningConfig         `json:"aptSigning,omitempty"`
			YumSigning       *repositoryv1alpha1.YumSigningConfig         `json:"yumSigning,omitempty"`
			AlpineSigning    *repositoryv1alpha1.AlpineSigningConfig      `json:"alpineSigning,omitempty"`
			TerraformSigning *repositoryv1alpha1.TerraformSigningConfig   `json:"terraformSigning,omitempty"`
		}{authentication, params.AptSigning, params.YumSigning, params.AlpineSigning, params.TerraformSigning}
	}

	var passwordRef *xpv2.LocalSecretKeySelector
	if authentication != nil {
		passwordRef = authentication.PasswordSecretRef
	}

	fingerprint, err := nexus.CredentialsFingerprint(ctx, kube, repoCR.GetNamespace(), spec, passwordRef)

	return fingerprint, errors.Wrap(err, errFingerprint)
}

// credentialsUpToDate reports whether the credentials last applied to Nexus
// are still the ones the Repository declares. Nexus redacts them on read, so
// this is the only way a rotated Secret, a repointed reference or a new
// signing key is detected. A Repository being deleted is always up to date,
// so that a deleted Secret cannot block its deletion.
func (e *external) credentialsUpToDate(ctx context.Context, repoCR *repositoryv1alpha1.Repository) (bool, error) {
	if meta.WasDeleted(repoCR) {
		return true, nil
	}

	fingerprint, err := e.credentialsFingerprint(ctx, e.kube, repoCR)
	if err != nil {
		return false, err
	}

	return fingerprint == repoCR.Status.AtProvider.CredentialsFingerprint, nil
}

// resolveHTTPClientPassword resolves the password from a Kubernetes secret if
// httpClient.authentication.passwordSecretRef is configured. It returns an
// empty password when the spec references no secret.
func resolveHTTPClientPassword(ctx context.Context, kube client.Reader, repoCR *repositoryv1alpha1.Repository) (string, error) {
	httpClient := repoCR.Spec.ForProvider.HTTPClient
	if httpClient == nil || httpClient.Authentication == nil || httpClient.Authentication.PasswordSecretRef == nil {
		return "", nil
	}

	return nexus.GetSecretValue(ctx, kube, repoCR.GetNamespace(), httpClient.Authentication.PasswordSecretRef)
}

// repositoryName returns the name that identifies the repository in Nexus.
//
// spec.forProvider.name is authoritative: it is immutable, as Nexus cannot
// rename a repository. helpers.WithSpecExternalName only seeds an empty
// crossplane.io/external-name annotation from it, so an annotation set
// earlier may differ until Observe or Create overwrites it with this name.
// The annotation is only a fallback for a spec that carries no name at all.
func repositoryName(repoCR *repositoryv1alpha1.Repository) string {
	if repoCR.Spec.ForProvider.Name != "" {
		return repoCR.Spec.ForProvider.Name
	}

	return meta.GetExternalName(repoCR)
}
