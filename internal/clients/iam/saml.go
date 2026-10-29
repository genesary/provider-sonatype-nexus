package iam

import (
	"context"
	"reflect"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

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

// GenerateSAML converts a SAML CR to the Nexus API type.
func GenerateSAML(samlCR *iamv1alpha1.SAML) v395.SamlConfigurationXO {
	return v395.SamlConfigurationXO{
		IdpMetadata:                samlCR.Spec.ForProvider.IdpMetadata,
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
func GenerateSAMLObservation(observed *v395.SamlConfigurationXO) iamv1alpha1.SAMLObservation {
	if observed == nil {
		return iamv1alpha1.SAMLObservation{}
	}

	return iamv1alpha1.SAMLObservation{
		IdpMetadata:                observed.IdpMetadata,
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

// samlObservationToSAML converts a SAMLObservation back to the API type.
func samlObservationToSAML(obs iamv1alpha1.SAMLObservation) v395.SamlConfigurationXO {
	return v395.SamlConfigurationXO{
		IdpMetadata:                obs.IdpMetadata,
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

// IsSAMLUpToDate reports whether the CR spec matches the observed SAML config.
func IsSAMLUpToDate(samlCR *iamv1alpha1.SAML) bool {
	desired := GenerateSAML(samlCR)
	observed := samlObservationToSAML(samlCR.Status.AtProvider)

	return reflect.DeepEqual(desired, observed)
}
