package instance

import (
	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

type IQServerClient interface {
	Get() (*nexussdk.IQServerConfiguration, error)
	Update(config nexussdk.IQServerConfiguration) error
	Disable() error
	Enable() error
	VerifyConnection() error
}

func NewIQServerClient(creds nexus.Credentials) (IQServerClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.IQServer, nil
}

func GenerateIQServerUpdate(params *instancev1alpha1.IQServerConfigurationParameters, password string) nexussdk.IQServerConfiguration {
	var passwd *string
	if password != "" {
		passwd = ptr.To(password)
	}

	return nexussdk.IQServerConfiguration{
		Enabled:             params.Enabled,
		ShowLink:            params.ShowLink,
		URL:                 params.URL,
		AuthenticationType:  params.AuthenticationMethod,
		Username:            params.Username,
		Password:            passwd,
		UseTrustStoreForURL: params.UseTrustStoreForURL,
		TimeoutSeconds:      params.TimeoutSeconds,
		Properties:          params.Properties,
		FailOpenModeEnabled: params.FailOpenModeEnabled,
	}
}

func IsIQServerUpToDate(cr *instancev1alpha1.IQServerConfiguration, observed *nexussdk.IQServerConfiguration) bool {
	params := cr.Spec.ForProvider

	if params.Enabled != observed.Enabled {
		return false
	}

	if params.ShowLink != observed.ShowLink {
		return false
	}

	if !ptrStringEqual(params.URL, observed.URL) {
		return false
	}

	if !ptrStringEqual(params.AuthenticationMethod, observed.AuthenticationType) {
		return false
	}

	if !ptrStringEqual(params.Username, observed.Username) {
		return false
	}

	if params.UseTrustStoreForURL != observed.UseTrustStoreForURL {
		return false
	}

	if !ptrIntEqual(params.TimeoutSeconds, observed.TimeoutSeconds) {
		return false
	}

	if !ptrStringEqual(params.Properties, observed.Properties) {
		return false
	}

	if params.FailOpenModeEnabled != observed.FailOpenModeEnabled {
		return false
	}

	return true
}

func GenerateIQServerObservation(observed *nexussdk.IQServerConfiguration) instancev1alpha1.IQServerConfigurationObservation {
	if observed == nil {
		return instancev1alpha1.IQServerConfigurationObservation{}
	}

	return instancev1alpha1.IQServerConfigurationObservation{
		Enabled:              observed.Enabled,
		ShowLink:             observed.ShowLink,
		URL:                  observed.URL,
		AuthenticationMethod: observed.AuthenticationType,
		Username:             observed.Username,
		UseTrustStoreForURL:  observed.UseTrustStoreForURL,
		TimeoutSeconds:       observed.TimeoutSeconds,
		Properties:           observed.Properties,
		FailOpenModeEnabled:  observed.FailOpenModeEnabled,
	}
}

func ptrStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return *a == *b
}

func ptrIntEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return *a == *b
}
