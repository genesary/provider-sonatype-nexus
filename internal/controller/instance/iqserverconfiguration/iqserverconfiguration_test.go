package iqserverconfiguration

import (
	"context"
	"errors"
	"testing"
	"time"

	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/iq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

func ptrTo[T any](v T) *T { return &v }

func newTestIQServer(enabled bool) *instancev1alpha1.IQServerConfiguration {
	return &instancev1alpha1.IQServerConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "iq-connection"},
		Spec: instancev1alpha1.IQServerConfigurationSpec{
			ForProvider: instancev1alpha1.IQServerConfigurationParameters{
				Enabled:              enabled,
				ShowLink:             true,
				URL:                  ptrTo("https://iq.example.com"),
				AuthenticationMethod: ptrTo("USER"),
				Username:             ptrTo("nexus-user"),
				TimeoutSeconds:       ptrTo(60),
			},
		},
	}
}

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	if err := instancev1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme(instance) failed: %v", err)
	}

	if err := nexusv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

func newObservedConfig() *nexussdk.IQServerConfiguration {
	return &nexussdk.IQServerConfiguration{
		Enabled:            true,
		ShowLink:           true,
		URL:                ptrTo("https://iq.example.com"),
		AuthenticationType: ptrTo("USER"),
		Username:           ptrTo("nexus-user"),
		TimeoutSeconds:     ptrTo(60),
	}
}

func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *instancev1alpha1.IQServerConfiguration
		mockSetup    func(*instancemocks.MockIQServerClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "DeletionTimestamp_ReportsAbsent",
			cr: func() *instancev1alpha1.IQServerConfiguration {
				cr := newTestIQServer(true)
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
			cr:   newTestIQServer(true),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.GetFn = func() (*nexussdk.IQServerConfiguration, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestIQServer(true),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.GetFn = func() (*nexussdk.IQServerConfiguration, error) {
					return newObservedConfig(), nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButEnabledDiffers",
			cr:   newTestIQServer(false),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.GetFn = func() (*nexussdk.IQServerConfiguration, error) {
					return newObservedConfig(), nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButURLDiffers",
			cr: func() *instancev1alpha1.IQServerConfiguration {
				cr := newTestIQServer(true)
				cr.Spec.ForProvider.URL = ptrTo("https://other-iq.example.com")

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.GetFn = func() (*nexussdk.IQServerConfiguration, error) {
					return newObservedConfig(), nil
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

			mc := instancemocks.NewMockIQServerClient()
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

func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockIQServerClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *instancev1alpha1.IQServerConfiguration
		mockSetup func(*instancemocks.MockIQServerClient)
		wantErr   bool
		validate  func(*testing.T, *instancemocks.MockIQServerClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestIQServer(true),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.UpdateFn = func(_ nexussdk.IQServerConfiguration) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *instancemocks.MockIQServerClient) {
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
			cr:   newTestIQServer(true),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.UpdateFn = func(_ nexussdk.IQServerConfiguration) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockIQServerClient()
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

func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockIQServerClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *instancev1alpha1.IQServerConfiguration
		mockSetup func(*instancemocks.MockIQServerClient)
		wantErr   bool
		validate  func(*testing.T, *instancemocks.MockIQServerClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestIQServer(false),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.UpdateFn = func(_ nexussdk.IQServerConfiguration) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *instancemocks.MockIQServerClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestIQServer(true),
			mockSetup: func(mc *instancemocks.MockIQServerClient) {
				mc.UpdateFn = func(_ nexussdk.IQServerConfiguration) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockIQServerClient()
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

func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockIQServerClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	cr := newTestIQServer(true)
	mc := instancemocks.NewMockIQServerClient()

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Errorf("Delete() returned unexpected error: %v", err)
	}

	if len(mc.UpdateCalls) != 0 {
		t.Error("Delete() should not call Update")
	}
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockIQServerClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotIQServerConfiguration {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotIQServerConfiguration)
	}
}

func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestIQServer(true)
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestIQServer(true)
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
