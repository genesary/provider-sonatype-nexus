// Package repository contains the controller for Repository resources.
package repository

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane-contrib/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/crossplane-contrib/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	errNotRepository    = "managed resource is not a Repository custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create new Nexus client"
	errGetRepository    = "cannot get repository from Nexus"
	errCreateRepository = "cannot create repository in Nexus"
	errUpdateRepository = "cannot update repository in Nexus"
	errDeleteRepository = "cannot delete repository from Nexus"
)

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

	return &external{client: nc}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
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

	var exists bool
	var upToDate bool

	format := cr.Spec.ForProvider.Format
	repoType := cr.Spec.ForProvider.Type

	switch format {
	case "maven2":
		exists, upToDate = e.observeMaven(ctx, cr, name, repoType)
	case "docker":
		exists, upToDate = e.observeDocker(ctx, cr, name, repoType)
	case "npm":
		exists, upToDate = e.observeNpm(ctx, cr, name, repoType)
	case "raw":
		exists, upToDate = e.observeRaw(ctx, cr, name, repoType)
	default:
		return managed.ExternalObservation{}, errors.Errorf("unsupported format: %s", format)
	}

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

	var err error
	switch format {
	case "maven2":
		err = e.createMaven(ctx, cr, repoType)
	case "docker":
		err = e.createDocker(ctx, cr, repoType)
	case "npm":
		err = e.createNpm(ctx, cr, repoType)
	case "raw":
		err = e.createRaw(ctx, cr, repoType)
	default:
		return managed.ExternalCreation{}, errors.Errorf("unsupported format: %s", format)
	}

	if err != nil {
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

	var err error
	switch format {
	case "maven2":
		err = e.updateMaven(ctx, cr, name, repoType)
	case "docker":
		err = e.updateDocker(ctx, cr, name, repoType)
	case "npm":
		err = e.updateNpm(ctx, cr, name, repoType)
	case "raw":
		err = e.updateRaw(ctx, cr, name, repoType)
	default:
		return managed.ExternalUpdate{}, errors.Errorf("unsupported format: %s", format)
	}

	if err != nil {
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

	var err error
	switch format {
	case "maven2":
		err = e.deleteMaven(ctx, name, repoType)
	case "docker":
		err = e.deleteDocker(ctx, name, repoType)
	case "npm":
		err = e.deleteNpm(ctx, name, repoType)
	case "raw":
		err = e.deleteRaw(ctx, name, repoType)
	default:
		return errors.Errorf("unsupported format: %s", format)
	}

	if err != nil && !isNotFound(err) {
		return errors.Wrap(err, errDeleteRepository)
	}

	return nil
}

// Maven repository operations

func (e *external) observeMaven(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) (exists, upToDate bool) {
	switch repoType {
	case "hosted":
		repo, err := e.client.Repository().GetMavenHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isMavenHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := e.client.Repository().GetMavenProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isMavenProxyUpToDate(cr, repo)
	case "group":
		repo, err := e.client.Repository().GetMavenGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isMavenGroupUpToDate(cr, repo)
	}
	return false, false
}

func (e *external) createMaven(ctx context.Context, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().CreateMavenHosted(ctx, generateMavenHosted(cr))
	case "proxy":
		return e.client.Repository().CreateMavenProxy(ctx, generateMavenProxy(cr))
	case "group":
		return e.client.Repository().CreateMavenGroup(ctx, generateMavenGroup(cr))
	}
	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

func (e *external) updateMaven(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().UpdateMavenHosted(ctx, name, generateMavenHosted(cr))
	case "proxy":
		return e.client.Repository().UpdateMavenProxy(ctx, name, generateMavenProxy(cr))
	case "group":
		return e.client.Repository().UpdateMavenGroup(ctx, name, generateMavenGroup(cr))
	}
	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

func (e *external) deleteMaven(ctx context.Context, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().DeleteMavenHosted(ctx, name)
	case "proxy":
		return e.client.Repository().DeleteMavenProxy(ctx, name)
	case "group":
		return e.client.Repository().DeleteMavenGroup(ctx, name)
	}
	return errors.Errorf("unsupported maven repository type: %s", repoType)
}

// Docker repository operations

func (e *external) observeDocker(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) (exists, upToDate bool) {
	switch repoType {
	case "hosted":
		repo, err := e.client.Repository().GetDockerHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isDockerHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := e.client.Repository().GetDockerProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isDockerProxyUpToDate(cr, repo)
	case "group":
		repo, err := e.client.Repository().GetDockerGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isDockerGroupUpToDate(cr, repo)
	}
	return false, false
}

func (e *external) createDocker(ctx context.Context, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().CreateDockerHosted(ctx, generateDockerHosted(cr))
	case "proxy":
		return e.client.Repository().CreateDockerProxy(ctx, generateDockerProxy(cr))
	case "group":
		return e.client.Repository().CreateDockerGroup(ctx, generateDockerGroup(cr))
	}
	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

func (e *external) updateDocker(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().UpdateDockerHosted(ctx, name, generateDockerHosted(cr))
	case "proxy":
		return e.client.Repository().UpdateDockerProxy(ctx, name, generateDockerProxy(cr))
	case "group":
		return e.client.Repository().UpdateDockerGroup(ctx, name, generateDockerGroup(cr))
	}
	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

func (e *external) deleteDocker(ctx context.Context, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().DeleteDockerHosted(ctx, name)
	case "proxy":
		return e.client.Repository().DeleteDockerProxy(ctx, name)
	case "group":
		return e.client.Repository().DeleteDockerGroup(ctx, name)
	}
	return errors.Errorf("unsupported docker repository type: %s", repoType)
}

// npm repository operations

func (e *external) observeNpm(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) (exists, upToDate bool) {
	switch repoType {
	case "hosted":
		repo, err := e.client.Repository().GetNpmHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isNpmHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := e.client.Repository().GetNpmProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isNpmProxyUpToDate(cr, repo)
	case "group":
		repo, err := e.client.Repository().GetNpmGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isNpmGroupUpToDate(cr, repo)
	}
	return false, false
}

func (e *external) createNpm(ctx context.Context, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().CreateNpmHosted(ctx, generateNpmHosted(cr))
	case "proxy":
		return e.client.Repository().CreateNpmProxy(ctx, generateNpmProxy(cr))
	case "group":
		return e.client.Repository().CreateNpmGroup(ctx, generateNpmGroup(cr))
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

func (e *external) updateNpm(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().UpdateNpmHosted(ctx, name, generateNpmHosted(cr))
	case "proxy":
		return e.client.Repository().UpdateNpmProxy(ctx, name, generateNpmProxy(cr))
	case "group":
		return e.client.Repository().UpdateNpmGroup(ctx, name, generateNpmGroup(cr))
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

func (e *external) deleteNpm(ctx context.Context, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().DeleteNpmHosted(ctx, name)
	case "proxy":
		return e.client.Repository().DeleteNpmProxy(ctx, name)
	case "group":
		return e.client.Repository().DeleteNpmGroup(ctx, name)
	}
	return errors.Errorf("unsupported npm repository type: %s", repoType)
}

// Raw repository operations

func (e *external) observeRaw(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) (exists, upToDate bool) {
	switch repoType {
	case "hosted":
		repo, err := e.client.Repository().GetRawHosted(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isRawHostedUpToDate(cr, repo)
	case "proxy":
		repo, err := e.client.Repository().GetRawProxy(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isRawProxyUpToDate(cr, repo)
	case "group":
		repo, err := e.client.Repository().GetRawGroup(ctx, name)
		if err != nil || repo == nil {
			return false, false
		}
		return true, isRawGroupUpToDate(cr, repo)
	}
	return false, false
}

func (e *external) createRaw(ctx context.Context, cr *v1alpha1.Repository, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().CreateRawHosted(ctx, generateRawHosted(cr))
	case "proxy":
		return e.client.Repository().CreateRawProxy(ctx, generateRawProxy(cr))
	case "group":
		return e.client.Repository().CreateRawGroup(ctx, generateRawGroup(cr))
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

func (e *external) updateRaw(ctx context.Context, cr *v1alpha1.Repository, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().UpdateRawHosted(ctx, name, generateRawHosted(cr))
	case "proxy":
		return e.client.Repository().UpdateRawProxy(ctx, name, generateRawProxy(cr))
	case "group":
		return e.client.Repository().UpdateRawGroup(ctx, name, generateRawGroup(cr))
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
}

func (e *external) deleteRaw(ctx context.Context, name, repoType string) error {
	switch repoType {
	case "hosted":
		return e.client.Repository().DeleteRawHosted(ctx, name)
	case "proxy":
		return e.client.Repository().DeleteRawProxy(ctx, name)
	case "group":
		return e.client.Repository().DeleteRawGroup(ctx, name)
	}
	return errors.Errorf("unsupported raw repository type: %s", repoType)
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
