package instance

import (
	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// IQServerClient provides methods for the Nexus IQ Server config.
type IQServerClient interface {
	Get() (*nexussdk.IQServerConfiguration, error)
	Update(config nexussdk.IQServerConfiguration) error
	Disable() error
}

// NewIQServerClient creates an IQServerClient for a live Nexus connection.
func NewIQServerClient(creds nexus.Credentials) (IQServerClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.IQServer, nil
}

// GenerateIQServerUpdate builds an IQServerConfiguration from the CR spec.
func GenerateIQServerUpdate(params *instancev1alpha1.IQServerConfigurationParameters, username, password string) nexussdk.IQServerConfiguration {
	url := params.URL
	config := nexussdk.IQServerConfiguration{
		Enabled:             ptr.Deref(params.Enabled, true),
		ShowLink:            ptr.Deref(params.ShowLink, false),
		URL:                 &url,
		AuthenticationType:  params.AuthenticationType,
		UseTrustStoreForURL: ptr.Deref(params.UseTrustStoreForURL, false),
		TimeoutSeconds:      params.TimeoutSeconds,
		Properties:          params.Properties,
		FailOpenModeEnabled: ptr.Deref(params.FailOpenModeEnabled, false),
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
func IsIQServerUpToDate(cr *instancev1alpha1.IQServerConfiguration, observed *nexussdk.IQServerConfiguration) bool {
	params := cr.Spec.ForProvider

	if ptr.Deref(params.Enabled, true) != observed.Enabled {
		return false
	}

	if ptr.Deref(params.ShowLink, false) != observed.ShowLink {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(observed.URL, params.URL) {
		return false
	}

	if !helpers.IsComparablePtrEqualComparablePtr(params.AuthenticationType, observed.AuthenticationType) {
		return false
	}

	if ptr.Deref(params.UseTrustStoreForURL, false) != observed.UseTrustStoreForURL {
		return false
	}

	if !helpers.IsComparablePtrEqualComparablePtr(params.TimeoutSeconds, observed.TimeoutSeconds) {
		return false
	}

	if !helpers.IsComparablePtrEqualComparablePtr(params.Properties, observed.Properties) {
		return false
	}

	if ptr.Deref(params.FailOpenModeEnabled, false) != observed.FailOpenModeEnabled {
		return false
	}

	return true
}

// GenerateIQServerObservation converts an IQ Server config to observation.
func GenerateIQServerObservation(observed *nexussdk.IQServerConfiguration) instancev1alpha1.IQServerConfigurationObservation {
	if observed == nil {
		return instancev1alpha1.IQServerConfigurationObservation{}
	}

	return instancev1alpha1.IQServerConfigurationObservation{
		Enabled:             observed.Enabled,
		ShowLink:            observed.ShowLink,
		URL:                 ptr.Deref(observed.URL, ""),
		AuthenticationType:  ptr.Deref(observed.AuthenticationType, ""),
		UseTrustStoreForURL: observed.UseTrustStoreForURL,
		TimeoutSeconds:      ptr.Deref(observed.TimeoutSeconds, 0),
		Properties:          ptr.Deref(observed.Properties, ""),
		FailOpenModeEnabled: observed.FailOpenModeEnabled,
	}
}
