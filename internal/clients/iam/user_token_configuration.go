package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// UserTokenConfigurationClient manages user token configuration.
type UserTokenConfigurationClient interface {
	GetUserTokenConfiguration(ctx context.Context) (*security.UserTokenConfiguration, error)
	UpdateUserTokenConfiguration(ctx context.Context, config security.UserTokenConfiguration) error
}

// NewUserTokenConfigurationClient returns a new UserTokenConfigurationClient.
func NewUserTokenConfigurationClient(creds nexus.Credentials) (UserTokenConfigurationClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
}

// GenerateUserTokenConfiguration converts the CR spec to Nexus config.
func GenerateUserTokenConfiguration(userTokenCfg *iamv1alpha1.UserTokenConfiguration) security.UserTokenConfiguration {
	config := security.UserTokenConfiguration{
		Enabled: userTokenCfg.Spec.ForProvider.Enabled,
	}

	if userTokenCfg.Spec.ForProvider.ProtectContent != nil {
		config.ProtectContent = *userTokenCfg.Spec.ForProvider.ProtectContent
	}

	if userTokenCfg.Spec.ForProvider.ExpirationEnabled != nil {
		config.ExpirationEnabled = *userTokenCfg.Spec.ForProvider.ExpirationEnabled
	}

	if userTokenCfg.Spec.ForProvider.ExpirationDays != nil {
		config.ExpirationDays = int(*userTokenCfg.Spec.ForProvider.ExpirationDays)
	}

	return config
}

// IsUserTokenConfigUpToDate reports whether the CR matches the observed config.
func IsUserTokenConfigUpToDate(
	userTokenCfg *iamv1alpha1.UserTokenConfiguration,
	observed *security.UserTokenConfiguration,
) bool {
	if userTokenCfg.Spec.ForProvider.Enabled != observed.Enabled {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ProtectContent != nil &&
		*userTokenCfg.Spec.ForProvider.ProtectContent != observed.ProtectContent {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ExpirationEnabled != nil &&
		*userTokenCfg.Spec.ForProvider.ExpirationEnabled != observed.ExpirationEnabled {
		return false
	}

	if userTokenCfg.Spec.ForProvider.ExpirationDays != nil &&
		int(*userTokenCfg.Spec.ForProvider.ExpirationDays) != observed.ExpirationDays {
		return false
	}

	return true
}
