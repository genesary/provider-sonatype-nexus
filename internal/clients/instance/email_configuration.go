// Package instance provides clients for instance-level Nexus resources.
package instance

import (
	"context"

	mailschema "github.com/datadrivers/go-nexus-client/nexus3/schema"
	"github.com/pkg/errors"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// EmailConfigurationClient manages Nexus SMTP email configuration.
type EmailConfigurationClient interface {
	GetEmailConfiguration(ctx context.Context) (*mailschema.MailConfig, error)
	UpdateEmailConfiguration(ctx context.Context, config mailschema.MailConfig) error
}

// NewEmailConfigurationClient returns a new EmailConfigurationClient.
func NewEmailConfigurationClient(creds nexus.Credentials) (EmailConfigurationClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.MailConfig(), nil
}

// GenerateEmailConfiguration converts an EmailConfiguration CR spec into the
// Nexus MailConfig API type. The password is resolved externally and passed in.
func GenerateEmailConfiguration(cr *instancev1alpha1.EmailConfiguration, password string) mailschema.MailConfig {
	spec := cr.Spec.ForProvider

	cfg := mailschema.MailConfig{
		Enabled:                       &spec.Enabled,
		Host:                          spec.Host,
		Port:                          spec.Port,
		FromAddress:                   spec.FromAddress,
		Username:                      spec.Username,
		SubjectPrefix:                 spec.SubjectPrefix,
		StartTlsEnabled:               spec.StartTlsEnabled,
		StartTlsRequired:              spec.StartTlsRequired,
		SslOnConnectEnabled:           spec.SslOnConnectEnabled,
		SslServerIdentityCheckEnabled: spec.SslServerIdentityCheckEnabled,
		NexusTrustStoreEnabled:        spec.NexusTrustStoreEnabled,
	}

	if password != "" {
		cfg.Password = &password
	}

	return cfg
}

// GenerateEmailConfigurationObservation converts Nexus API state into the
// observation type stored in the CR status.
func GenerateEmailConfigurationObservation(config *mailschema.MailConfig) instancev1alpha1.EmailConfigurationObservation {
	if config == nil {
		return instancev1alpha1.EmailConfigurationObservation{}
	}

	obs := instancev1alpha1.EmailConfigurationObservation{
		Host:        config.Host,
		Port:        config.Port,
		FromAddress: config.FromAddress,
	}

	if config.Enabled != nil {
		obs.Enabled = *config.Enabled
	}

	if config.Username != nil {
		obs.Username = *config.Username
	}

	if config.SubjectPrefix != nil {
		obs.SubjectPrefix = *config.SubjectPrefix
	}

	if config.StartTlsEnabled != nil {
		obs.StartTlsEnabled = *config.StartTlsEnabled
	}

	if config.StartTlsRequired != nil {
		obs.StartTlsRequired = *config.StartTlsRequired
	}

	if config.SslOnConnectEnabled != nil {
		obs.SslOnConnectEnabled = *config.SslOnConnectEnabled
	}

	if config.SslServerIdentityCheckEnabled != nil {
		obs.SslServerIdentityCheckEnabled = *config.SslServerIdentityCheckEnabled
	}

	if config.NexusTrustStoreEnabled != nil {
		obs.NexusTrustStoreEnabled = *config.NexusTrustStoreEnabled
	}

	return obs
}

// IsEmailConfigurationUpToDate reports whether the CR spec matches the
// observed state. The password is never returned by the Nexus API, so it is
// excluded from drift detection.
func IsEmailConfigurationUpToDate(cr *instancev1alpha1.EmailConfiguration) bool {
	obs := cr.Status.AtProvider
	spec := cr.Spec.ForProvider

	if !scalarsUpToDate(spec, obs) {
		return false
	}

	if !boolPtrMatchesObs(spec.StartTlsEnabled, obs.StartTlsEnabled) ||
		!boolPtrMatchesObs(spec.StartTlsRequired, obs.StartTlsRequired) ||
		!boolPtrMatchesObs(spec.SslOnConnectEnabled, obs.SslOnConnectEnabled) ||
		!boolPtrMatchesObs(spec.SslServerIdentityCheckEnabled, obs.SslServerIdentityCheckEnabled) ||
		!boolPtrMatchesObs(spec.NexusTrustStoreEnabled, obs.NexusTrustStoreEnabled) {
		return false
	}

	return strPtrMatchesObs(spec.Username, obs.Username) &&
		strPtrMatchesObs(spec.SubjectPrefix, obs.SubjectPrefix)
}

// scalarsUpToDate reports whether the non-pointer scalar fields match.
func scalarsUpToDate(
	spec instancev1alpha1.EmailConfigurationParameters,
	obs instancev1alpha1.EmailConfigurationObservation,
) bool {
	return obs.Enabled == spec.Enabled &&
		obs.Host == spec.Host &&
		obs.Port == spec.Port &&
		obs.FromAddress == spec.FromAddress
}

// boolPtrMatchesObs returns true when desired is nil or equals observed.
func boolPtrMatchesObs(desired *bool, observed bool) bool {
	return desired == nil || *desired == observed
}

// strPtrMatchesObs returns true when desired is nil or equals observed.
func strPtrMatchesObs(desired *string, observed string) bool {
	return desired == nil || *desired == observed
}
