package iam

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
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

	if userRes.Spec.ForProvider.Status != nil {
		nexusUser.Status = *userRes.Spec.ForProvider.Status
	}

	if userRes.Spec.ForProvider.Source != nil {
		nexusUser.Source = *userRes.Spec.ForProvider.Source
	}

	return nexusUser
}

// IsUserUpToDate reports whether the CR matches the observed User.
func IsUserUpToDate(userRes *iamv1alpha1.User, observed *security.User) bool {
	if userRes.Spec.ForProvider.FirstName != observed.FirstName {
		return false
	}

	if userRes.Spec.ForProvider.LastName != observed.LastName {
		return false
	}

	if userRes.Spec.ForProvider.EmailAddress != observed.EmailAddress {
		return false
	}

	if userRes.Spec.ForProvider.Status != nil &&
		*userRes.Spec.ForProvider.Status != observed.Status {
		return false
	}

	if !StringSlicesEqual(userRes.Spec.ForProvider.Roles, observed.Roles) {
		return false
	}

	return true
}
