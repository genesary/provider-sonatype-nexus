package capability

import (
	"context"
	"errors"
	"testing"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	capabilitymocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// newTestCapability returns a minimal Capability for tests.
func newTestCapability(name, typeID string) *instancev1alpha1.Capability {
	return &instancev1alpha1.Capability{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: instancev1alpha1.CapabilitySpec{
			ForProvider: instancev1alpha1.CapabilityParameters{
				TypeId:  typeID,
				Enabled: true,
			},
		},
	}
}

// newTestCapabilityWithID returns a Capability with the external-name
// annotation set.
func newTestCapabilityWithID(name, typeID, id string) *instancev1alpha1.Capability {
	cr := newTestCapability(name, typeID)
	meta.SetExternalName(cr, id)

	return cr
}

// TestObserve covers the Observe method across all meaningful branches.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *instancev1alpha1.Capability
		mockSetup    func(*capabilitymocks.MockCapabilityClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:         "NoExternalName_NotExists",
			cr:           newTestCapability("cap", "DockerBearerTokenRealm"),
			mockSetup:    func(_ *capabilitymocks.MockCapabilityClient) {},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError_NotFound",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return nil, errors.New("resource not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError_General",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "NilCapabilityReturned",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					//nolint:nilnil // intentionally testing nil capability with nil error case
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return &nexussdk.Capability{
						ID:         "abc123",
						Type:       "DockerBearerTokenRealm",
						Enabled:    true,
						Properties: map[string]string{},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated_DifferentEnabled",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return &nexussdk.Capability{
						ID:         "abc123",
						Type:       "DockerBearerTokenRealm",
						Enabled:    false,
						Properties: map[string]string{},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated_DifferentNotes",
			cr: func() *instancev1alpha1.Capability {
				cr := newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123")
				cr.Spec.ForProvider.Notes = "new notes"

				return cr
			}(),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return &nexussdk.Capability{
						ID:         "abc123",
						Type:       "DockerBearerTokenRealm",
						Enabled:    true,
						Notes:      "old notes",
						Properties: map[string]string{},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated_DifferentProperties",
			cr: func() *instancev1alpha1.Capability {
				cr := newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123")
				cr.Spec.ForProvider.Properties = map[string]string{"key": "value"}

				return cr
			}(),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
					return &nexussdk.Capability{
						ID:         "abc123",
						Type:       "DockerBearerTokenRealm",
						Enabled:    true,
						Properties: map[string]string{},
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

			mc := capabilitymocks.NewMockCapabilityClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			got, err := e.Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Observe() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if got.ResourceExists != tt.wantExists {
				t.Errorf("ResourceExists = %v, want %v", got.ResourceExists, tt.wantExists)
			}

			if got.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("ResourceUpToDate = %v, want %v", got.ResourceUpToDate, tt.wantUpToDate)
			}
		})
	}
}

// TestObserve_WrongType verifies Observe returns an error for non-Capability
// resources.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: capabilitymocks.NewMockCapabilityClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Fatal("Observe() expected error for wrong type, got nil")
	}
}

// TestObserve_UpdatesAtProvider verifies Observe populates AtProvider.
func TestObserve_UpdatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123")

	mc := capabilitymocks.NewMockCapabilityClient()
	mc.GetFn = func(_ string) (*nexussdk.Capability, error) {
		return &nexussdk.Capability{
			ID:         "abc123",
			Type:       "DockerBearerTokenRealm",
			Enabled:    true,
			Properties: map[string]string{},
		}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.ID != "abc123" {
		t.Errorf("AtProvider.ID = %q, want %q", cr.Status.AtProvider.ID, "abc123")
	}
}

// TestCreate covers the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cr          *instancev1alpha1.Capability
		mockSetup   func(*capabilitymocks.MockCapabilityClient)
		wantErr     bool
		wantExtName string
	}{
		{
			name: "Success",
			cr:   newTestCapability("cap", "DockerBearerTokenRealm"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.CreateFn = func(_ nexussdk.CapabilityCreate) (*nexussdk.Capability, error) {
					return &nexussdk.Capability{ID: "server-id-123", Type: "DockerBearerTokenRealm"}, nil
				}
			},
			wantErr:     false,
			wantExtName: "server-id-123",
		},
		{
			name: "CreateError",
			cr:   newTestCapability("cap", "DockerBearerTokenRealm"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.CreateFn = func(_ nexussdk.CapabilityCreate) (*nexussdk.Capability, error) {
					return nil, errors.New("nexus unavailable")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := capabilitymocks.NewMockCapabilityClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if !tt.wantErr && meta.GetExternalName(tt.cr) != tt.wantExtName {
				t.Errorf("external-name = %q, want %q", meta.GetExternalName(tt.cr), tt.wantExtName)
			}
		})
	}
}

// TestCreate_WrongType verifies Create returns an error for non-Capability
// resources.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: capabilitymocks.NewMockCapabilityClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("Create() expected error for wrong type, got nil")
	}
}

// TestUpdate covers the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *instancev1alpha1.Capability
		mockSetup func(*capabilitymocks.MockCapabilityClient)
		wantErr   bool
	}{
		{
			name: "Success",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.UpdateFn = func(_ string, _ nexussdk.CapabilityUpdate) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateError",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.UpdateFn = func(_ string, _ nexussdk.CapabilityUpdate) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := capabilitymocks.NewMockCapabilityClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdate_WrongType verifies Update returns an error for non-Capability
// resources.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: capabilitymocks.NewMockCapabilityClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("Update() expected error for wrong type, got nil")
	}
}

// TestDelete covers the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *instancev1alpha1.Capability
		mockSetup func(*capabilitymocks.MockCapabilityClient)
		wantErr   bool
	}{
		{
			name:      "NoID_Noop",
			cr:        newTestCapability("cap", "DockerBearerTokenRealm"),
			mockSetup: func(_ *capabilitymocks.MockCapabilityClient) {},
			wantErr:   false,
		},
		{
			name: "Success",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "NotFound_Noop",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("resource not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestCapabilityWithID("cap", "DockerBearerTokenRealm", "abc123"),
			mockSetup: func(mc *capabilitymocks.MockCapabilityClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("connection refused")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := capabilitymocks.NewMockCapabilityClient()
			tt.mockSetup(mc)

			e := &external{client: mc}

			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestDelete_WrongType verifies Delete returns an error for non-Capability
// resources.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: capabilitymocks.NewMockCapabilityClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Fatal("Delete() expected error for wrong type, got nil")
	}
}

// newTestScheme registers nexus v1alpha1 types in a new scheme.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestConnect_WrongType verifies Connect returns an error for non-Capability
// resources.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Fatal("Connect() expected error for wrong type, got nil")
	}

	if err.Error() != errNotCapability {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotCapability)
	}
}

// TestConnect_TrackError verifies Connect returns an error when ProviderConfig
// tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &instancev1alpha1.Capability{
		ObjectMeta: metav1.ObjectMeta{Name: "cap", Namespace: "default"},
		Spec: instancev1alpha1.CapabilitySpec{
			ForProvider: instancev1alpha1.CapabilityParameters{TypeId: "DockerBearerTokenRealm"},
		},
	}
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Fatal("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError verifies Connect returns an error when
// ProviderConfig is missing.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &instancev1alpha1.Capability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cap",
			Namespace: "default",
			UID:       types.UID("test-uid-1234"),
		},
		Spec: instancev1alpha1.CapabilitySpec{
			ForProvider: instancev1alpha1.CapabilityParameters{TypeId: "DockerBearerTokenRealm"},
		},
	}
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Fatal("Connect() should fail without ProviderConfig in store")
	}
}

// TestDisconnect verifies Disconnect returns no error.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: capabilitymocks.NewMockCapabilityClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() unexpected error: %v", err)
	}
}
