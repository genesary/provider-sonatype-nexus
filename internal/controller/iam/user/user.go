// Package user manages User resources.
package user

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iamclient "github.com/genesary/provider-sonatype-nexus/internal/clients/iam"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

const (
	// errNotUser means the managed resource is not a User custom resource.
	errNotUser = "managed resource is not a User custom resource"
	// errTrackPCUsage is returned when tracking ProviderConfig usage fails.
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC is returned when retrieving the ProviderConfig fails.
	errGetPC = "cannot get ProviderConfig"
	// errNewClient is returned when creating the Nexus client fails.
	errNewClient = "cannot create new Nexus client"
	// errGetUser is returned when retrieving a User fails.
	errGetUser = "cannot get user from Nexus"
	// errCreateUser is returned when creating a User fails.
	errCreateUser = "cannot create user in Nexus"
	// errUpdateUser is returned when updating a User fails.
	errUpdateUser = "cannot update user in Nexus"
	// errDeleteUser is returned when deleting a User fails.
	errDeleteUser = "cannot delete user from Nexus"
	// errGetPassword is returned when reading the password secret fails.
	errGetPassword = "cannot get password from secret"
	// errChangePassword is returned when changing the user password fails.
	errChangePassword = "cannot change user password"

	// defaultPasswordKey is the default key used when reading a password
	// secret and no explicit key is specified in the SecretKeySelector.
	defaultPasswordKey = "password"
)

// Setup adds a controller that reconciles User resources.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.UserGroupKind)

	reconciler := managed.NewReconciler(mgr,
		resource.ManagedKind(iamv1alpha1.UserGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &nexusv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(opts.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))) //nolint:deprecated // no replacement yet

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.User{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}

// connector implements managed.ExternalConnector.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

// Connect produces an ExternalClient for the given managed resource.
func (c *connector) Connect(ctx context.Context, managedRes resource.Managed) (managed.ExternalClient, error) {
	_, isUser := managedRes.(*iamv1alpha1.User)
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

	creds, err := nexus.GetCredentials(ctx, c.kube, modernMG)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	userClient, err := iamclient.NewUserClient(creds)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: userClient, kube: c.kube}, nil
}

// external implements managed.ExternalClient.
type external struct {
	client iamclient.UserClient
	kube   client.Client
}

// Observe checks whether the external resource exists and is up-to-date.
func (e *external) Observe(ctx context.Context, managedRes resource.Managed) (managed.ExternalObservation, error) {
	userRes, isUser := managedRes.(*iamv1alpha1.User)
	if !isUser {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userRes)
	if userID == "" {
		userID = userRes.Spec.ForProvider.UserID
	}

	observed, err := e.client.GetUser(ctx, userID)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	if observed == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	userRes.SetConditions(nexusv1alpha1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: iamclient.IsUserUpToDate(userRes, observed),
	}, nil
}

// Create creates the desired User in Nexus.
func (e *external) Create(ctx context.Context, managedRes resource.Managed) (managed.ExternalCreation, error) {
	userRes, isUser := managedRes.(*iamv1alpha1.User)
	if !isUser {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	password, err := e.getPassword(ctx, userRes)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPassword)
	}

	userData := iamclient.GenerateUser(userRes, password)

	err = e.client.CreateUser(ctx, userData)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	meta.SetExternalName(userRes, userRes.Spec.ForProvider.UserID)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the User to the desired state.
func (e *external) Update(ctx context.Context, managedRes resource.Managed) (managed.ExternalUpdate, error) {
	userRes, isUser := managedRes.(*iamv1alpha1.User)
	if !isUser {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userRes)
	if userID == "" {
		userID = userRes.Spec.ForProvider.UserID
	}

	userData := iamclient.GenerateUser(userRes, "")

	err := e.client.UpdateUser(ctx, userID, userData)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	if userRes.Spec.ForProvider.PasswordSecretRef != nil {
		password, err := e.getPassword(ctx, userRes)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errGetPassword)
		}

		if password != "" {
			err = e.client.ChangePassword(ctx, userID, password)
			if err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, errChangePassword)
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the User from Nexus.
func (e *external) Delete(ctx context.Context, managedRes resource.Managed) (managed.ExternalDelete, error) {
	userRes, isUser := managedRes.(*iamv1alpha1.User)
	if !isUser {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(userRes)
	if userID == "" {
		userID = userRes.Spec.ForProvider.UserID
	}

	err := e.client.DeleteUser(ctx, userID)
	if err != nil {
		if iamclient.IsNotFound(err) {
			return managed.ExternalDelete{}, nil
		}

		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteUser)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect is a no-op; the Nexus HTTP client has no persistent connection.
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// getPassword retrieves the password from the referenced secret.
func (e *external) getPassword(
	ctx context.Context,
	userRes *iamv1alpha1.User,
) (string, error) {
	if userRes.Spec.ForProvider.PasswordSecretRef == nil {
		return "", nil
	}

	secretRef := userRes.Spec.ForProvider.PasswordSecretRef
	secret := &corev1.Secret{}

	err := e.kube.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}, secret)
	if err != nil {
		return "", err
	}

	secretKey := secretRef.Key
	if secretKey == "" {
		secretKey = defaultPasswordKey
	}

	data, hasKey := secret.Data[secretKey]
	if !hasKey {
		return "", errors.Errorf("secret does not contain key %q", secretKey)
	}

	return string(data), nil
}
