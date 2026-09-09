package iam

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// UserTokenConfigurationClient manages user token configuration.
type UserTokenConfigurationClient interface {
	Get() (*security.UserTokenConfiguration, error)
	Configure(config security.UserTokenConfiguration) error
}

// NewUserTokenConfigurationClient returns a new UserTokenConfigurationClient.
func NewUserTokenConfigurationClient(creds nexus.Credentials) (UserTokenConfigurationClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Security.UserTokens, nil
}

// GenerateUserTokenConfiguration converts the CR spec to Nexus config.
func GenerateUserTokenConfiguration(userTokenCfg *iamv1alpha1.UserTokenConfiguration) security.UserTokenConfiguration {
	config := security.UserTokenConfiguration{
		Enabled: userTokenCfg.Spec.ForProvider.Enabled,
	}

	helpers.AssignIfNonNil(&config.ProtectContent, userTokenCfg.Spec.ForProvider.ProtectContent)
	helpers.AssignIfNonNil(&config.ExpirationEnabled, userTokenCfg.Spec.ForProvider.ExpirationEnabled)

	if userTokenCfg.Spec.ForProvider.ExpirationDays != nil {
		config.ExpirationDays = int(*userTokenCfg.Spec.ForProvider.ExpirationDays)
	}

	return config
}

// GenerateUserTokenConfigObservation returns the observed state.
func GenerateUserTokenConfigObservation(config *security.UserTokenConfiguration) iamv1alpha1.UserTokenConfigurationObservation {
	if config == nil {
		return iamv1alpha1.UserTokenConfigurationObservation{}
	}

	return iamv1alpha1.UserTokenConfigurationObservation{
		Enabled:           config.Enabled,
		ProtectContent:    config.ProtectContent,
		ExpirationEnabled: config.ExpirationEnabled,
		ExpirationDays:    config.ExpirationDays,
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
