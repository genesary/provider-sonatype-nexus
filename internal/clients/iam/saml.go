package iam

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"io"
	"reflect"
	"unicode/utf8"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// samlMetadataNamespace is the XML namespace of SAML 2.0 metadata.
const samlMetadataNamespace = "urn:oasis:names:tc:SAML:2.0:metadata"

// SAMLClient manages Nexus SAML SSO configuration.
type SAMLClient interface {
	Read(ctx context.Context) (*v395.SamlConfigurationXO, error)
	Apply(ctx context.Context, saml v395.SamlConfigurationXO) error
	Delete(ctx context.Context) error
}

// samlClient implements SAMLClient over the Nexus API.
type samlClient struct {
	api *v395.SecurityManagementSAMLAPIService
}

// NewSAMLClient returns a new SAMLClient.
func NewSAMLClient(creds nexus.Credentials) (SAMLClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &samlClient{api: nc.SecurityManagementSAMLAPI}, nil
}

// Read returns the SAML configuration.
func (c *samlClient) Read(ctx context.Context) (*v395.SamlConfigurationXO, error) {
	saml, resp, err := c.api.ListSecuritySaml(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return saml, nil
}

// Apply creates or replaces the SAML configuration.
func (c *samlClient) Apply(ctx context.Context, saml v395.SamlConfigurationXO) error {
	return nexus.APIError(c.api.UpdateSecuritySaml(ctx).SamlConfigurationXO(saml).Execute())
}

// Delete removes the SAML configuration.
func (c *samlClient) Delete(ctx context.Context) error {
	return nexus.APIError(c.api.DeleteSecuritySaml(ctx).Execute())
}

// ErrNotIdPMetadata is returned by ParseIdPMetadata for anything but SAML
// metadata.
var ErrNotIdPMetadata = errors.New("not a SAML metadata XML document")

// ParseIdPMetadata checks that data is SAML metadata, and returns it without
// byte order mark and surrounding whitespace, neither of which may precede an
// XML declaration.
//
// It rejects what a misconfigured source serves instead, such as a login page
// or another key of a Secret, whose content must not be sent to Nexus. Nexus
// validates the metadata itself.
func ParseIdPMetadata(data []byte) ([]byte, error) {
	data = bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))

	// The metadata is sent as JSON, which would mangle anything but UTF-8.
	if !utf8.Valid(data) {
		return nil, ErrNotIdPMetadata
	}

	// The declared encoding does not matter: the data is UTF-8.
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, ErrNotIdPMetadata
		}

		root, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if root.Name.Space != samlMetadataNamespace {
			return nil, ErrNotIdPMetadata
		}

		return data, nil
	}
}

// GenerateSAML converts a SAML CR to the Nexus API type, with the identity
// provider metadata read from the source the CR names.
func GenerateSAML(samlCR *iamv1alpha1.SAML, idpMetadata string) v395.SamlConfigurationXO {
	return v395.SamlConfigurationXO{
		IdpMetadata:                idpMetadata,
		EntityId:                   new(samlCR.Spec.ForProvider.EntityId),
		UsernameAttribute:          samlCR.Spec.ForProvider.UsernameAttribute,
		FirstNameAttribute:         samlCR.Spec.ForProvider.FirstNameAttribute,
		LastNameAttribute:          samlCR.Spec.ForProvider.LastNameAttribute,
		EmailAttribute:             samlCR.Spec.ForProvider.EmailAttribute,
		GroupsAttribute:            samlCR.Spec.ForProvider.GroupsAttribute,
		ValidateResponseSignature:  samlCR.Spec.ForProvider.ValidateResponseSignature,
		ValidateAssertionSignature: samlCR.Spec.ForProvider.ValidateAssertionSignature,
	}
}

// GenerateSAMLObservation converts an observed SAML to an observation struct.
// The identity provider metadata is recorded as its digest only.
func GenerateSAMLObservation(observed *v395.SamlConfigurationXO) iamv1alpha1.SAMLObservation {
	if observed == nil {
		return iamv1alpha1.SAMLObservation{}
	}

	return iamv1alpha1.SAMLObservation{
		IdpMetadataSHA256:          IdPMetadataSHA256(observed.IdpMetadata),
		EntityId:                   observed.GetEntityId(),
		UsernameAttribute:          observed.UsernameAttribute,
		FirstNameAttribute:         observed.FirstNameAttribute,
		LastNameAttribute:          observed.LastNameAttribute,
		EmailAttribute:             observed.EmailAttribute,
		GroupsAttribute:            observed.GroupsAttribute,
		ValidateResponseSignature:  observed.ValidateResponseSignature,
		ValidateAssertionSignature: observed.ValidateAssertionSignature,
	}
}

// IdPMetadataSHA256 returns the hex encoded SHA-256 digest of identity
// provider metadata.
func IdPMetadataSHA256(idpMetadata string) string {
	sum := sha256.Sum256([]byte(idpMetadata))

	return hex.EncodeToString(sum[:])
}

// samlObservationToSAML converts a SAMLObservation back to the API type,
// without the identity provider metadata it holds the digest of only.
func samlObservationToSAML(obs iamv1alpha1.SAMLObservation) v395.SamlConfigurationXO {
	return v395.SamlConfigurationXO{
		EntityId:                   new(obs.EntityId),
		UsernameAttribute:          obs.UsernameAttribute,
		FirstNameAttribute:         obs.FirstNameAttribute,
		LastNameAttribute:          obs.LastNameAttribute,
		EmailAttribute:             obs.EmailAttribute,
		GroupsAttribute:            obs.GroupsAttribute,
		ValidateResponseSignature:  obs.ValidateResponseSignature,
		ValidateAssertionSignature: obs.ValidateAssertionSignature,
	}
}

// IsSAMLUpToDate reports whether the CR spec, with the identity provider
// metadata read from the source it names, matches the observed SAML config.
func IsSAMLUpToDate(samlCR *iamv1alpha1.SAML, idpMetadata string) bool {
	desired := GenerateSAML(samlCR, "")
	observed := samlObservationToSAML(samlCR.Status.AtProvider)

	return samlCR.Status.AtProvider.IdpMetadataSHA256 == IdPMetadataSHA256(idpMetadata) &&
		reflect.DeepEqual(desired, observed)
}
