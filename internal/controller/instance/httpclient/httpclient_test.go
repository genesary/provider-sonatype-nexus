package httpclient

import (
	"context"
	"errors"
	"testing"
	"time"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// newTestHTTPClient returns an HTTPClient with a timeout and an
// unauthenticated HTTP proxy.
func newTestHTTPClient() *instancev1alpha1.HTTPClient {
	return &instancev1alpha1.HTTPClient{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: instancev1alpha1.HTTPClientSpec{
			ForProvider: instancev1alpha1.HTTPClientParameters{
				TimeoutSeconds: new(60),
				HTTPProxy: &instancev1alpha1.HTTPClientProxy{
					Host: "proxy.internal",
					Port: 3128,
				},
			},
		},
	}
}

// newObservedSettings returns the Nexus settings matching newTestHTTPClient.
func newObservedSettings() *v395.HttpSettingsXo {
	return &v395.HttpSettingsXo{
		HttpProxy: *v395.NewNullableProxySettingsXo(&v395.ProxySettingsXo{
			Enabled: true,
			Host:    "proxy.internal",
			Port:    "3128",
		}),
		Timeout: 60,
		Retries: 2,
	}
}

// newTestScheme returns a runtime.Scheme with all required types registered.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{instancev1alpha1.AddToScheme, nexusv1alpha1.AddToScheme, corev1.AddToScheme} {
		err := add(s)
		if err != nil {
			t.Fatalf("AddToScheme failed: %v", err)
		}
	}

	return s
}

// newFakeKube returns a fake Kubernetes client holding objs.
func newFakeKube(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	return fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(objs...).Build()
}

// TestObserve covers the Observe method across all meaningful branches.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *instancev1alpha1.HTTPClient
		observed     *v395.HttpSettingsXo
		getErr       error
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
		wantGets     int
	}{
		{
			name:     "GetError",
			cr:       newTestHTTPClient(),
			getErr:   errors.New("connection refused"),
			wantErr:  true,
			wantGets: 1,
		},
		{
			name:         "UpToDate",
			cr:           newTestHTTPClient(),
			observed:     newObservedSettings(),
			wantExists:   true,
			wantUpToDate: true,
			wantGets:     1,
		},
		{
			name: "Drifted",
			cr:   newTestHTTPClient(),
			observed: func() *v395.HttpSettingsXo {
				settings := newObservedSettings()
				settings.Timeout = 20

				return settings
			}(),
			wantExists:   true,
			wantUpToDate: false,
			wantGets:     1,
		},
		{
			// Delete has not reset the settings yet, so they must be
			// reported present for the reconciler to call it.
			name: "Deleting_NotReset_ReportsPresent",
			cr: func() *instancev1alpha1.HTTPClient {
				cr := newTestHTTPClient()
				now := metav1.NewTime(time.Now())
				cr.DeletionTimestamp = &now

				return cr
			}(),
			wantExists: true,
			wantGets:   0,
		},
		{
			name: "Deleting_Reset_ReportsAbsent",
			cr: func() *instancev1alpha1.HTTPClient {
				cr := newTestHTTPClient()
				now := metav1.NewTime(time.Now())
				cr.DeletionTimestamp = &now
				cr.Status.AtProvider.ResetToDefaults = true

				return cr
			}(),
			wantExists: false,
			wantGets:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockHTTPSettingsClient()
			mc.GetFn = func() (*v395.HttpSettingsXo, error) {
				if tt.getErr != nil {
					return nil, tt.getErr
				}

				return tt.observed, nil
			}

			e := &external{client: mc, kube: newFakeKube(t)}

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

			if mc.GetCalls != tt.wantGets {
				t.Errorf("Observe() made %d Get calls, want %d", mc.GetCalls, tt.wantGets)
			}
		})
	}
}

// TestUpdate verifies Update carries unset settings over from Nexus and
// applies the spec.
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.GetFn = func() (*v395.HttpSettingsXo, error) {
			return &v395.HttpSettingsXo{Timeout: 20, Retries: 7}, nil
		}
		mc.UpdateFn = func(_ v395.HttpSettingsXo) error { return nil }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), newTestHTTPClient())
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if len(mc.UpdateCalls) != 1 {
			t.Fatalf("expected 1 Update call, got %d", len(mc.UpdateCalls))
		}

		got := mc.UpdateCalls[0]
		if got.Timeout != 60 || got.Retries != 7 {
			t.Errorf("Timeout, Retries = %d, %d, want the spec 60 and the Nexus 7", got.Timeout, got.Retries)
		}

		if proxy := got.HttpProxy.Get(); proxy == nil || proxy.Host != "proxy.internal" || proxy.AuthInfo.Enabled {
			t.Errorf("HttpProxy = %+v, want the unauthenticated spec proxy", proxy)
		}
	})

	t.Run("GetError", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.GetFn = func() (*v395.HttpSettingsXo, error) { return nil, errors.New("get failed") }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), newTestHTTPClient())
		if err == nil {
			t.Error("Update() expected an error")
		}

		if len(mc.UpdateCalls) != 0 {
			t.Errorf("expected no Update call without the current settings, got %d", len(mc.UpdateCalls))
		}
	})

	t.Run("UpdateError", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.GetFn = func() (*v395.HttpSettingsXo, error) { return newObservedSettings(), nil }
		mc.UpdateFn = func(_ v395.HttpSettingsXo) error { return errors.New("update failed") }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), newTestHTTPClient())
		if err == nil {
			t.Error("Update() expected an error")
		}
	})

	t.Run("MissingPasswordSecret", func(t *testing.T) {
		t.Parallel()

		cr := newTestHTTPClient()
		cr.Spec.ForProvider.HTTPProxy.Authentication = &instancev1alpha1.HTTPClientProxyAuthentication{
			Username: "proxy-user",
			PasswordSecretRef: xpv2.LocalSecretKeySelector{
				LocalSecretReference: xpv2.LocalSecretReference{Name: "missing"},
				Key:                  "password",
			},
		}

		mc := instancemocks.NewMockHTTPSettingsClient()

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), cr)
		if err == nil {
			t.Error("Update() expected an error for a missing password Secret")
		}

		if len(mc.UpdateCalls) != 0 {
			t.Errorf("expected no Update call, got %d", len(mc.UpdateCalls))
		}
	})
}

// TestCreate verifies Create applies the spec like Update.
func TestCreate(t *testing.T) {
	t.Parallel()

	mc := instancemocks.NewMockHTTPSettingsClient()
	mc.GetFn = func() (*v395.HttpSettingsXo, error) { return newObservedSettings(), nil }
	mc.UpdateFn = func(_ v395.HttpSettingsXo) error { return nil }

	_, err := (&external{client: mc, kube: newFakeKube(t)}).Create(context.Background(), newTestHTTPClient())
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if len(mc.UpdateCalls) != 1 {
		t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
	}
}

// TestDelete verifies Delete resets the settings and records it, and that a
// failed reset is not recorded.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.ResetFn = func() error { return nil }

		cr := newTestHTTPClient()

		_, err := (&external{client: mc}).Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if mc.ResetCalls != 1 {
			t.Errorf("expected 1 Reset call, got %d", mc.ResetCalls)
		}

		if !cr.Status.AtProvider.ResetToDefaults {
			t.Error("Delete() did not record the reset in status")
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.ResetFn = func() error { return errors.New("reset failed") }

		cr := newTestHTTPClient()

		_, err := (&external{client: mc}).Delete(context.Background(), cr)
		if err == nil {
			t.Error("Delete() expected an error")
		}

		if cr.Status.AtProvider.ResetToDefaults {
			t.Error("Delete() recorded a reset that failed")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockHTTPSettingsClient()
		mc.ResetFn = func() error { return nexus.ErrNotFound }

		cr := newTestHTTPClient()

		_, err := (&external{client: mc}).Delete(context.Background(), cr)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if !cr.Status.AtProvider.ResetToDefaults {
			t.Error("Delete() did not record the reset in status")
		}
	})
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockHTTPSettingsClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_TrackError verifies Connect fails without ProviderConfig Kind.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := newFakeKube(t)
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestHTTPClient()
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

	fakeClient := newFakeKube(t)
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestHTTPClient()
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
