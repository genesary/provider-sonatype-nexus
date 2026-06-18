// Package saml contains the controller for SAML resources.
package saml

import (
	"context"
	"reflect"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotSAML is returned when the managed resource is not a SAML.
	errNotSAML = "managed resource is not a SAML custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetSAML is returned when retrieving SAML configuration fails.
	errGetSAML = "cannot get SAML configuration from Nexus"
	// errApplySAML is returned when applying SAML configuration fails.
	errApplySAML = "cannot apply SAML configuration in Nexus"
	// errDeleteSAML is returned when deleting SAML configuration fails.
	errDeleteSAML = "cannot delete SAML configuration from Nexus"
)

// Setup creates a controller for SAML resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.SAMLGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.SAMLGroupVersionKind),
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
		For(&v1alpha1.SAML{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the SAML controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isSAML := managedRes.(*v1alpha1.SAML)
	if !isSAML {
		return nil, errors.New(errNotSAML)
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

	return &external{client: nexusClient}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
}

// Observe checks if the SAML resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	samlCR, isSAML := managedRes.(*v1alpha1.SAML)
	if !isSAML {
		return managed.ExternalObservation{}, errors.New(errNotSAML)
	}

	samlResult, err := e.client.Security().GetSAML(ctx)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetSAML)
	}

	if samlResult == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	samlCR.SetConditions(v1alpha1.Available())

	upToDate := isSAMLUpToDate(samlCR, samlResult)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new SAML resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	samlCR, isSAML := managedRes.(*v1alpha1.SAML)
	if !isSAML {
		return managed.ExternalCreation{}, errors.New(errNotSAML)
	}

	samlCfg := generateSAML(samlCR)

	err := e.client.Security().ApplySAML(ctx, samlCfg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing SAML resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	samlCR, isSAML := managedRes.(*v1alpha1.SAML)
	if !isSAML {
		return managed.ExternalUpdate{}, errors.New(errNotSAML)
	}

	samlCfg := generateSAML(samlCR)

	err := e.client.Security().ApplySAML(ctx, samlCfg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errApplySAML)
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing SAML resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	_, isSAML := managedRes.(*v1alpha1.SAML)
	if !isSAML {
		return managed.ExternalDelete{}, errors.New(errNotSAML)
	}

	err := e.client.Security().DeleteSAML(ctx)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteSAML)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// generateSAML generates SAML configuration from the CR spec.
func generateSAML(samlCR *v1alpha1.SAML) security.SAML {
	return security.SAML{
		IdpMetadata:                samlCR.Spec.ForProvider.IdpMetadata,
		EntityId:                   samlCR.Spec.ForProvider.EntityId,
		UsernameAttribute:          samlCR.Spec.ForProvider.UsernameAttribute,
		FirstNameAttribute:         samlCR.Spec.ForProvider.FirstNameAttribute,
		LastNameAttribute:          samlCR.Spec.ForProvider.LastNameAttribute,
		EmailAttribute:             samlCR.Spec.ForProvider.EmailAttribute,
		GroupsAttribute:            samlCR.Spec.ForProvider.GroupsAttribute,
		ValidateResponseSignature:  samlCR.Spec.ForProvider.ValidateResponseSignature,
		ValidateAssertionSignature: samlCR.Spec.ForProvider.ValidateAssertionSignature,
	}
}

// isSAMLUpToDate checks if SAML configuration is up to date.
func isSAMLUpToDate(samlCR *v1alpha1.SAML, samlResult *security.SAML) bool {
	desired := generateSAML(samlCR)

	return reflect.DeepEqual(desired, *samlResult)
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
