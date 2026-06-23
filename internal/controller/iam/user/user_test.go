package user

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

// newTestUser returns a minimal User for tests.
func newTestUser(userID, firstName, lastName string) *iamv1alpha1.User {
	return &iamv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: userID},
		Spec: iamv1alpha1.UserSpec{
			ForProvider: iamv1alpha1.UserParameters{
				UserID:       userID,
				FirstName:    firstName,
				LastName:     lastName,
				EmailAddress: userID + "@example.com",
			},
		},
	}
}

// newTestScheme registers iam, nexus v1alpha1, and corev1 types.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := iamv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(iam) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	err = corev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *iamv1alpha1.User
		mockSetup    func(*iammocks.MockUserClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.GetFn = func(_ string) (*security.User, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.GetFn = func(_ string) (*security.User, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "NilUserReturned",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.GetFn = func(_ string) (*security.User, error) {
					//nolint:nilnil // intentionally testing nil user with nil error
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.GetFn = func(_ string) (*security.User, error) {
					return &security.User{
						UserID:       "alice",
						FirstName:    "Alice",
						LastName:     "Smith",
						EmailAddress: "alice@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr:   newTestUser("alice", "Alice", "NewLastName"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.GetFn = func(_ string) (*security.User, error) {
					return &security.User{
						UserID:       "alice",
						FirstName:    "Alice",
						LastName:     "OldLastName",
						EmailAddress: "alice@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockUserClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			obs, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

// TestObserve_WrongType tests Observe with wrong resource type.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockUserClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.User
		mockSetup func(*iammocks.MockUserClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockUserClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.CreateFn = func(_ security.User) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockUserClient) {
				t.Helper()

				if len(mc.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateCalls))
				}

				if mc.CreateCalls[0].UserID != "alice" {
					t.Errorf("wrong user ID: %v", mc.CreateCalls[0].UserID)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.CreateFn = func(_ security.User) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockUserClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestCreate_WrongType tests Create with wrong resource type.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockUserClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

// TestCreate_WithPassword tests Create when a password secret reference is set.
func TestCreate_WithPassword(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alice-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			defaultPasswordKey: []byte("s3cr3t"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	mc := iammocks.NewMockUserClient()
	mc.CreateFn = func(user security.User) error {
		if user.Password != "s3cr3t" {
			return errors.New("unexpected password value")
		}

		return nil
	}

	cr := newTestUser("alice", "Alice", "Smith")
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "alice-secret",
			Namespace: "default",
		},
		Key: defaultPasswordKey,
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Errorf("Create() with password returned unexpected error: %v", err)
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.User
		mockSetup func(*iammocks.MockUserClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockUserClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.UpdateFn = func(_ string, _ security.User) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockUserClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.UpdateFn = func(_ string, _ security.User) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockUserClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestUpdate_WrongType tests Update with wrong resource type.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockUserClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestUpdate_WithPassword tests Update with a password change.
func TestUpdate_WithPassword(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alice-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			defaultPasswordKey: []byte("newpass"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	mc := iammocks.NewMockUserClient()
	mc.UpdateFn = func(_ string, _ security.User) error {
		return nil
	}
	mc.ChangePasswordFn = func(id, password string) error {
		if password != "newpass" {
			return errors.New("unexpected password")
		}

		return nil
	}

	cr := newTestUser("alice", "Alice", "Smith")
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "alice-secret",
			Namespace: "default",
		},
		Key: defaultPasswordKey,
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Update(context.Background(), cr)
	if err != nil {
		t.Errorf("Update() with password returned unexpected error: %v", err)
	}

	if len(mc.ChangePasswordCalls) != 1 {
		t.Errorf("expected 1 ChangePassword call, got %d", len(mc.ChangePasswordCalls))
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.User
		mockSetup func(*iammocks.MockUserClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockUserClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockUserClient) {
				t.Helper()

				if len(mc.DeleteCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeleteCalls))
				}

				if mc.DeleteCalls[0] != "alice" {
					t.Errorf("wrong id passed to Delete: %v", mc.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestUser("alice", "Alice", "Smith"),
			mockSetup: func(mc *iammocks.MockUserClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockUserClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestDelete_WrongType tests Delete with wrong resource type.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockUserClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockUserClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_WrongType tests Connect with wrong resource type.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotUser {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotUser)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestUser("alice", "Alice", "Smith")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests ProviderConfig get failure.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestUser("alice", "Alice", "Smith")
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	conn := &connector{kube: fakeClient, usage: usage}

	_, err := conn.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}
