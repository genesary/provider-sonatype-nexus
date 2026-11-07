package instance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// licenseAPIPath is the path of the Nexus license endpoint, relative to the
// REST API base.
const licenseAPIPath = "/v1/system/license"

// ErrNoLicense is returned by GetLicense when no license is installed.
var ErrNoLicense = errors.New("no license installed on Nexus")

// LicenseInfo contains the license metadata returned by the Nexus API.
type LicenseInfo struct {
	ContactEmail   string
	ContactCompany string
	ContactName    string
	EffectiveDate  string
	ExpirationDate string
	LicenseType    string
	LicensedUsers  string
	Fingerprint    string
}

// LicenseClient manages the Nexus license via the REST API.
type LicenseClient interface {
	GetLicense(ctx context.Context) (*LicenseInfo, error)
	InstallLicense(ctx context.Context, licenseData []byte) error
	DeleteLicense(ctx context.Context) error
}

// licenseClient implements LicenseClient over the Nexus API.
type licenseClient struct {
	client *v395.APIClient
}

// NewLicenseClient creates a LicenseClient from Nexus credentials.
func NewLicenseClient(creds nexus.Credentials) (LicenseClient, error) {
	client, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &licenseClient{client: client}, nil
}

// GetLicense returns the installed Nexus license. It returns ErrNoLicense
// when no license is installed, which Nexus reports as 402 Payment Required,
// or as 404 on older versions.
func (c *licenseClient) GetLicense(ctx context.Context) (*LicenseInfo, error) {
	details, resp, err := c.client.ProductLicensingAPI.ListSystemLicense(ctx).Execute()
	if resp != nil && (resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusNotFound) {
		return nil, ErrNoLicense
	}

	if err != nil {
		return nil, errors.Wrap(nexus.APIError(resp, err), "cannot get license")
	}

	return &LicenseInfo{
		ContactEmail:   details.GetContactEmail(),
		ContactCompany: details.GetContactCompany(),
		ContactName:    details.GetContactName(),
		EffectiveDate:  formatLicenseDate(details.EffectiveDate),
		ExpirationDate: formatLicenseDate(details.ExpirationDate),
		LicenseType:    details.GetLicenseType(),
		LicensedUsers:  details.GetLicensedUsers(),
		Fingerprint:    details.GetFingerprint(),
	}, nil
}

// InstallLicense uploads and installs the given license data on Nexus.
//
// The request is built by hand rather than through the generated client,
// which models the upload as a JSON object, whereas Nexus expects the raw
// license file as an octet stream. It still goes through the generated
// client's configuration, so it shares its authentication, transport and
// limits.
func (c *licenseClient) InstallLicense(ctx context.Context, licenseData []byte) error {
	cfg := c.client.GetConfig()

	baseURL, err := cfg.ServerURLWithContext(ctx, "ProductLicensingAPIService.CreateSystemLicense")
	if err != nil {
		return errors.Wrap(err, "cannot resolve the Nexus API URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+licenseAPIPath, bytes.NewReader(licenseData))
	if err != nil {
		return errors.Wrap(err, "cannot build POST license request")
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "POST license request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errors.Errorf("POST license returned status %d: %s", resp.StatusCode, nexus.ErrorBody(resp.Body))
	}

	return nil
}

// DeleteLicense removes the installed Nexus license. A license that is
// already gone is not an error, whether Nexus reports it as 402 Payment
// Required or as 404, as GetLicense does.
func (c *licenseClient) DeleteLicense(ctx context.Context) error {
	resp, err := c.client.ProductLicensingAPI.DeleteSystemLicense(ctx).Execute()
	if resp != nil && resp.StatusCode == http.StatusPaymentRequired {
		return nil
	}

	return errors.Wrap(nexus.IgnoreNotFound(nexus.APIError(resp, err)), "cannot delete license")
}

// formatLicenseDate renders a license date as RFC 3339, or as an empty string
// when Nexus reports none.
func formatLicenseDate(date *time.Time) string {
	if date == nil {
		return ""
	}

	return date.Format(time.RFC3339)
}

// GenerateLicenseObservation builds an observation from Nexus license info.
// installedHash is preserved from the previous observation.
func GenerateLicenseObservation(info *LicenseInfo, installedHash string) instancev1alpha1.LicenseObservation {
	if info == nil {
		return instancev1alpha1.LicenseObservation{InstalledHash: installedHash}
	}

	return instancev1alpha1.LicenseObservation{
		Fingerprint:    info.Fingerprint,
		ContactEmail:   info.ContactEmail,
		ContactName:    info.ContactName,
		ContactCompany: info.ContactCompany,
		EffectiveDate:  info.EffectiveDate,
		ExpirationDate: info.ExpirationDate,
		LicenseType:    info.LicenseType,
		LicensedUsers:  info.LicensedUsers,
		InstalledHash:  installedHash,
	}
}

// IsLicenseUpToDate reports whether the installed license matches the desired.
// currentHash is the SHA-256 of the current desired license bytes.
func IsLicenseUpToDate(cr *instancev1alpha1.License, currentHash string) bool {
	obs := cr.Status.AtProvider

	return obs.Fingerprint != "" && obs.InstalledHash == currentHash
}

// HashLicense returns the SHA-256 hex-encoded hash of the license bytes.
func HashLicense(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
