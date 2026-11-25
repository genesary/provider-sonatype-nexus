// Package license manages License resources.
package license

import (
	"context"

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
	"github.com/genesary/provider-sonatype-nexus/internal/document"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// licenseSecretCacheKey is the data key used when caching the license in the
// writeConnectionSecretToRef secret.
const licenseSecretCacheKey = "license"

const (
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetLicense is returned when retrieving the license from Nexus fails.
	errGetLicense = "cannot get license from Nexus"
	// errInstallLicense is returned when installing the license on Nexus fails.
	errInstallLicense = "cannot install license on Nexus"
	// errDeleteLicense is returned when removing the license from Nexus fails.
	errDeleteLicense = "cannot delete license from Nexus"
	// errFetchLicense is returned when fetching the desired license bytes fails.
	errFetchLicense = "cannot fetch license bytes"
)

// Setup adds a controller that reconciles License resources.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	name := managed.ControllerName(instancev1alpha1.LicenseGroupKind)

	reconcilerOpts := helpers.ReconcilerOptions(mgr, name, opts,
		managed.WithTypedExternalConnector[*instancev1alpha1.License](&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
	)

	reconciler := managed.NewReconciler(mgr, resource.ManagedKind(instancev1alpha1.LicenseGroupVersionKind), reconcilerOpts...)

	err := helpers.AddMRStateRecorder(mgr, opts, &instancev1alpha1.LicenseList{})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&instancev1alpha1.License{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.TypedExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces a TypedExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.TypedExternalClient[*instancev1alpha1.License], error) {
	err := c.usage.Track(ctx, licenseCR)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	creds, err := nexus.GetCredentials(ctx, c.kube, licenseCR)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	licenseClient, err := instanceclient.NewLicenseClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		client: licenseClient,
		kube:   c.kube,
	}, nil
}

// external implements managed.TypedExternalClient.
type external struct {
	client instanceclient.LicenseClient
	kube   client.Client
}

// Observe checks whether the license is installed and up to date.
//
// While the License is being deleted, Observe only reports whether Nexus has a
// license installed, so the reconciler calls Delete when there is one. It does
// not read the desired license then: its Secret or endpoint may already be
// gone, and an error would block the finalizer.
func (e *external) Observe(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.ExternalObservation, error) {
	if licenseCR.GetDeletionTimestamp() != nil {
		return e.observeInstalled(ctx, licenseCR)
	}

	license, err := e.readLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	currentHash := instanceclient.HashLicense(license.Data)

	observation, err := e.observeInstalled(ctx, licenseCR)
	if err != nil || !observation.ResourceExists {
		return observation, err
	}

	observation.ResourceUpToDate = instanceclient.IsLicenseUpToDate(licenseCR, currentHash)
	observation.ConnectionDetails = license.ConnectionDetails

	return observation, nil
}

// Create installs the license on Nexus for the first time.
func (e *external) Create(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.ExternalCreation, error) {
	details, err := e.applyLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{ConnectionDetails: details}, nil
}

// Update reinstalls the license when it drifts from the desired state.
func (e *external) Update(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.ExternalUpdate, error) {
	details, err := e.applyLicense(ctx, licenseCR)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: details}, nil
}

// Delete removes the license from Nexus.
func (e *external) Delete(ctx context.Context, _ *instancev1alpha1.License) (managed.ExternalDelete, error) {
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

// observeInstalled reports whether Nexus has a license installed and records
// it in Status.AtProvider. It marks the License Available only when one is.
func (e *external) observeInstalled(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.ExternalObservation, error) {
	prevHash := licenseCR.Status.AtProvider.InstalledHash

	nexusInfo, err := e.client.GetLicense(ctx)
	if errors.Is(err, instanceclient.ErrNoLicense) {
		licenseCR.Status.AtProvider = instanceclient.GenerateLicenseObservation(nil, prevHash)

		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetLicense)
	}

	licenseCR.Status.AtProvider = instanceclient.GenerateLicenseObservation(nexusInfo, prevHash)
	licenseCR.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

// applyLicense reads the desired license, installs it on Nexus, records the
// hash in Status.AtProvider, and returns the ConnectionDetails that cache it.
func (e *external) applyLicense(ctx context.Context, licenseCR *instancev1alpha1.License) (managed.ConnectionDetails, error) {
	license, err := e.readLicense(ctx, licenseCR)
	if err != nil {
		return nil, err
	}

	err = e.client.InstallLicense(ctx, license.Data)
	if err != nil {
		return nil, errors.Wrap(err, errInstallLicense)
	}

	hash := instanceclient.HashLicense(license.Data)

	before := licenseCR.DeepCopy()
	licenseCR.Status.AtProvider.InstalledHash = hash

	// Non-fatal: next Observe will retry reconciliation.
	_ = e.kube.Status().Patch(ctx, licenseCR, client.MergeFrom(before))

	return license.ConnectionDetails, nil
}

// readLicense reads the desired license from licenseSecretRef or endpointUrl.
func (e *external) readLicense(ctx context.Context, licenseCR *instancev1alpha1.License) (document.Document, error) {
	params := licenseCR.Spec.ForProvider

	source := document.Source{
		From: &nexusv1alpha1.DocumentSource{
			SecretRef:           params.LicenseSecretRef,
			EndpointURL:         params.EndpointURL,
			EndpointCredentials: params.EndpointCredentials,
		},
		Key: licenseSecretCacheKey,
	}

	license, err := source.Read(ctx, e.kube, licenseCR)

	return license, errors.Wrap(err, errFetchLicense)
}
