// Package iam provides clients for Nexus IAM resources.
package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// AnonymousAccessClient manages the Nexus anonymous access configuration.
type AnonymousAccessClient interface {
	Read(ctx context.Context) (*v395.AnonymousAccessSettingsXO, error)
	Update(ctx context.Context, settings v395.AnonymousAccessSettingsXO) error
}

// anonymousAccessClient implements AnonymousAccessClient over the Nexus API.
type anonymousAccessClient struct {
	api *v395.SecurityManagementAnonymousAccessAPIService
}

// NewAnonymousAccessClient returns an AnonymousAccessClient backed by a
// live Nexus connection.
func NewAnonymousAccessClient(creds nexus.Credentials) (AnonymousAccessClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &anonymousAccessClient{api: nc.SecurityManagementAnonymousAccessAPI}, nil
}

// Read returns the anonymous access settings.
func (c *anonymousAccessClient) Read(ctx context.Context) (*v395.AnonymousAccessSettingsXO, error) {
	settings, resp, err := c.api.ListSecurityAnonymous(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return settings, nil
}

// Update replaces the anonymous access settings.
func (c *anonymousAccessClient) Update(ctx context.Context, settings v395.AnonymousAccessSettingsXO) error {
	_, resp, err := c.api.UpdateSecurityAnonymous(ctx).AnonymousAccessSettingsXO(settings).Execute()

	return nexus.APIError(resp, err)
}

// GenerateAnonymousAccessSettings builds an AnonymousAccessSettings
// from the CR spec.
func GenerateAnonymousAccessSettings(cr *iamv1alpha1.AnonymousAccess) v395.AnonymousAccessSettingsXO {
	return v395.AnonymousAccessSettingsXO{
		Enabled:   new(cr.Spec.ForProvider.Enabled),
		UserId:    cr.Spec.ForProvider.UserID,
		RealmName: cr.Spec.ForProvider.RealmName,
	}
}

// GenerateAnonymousAccessObservation converts observed settings to an
// observation struct.
func GenerateAnonymousAccessObservation(settings *v395.AnonymousAccessSettingsXO) iamv1alpha1.AnonymousAccessObservation {
	if settings == nil {
		return iamv1alpha1.AnonymousAccessObservation{}
	}

	return iamv1alpha1.AnonymousAccessObservation{
		Enabled:   settings.GetEnabled(),
		UserID:    settings.UserId,
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
