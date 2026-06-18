package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// AnonymousAccessClient manages anonymous access settings.
type AnonymousAccessClient interface {
	GetAnonymousAccess(ctx context.Context) (*security.AnonymousAccessSettings, error)
	UpdateAnonymousAccess(ctx context.Context, settings security.AnonymousAccessSettings) error
}

// NewAnonymousAccessClient returns a new AnonymousAccessClient.
func NewAnonymousAccessClient(creds nexus.Credentials) (AnonymousAccessClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
}

// GenerateAnonymousAccessSettings converts the CR spec to Nexus settings.
func GenerateAnonymousAccessSettings(cr *iamv1alpha1.AnonymousAccess) security.AnonymousAccessSettings {
	return security.AnonymousAccessSettings{
		Enabled:   cr.Spec.ForProvider.Enabled,
		UserID:    cr.Spec.ForProvider.UserID,
		RealmName: cr.Spec.ForProvider.RealmName,
	}
}

// GenerateAnonymousAccessObservation returns the observed state.
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

// IsAnonymousAccessUpToDate reports whether the CR spec matches observed.
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
