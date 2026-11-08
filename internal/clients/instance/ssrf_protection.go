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

// ErrNoSSRFProtection is returned when Nexus answers without SSRF protection
// settings.
var ErrNoSSRFProtection = errors.New("nexus returned no SSRF protection settings")

// SSRFProtectionClient provides methods for the Nexus SSRF protection
// settings.
type SSRFProtectionClient interface {
	Get(ctx context.Context) (*v395.SsrfProtectionConfigurationXO, error)
	Update(ctx context.Context, config v395.SsrfProtectionConfigurationXO) error
}

// ssrfProtectionClient implements SSRFProtectionClient over the Nexus API.
type ssrfProtectionClient struct {
	api *v395.SecurityManagementSSRFProtectionAPIService
}

// NewSSRFProtectionClient creates an SSRFProtectionClient for a live Nexus
// connection.
func NewSSRFProtectionClient(creds nexus.Credentials) (SSRFProtectionClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &ssrfProtectionClient{api: nc.SecurityManagementSSRFProtectionAPI}, nil
}

// Get returns the SSRF protection settings.
func (c *ssrfProtectionClient) Get(ctx context.Context) (*v395.SsrfProtectionConfigurationXO, error) {
	config, resp, err := c.api.ListSecuritySsrfProtection(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	if config == nil {
		return nil, ErrNoSSRFProtection
	}

	return config, nil
}

// Update replaces the SSRF protection settings. Nexus drops any exception the
// update leaves out.
func (c *ssrfProtectionClient) Update(ctx context.Context, config v395.SsrfProtectionConfigurationXO) error {
	_, resp, err := c.api.UpdateSecuritySsrfProtection(ctx).SsrfProtectionConfigurationXO(config).Execute()

	return nexus.APIError(resp, err)
}

// DefaultSSRFProtection is the Nexus default SSRF protection configuration:
// protection enabled, with no exceptions.
func DefaultSSRFProtection() v395.SsrfProtectionConfigurationXO {
	return v395.SsrfProtectionConfigurationXO{Enabled: true}
}

// IsDefaultSSRFProtection reports whether observed is the Nexus default SSRF
// protection configuration.
func IsDefaultSSRFProtection(observed *v395.SsrfProtectionConfigurationXO) bool {
	return observed.Enabled && len(observed.AllowedIPs) == 0 && len(observed.AllowedDomains) == 0
}

// GenerateSSRFProtectionUpdate builds the SSRF protection configuration from
// the CR spec.
func GenerateSSRFProtectionUpdate(params *instancev1alpha1.SSRFProtectionParameters) v395.SsrfProtectionConfigurationXO {
	return v395.SsrfProtectionConfigurationXO{
		Enabled:        ptr.Deref(params.Enabled, true),
		AllowedIPs:     params.AllowedIPs,
		AllowedDomains: params.AllowedDomains,
	}
}

// IsSSRFProtectionUpToDate returns true when the Nexus SSRF protection
// settings match the CR spec. Nexus trims, de-duplicates and reorders the
// exceptions, so they are compared as sets.
func IsSSRFProtectionUpToDate(cr *instancev1alpha1.SSRFProtection, observed *v395.SsrfProtectionConfigurationXO) bool {
	params := cr.Spec.ForProvider

	if ptr.Deref(params.Enabled, true) != observed.Enabled {
		return false
	}

	if !helpers.AreStringSlicesEqualDeDuped(params.AllowedIPs, observed.AllowedIPs) {
		return false
	}

	return helpers.AreStringSlicesEqualDeDuped(params.AllowedDomains, observed.AllowedDomains)
}

// GenerateSSRFProtectionObservation converts the SSRF protection settings to
// an observation.
func GenerateSSRFProtectionObservation(observed *v395.SsrfProtectionConfigurationXO) instancev1alpha1.SSRFProtectionObservation {
	if observed == nil {
		return instancev1alpha1.SSRFProtectionObservation{}
	}

	return instancev1alpha1.SSRFProtectionObservation{
		Enabled:        observed.Enabled,
		AllowedIPs:     observed.AllowedIPs,
		AllowedDomains: observed.AllowedDomains,
	}
}
