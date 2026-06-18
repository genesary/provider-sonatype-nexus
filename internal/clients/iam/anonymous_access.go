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

// IsAnonymousAccessUpToDate reports whether the CR matches the observed state.
func IsAnonymousAccessUpToDate(
	anonAccess *iamv1alpha1.AnonymousAccess,
	observed *security.AnonymousAccessSettings,
) bool {
	if anonAccess.Spec.ForProvider.Enabled != observed.Enabled {
		return false
	}

	if anonAccess.Spec.ForProvider.UserID != observed.UserID {
		return false
	}

	if anonAccess.Spec.ForProvider.RealmName != observed.RealmName {
		return false
	}

	return true
}
