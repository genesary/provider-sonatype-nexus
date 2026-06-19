package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// UserClient manages Nexus users.
type UserClient interface {
	GetUser(ctx context.Context, id string) (*security.User, error)
	CreateUser(ctx context.Context, user security.User) error
	UpdateUser(ctx context.Context, id string, user security.User) error
	DeleteUser(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, id, password string) error
}

// NewUserClient returns a new UserClient.
func NewUserClient(creds nexus.Credentials) (UserClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create nexus client")
	}

	return nexusClient.Security(), nil
}

// GenerateUser converts a User CR to the Nexus API type.
func GenerateUser(userRes *iamv1alpha1.User, password string) security.User {
	nexusUser := security.User{
		UserID:       userRes.Spec.ForProvider.UserID,
		FirstName:    userRes.Spec.ForProvider.FirstName,
		LastName:     userRes.Spec.ForProvider.LastName,
		EmailAddress: userRes.Spec.ForProvider.EmailAddress,
		Password:     password,
		Status:       "active",
		Source:       "default",
		Roles:        userRes.Spec.ForProvider.Roles,
	}

	helpers.AssignIfNonNil(&nexusUser.Status, userRes.Spec.ForProvider.Status)
	helpers.AssignIfNonNil(&nexusUser.Source, userRes.Spec.ForProvider.Source)

	return nexusUser
}

// GenerateUserObservation returns the observed User state.
// Note: the upstream go-nexus-client security.User struct does not expose
// the readOnly or externalRoles fields returned by the Nexus REST API, so
// UserObservation.ReadOnly and UserObservation.ExternalRoles remain nil until
// the upstream library is updated.
func GenerateUserObservation(observed *security.User) iamv1alpha1.UserObservation {
	if observed == nil {
		return iamv1alpha1.UserObservation{}
	}

	return iamv1alpha1.UserObservation{
		FirstName:    observed.FirstName,
		LastName:     observed.LastName,
		EmailAddress: observed.EmailAddress,
		Status:       observed.Status,
		Roles:        observed.Roles,
	}
}

// IsUserUpToDate reports whether the CR spec matches the observed User.
func IsUserUpToDate(userRes *iamv1alpha1.User) bool {
	obs := userRes.Status.AtProvider

	if userRes.Spec.ForProvider.FirstName != obs.FirstName {
		return false
	}

	if userRes.Spec.ForProvider.LastName != obs.LastName {
		return false
	}

	if userRes.Spec.ForProvider.EmailAddress != obs.EmailAddress {
		return false
	}

	if !helpers.IsComparablePtrEqualComparable(userRes.Spec.ForProvider.Status, obs.Status) {
		return false
	}

	if !helpers.AreStringSlicesEqual(userRes.Spec.ForProvider.Roles, obs.Roles) {
		return false
	}

	return true
}
