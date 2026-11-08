package ssrfprotection

import (
	"context"
	"errors"
	"testing"
	"time"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
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

// newTestSSRF returns an SSRFProtection allowing one IP and one domain.
func newTestSSRF() *instancev1alpha1.SSRFProtection {
	return &instancev1alpha1.SSRFProtection{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: instancev1alpha1.SSRFProtectionSpec{
			ForProvider: instancev1alpha1.SSRFProtectionParameters{
				Enabled:        new(true),
				AllowedIPs:     []string{"10.0.0.50"},
				AllowedDomains: []string{"registry.local"},
			},
		},
	}
}

// newDeletedTestSSRF returns an SSRFProtection that is being deleted, whose
// status still reports the exceptions observed before the deletion.
func newDeletedTestSSRF() *instancev1alpha1.SSRFProtection {
	cr := newTestSSRF()
	now := metav1.NewTime(time.Now())
	cr.DeletionTimestamp = &now
	cr.Status.AtProvider = instancev1alpha1.SSRFProtectionObservation{
		Enabled:        true,
		AllowedIPs:     []string{"10.0.0.50"},
		AllowedDomains: []string{"registry.local"},
	}

	return cr
}

// newObservedSSRF returns the Nexus settings matching newTestSSRF.
func newObservedSSRF() *v395.SsrfProtectionConfigurationXO {
	return &v395.SsrfProtectionConfigurationXO{
		Enabled:        true,
		AllowedIPs:     []string{"10.0.0.50"},
		AllowedDomains: []string{"registry.local"},
	}
}

// newTestScheme returns a runtime.Scheme with all required types registered.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := instancev1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(instance) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestObserve covers the Observe method across all meaningful branches.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *instancev1alpha1.SSRFProtection
		observed     *v395.SsrfProtectionConfigurationXO
		getErr       error
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:    "GetError",
			cr:      newTestSSRF(),
			getErr:  errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:         "UpToDate",
			cr:           newTestSSRF(),
			observed:     newObservedSSRF(),
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name: "Drifted",
			cr:   newTestSSRF(),
			observed: &v395.SsrfProtectionConfigurationXO{
				Enabled:    true,
				AllowedIPs: []string{"10.0.0.50", "10.0.0.51"},
			},
			wantExists:   true,
			wantUpToDate: false,
		},
		{
			// A singleton that is already at its default needs no Create.
			name:         "DefaultSettingsStillExist",
			cr:           newTestSSRF(),
			observed:     &v395.SsrfProtectionConfigurationXO{Enabled: true},
			wantExists:   true,
			wantUpToDate: false,
		},
		{
			name:         "Deleting_DefaultsRestored_ReportsAbsent",
			cr:           newDeletedTestSSRF(),
			observed:     &v395.SsrfProtectionConfigurationXO{Enabled: true},
			wantExists:   false,
			wantUpToDate: false,
		},
		{
			name:         "Deleting_StillCustomised_ReportsPresent",
			cr:           newDeletedTestSSRF(),
			observed:     newObservedSSRF(),
			wantExists:   true,
			wantUpToDate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockSSRFProtectionClient()
			mc.GetFn = func() (*v395.SsrfProtectionConfigurationXO, error) {
				if tt.getErr != nil {
					return nil, tt.getErr
				}

				return tt.observed, nil
			}

			e := &external{client: mc}

			obs, err := e.Observe(context.Background(), tt.cr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}

			// The status follows Nexus on every successful Get, including the
			// one that finds the defaults restored during deletion.
			if tt.getErr == nil && len(tt.cr.Status.AtProvider.AllowedIPs) != len(tt.observed.AllowedIPs) {
				t.Errorf("Observe() AtProvider.AllowedIPs = %v, want %v", tt.cr.Status.AtProvider.AllowedIPs, tt.observed.AllowedIPs)
			}
		})
	}
}

// TestApply covers Create and Update, which both replace the settings with
// the spec.
func TestApply(t *testing.T) {
	t.Parallel()

	apply := map[string]func(*external, *instancev1alpha1.SSRFProtection) error{
		"Create": func(e *external, cr *instancev1alpha1.SSRFProtection) error {
			_, err := e.Create(context.Background(), cr)

			return err
		},
		"Update": func(e *external, cr *instancev1alpha1.SSRFProtection) error {
			_, err := e.Update(context.Background(), cr)

			return err
		},
	}

	for name, call := range apply {
		t.Run(name+"_Success", func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockSSRFProtectionClient()
			mc.UpdateFn = func(_ v395.SsrfProtectionConfigurationXO) error { return nil }

			err := call(&external{client: mc}, newTestSSRF())
			if err != nil {
				t.Fatalf("%s() unexpected error: %v", name, err)
			}

			if len(mc.UpdateCalls) != 1 {
				t.Fatalf("expected 1 Update call, got %d", len(mc.UpdateCalls))
			}

			got := mc.UpdateCalls[0]
			if !got.Enabled || len(got.AllowedIPs) != 1 || len(got.AllowedDomains) != 1 {
				t.Errorf("%s() applied %+v, want the spec", name, got)
			}
		})

		t.Run(name+"_Error", func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockSSRFProtectionClient()
			mc.UpdateFn = func(_ v395.SsrfProtectionConfigurationXO) error { return errors.New("update failed") }

			err := call(&external{client: mc}, newTestSSRF())
			if err == nil {
				t.Errorf("%s() expected an error", name)
			}
		})
	}
}

// TestDelete verifies Delete restores the Nexus default settings.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("RestoresDefaults", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockSSRFProtectionClient()
		mc.UpdateFn = func(_ v395.SsrfProtectionConfigurationXO) error { return nil }

		_, err := (&external{client: mc}).Delete(context.Background(), newDeletedTestSSRF())
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if len(mc.UpdateCalls) != 1 {
			t.Fatalf("expected 1 Update call, got %d", len(mc.UpdateCalls))
		}

		got := mc.UpdateCalls[0]
		if !got.Enabled || len(got.AllowedIPs) != 0 || len(got.AllowedDomains) != 0 {
			t.Errorf("Delete() applied %+v, want enabled protection without exceptions", got)
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockSSRFProtectionClient()
		mc.UpdateFn = func(_ v395.SsrfProtectionConfigurationXO) error { return errors.New("update failed") }

		_, err := (&external{client: mc}).Delete(context.Background(), newDeletedTestSSRF())
		if err == nil {
			t.Error("Delete() expected an error")
		}
	})
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockSSRFProtectionClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_TrackError verifies Connect fails without ProviderConfig Kind.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestSSRF()
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError verifies Connect needs ProviderConfig.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestSSRF()
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
