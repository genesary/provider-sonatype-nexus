package iam

import (
	nexuspkgsecurity "github.com/datadrivers/go-nexus-client/nexus3/pkg/security"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// UserClient manages Nexus users.
type UserClient interface {
	Get(id string) (*security.User, error)
	Create(user security.User) error
	Update(id string, user security.User) error
	Delete(id string) error
	ChangePassword(id, password string) error
}

// userClientImpl wraps SecurityUserService to hide the optional source
// parameter from Get.
type userClientImpl struct {
	svc *nexuspkgsecurity.SecurityUserService
}

// Get returns the user with the given ID.
func (c *userClientImpl) Get(id string) (*security.User, error) {
	return c.svc.Get(id, nil)
}

// Create creates a new Nexus user.
func (c *userClientImpl) Create(user security.User) error {
	return c.svc.Create(user)
}

// Update updates the user with the given ID.
func (c *userClientImpl) Update(id string, user security.User) error {
	return c.svc.Update(id, user)
}

// Delete deletes the user with the given ID.
func (c *userClientImpl) Delete(id string) error {
	return c.svc.Delete(id)
}

// ChangePassword changes the password for the user with the given ID.
func (c *userClientImpl) ChangePassword(id, password string) error {
	return c.svc.ChangePassword(id, password)
}

// NewUserClient returns a new UserClient.
func NewUserClient(creds nexus.Credentials) (UserClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &userClientImpl{svc: nc.Security.User}, nil
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
