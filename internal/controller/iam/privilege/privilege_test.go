package privilege

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

// newTestPrivilege returns a minimal application Privilege for tests.
func newTestPrivilege(name string) *iamv1alpha1.Privilege {
	return &iamv1alpha1.Privilege{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.PrivilegeSpec{
			ForProvider: iamv1alpha1.PrivilegeParameters{
				Name: name,
				Type: "application",
			},
		},
	}
}

// newTestScheme registers iam and nexus v1alpha1 types.
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
		cr           *iamv1alpha1.Privilege
		mockSetup    func(*iammocks.MockPrivilegeClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.GetPrivilegeFn = func(_ context.Context, _ string) (*security.Privilege, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.GetPrivilegeFn = func(_ context.Context, _ string) (*security.Privilege, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "NilPrivilegeReturned",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.GetPrivilegeFn = func(_ context.Context, _ string) (*security.Privilege, error) {
					//nolint:nilnil // intentionally testing nil privilege with nil error
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.GetPrivilegeFn = func(_ context.Context, _ string) (*security.Privilege, error) {
					return &security.Privilege{
						Name: "my-priv",
						Type: "application",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr: func() *iamv1alpha1.Privilege {
				cr := newTestPrivilege("my-priv")
				actions := []string{"READ"}
				cr.Spec.ForProvider.Actions = actions

				return cr
			}(),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.GetPrivilegeFn = func(_ context.Context, _ string) (*security.Privilege, error) {
					return &security.Privilege{
						Name:    "my-priv",
						Type:    "application",
						Actions: []string{"READ", "WRITE"},
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

			mc := iammocks.NewMockPrivilegeClient()
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

	e := &external{client: iammocks.NewMockPrivilegeClient()}

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
		cr        *iamv1alpha1.Privilege
		mockSetup func(*iammocks.MockPrivilegeClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockPrivilegeClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestPrivilege("new-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.CreatePrivilegeFn = func(_ context.Context, _ *iamv1alpha1.Privilege) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockPrivilegeClient) {
				t.Helper()

				if len(mc.CreatePrivilegeCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreatePrivilegeCalls))
				}

				if mc.CreatePrivilegeCalls[0].Spec.ForProvider.Name != "new-priv" {
					t.Errorf("wrong name: %v", mc.CreatePrivilegeCalls[0].Spec.ForProvider.Name)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestPrivilege("new-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.CreatePrivilegeFn = func(_ context.Context, _ *iamv1alpha1.Privilege) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockPrivilegeClient()
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

	e := &external{client: iammocks.NewMockPrivilegeClient()}

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
		cr        *iamv1alpha1.Privilege
		mockSetup func(*iammocks.MockPrivilegeClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockPrivilegeClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.UpdatePrivilegeFn = func(_ context.Context, _ string, _ *iamv1alpha1.Privilege) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockPrivilegeClient) {
				t.Helper()

				if len(mc.UpdatePrivilegeCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdatePrivilegeCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.UpdatePrivilegeFn = func(_ context.Context, _ string, _ *iamv1alpha1.Privilege) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockPrivilegeClient()
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

	e := &external{client: iammocks.NewMockPrivilegeClient()}

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
		cr        *iamv1alpha1.Privilege
		mockSetup func(*iammocks.MockPrivilegeClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockPrivilegeClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.DeletePrivilegeFn = func(_ context.Context, _ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockPrivilegeClient) {
				t.Helper()

				if len(mc.DeletePrivilegeCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeletePrivilegeCalls))
				}

				if mc.DeletePrivilegeCalls[0] != "my-priv" {
					t.Errorf("wrong id passed to Delete: %v", mc.DeletePrivilegeCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.DeletePrivilegeFn = func(_ context.Context, _ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestPrivilege("my-priv"),
			mockSetup: func(mc *iammocks.MockPrivilegeClient) {
				mc.DeletePrivilegeFn = func(_ context.Context, _ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockPrivilegeClient()
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

	e := &external{client: iammocks.NewMockPrivilegeClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockPrivilegeClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_WrongType tests Connect with wrong resource type.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	conn := &connector{}

	_, err := conn.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotPrivilege {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotPrivilege)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestPrivilege("my-priv")
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

	cr := newTestPrivilege("my-priv")
	cr.UID = types.UID("test-uid-5678")
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
