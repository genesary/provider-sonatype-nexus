// Package emailconfiguration manages EmailConfiguration resources.
package emailconfiguration

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
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
	// errGetEmail is returned when retrieving the email server configuration
	// fails.
	errGetEmail = "cannot get email server configuration from Nexus"
	// errUpdateEmail is returned when updating the email server configuration
	// fails.
	errUpdateEmail = "cannot update email server configuration in Nexus"
	// errResetEmail is returned when resetting the email server configuration
	// fails.
	errResetEmail = "cannot reset email server configuration in Nexus"
	// errGetPassword is returned when reading the SMTP password fails.
	errGetPassword = "cannot get SMTP password from secret"
	// errFingerprint is returned when the password Secret cannot be
	// fingerprinted.
	errFingerprint = "cannot fingerprint SMTP password secret"
)

// Setup adds a controller that reconciles EmailConfiguration resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.EmailConfigurationGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.EmailConfiguration](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.EmailConfigurationGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.EmailConfigurationList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.EmailConfiguration{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector creates an external client for EmailConfiguration resources.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates the external email server configuration client.
func (c *connector) Connect(ctx context.Context, email *instancev1alpha1.EmailConfiguration) (managed.TypedExternalClient[*instancev1alpha1.EmailConfiguration], error) {
	err := c.usage.Track(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, email)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	emailClient, err := instanceclient.NewEmailClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: emailClient, kube: c.kube}, nil
}

// external implements managed.TypedExternalClient for EmailConfiguration.
type external struct {
	client instanceclient.EmailClient
	kube   client.Client
}

// Observe checks the Nexus email server configuration against desired state.
//
// The configuration always exists. While the resource is being deleted it
// counts as absent once Nexus holds the reset configuration, which is the
// same on every instance. The password Secret is not read then, so that a
// deleted Secret cannot block the deletion.
func (e *external) Observe(ctx context.Context, email *instancev1alpha1.EmailConfiguration) (managed.ExternalObservation, error) {
	observed, err := e.client.Get(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetEmail)
	}

	appliedFingerprint := email.Status.AtProvider.CredentialsFingerprint
	email.Status.AtProvider = instanceclient.GenerateEmailConfigurationObservation(observed)
	email.Status.AtProvider.CredentialsFingerprint = appliedFingerprint

	if meta.WasDeleted(email) {
		return managed.ExternalObservation{ResourceExists: !instanceclient.IsResetEmailConfiguration(observed)}, nil
	}

	fingerprint, err := credentialsFingerprint(ctx, e.kube, email)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	email.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: fingerprint == appliedFingerprint && instanceclient.IsEmailConfigurationUpToDate(email, observed),
	}, nil
}

// Create applies the email server configuration in Nexus. It is a
// singleton, so Create applies the spec like Update.
func (e *external) Create(ctx context.Context, email *instancev1alpha1.EmailConfiguration) (managed.ExternalCreation, error) {
	fingerprint, err := e.apply(ctx, email)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if fingerprint != "" {
		// Non-fatal: without a recorded fingerprint the next Observe reports
		// drift and Update re-applies the password.
		_ = nexus.PatchStatus(ctx, e.kube, email, func(c *instancev1alpha1.EmailConfiguration) {
			c.Status.AtProvider.CredentialsFingerprint = fingerprint
		})
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the email server configuration in Nexus.
func (e *external) Update(ctx context.Context, email *instancev1alpha1.EmailConfiguration) (managed.ExternalUpdate, error) {
	fingerprint, err := e.apply(ctx, email)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	email.Status.AtProvider.CredentialsFingerprint = fingerprint

	return managed.ExternalUpdate{}, nil
}

// Delete resets the email server configuration, which disables email. The
// configuration itself cannot be removed.
func (e *external) Delete(ctx context.Context, _ *instancev1alpha1.EmailConfiguration) (managed.ExternalDelete, error) {
	err := e.client.Reset(ctx)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errResetEmail)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// apply resolves the password and replaces the Nexus email server
// configuration. It returns the fingerprint of the password it applied.
func (e *external) apply(ctx context.Context, email *instancev1alpha1.EmailConfiguration) (string, error) {
	// One read of the Secret serves both the fingerprint and the password,
	// so the recorded fingerprint describes exactly the password applied.
	secrets := nexus.MemoizeSecrets(e.kube)

	fingerprint, err := credentialsFingerprint(ctx, secrets, email)
	if err != nil {
		return "", err
	}

	var password string

	ref := email.Spec.ForProvider.PasswordSecretRef
	if ref != nil {
		password, err = nexus.GetSecretValue(ctx, secrets, email.GetNamespace(), ref)
		if err != nil {
			return "", errors.Wrap(err, errGetPassword)
		}
	}

	config := instanceclient.GenerateEmailConfigurationUpdate(&email.Spec.ForProvider, password)

	err = e.client.Update(ctx, config)
	if err != nil {
		return "", errors.Wrap(err, errUpdateEmail)
	}

	return fingerprint, nil
}

// credentialsFingerprint fingerprints the password Secret the
// EmailConfiguration references, reading it through kube.
func credentialsFingerprint(ctx context.Context, kube client.Reader, email *instancev1alpha1.EmailConfiguration) (string, error) {
	fingerprint, err := nexus.CredentialsFingerprint(ctx, kube, email.GetNamespace(), nil, email.Spec.ForProvider.PasswordSecretRef)

	return fingerprint, errors.Wrap(err, errFingerprint)
}
