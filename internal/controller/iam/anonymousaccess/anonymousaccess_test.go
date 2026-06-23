package anonymousaccess

import (
	"context"
	"errors"
	"testing"
	"time"

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

// newTestAnonymousAccess returns a minimal AnonymousAccess for tests.
func newTestAnonymousAccess(enabled bool, userID, realmName string) *iamv1alpha1.AnonymousAccess {
	return &iamv1alpha1.AnonymousAccess{
		ObjectMeta: metav1.ObjectMeta{Name: "singleton"},
		Spec: iamv1alpha1.AnonymousAccessSpec{
			ForProvider: iamv1alpha1.AnonymousAccessParameters{
				Enabled:   enabled,
				UserID:    userID,
				RealmName: realmName,
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
		cr           *iamv1alpha1.AnonymousAccess
		mockSetup    func(*iammocks.MockAnonymousAccessClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "DeletionTimestamp_ReportsAbsent",
			cr: func() *iamv1alpha1.AnonymousAccess {
				cr := newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm")
				now := metav1.NewTime(time.Now())
				cr.DeletionTimestamp = &now

				return cr
			}(),
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.ReadFn = func() (*security.AnonymousAccessSettings, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.ReadFn = func() (*security.AnonymousAccessSettings, error) {
					return &security.AnonymousAccessSettings{
						Enabled:   true,
						UserID:    "anonymous",
						RealmName: "NexusAuthorizingRealm",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButEnabledDiffers",
			cr:   newTestAnonymousAccess(false, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.ReadFn = func() (*security.AnonymousAccessSettings, error) {
					return &security.AnonymousAccessSettings{
						Enabled:   true,
						UserID:    "anonymous",
						RealmName: "NexusAuthorizingRealm",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButUserIDDiffers",
			cr:   newTestAnonymousAccess(true, "guest", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.ReadFn = func() (*security.AnonymousAccessSettings, error) {
					return &security.AnonymousAccessSettings{
						Enabled:   true,
						UserID:    "anonymous",
						RealmName: "NexusAuthorizingRealm",
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

			mc := iammocks.NewMockAnonymousAccessClient()
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

	e := &external{client: iammocks.NewMockAnonymousAccessClient()}

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
		cr        *iamv1alpha1.AnonymousAccess
		mockSetup func(*iammocks.MockAnonymousAccessClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockAnonymousAccessClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.UpdateFn = func(_ security.AnonymousAccessSettings) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockAnonymousAccessClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}

				if !mc.UpdateCalls[0].Enabled {
					t.Error("expected Enabled=true in Update call")
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.UpdateFn = func(_ security.AnonymousAccessSettings) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockAnonymousAccessClient()
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

	e := &external{client: iammocks.NewMockAnonymousAccessClient()}

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
		cr        *iamv1alpha1.AnonymousAccess
		mockSetup func(*iammocks.MockAnonymousAccessClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockAnonymousAccessClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestAnonymousAccess(false, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.UpdateFn = func(_ security.AnonymousAccessSettings) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockAnonymousAccessClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm"),
			mockSetup: func(mc *iammocks.MockAnonymousAccessClient) {
				mc.UpdateFn = func(_ security.AnonymousAccessSettings) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockAnonymousAccessClient()
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

	e := &external{client: iammocks.NewMockAnonymousAccessClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests that Delete is a no-op for the singleton AnonymousAccess.
func TestDelete(t *testing.T) {
	t.Parallel()

	cr := newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm")
	mc := iammocks.NewMockAnonymousAccessClient()

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Errorf("Delete() returned unexpected error: %v", err)
	}

	if len(mc.UpdateCalls) != 0 {
		t.Error("Delete() should not call Update")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockAnonymousAccessClient()}

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

	if err.Error() != errNotAnonymousAccess {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotAnonymousAccess)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm")
	// Kind is empty → Track will fail early with missingRefError.
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

	cr := newTestAnonymousAccess(true, "anonymous", "NexusAuthorizingRealm")
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
