package iam

import (
	"context"

	"github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

const (
	// defaultUserStatus is the account status Nexus is asked for when the
	// spec sets none. It mirrors the CRD default.
	defaultUserStatus = "active"
	// defaultUserSource is the authentication source Nexus is asked for when
	// the spec sets none. It mirrors the CRD default.
	defaultUserSource = "default"
)

// UserClient manages Nexus users.
type UserClient interface {
	Get(ctx context.Context, id string) (*v395.ApiUser, error)
	Create(ctx context.Context, user v395.ApiCreateUser) error
	Update(ctx context.Context, id string, user v395.ApiUser) error
	Delete(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, id, password string) error
}

// userClient implements UserClient over the Nexus API.
type userClient struct {
	api *v395.SecurityManagementUsersAPIService
}

// NewUserClient returns a new UserClient.
func NewUserClient(creds nexus.Credentials) (UserClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &userClient{api: nc.SecurityManagementUsersAPI}, nil
}

// Get returns the user with the given ID, or an error wrapping
// nexus.ErrNotFound when there is none. Nexus matches the userId filter as a
// prefix, so the result is narrowed down to the exact ID.
func (c *userClient) Get(ctx context.Context, userID string) (*v395.ApiUser, error) {
	users, resp, err := c.api.ListSecurityUsers(ctx).UserId(userID).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	for i := range users {
		if users[i].UserId == userID {
			return &users[i], nil
		}
	}

	return nil, errors.Wrapf(nexus.ErrNotFound, "user %q", userID)
}

// Create creates a new Nexus user.
func (c *userClient) Create(ctx context.Context, user v395.ApiCreateUser) error {
	_, resp, err := c.api.CreateSecurityUsers(ctx).ApiCreateUser(user).Execute()

	return nexus.APIError(resp, err)
}

// Update updates the user with the given ID.
func (c *userClient) Update(ctx context.Context, id string, user v395.ApiUser) error {
	return nexus.APIError(c.api.UpdateSecurityUsers(ctx, id).ApiUser(user).Execute())
}

// Delete deletes the user with the given ID.
func (c *userClient) Delete(ctx context.Context, id string) error {
	return nexus.APIError(c.api.DeleteSecurityUsers(ctx, id).Execute())
}

// ChangePassword changes the password for the user with the given ID.
func (c *userClient) ChangePassword(ctx context.Context, id, password string) error {
	return nexus.APIError(c.api.UpdateSecurityUsersChangePassword(ctx, id).Body(password).Execute())
}

// GenerateUser converts a User CR to the Nexus API creation request. Nexus
// creates every user in the default source, so the spec's source only takes
// part in updates.
func GenerateUser(userRes *iamv1alpha1.User, password string) v395.ApiCreateUser {
	spec := userRes.Spec.ForProvider

	return v395.ApiCreateUser{
		UserId:       spec.UserID,
		FirstName:    spec.FirstName,
		LastName:     spec.LastName,
		EmailAddress: spec.EmailAddress,
		Password:     password,
		Status:       ptr.Deref(spec.Status, defaultUserStatus),
		Roles:        spec.Roles,
	}
}

// GenerateUserUpdate converts a User CR to the Nexus API update request.
func GenerateUserUpdate(userRes *iamv1alpha1.User) v395.ApiUser {
	spec := userRes.Spec.ForProvider

	return v395.ApiUser{
		UserId:       spec.UserID,
		FirstName:    *v395.NewNullableString(new(spec.FirstName)),
		LastName:     *v395.NewNullableString(new(spec.LastName)),
		EmailAddress: *v395.NewNullableString(new(spec.EmailAddress)),
		Status:       ptr.Deref(spec.Status, defaultUserStatus),
		Source:       ptr.Deref(spec.Source, defaultUserSource),
		Roles:        spec.Roles,
	}
}

// GenerateUserObservation returns the observed User state.
func GenerateUserObservation(observed *v395.ApiUser) iamv1alpha1.UserObservation {
	if observed == nil {
		return iamv1alpha1.UserObservation{}
	}

	return iamv1alpha1.UserObservation{
		UserID:        observed.UserId,
		Source:        observed.Source,
		FirstName:     observed.GetFirstName(),
		LastName:      observed.GetLastName(),
		EmailAddress:  observed.GetEmailAddress(),
		Status:        observed.Status,
		Roles:         observed.Roles,
		ReadOnly:      observed.ReadOnly,
		ExternalRoles: observed.ExternalRoles,
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

	// GenerateUser falls back to the same defaults the CRD declares, so an
	// unset status or source still carries an intent to compare against.
	if ptr.Deref(userRes.Spec.ForProvider.Status, defaultUserStatus) != obs.Status {
		return false
	}

	if ptr.Deref(userRes.Spec.ForProvider.Source, defaultUserSource) != obs.Source {
		return false
	}

	if !helpers.AreStringSlicesEqual(userRes.Spec.ForProvider.Roles, obs.Roles) {
		return false
	}

	return true
}
