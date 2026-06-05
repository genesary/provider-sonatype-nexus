package iam

import (
	"context"
	"reflect"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// SAMLClient manages Nexus SAML SSO configuration.
type SAMLClient interface {
	GetSAML(ctx context.Context) (*security.SAML, error)
	ApplySAML(ctx context.Context, saml security.SAML) error
	DeleteSAML(ctx context.Context) error
}

// NewSAMLClient returns a new SAMLClient.
func NewSAMLClient(creds nexus.Credentials) (SAMLClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
}

// GenerateSAML converts a SAML CR to the Nexus API type.
func GenerateSAML(samlCR *iamv1alpha1.SAML) security.SAML {
	return security.SAML{
		IdpMetadata:                samlCR.Spec.ForProvider.IdpMetadata,
		EntityId:                   samlCR.Spec.ForProvider.EntityId,
		UsernameAttribute:          samlCR.Spec.ForProvider.UsernameAttribute,
		FirstNameAttribute:         samlCR.Spec.ForProvider.FirstNameAttribute,
		LastNameAttribute:          samlCR.Spec.ForProvider.LastNameAttribute,
		EmailAttribute:             samlCR.Spec.ForProvider.EmailAttribute,
		GroupsAttribute:            samlCR.Spec.ForProvider.GroupsAttribute,
		ValidateResponseSignature:  samlCR.Spec.ForProvider.ValidateResponseSignature,
		ValidateAssertionSignature: samlCR.Spec.ForProvider.ValidateAssertionSignature,
	}
}

// IsSAMLUpToDate reports whether the CR matches the observed SAML config.
func IsSAMLUpToDate(samlCR *iamv1alpha1.SAML, observed *security.SAML) bool {
	desired := GenerateSAML(samlCR)

	return reflect.DeepEqual(desired, *observed)
}
