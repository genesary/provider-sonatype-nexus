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
	errNotUser        = "managed resource is not a User custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create new Nexus client"
	errGetUser        = "cannot get user from Nexus"
	errCreateUser     = "cannot create user in Nexus"
	errUpdateUser     = "cannot update user in Nexus"
	errDeleteUser     = "cannot delete user from Nexus"
	errGetPassword    = "cannot get password from secret"
	errChangePassword = "cannot change user password"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.User{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
	}

	modernMG, ok := mg.(resource.ModernManaged)
	if !ok {
		return nil, errors.New("managed resource is not a ModernManaged")
	}

	if err := c.usage.Track(ctx, modernMG); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: modernMG.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	creds, err := nexus.GetCredentialsFromSecret(ctx, c.kube, pc)
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

// Observe the external resource.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		userID = cr.Spec.ForProvider.UserID
	}

	user, err := e.client.Security().GetUser(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	if user == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetConditions(v1alpha1.Available())

	upToDate := isUserUpToDate(cr, user)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create the external resource.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	password, err := e.getPassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPassword)
	}

	user := generateUser(cr, password)
	if err := e.client.Security().CreateUser(ctx, user); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	meta.SetExternalName(cr, cr.Spec.ForProvider.UserID)

	return managed.ExternalCreation{}, nil
}

// Update the external resource.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		userID = cr.Spec.ForProvider.UserID
	}

	// Update user info (without password)
	user := generateUser(cr, "")

	err := e.client.Security().UpdateUser(ctx, userID, user)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	// Handle password change if secret is specified
	if cr.Spec.ForProvider.PasswordSecretRef != nil {
		password, err := e.getPassword(ctx, cr)
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

// Delete the external resource.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		userID = cr.Spec.ForProvider.UserID
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
func (e *external) getPassword(ctx context.Context, cr *v1alpha1.User) (string, error) {
	if cr.Spec.ForProvider.PasswordSecretRef == nil {
		return "", nil
	}

	secret := &corev1.Secret{}

	err := e.kube.Get(ctx, types.NamespacedName{
		Name:      cr.Spec.ForProvider.PasswordSecretRef.Name,
		Namespace: cr.Spec.ForProvider.PasswordSecretRef.Namespace,
	}, secret)
	if err != nil {
		return "", err
	}

	key := cr.Spec.ForProvider.PasswordSecretRef.Key
	if key == "" {
		key = "password"
	}

	data, ok := secret.Data[key]
	if !ok {
		return "", errors.Errorf("secret does not contain key %q", key)
	}

	return string(data), nil
}

// generateUser generates a User from the CR spec.
func generateUser(cr *v1alpha1.User, password string) security.User {
	user := security.User{
		UserID:       cr.Spec.ForProvider.UserID,
		FirstName:    cr.Spec.ForProvider.FirstName,
		LastName:     cr.Spec.ForProvider.LastName,
		EmailAddress: cr.Spec.ForProvider.EmailAddress,
		Password:     password,
		Status:       "active",
		Source:       "default",
		Roles:        cr.Spec.ForProvider.Roles,
	}

	if cr.Spec.ForProvider.Status != nil {
		user.Status = *cr.Spec.ForProvider.Status
	}

	if cr.Spec.ForProvider.Source != nil {
		user.Source = *cr.Spec.ForProvider.Source
	}

	return user
}

// isUserUpToDate checks if a User is up to date.
func isUserUpToDate(cr *v1alpha1.User, user *security.User) bool {
	if cr.Spec.ForProvider.FirstName != user.FirstName {
		return false
	}

	if cr.Spec.ForProvider.LastName != user.LastName {
		return false
	}

	if cr.Spec.ForProvider.EmailAddress != user.EmailAddress {
		return false
	}

	if cr.Spec.ForProvider.Status != nil && *cr.Spec.ForProvider.Status != user.Status {
		return false
	}

	if !stringSlicesEqual(cr.Spec.ForProvider.Roles, user.Roles) {
		return false
	}

	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
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
