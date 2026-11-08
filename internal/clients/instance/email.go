package instance

import (
	"context"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// ErrNoEmailConfiguration is returned when Nexus answers without an email
// server configuration.
var ErrNoEmailConfiguration = errors.New("nexus returned no email server configuration")

// EmailClient provides methods for the Nexus email server configuration.
type EmailClient interface {
	Get(ctx context.Context) (*v395.ApiEmailConfiguration, error)
	Update(ctx context.Context, config v395.ApiEmailConfiguration) error
	Reset(ctx context.Context) error
}

// emailClient implements EmailClient over the Nexus API.
type emailClient struct {
	api *v395.EmailAPIService
}

// NewEmailClient creates an EmailClient for a live Nexus connection.
func NewEmailClient(creds nexus.Credentials) (EmailClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &emailClient{api: nc.EmailAPI}, nil
}

// Get returns the email server configuration. Nexus never returns the
// password.
func (c *emailClient) Get(ctx context.Context) (*v395.ApiEmailConfiguration, error) {
	config, resp, err := c.api.ListEmail(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	if config == nil {
		return nil, ErrNoEmailConfiguration
	}

	return config, nil
}

// Update replaces the email server configuration. Nexus resolves the host
// first and refuses one it cannot resolve, or one SSRF protection blocks.
func (c *emailClient) Update(ctx context.Context, config v395.ApiEmailConfiguration) error {
	return nexus.APIError(c.api.UpdateEmail(ctx).ApiEmailConfiguration(config).Execute())
}

// Reset clears the email server configuration, which disables email.
func (c *emailClient) Reset(ctx context.Context) error {
	return nexus.APIError(c.api.DeleteEmail(ctx).Execute())
}

// IsResetEmailConfiguration reports whether observed is the configuration
// Nexus holds once it has been reset: disabled, without a host or a sender
// address. A fresh Nexus instance does not start in this state; it holds a
// disabled placeholder configuration for localhost instead.
func IsResetEmailConfiguration(observed *v395.ApiEmailConfiguration) bool {
	return !observed.GetEnabled() && observed.Host == "" && observed.FromAddress == ""
}

// GenerateEmailConfigurationUpdate builds the email server configuration
// from the CR spec and the resolved password, empty when the spec references
// none.
//
// An empty password is sent as no password at all. Nexus does not keep the
// stored password then: a PUT without one clears it, so removing
// passwordSecretRef or emptying the Secret value takes effect (verified on
// Nexus 3.96, which treats an omitted, null, empty or blank password alike).
func GenerateEmailConfigurationUpdate(params *instancev1alpha1.EmailConfigurationParameters, password string) v395.ApiEmailConfiguration {
	config := v395.ApiEmailConfiguration{
		Enabled:                       new(ptr.Deref(params.Enabled, true)),
		Host:                          params.Host,
		Port:                          helpers.ClampInt32(params.Port),
		Username:                      params.Username,
		FromAddress:                   params.FromAddress,
		SubjectPrefix:                 params.SubjectPrefix,
		StartTlsEnabled:               new(ptr.Deref(params.StartTLSEnabled, false)),
		StartTlsRequired:              new(ptr.Deref(params.StartTLSRequired, false)),
		SslOnConnectEnabled:           new(ptr.Deref(params.SSLOnConnectEnabled, false)),
		SslServerIdentityCheckEnabled: new(ptr.Deref(params.SSLServerIdentityCheckEnabled, true)),
		NexusTrustStoreEnabled:        new(ptr.Deref(params.NexusTrustStoreEnabled, false)),
	}

	if password != "" {
		config.Password = &password
	}

	return config
}

// IsEmailConfigurationUpToDate returns true when the Nexus email server
// configuration matches the CR spec. The password is not compared: Nexus
// never returns it, so it is tracked by the credentials fingerprint. Nexus
// reports an unset user name or subject prefix as either null or an empty
// string, so both count as unset.
func IsEmailConfigurationUpToDate(cr *instancev1alpha1.EmailConfiguration, observed *v395.ApiEmailConfiguration) bool {
	// The desired configuration goes through the same conversion as the
	// observed one, so the spec defaults are stated once, in
	// GenerateEmailConfigurationUpdate.
	desired := GenerateEmailConfigurationUpdate(&cr.Spec.ForProvider, "")

	return GenerateEmailConfigurationObservation(&desired) == GenerateEmailConfigurationObservation(observed)
}

// GenerateEmailConfigurationObservation converts the email server
// configuration to an observation.
func GenerateEmailConfigurationObservation(observed *v395.ApiEmailConfiguration) instancev1alpha1.EmailConfigurationObservation {
	if observed == nil {
		return instancev1alpha1.EmailConfigurationObservation{}
	}

	return instancev1alpha1.EmailConfigurationObservation{
		Enabled:                       observed.GetEnabled(),
		Host:                          observed.Host,
		Port:                          int(observed.Port),
		Username:                      observed.GetUsername(),
		FromAddress:                   observed.FromAddress,
		SubjectPrefix:                 observed.GetSubjectPrefix(),
		StartTLSEnabled:               observed.GetStartTlsEnabled(),
		StartTLSRequired:              observed.GetStartTlsRequired(),
		SSLOnConnectEnabled:           observed.GetSslOnConnectEnabled(),
		SSLServerIdentityCheckEnabled: observed.GetSslServerIdentityCheckEnabled(),
		NexusTrustStoreEnabled:        observed.GetNexusTrustStoreEnabled(),
	}
}
