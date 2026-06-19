// Package license manages License resources.
package license

import (
	"context"
	"io"
	"net/http"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// licenseSecretCacheKey is the data key used when caching the license in the
// writeConnectionSecretToRef secret.
const licenseSecretCacheKey = "license"

const (
	// errNotLicense means the managed resource is not a License.
	errNotLicense = "managed resource is not a License custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetLicense is returned when retrieving the license from Nexus fails.
	errGetLicense = "cannot get license from Nexus"
	// errInstallLicense is returned when installing the license on Nexus fails.
	errInstallLicense = "cannot install license on Nexus"
	// errDeleteLicense is returned when removing the license from Nexus fails.
	errDeleteLicense = "cannot delete license from Nexus"
	// errFetchLicense is returned when fetching the desired license bytes fails.
	errFetchLicense = "cannot fetch license bytes"
	// errNoLicenseSource is returned when neither source is configured.
	errNoLicenseSource = "neither licenseSecretRef nor endpointUrl is configured"
)

// Setup adds a controller that reconciles License resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.LicenseGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.LicenseGroupVersionKind),
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
		For(&iamv1alpha1.License{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isLicense := managedRes.(*iamv1alpha1.License)
	if !isLicense {
		return nil, errors.New(errNotLicense)
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

	return &external{
		client: iamclient.NewLicenseClient(creds),
		kube:   c.kube,
	}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.LicenseClient
	kube   client.Client
}

// Observe checks whether the license is installed and up to date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	licenseCR, isLicense := managedRes.(*iamv1alpha1.License)
	if !isLicense {
		return managed.ExternalObservation{}, errors.New(errNotLicense)
	}

	if licenseCR.GetDeletionTimestamp() != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	desiredBytes, err := e.fetchDesiredLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errFetchLicense)
	}

	currentHash := iamclient.HashLicense(desiredBytes)

	nexusInfo, err := e.client.GetLicense(ctx)
	if errors.Is(err, iamclient.ErrNoLicense) {
		prevHash := licenseCR.Status.AtProvider.InstalledHash
		licenseCR.Status.AtProvider = iamclient.GenerateLicenseObservation(nil, prevHash)
		licenseCR.SetConditions(nexusv1alpha1.Available())

		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLicense)
	}

	prevHash := licenseCR.Status.AtProvider.InstalledHash
	licenseCR.Status.AtProvider = iamclient.GenerateLicenseObservation(nexusInfo, prevHash)
	licenseCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsLicenseUpToDate(licenseCR, currentHash),
	}, nil
}

// Create installs the license on Nexus for the first time.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	licenseCR, isLicense := managedRes.(*iamv1alpha1.License)
	if !isLicense {
		return managed.ExternalCreation{}, errors.New(errNotLicense)
	}

	data, err := e.applyLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{licenseSecretCacheKey: data},
	}, nil
}

// Update reinstalls the license when it drifts from the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	licenseCR, isLicense := managedRes.(*iamv1alpha1.License)
	if !isLicense {
		return managed.ExternalUpdate{}, errors.New(errNotLicense)
	}

	data, err := e.applyLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{licenseSecretCacheKey: data},
	}, nil
}

// Delete removes the license from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	_, isLicense := managedRes.(*iamv1alpha1.License)
	if !isLicense {
		return managed.ExternalDelete{}, errors.New(errNotLicense)
	}

	err := e.client.DeleteLicense(ctx)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteLicense)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// applyLicense fetches the desired license, installs it on Nexus, records the
// hash in Status.AtProvider, and returns the raw bytes for ConnectionDetails.
func (e *external) applyLicense(ctx context.Context, licenseCR *iamv1alpha1.License) ([]byte, error) {
	data, err := e.fetchDesiredLicense(ctx, licenseCR)
	if err != nil {
		return nil, errors.Wrap(err, errFetchLicense)
	}

	err = e.client.InstallLicense(ctx, data)
	if err != nil {
		return nil, errors.Wrap(err, errInstallLicense)
	}

	hash := iamclient.HashLicense(data)

	before := licenseCR.DeepCopy()
	licenseCR.Status.AtProvider.InstalledHash = hash

	err = e.kube.Status().Patch(ctx, licenseCR, client.MergeFrom(before))
	if err != nil {
		// Non-fatal: next Observe will retry reconciliation.
		_ = err
	}

	return data, nil
}

// fetchDesiredLicense returns the license bytes from the configured source.
// Behavior 1: reads from LicenseSecretRef.
// Behavior 2: fetches from EndpointURL, caches in writeConnectionSecretToRef.
func (e *external) fetchDesiredLicense(ctx context.Context, licenseCR *iamv1alpha1.License) ([]byte, error) {
	params := licenseCR.Spec.ForProvider

	if params.LicenseSecretRef != nil {
		return nexus.GetSecretBytes(ctx, e.kube, params.LicenseSecretRef)
	}

	if params.EndpointURL != nil {
		return e.fetchFromEndpoint(ctx, licenseCR)
	}

	return nil, errors.New(errNoLicenseSource)
}

// fetchFromEndpoint fetches the license from the HTTP endpoint.
// Falls back to the writeConnectionSecretToRef cache when the endpoint is
// unavailable; the cache is written by the reconciler via ConnectionDetails.
func (e *external) fetchFromEndpoint(ctx context.Context, licenseCR *iamv1alpha1.License) ([]byte, error) {
	params := licenseCR.Spec.ForProvider

	data, endpointErr := e.doEndpointRequest(ctx, *params.EndpointURL, params.EndpointCredentials)
	if endpointErr == nil {
		return data, nil
	}

	// Endpoint unavailable: try the connection secret written by a prior reconcile.
	cacheRef := licenseCR.GetWriteConnectionSecretToReference()
	if cacheRef != nil {
		cached, cacheErr := nexus.GetLocalConnectionSecretBytes(ctx, e.kube, licenseCR, cacheRef, licenseSecretCacheKey)
		if cacheErr == nil {
			return cached, nil
		}
	}

	return nil, errors.Wrap(endpointErr, "endpoint unavailable and no cached license")
}

// doEndpointRequest performs the HTTP GET for the license file.
func (e *external) doEndpointRequest(ctx context.Context, url string, creds *iamv1alpha1.LicenseEndpointCredentials) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build endpoint request")
	}

	err = e.applyEndpointCredentials(ctx, req, creds)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "endpoint request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("endpoint returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read endpoint response")
	}

	return data, nil
}

// applyEndpointCredentials sets HTTP auth headers on the request.
func (e *external) applyEndpointCredentials(ctx context.Context, req *http.Request, creds *iamv1alpha1.LicenseEndpointCredentials) error {
	if creds == nil {
		return nil
	}

	if creds.UsernameSecretRef != nil && creds.PasswordSecretRef != nil {
		username, err := nexus.GetSecretValue(ctx, e.kube, creds.UsernameSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get endpoint username")
		}

		password, err := nexus.GetSecretValue(ctx, e.kube, creds.PasswordSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get endpoint password")
		}

		req.SetBasicAuth(username, password)

		return nil
	}

	if creds.PasswordSecretRef != nil {
		token, err := nexus.GetSecretValue(ctx, e.kube, creds.PasswordSecretRef)
		if err != nil {
			return errors.Wrap(err, "cannot get endpoint token")
		}

		req.Header.Set("Authorization", "Bearer "+token)
	}

	return nil
}
