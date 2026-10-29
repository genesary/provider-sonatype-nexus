package instance

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// IQServerClient provides methods for the Nexus IQ Server config.
type IQServerClient interface {
	Get(ctx context.Context) (*v395.IqConnectionXo, error)
	Update(ctx context.Context, config v395.IqConnectionXo) error
	Disable(ctx context.Context) error
}

// iqServerClient implements IQServerClient over the Nexus API.
type iqServerClient struct {
	api *v395.ManageSonatypeRepositoryFirewallConfigurationAPIService
}

// NewIQServerClient creates an IQServerClient for a live Nexus connection.
func NewIQServerClient(creds nexus.Credentials) (IQServerClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &iqServerClient{api: nc.ManageSonatypeRepositoryFirewallConfigurationAPI}, nil
}

// Get returns the IQ Server connection.
func (c *iqServerClient) Get(ctx context.Context) (*v395.IqConnectionXo, error) {
	config, resp, err := c.api.ListIq(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return config, nil
}

// Update replaces the IQ Server connection.
func (c *iqServerClient) Update(ctx context.Context, config v395.IqConnectionXo) error {
	_, resp, err := c.api.UpdateIq(ctx).IqConnectionXo(config).Execute()

	return nexus.APIError(resp, err)
}

// Disable disables the IQ Server connection.
func (c *iqServerClient) Disable(ctx context.Context) error {
	return nexus.APIError(c.api.CreateIqDisable(ctx).Execute())
}

// GenerateIQServerUpdate builds an IQServerConfiguration from the CR spec.
func GenerateIQServerUpdate(params *instancev1alpha1.IQServerConfigurationParameters, username, password string) v395.IqConnectionXo {
	config := v395.IqConnectionXo{
		Enabled:             new(ptr.Deref(params.Enabled, true)),
		ShowLink:            new(ptr.Deref(params.ShowLink, false)),
		Url:                 params.URL,
		AuthenticationType:  ptr.Deref(params.AuthenticationType, ""),
		UseTrustStoreForUrl: new(ptr.Deref(params.UseTrustStoreForURL, false)),
		TimeoutSeconds:      toInt32Ptr(params.TimeoutSeconds),
		Properties:          params.Properties,
		FailOpenModeEnabled: new(ptr.Deref(params.FailOpenModeEnabled, false)),
	}

	if username != "" {
		config.Username = &username
	}

	if password != "" {
		config.Password = &password
	}

	return config
}

// IsIQServerUpToDate returns true when Nexus IQ Server matches CR spec.
func IsIQServerUpToDate(cr *instancev1alpha1.IQServerConfiguration, observed *v395.IqConnectionXo) bool {
	params := cr.Spec.ForProvider

	if ptr.Deref(params.Enabled, true) != observed.GetEnabled() {
		return false
	}

	if ptr.Deref(params.ShowLink, false) != observed.GetShowLink() {
		return false
	}

	// The spec URL is required, so it is compared against whatever Nexus
	// reports - including nothing at all.
	if observed.Url != params.URL {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(params.AuthenticationType, observed.AuthenticationType) {
		return false
	}

	if ptr.Deref(params.UseTrustStoreForURL, false) != observed.GetUseTrustStoreForUrl() {
		return false
	}

	if !helpers.IsComparablePtrEqualComparablePtr(toInt32Ptr(params.TimeoutSeconds), observed.TimeoutSeconds) {
		return false
	}

	if !helpers.IsComparablePtrEqualComparablePtr(params.Properties, observed.Properties) {
		return false
	}

	if ptr.Deref(params.FailOpenModeEnabled, false) != observed.GetFailOpenModeEnabled() {
		return false
	}

	return true
}

// GenerateIQServerObservation converts an IQ Server config to observation.
func GenerateIQServerObservation(observed *v395.IqConnectionXo) instancev1alpha1.IQServerConfigurationObservation {
	if observed == nil {
		return instancev1alpha1.IQServerConfigurationObservation{}
	}

	return instancev1alpha1.IQServerConfigurationObservation{
		Enabled:             observed.GetEnabled(),
		ShowLink:            observed.GetShowLink(),
		URL:                 observed.Url,
		AuthenticationType:  observed.AuthenticationType,
		UseTrustStoreForURL: observed.GetUseTrustStoreForUrl(),
		TimeoutSeconds:      int(observed.GetTimeoutSeconds()),
		Properties:          observed.GetProperties(),
		FailOpenModeEnabled: observed.GetFailOpenModeEnabled(),
	}
}

// toInt32Ptr narrows an optional int, keeping nil as nil.
func toInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}

	return new(helpers.ClampInt32(*value))
}
