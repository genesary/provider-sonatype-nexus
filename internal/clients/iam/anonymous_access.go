// Package iam provides clients for Nexus IAM resources.
package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// AnonymousAccessClient manages the Nexus anonymous access configuration.
type AnonymousAccessClient interface {
	Read() (*security.AnonymousAccessSettings, error)
	Update(settings security.AnonymousAccessSettings) error
}

// NewAnonymousAccessClient returns an AnonymousAccessClient backed by a
// live Nexus connection.
func NewAnonymousAccessClient(creds nexus.Credentials) (AnonymousAccessClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Security.Anonymous, nil
}

// GenerateAnonymousAccessSettings builds an AnonymousAccessSettings
// from the CR spec.
func GenerateAnonymousAccessSettings(cr *iamv1alpha1.AnonymousAccess) security.AnonymousAccessSettings {
	return security.AnonymousAccessSettings{
		Enabled:   cr.Spec.ForProvider.Enabled,
		UserID:    cr.Spec.ForProvider.UserID,
		RealmName: cr.Spec.ForProvider.RealmName,
	}
}

// GenerateAnonymousAccessObservation converts observed settings to an
// observation struct.
func GenerateAnonymousAccessObservation(settings *security.AnonymousAccessSettings) iamv1alpha1.AnonymousAccessObservation {
	if settings == nil {
		return iamv1alpha1.AnonymousAccessObservation{}
	}

	return iamv1alpha1.AnonymousAccessObservation{
		Enabled:   settings.Enabled,
		UserID:    settings.UserID,
		RealmName: settings.RealmName,
	}
}

// IsAnonymousAccessUpToDate reports whether the CR spec matches
// the observed state.
func IsAnonymousAccessUpToDate(anonAccess *iamv1alpha1.AnonymousAccess) bool {
	obs := anonAccess.Status.AtProvider

	if anonAccess.Spec.ForProvider.Enabled != obs.Enabled {
		return false
	}

	if anonAccess.Spec.ForProvider.UserID != obs.UserID {
		return false
	}

	if anonAccess.Spec.ForProvider.RealmName != obs.RealmName {
		return false
	}

	return true
}
