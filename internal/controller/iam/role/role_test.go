package role

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
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

// newTestRole returns a minimal Role for tests.
func newTestRole(id, name string) *iamv1alpha1.Role {
	return &iamv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.RoleSpec{
			ForProvider: iamv1alpha1.RoleParameters{
				ID:   id,
				Name: name,
			},
		},
	}
}

// newTestScheme registers iam and nexus v1alpha1 types in a new scheme.
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

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *iamv1alpha1.Role
		mockSetup    func(*iammocks.MockRoleClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestRole("my-role", "My Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, _ string) (*security.Role, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestRole("my-role", "My Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, _ string) (*security.Role, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "NilRoleReturned",
			cr:   newTestRole("my-role", "My Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, _ string) (*security.Role, error) {
					//nolint:nilnil // intentionally testing nil role with nil error
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestRole("my-role", "My Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, _ string) (*security.Role, error) {
					return &security.Role{
						ID:   "my-role",
						Name: "My Role",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr:   newTestRole("my-role", "New Name"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, _ string) (*security.Role, error) {
					return &security.Role{
						ID:   "my-role",
						Name: "Old Name",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UsesExternalNameAnnotation",
			cr: func() *iamv1alpha1.Role {
				cr := newTestRole("my-role", "My Role")
				cr.Annotations = map[string]string{
					"crossplane.io/external-name": "external-role-id",
				}

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.GetRoleFn = func(_ context.Context, id string) (*security.Role, error) {
					if id != "external-role-id" {
						return nil, errors.New("wrong id called")
					}

					return &security.Role{
						ID:   "external-role-id",
						Name: "My Role",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockRoleClient()
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

	e := &external{client: iammocks.NewMockRoleClient()}

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
		cr        *iamv1alpha1.Role
		mockSetup func(*iammocks.MockRoleClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockRoleClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestRole("new-role", "New Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.CreateRoleFn = func(_ context.Context, _ security.Role) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockRoleClient) {
				t.Helper()

				if len(mc.CreateRoleCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateRoleCalls))
				}

				if mc.CreateRoleCalls[0].ID != "new-role" {
					t.Errorf("wrong role ID: %v", mc.CreateRoleCalls[0].ID)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestRole("new-role", "New Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.CreateRoleFn = func(_ context.Context, _ security.Role) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockRoleClient()
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

	e := &external{client: iammocks.NewMockRoleClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.Role
		mockSetup func(*iammocks.MockRoleClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockRoleClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestRole("existing-role", "Updated Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.UpdateRoleFn = func(_ context.Context, _ string, _ security.Role) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockRoleClient) {
				t.Helper()

				if len(mc.UpdateRoleCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateRoleCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestRole("existing-role", "Updated Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.UpdateRoleFn = func(_ context.Context, _ string, _ security.Role) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockRoleClient()
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

	e := &external{client: iammocks.NewMockRoleClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.Role
		mockSetup func(*iammocks.MockRoleClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockRoleClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestRole("old-role", "Old Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.DeleteRoleFn = func(_ context.Context, _ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockRoleClient) {
				t.Helper()

				if len(mc.DeleteRoleCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeleteRoleCalls))
				}

				if mc.DeleteRoleCalls[0] != "old-role" {
					t.Errorf("wrong id passed to Delete: %v", mc.DeleteRoleCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestRole("old-role", "Old Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.DeleteRoleFn = func(_ context.Context, _ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestRole("old-role", "Old Role"),
			mockSetup: func(mc *iammocks.MockRoleClient) {
				mc.DeleteRoleFn = func(_ context.Context, _ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockRoleClient()
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

	e := &external{client: iammocks.NewMockRoleClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockRoleClient()}

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

	if err.Error() != errNotRole {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotRole)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestRole("my-role", "My Role")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests ProviderConfig get failure.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestRole("my-role", "My Role")
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}
