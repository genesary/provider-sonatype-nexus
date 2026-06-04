// Package user contains the controller for User resources.
package user

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotUser is returned when the managed resource is not a User.
	errNotUser = "managed resource is not a User custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when getting the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errGetCreds is returned when getting credentials fails.
	errGetCreds = "cannot get credentials"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetUser is returned when getting the user from Nexus fails.
	errGetUser = "cannot get user from Nexus"
	// errCreateUser is returned when creating the user in Nexus fails.
	errCreateUser = "cannot create user in Nexus"
	// errUpdateUser is returned when updating the user in Nexus fails.
	errUpdateUser = "cannot update user in Nexus"
	// errDeleteUser is returned when deleting the user from Nexus fails.
	errDeleteUser = "cannot delete user from Nexus"
	// errGetPassword is returned when getting the password from a secret fails.
	errGetPassword = "cannot get password from secret"
	// errChangePassword is returned when changing the user password fails.
	errChangePassword = "cannot change user password"

	// userStatusActive is the active status string for Nexus users.
	userStatusActive = "active"
	// userSourceDefault is the default source string for Nexus users.
	userSourceDefault = "default"
)

// Setup creates a controller for User resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.User{}).
		Complete(ratelimiter.NewReconciler(name, rec, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect creates an ExternalClient for the User controller.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isUser := managedRes.(*v1alpha1.User)
	if !isUser {
		return nil, errors.New(errNotUser)
	}

	modernMG, isModern := managedRes.(resource.ModernManaged)
	if !isModern {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	err := c.usage.Track(ctx, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	providerConfig := &v1alpha1.ProviderConfig{}

	err = c.kube.Get(ctx, client.ObjectKey{Name: modernMG.GetProviderConfigReference().Name}, providerConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, providerConfig)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: nc, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client nexus.Client
	kube   client.Client
}

// Observe checks if the User resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	userCR, isUser := managedRes.(*v1alpha1.User)
	if !isUser {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userCR)
	if userID == "" {
		userID = userCR.Spec.ForProvider.UserID
	}

	nexusUser, err := e.client.Security().GetUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	if nexusUser == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	userCR.SetConditions(v1alpha1.Available())

	upToDate := isUserUpToDate(userCR, nexusUser)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create creates a new User resource.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	userCR, isUser := managedRes.(*v1alpha1.User)
	if !isUser {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	password, err := e.getPassword(ctx, userCR)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPassword)
	}

	nexusUser := generateUser(userCR, password)

	err = e.client.Security().CreateUser(ctx, nexusUser)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	meta.SetExternalName(userCR, userCR.Spec.ForProvider.UserID)

	return managed.ExternalCreation{}, nil
}

// Update modifies an existing User resource.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	userCR, isUser := managedRes.(*v1alpha1.User)
	if !isUser {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userCR)
	if userID == "" {
		userID = userCR.Spec.ForProvider.UserID
	}

	// Update user info (without password)
	nexusUser := generateUser(userCR, "")

	err := e.client.Security().UpdateUser(ctx, userID, nexusUser)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	// Handle password change if secret is specified
	if userCR.Spec.ForProvider.PasswordSecretRef != nil {
		password, err := e.getPassword(ctx, userCR)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errGetPassword)
		}

		if password != "" {
			err := e.client.Security().ChangePassword(ctx, userID, password)
			if err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, errChangePassword)
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes an existing User resource.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	userCR, isUser := managedRes.(*v1alpha1.User)
	if !isUser {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userCR)
	if userID == "" {
		userID = userCR.Spec.ForProvider.UserID
	}

	err := e.client.Security().DeleteUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect from the provider.
func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

// getPassword retrieves the password from the secret reference.
func (e *external) getPassword(ctx context.Context, userCR *v1alpha1.User) (string, error) {
	if userCR.Spec.ForProvider.PasswordSecretRef == nil {
		return "", nil
	}

	secret := &corev1.Secret{}

	err := e.kube.Get(ctx, types.NamespacedName{
		Name:      userCR.Spec.ForProvider.PasswordSecretRef.Name,
		Namespace: userCR.Spec.ForProvider.PasswordSecretRef.Namespace,
	}, secret)
	if err != nil {
		return "", err
	}

	key := userCR.Spec.ForProvider.PasswordSecretRef.Key
	if key == "" {
		key = "password"
	}

	data, hasKey := secret.Data[key]
	if !hasKey {
		return "", errors.Errorf("secret does not contain key %q", key)
	}

	return string(data), nil
}

// generateUser generates a User from the CR spec.
func generateUser(userCR *v1alpha1.User, password string) security.User {
	nexusUser := security.User{
		UserID:       userCR.Spec.ForProvider.UserID,
		FirstName:    userCR.Spec.ForProvider.FirstName,
		LastName:     userCR.Spec.ForProvider.LastName,
		EmailAddress: userCR.Spec.ForProvider.EmailAddress,
		Password:     password,
		Status:       userStatusActive,
		Source:       userSourceDefault,
		Roles:        userCR.Spec.ForProvider.Roles,
	}

	if userCR.Spec.ForProvider.Status != nil {
		nexusUser.Status = *userCR.Spec.ForProvider.Status
	}

	if userCR.Spec.ForProvider.Source != nil {
		nexusUser.Source = *userCR.Spec.ForProvider.Source
	}

	return nexusUser
}

// isUserUpToDate checks if a User is up to date.
func isUserUpToDate(userCR *v1alpha1.User, nexusUser *security.User) bool {
	if userCR.Spec.ForProvider.FirstName != nexusUser.FirstName {
		return false
	}

	if userCR.Spec.ForProvider.LastName != nexusUser.LastName {
		return false
	}

	if userCR.Spec.ForProvider.EmailAddress != nexusUser.EmailAddress {
		return false
	}

	if userCR.Spec.ForProvider.Status != nil && *userCR.Spec.ForProvider.Status != nexusUser.Status {
		return false
	}

	if !stringSlicesEqual(userCR.Spec.ForProvider.Roles, nexusUser.Roles) {
		return false
	}

	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(sliceA, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}

	for idx := range sliceA {
		if sliceA[idx] != sliceB[idx] {
			return false
		}
	}

	return true
}

// isNotFound checks if an error indicates a resource was not found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "does not exist")
}
