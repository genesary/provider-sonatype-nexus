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

// GenerateSAMLObservation converts an observed SAML to an observation struct.
func GenerateSAMLObservation(observed *security.SAML) iamv1alpha1.SAMLObservation {
	if observed == nil {
		return iamv1alpha1.SAMLObservation{}
	}

	return iamv1alpha1.SAMLObservation{
		IdpMetadata:                observed.IdpMetadata,
		EntityId:                   observed.EntityId,
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
func samlObservationToSAML(obs iamv1alpha1.SAMLObservation) security.SAML {
	return security.SAML{
		IdpMetadata:                obs.IdpMetadata,
		EntityId:                   obs.EntityId,
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
