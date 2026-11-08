// Package httpclient manages HTTPClient resources.
package httpclient

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
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
	// errGetHTTP is returned when retrieving the HTTP client settings fails.
	errGetHTTP = "cannot get HTTP client settings from Nexus"
	// errUpdateHTTP is returned when updating the HTTP client settings fails.
	errUpdateHTTP = "cannot update HTTP client settings in Nexus"
	// errResetHTTP is returned when resetting the HTTP client settings fails.
	errResetHTTP = "cannot reset HTTP client settings in Nexus"
	// errGetHTTPPassword is returned when reading the HTTP proxy password
	// fails.
	errGetHTTPPassword = "cannot get HTTP proxy password from secret"
	// errGetHTTPSPassword is returned when reading the HTTPS proxy password
	// fails.
	errGetHTTPSPassword = "cannot get HTTPS proxy password from secret"
	// errFingerprint is returned when the proxy credential Secrets cannot be
	// fingerprinted.
	errFingerprint = "cannot fingerprint proxy credential secrets"
)

// Setup adds a controller that reconciles HTTPClient resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.HTTPClientGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.HTTPClient](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.HTTPClientGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.HTTPClientList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.HTTPClient{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector creates an external client for HTTPClient resources.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates the external HTTP client settings client.
func (c *connector) Connect(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (managed.TypedExternalClient[*instancev1alpha1.HTTPClient], error) {
	err := c.usage.Track(ctx, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	settingsClient, err := instanceclient.NewHTTPSettingsClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: settingsClient, kube: c.kube}, nil
}

// external implements managed.TypedExternalClient for HTTPClient.
type external struct {
	client instanceclient.HTTPSettingsClient
	kube   client.Client
}

// Observe checks the Nexus HTTP client settings against desired state.
//
// The settings always exist. A resource being deleted reports them present
// until Delete has reset them, which it records in status: the defaults
// Nexus resets them to are configurable per instance, so the reset cannot be
// recognised from the settings themselves.
func (e *external) Observe(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (managed.ExternalObservation, error) {
	if meta.WasDeleted(httpClient) {
		return managed.ExternalObservation{ResourceExists: !httpClient.Status.AtProvider.ResetToDefaults}, nil
	}

	observed, err := e.client.Get(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetHTTP)
	}

	fingerprint, err := credentialsFingerprint(ctx, e.kube, httpClient)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	appliedFingerprint := httpClient.Status.AtProvider.CredentialsFingerprint
	httpClient.Status.AtProvider = instanceclient.GenerateHTTPSettingsObservation(observed)
	httpClient.Status.AtProvider.CredentialsFingerprint = appliedFingerprint
	httpClient.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: fingerprint == appliedFingerprint && instanceclient.IsHTTPSettingsUpToDate(httpClient, observed),
	}, nil
}

// Create applies the HTTP client settings in Nexus. They are a singleton,
// so Create applies the spec like Update.
func (e *external) Create(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (managed.ExternalCreation, error) {
	fingerprint, err := e.applySettings(ctx, httpClient)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if fingerprint != "" {
		// Non-fatal: without a recorded fingerprint the next Observe reports
		// drift and Update re-applies the credentials.
		_ = nexus.PatchStatus(ctx, e.kube, httpClient, func(c *instancev1alpha1.HTTPClient) {
			c.Status.AtProvider.CredentialsFingerprint = fingerprint
		})
	}

	return managed.ExternalCreation{}, nil
}

// Update applies the HTTP client settings in Nexus.
func (e *external) Update(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (managed.ExternalUpdate, error) {
	fingerprint, err := e.applySettings(ctx, httpClient)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	httpClient.Status.AtProvider.CredentialsFingerprint = fingerprint

	return managed.ExternalUpdate{}, nil
}

// Delete resets the HTTP client settings to the Nexus defaults; they cannot
// be removed. The reconciler saves the status after a successful Delete, so
// the next Observe sees the recorded reset and reports the settings absent.
func (e *external) Delete(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (managed.ExternalDelete, error) {
	err := e.client.Reset(ctx)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errResetHTTP)
	}

	httpClient.Status.AtProvider.ResetToDefaults = true

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// applySettings resolves the proxy passwords and replaces the Nexus HTTP
// client settings. It returns the fingerprint of the credentials it applied.
func (e *external) applySettings(ctx context.Context, httpClient *instancev1alpha1.HTTPClient) (string, error) {
	// One read per Secret serves both the fingerprint and the passwords, so
	// the recorded fingerprint describes exactly the passwords applied.
	secrets := nexus.MemoizeSecrets(e.kube)

	fingerprint, err := credentialsFingerprint(ctx, secrets, httpClient)
	if err != nil {
		return "", err
	}

	passwords, err := resolvePasswords(ctx, secrets, httpClient)
	if err != nil {
		return "", err
	}

	// The update replaces every setting, so the ones the spec leaves unset
	// are carried over from Nexus.
	current, err := e.client.Get(ctx)
	if err != nil {
		return "", errors.Wrap(err, errGetHTTP)
	}

	settings := instanceclient.GenerateHTTPSettingsUpdate(&httpClient.Spec.ForProvider, current, passwords)

	err = e.client.Update(ctx, settings)
	if err != nil {
		return "", errors.Wrap(err, errUpdateHTTP)
	}

	return fingerprint, nil
}

// credentialsFingerprint fingerprints the proxy password Secrets the
// HTTPClient references, reading them through kube.
func credentialsFingerprint(ctx context.Context, kube client.Reader, httpClient *instancev1alpha1.HTTPClient) (string, error) {
	params := httpClient.Spec.ForProvider

	fingerprint, err := nexus.CredentialsFingerprint(ctx, kube, httpClient.GetNamespace(), nil,
		passwordSecretRef(params.HTTPProxy), passwordSecretRef(params.HTTPSProxy))

	return fingerprint, errors.Wrap(err, errFingerprint)
}

// resolvePasswords reads the proxy passwords from Kubernetes Secrets through
// kube.
func resolvePasswords(ctx context.Context, kube client.Reader, httpClient *instancev1alpha1.HTTPClient) (instanceclient.HTTPProxyPasswords, error) {
	var passwords instanceclient.HTTPProxyPasswords

	params := httpClient.Spec.ForProvider

	if ref := passwordSecretRef(params.HTTPProxy); ref != nil {
		password, err := nexus.GetSecretValue(ctx, kube, httpClient.GetNamespace(), ref)
		if err != nil {
			return passwords, errors.Wrap(err, errGetHTTPPassword)
		}

		passwords.HTTP = password
	}

	if ref := passwordSecretRef(params.HTTPSProxy); ref != nil {
		password, err := nexus.GetSecretValue(ctx, kube, httpClient.GetNamespace(), ref)
		if err != nil {
			return passwords, errors.Wrap(err, errGetHTTPSPassword)
		}

		passwords.HTTPS = password
	}

	return passwords, nil
}

// passwordSecretRef returns the password Secret reference of a proxy, or nil
// when the proxy or its authentication is unset.
func passwordSecretRef(proxy *instancev1alpha1.HTTPClientProxy) *xpv2.LocalSecretKeySelector {
	if proxy == nil || proxy.Authentication == nil {
		return nil
	}

	return &proxy.Authentication.PasswordSecretRef
}
