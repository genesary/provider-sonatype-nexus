package emailconfiguration

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
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// newTestEmailConfiguration returns an EmailConfiguration without a
// password.
func newTestEmailConfiguration() *instancev1alpha1.EmailConfiguration {
	return &instancev1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "smtp", Namespace: "default"},
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Enabled:         new(true),
				Host:            "smtp.example.com",
				Port:            587,
				Username:        new("nexus"),
				FromAddress:     "nexus@example.com",
				StartTLSEnabled: new(true),
			},
		},
	}
}

// newObservedConfiguration returns the Nexus configuration matching
// newTestEmailConfiguration.
func newObservedConfiguration() *v395.ApiEmailConfiguration {
	return &v395.ApiEmailConfiguration{
		Enabled:         new(true),
		Host:            "smtp.example.com",
		Port:            587,
		Username:        new("nexus"),
		FromAddress:     "nexus@example.com",
		StartTlsEnabled: new(true),
	}
}

// withPassword makes cr reference the password key of the smtp Secret.
func withPassword(cr *instancev1alpha1.EmailConfiguration) *instancev1alpha1.EmailConfiguration {
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: "smtp"},
		Key:                  "password",
	}

	return cr
}

// newPasswordSecret returns the smtp Secret holding password.
func newPasswordSecret(password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "smtp", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte(password)},
	}
}

// deleting marks cr as being deleted.
func deleting(cr *instancev1alpha1.EmailConfiguration) *instancev1alpha1.EmailConfiguration {
	now := metav1.NewTime(time.Now())
	cr.DeletionTimestamp = &now

	return cr
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

	reset := &v395.ApiEmailConfiguration{Enabled: new(false)}

	tests := []struct {
		name         string
		cr           *instancev1alpha1.EmailConfiguration
		secrets      []client.Object
		observed     *v395.ApiEmailConfiguration
		getErr       error
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:    "GetError",
			cr:      newTestEmailConfiguration(),
			getErr:  errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:         "UpToDate",
			cr:           newTestEmailConfiguration(),
			observed:     newObservedConfiguration(),
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name: "Drifted",
			cr:   newTestEmailConfiguration(),
			observed: func() *v395.ApiEmailConfiguration {
				observed := newObservedConfiguration()
				observed.Port = 25

				return observed
			}(),
			wantExists: true,
		},
		{
			// A reset configuration still exists for a live resource: the
			// spec must be applied to it.
			name:       "Reset_NotDeleting_ReportsPresent",
			cr:         newTestEmailConfiguration(),
			observed:   reset,
			wantExists: true,
		},
		{
			// No password has been applied yet, so it must be.
			name:       "PasswordNotApplied",
			cr:         withPassword(newTestEmailConfiguration()),
			secrets:    []client.Object{newPasswordSecret("secret")},
			observed:   newObservedConfiguration(),
			wantExists: true,
		},
		{
			name:     "MissingPasswordSecret",
			cr:       withPassword(newTestEmailConfiguration()),
			observed: newObservedConfiguration(),
			wantErr:  true,
		},
		{
			name:       "Deleting_NotReset_ReportsPresent",
			cr:         deleting(newTestEmailConfiguration()),
			observed:   newObservedConfiguration(),
			wantExists: true,
		},
		{
			name:     "Deleting_Reset_ReportsAbsent",
			cr:       deleting(newTestEmailConfiguration()),
			observed: reset,
		},
		{
			// A deleted password Secret must not block the deletion.
			name:     "Deleting_MissingPasswordSecret_ReportsAbsent",
			cr:       deleting(withPassword(newTestEmailConfiguration())),
			observed: reset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockEmailClient()
			mc.GetFn = func() (*v395.ApiEmailConfiguration, error) {
				if tt.getErr != nil {
					return nil, tt.getErr
				}

				return tt.observed, nil
			}

			e := &external{client: mc, kube: newFakeKube(t, tt.secrets...)}

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
		})
	}
}

// TestObserve_KeepsAppliedFingerprint verifies Observe replaces the
// observation with what Nexus reports but keeps the applied fingerprint,
// which Nexus cannot report.
func TestObserve_KeepsAppliedFingerprint(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfiguration()
	cr.Status.AtProvider.CredentialsFingerprint = "applied"
	cr.Status.AtProvider.SubjectPrefix = "stale"

	mc := instancemocks.NewMockEmailClient()
	mc.GetFn = func() (*v395.ApiEmailConfiguration, error) { return newObservedConfiguration(), nil }

	_, err := (&external{client: mc, kube: newFakeKube(t)}).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	got := cr.Status.AtProvider
	if got.CredentialsFingerprint != "applied" || got.SubjectPrefix != "" || got.Host != "smtp.example.com" {
		t.Errorf("AtProvider = %+v, want the Nexus configuration with the applied fingerprint", got)
	}
}

// TestUpdate verifies Update applies the spec with the resolved password and
// records its fingerprint.
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.UpdateFn = func(_ v395.ApiEmailConfiguration) error { return nil }

		cr := withPassword(newTestEmailConfiguration())

		_, err := (&external{client: mc, kube: newFakeKube(t, newPasswordSecret("secret"))}).Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if len(mc.UpdateCalls) != 1 {
			t.Fatalf("expected 1 Update call, got %d", len(mc.UpdateCalls))
		}

		got := mc.UpdateCalls[0]
		if got.Host != "smtp.example.com" || got.Port != 587 || got.GetPassword() != "secret" {
			t.Errorf("Update() sent %+v, want the spec with the Secret password", got)
		}

		if cr.Status.AtProvider.CredentialsFingerprint == "" {
			t.Error("Update() did not record the credentials fingerprint")
		}
	})

	t.Run("WithoutPassword", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.UpdateFn = func(_ v395.ApiEmailConfiguration) error { return nil }

		cr := newTestEmailConfiguration()
		cr.Status.AtProvider.CredentialsFingerprint = "stale"

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), cr)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if len(mc.UpdateCalls) != 1 || mc.UpdateCalls[0].Password != nil {
			t.Errorf("Update() calls = %+v, want one without a password", mc.UpdateCalls)
		}

		if cr.Status.AtProvider.CredentialsFingerprint != "" {
			t.Errorf("CredentialsFingerprint = %q, want it cleared", cr.Status.AtProvider.CredentialsFingerprint)
		}
	})

	t.Run("UpdateError", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.UpdateFn = func(_ v395.ApiEmailConfiguration) error { return errors.New("update failed") }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), newTestEmailConfiguration())
		if err == nil {
			t.Error("Update() expected an error")
		}
	})

	t.Run("MissingPasswordSecret", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Update(context.Background(), withPassword(newTestEmailConfiguration()))
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

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.UpdateFn = func(_ v395.ApiEmailConfiguration) error { return nil }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Create(context.Background(), newTestEmailConfiguration())
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if len(mc.UpdateCalls) != 1 {
			t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.UpdateFn = func(_ v395.ApiEmailConfiguration) error { return errors.New("host not resolvable") }

		_, err := (&external{client: mc, kube: newFakeKube(t)}).Create(context.Background(), newTestEmailConfiguration())
		if err == nil {
			t.Error("Create() expected an error")
		}
	})
}

// TestDelete verifies Delete resets the configuration.
func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.ResetFn = func() error { return nil }

		_, err := (&external{client: mc}).Delete(context.Background(), newTestEmailConfiguration())
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		if mc.ResetCalls != 1 {
			t.Errorf("expected 1 Reset call, got %d", mc.ResetCalls)
		}
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockEmailClient()
		mc.ResetFn = func() error { return errors.New("reset failed") }

		_, err := (&external{client: mc}).Delete(context.Background(), newTestEmailConfiguration())
		if err == nil {
			t.Error("Delete() expected an error")
		}
	})
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockEmailClient()}

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

	cr := newTestEmailConfiguration()
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

	cr := newTestEmailConfiguration()
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
