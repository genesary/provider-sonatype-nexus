package securityrealm

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

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

// newTestSecurityRealm returns a minimal SecurityRealm for tests.
func newTestSecurityRealm(name string, activeRealms []string) *iamv1alpha1.SecurityRealm {
	return &iamv1alpha1.SecurityRealm{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.SecurityRealmSpec{
			ForProvider: iamv1alpha1.SecurityRealmParameters{
				ActiveRealms: activeRealms,
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
		cr           *iamv1alpha1.SecurityRealm
		mockSetup    func(*iammocks.MockSecurityRealmClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "DeletionTimestamp_ReportsAbsent",
			cr: func() *iamv1alpha1.SecurityRealm {
				cr := newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"})
				now := metav1.NewTime(time.Now())
				cr.DeletionTimestamp = &now

				return cr
			}(),
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ListActiveRealmsError",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ListActiveFn = func() ([]string, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ListActiveFn = func() ([]string, error) {
					return []string{"NexusAuthenticatingRealm"}, nil
				}
				mc.ListAvailableFn = func() ([]security.Realm, error) {
					return []security.Realm{{ID: "NexusAuthenticatingRealm", Name: "Local Authenticating Realm"}}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm", "LdapRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ListActiveFn = func() ([]string, error) {
					return []string{"NexusAuthenticatingRealm"}, nil
				}
				mc.ListAvailableFn = func() ([]security.Realm, error) {
					return nil, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ListAvailableRealmsError_StillObserves",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ListActiveFn = func() ([]string, error) {
					return []string{"NexusAuthenticatingRealm"}, nil
				}
				mc.ListAvailableFn = func() ([]security.Realm, error) {
					return nil, errors.New("unavailable")
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

			mc := iammocks.NewMockSecurityRealmClient()
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

	e := &external{client: iammocks.NewMockSecurityRealmClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

// TestObserve_PopulatesAtProvider checks that Observe sets AtProvider.
func TestObserve_PopulatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"})

	mc := iammocks.NewMockSecurityRealmClient()
	mc.ListActiveFn = func() ([]string, error) {
		return []string{"NexusAuthenticatingRealm"}, nil
	}
	mc.ListAvailableFn = func() ([]security.Realm, error) {
		return []security.Realm{
			{ID: "NexusAuthenticatingRealm", Name: "Local Authenticating Realm"},
			{ID: "LdapRealm", Name: "LDAP Realm"},
		}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if len(cr.Status.AtProvider.AvailableRealms) != 2 {
		t.Errorf("AtProvider.AvailableRealms length = %d, want 2", len(cr.Status.AtProvider.AvailableRealms))
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.SecurityRealm
		mockSetup func(*iammocks.MockSecurityRealmClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockSecurityRealmClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ActivateFn = func(_ []string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockSecurityRealmClient) {
				t.Helper()

				if len(mc.ActivateCalls) != 1 {
					t.Errorf("expected 1 ActivateRealms call, got %d", len(mc.ActivateCalls))
				}

				if len(mc.ActivateCalls[0]) != 1 || mc.ActivateCalls[0][0] != "NexusAuthenticatingRealm" {
					t.Errorf("unexpected realms passed: %v", mc.ActivateCalls[0])
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ActivateFn = func(_ []string) error {
					return errors.New("activate failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockSecurityRealmClient()
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

	e := &external{client: iammocks.NewMockSecurityRealmClient()}

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
		cr        *iamv1alpha1.SecurityRealm
		mockSetup func(*iammocks.MockSecurityRealmClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockSecurityRealmClient)
	}{
		{
			name: "UpdateSuccess",
			cr: newTestSecurityRealm("singleton", []string{
				"NexusAuthenticatingRealm",
				"LdapRealm",
			}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ActivateFn = func(_ []string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockSecurityRealmClient) {
				t.Helper()

				if len(mc.ActivateCalls) != 1 {
					t.Errorf("expected 1 ActivateRealms call, got %d", len(mc.ActivateCalls))
				}

				if len(mc.ActivateCalls[0]) != 2 {
					t.Errorf("expected 2 realms, got %d", len(mc.ActivateCalls[0]))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"}),
			mockSetup: func(mc *iammocks.MockSecurityRealmClient) {
				mc.ActivateFn = func(_ []string) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockSecurityRealmClient()
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

	e := &external{client: iammocks.NewMockSecurityRealmClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests that Delete is a no-op for the singleton SecurityRealm.
func TestDelete(t *testing.T) {
	t.Parallel()

	cr := newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"})
	mc := iammocks.NewMockSecurityRealmClient()

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Errorf("Delete() returned unexpected error: %v", err)
	}

	if len(mc.ActivateCalls) != 0 {
		t.Error("Delete() should not call ActivateRealms")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockSecurityRealmClient()}

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

	if err.Error() != errNotSecurityRealm {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotSecurityRealm)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"})
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

	cr := newTestSecurityRealm("singleton", []string{"NexusAuthenticatingRealm"})
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
