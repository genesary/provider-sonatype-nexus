package instance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// licenseAPIPath is the Nexus REST API path for license management.
const licenseAPIPath = "/service/rest/v1/system/license"

// ErrNoLicense is returned by GetLicense when no license is installed.
var ErrNoLicense = errors.New("no license installed on Nexus")

// LicenseInfo contains the license metadata returned by the Nexus API.
type LicenseInfo struct {
	ContactEmail   string `json:"contactEmail"`
	ContactCompany string `json:"contactCompany"`
	ContactName    string `json:"contactName"`
	EffectiveDate  string `json:"effectiveDate"`
	ExpirationDate string `json:"expirationDate"`
	LicenseType    string `json:"licenseType"`
	LicensedUsers  string `json:"licensedUsers"`
	Fingerprint    string `json:"fingerprint"`
}

// LicenseClient manages the Nexus license via the REST API.
type LicenseClient interface {
	GetLicense(ctx context.Context) (*LicenseInfo, error)
	InstallLicense(ctx context.Context, licenseData []byte) error
	DeleteLicense(ctx context.Context) error
}

// licenseHTTPClient implements LicenseClient using raw HTTP calls.
type licenseHTTPClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewLicenseClient creates a LicenseClient from Nexus credentials.
func NewLicenseClient(creds nexus.Credentials) LicenseClient {
	transport := http.DefaultTransport

	if creds.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // user opt-in
		}
	}

	return &licenseHTTPClient{
		baseURL:    creds.URL,
		username:   creds.Username,
		password:   creds.Password,
		httpClient: &http.Client{Transport: transport},
	}
}

// GetLicense returns the installed Nexus license.
// Returns ErrNoLicense when no license is installed.
func (c *licenseHTTPClient) GetLicense(ctx context.Context) (*LicenseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+licenseAPIPath, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "cannot build GET license request")
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "GET license request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoLicense
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET license returned status %d", resp.StatusCode)
	}

	var info LicenseInfo

	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode license response")
	}

	return &info, nil
}

// InstallLicense uploads and installs the given license data on Nexus.
func (c *licenseHTTPClient) InstallLicense(ctx context.Context, licenseData []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+licenseAPIPath, bytes.NewReader(licenseData))
	if err != nil {
		return errors.Wrap(err, "cannot build POST license request")
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "POST license request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)

		return errors.Errorf("POST license returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteLicense removes the installed Nexus license.
func (c *licenseHTTPClient) DeleteLicense(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+licenseAPIPath, http.NoBody)
	if err != nil {
		return errors.Wrap(err, "cannot build DELETE license request")
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "DELETE license request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable here

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusNotFound {
		return errors.Errorf("DELETE license returned status %d", resp.StatusCode)
	}

	return nil
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
