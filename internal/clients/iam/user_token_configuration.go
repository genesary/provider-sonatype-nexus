package iam

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// UserTokenConfigurationClient manages user token configuration.
type UserTokenConfigurationClient interface {
	Get(ctx context.Context) (*v395.UserTokensApiModel, error)
	Configure(ctx context.Context, config v395.UserTokensApiModel) error
}

// userTokenConfigurationClient implements UserTokenConfigurationClient over
// the Nexus API.
type userTokenConfigurationClient struct {
	api *v395.SecurityManagementUserTokensAPIService
}

// NewUserTokenConfigurationClient returns a new UserTokenConfigurationClient.
func NewUserTokenConfigurationClient(creds nexus.Credentials) (UserTokenConfigurationClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &userTokenConfigurationClient{api: nc.SecurityManagementUserTokensAPI}, nil
}

// Get returns the user token configuration.
func (c *userTokenConfigurationClient) Get(ctx context.Context) (*v395.UserTokensApiModel, error) {
	config, resp, err := c.api.ListSecurityUserTokens(ctx).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return config, nil
}

// Configure replaces the user token configuration.
func (c *userTokenConfigurationClient) Configure(ctx context.Context, config v395.UserTokensApiModel) error {
	_, resp, err := c.api.UpdateSecurityUserTokens(ctx).UserTokensApiModel(config).Execute()

	return nexus.APIError(resp, err)
}

// GenerateUserTokenConfiguration converts the CR spec to Nexus config.
//
// Every setting is submitted, an unset optional as its zero value, which is
// what IsUserTokenConfigUpToDate compares against.
func GenerateUserTokenConfiguration(userTokenCfg *iamv1alpha1.UserTokenConfiguration) v395.UserTokensApiModel {
	spec := userTokenCfg.Spec.ForProvider

	return v395.UserTokensApiModel{
		Enabled:           new(spec.Enabled),
		ProtectContent:    new(ptr.Deref(spec.ProtectContent, false)),
		ExpirationEnabled: new(ptr.Deref(spec.ExpirationEnabled, false)),
		ExpirationDays:    new(ptr.Deref(spec.ExpirationDays, 0)),
	}
}

// GenerateUserTokenConfigObservation returns the observed state.
func GenerateUserTokenConfigObservation(config *v395.UserTokensApiModel) iamv1alpha1.UserTokenConfigurationObservation {
	if config == nil {
		return iamv1alpha1.UserTokenConfigurationObservation{}
	}

	return iamv1alpha1.UserTokenConfigurationObservation{
		Enabled:           config.GetEnabled(),
		ProtectContent:    config.GetProtectContent(),
		ExpirationEnabled: config.GetExpirationEnabled(),
		ExpirationDays:    int(config.GetExpirationDays()),
	}
}

// IsUserTokenConfigUpToDate reports whether the CR spec matches observed.
//
// The Nexus payload carries every setting on every write, so an unset optional
// still submits its zero value: turning one off by removing it from the spec
// has to be reported as drift.
func IsUserTokenConfigUpToDate(userTokenCfg *iamv1alpha1.UserTokenConfiguration) bool {
	spec := userTokenCfg.Spec.ForProvider
	obs := userTokenCfg.Status.AtProvider

	if spec.Enabled != obs.Enabled {
		return false
	}

	if ptr.Deref(spec.ProtectContent, false) != obs.ProtectContent {
		return false
	}

	if ptr.Deref(spec.ExpirationEnabled, false) != obs.ExpirationEnabled {
		return false
	}

	return int(ptr.Deref(spec.ExpirationDays, 0)) == obs.ExpirationDays
}
