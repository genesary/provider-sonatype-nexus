package emailconfiguration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mailschema "github.com/datadrivers/go-nexus-client/nexus3/schema"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	nexusclient "github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

// newTestEmailConfig returns a minimal EmailConfiguration for tests.
func newTestEmailConfig() *instancev1alpha1.EmailConfiguration {
	return &instancev1alpha1.EmailConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "singleton"},
		Spec: instancev1alpha1.EmailConfigurationSpec{
			ForProvider: instancev1alpha1.EmailConfigurationParameters{
				Enabled:     true,
				Host:        "smtp.example.com",
				Port:        587,
				FromAddress: "nexus@example.com",
			},
		},
	}
}

// newTestScheme registers instance and nexus v1alpha1 types in a new scheme.
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

	err = clientgoscheme.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(clientgo) failed: %v", err)
	}

	return s
}

// truePtr returns a pointer to true.
func truePtr() *bool {
	v := true

	return &v
}

// falsePtr returns a pointer to false.
func falsePtr() *bool {
	v := false

	return &v
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) { //nolint:maintidx // table-driven test with many sub-cases
	t.Parallel()

	tests := []struct {
		name         string
		cr           *instancev1alpha1.EmailConfiguration
		mockSetup    func(*instancemocks.MockEmailConfigurationClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "GetError",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					return nil, errors.New("nexus error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:     &enabled,
						Host:        "smtp.example.com",
						Port:        587,
						FromAddress: "nexus@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButHostDiffers",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:     &enabled,
						Host:        "other.smtp.com",
						Port:        587,
						FromAddress: "nexus@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButPortDiffers",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:     &enabled,
						Host:        "smtp.example.com",
						Port:        25,
						FromAddress: "nexus@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButEnabledDiffers",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					disabled := false

					return &mailschema.MailConfig{
						Enabled:     &disabled,
						Host:        "smtp.example.com",
						Port:        587,
						FromAddress: "nexus@example.com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButUsernameDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				user := "admin"
				cr.Spec.ForProvider.Username = &user

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true
					other := "other-user"

					return &mailschema.MailConfig{
						Enabled:     &enabled,
						Host:        "smtp.example.com",
						Port:        587,
						FromAddress: "nexus@example.com",
						Username:    &other,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButSubjectPrefixDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				pfx := "[Nexus]"
				cr.Spec.ForProvider.SubjectPrefix = &pfx

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true
					pfx := "[Other]"

					return &mailschema.MailConfig{
						Enabled:       &enabled,
						Host:          "smtp.example.com",
						Port:          587,
						FromAddress:   "nexus@example.com",
						SubjectPrefix: &pfx,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButStartTlsEnabledDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				cr.Spec.ForProvider.StartTlsEnabled = truePtr()

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:         &enabled,
						Host:            "smtp.example.com",
						Port:            587,
						FromAddress:     "nexus@example.com",
						StartTlsEnabled: falsePtr(),
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButStartTlsRequiredDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				cr.Spec.ForProvider.StartTlsRequired = truePtr()

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:          &enabled,
						Host:             "smtp.example.com",
						Port:             587,
						FromAddress:      "nexus@example.com",
						StartTlsRequired: falsePtr(),
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButSslOnConnectEnabledDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				cr.Spec.ForProvider.SslOnConnectEnabled = truePtr()

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:             &enabled,
						Host:                "smtp.example.com",
						Port:                587,
						FromAddress:         "nexus@example.com",
						SslOnConnectEnabled: falsePtr(),
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButSslIdentityCheckDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				cr.Spec.ForProvider.SslServerIdentityCheckEnabled = truePtr()

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:                       &enabled,
						Host:                          "smtp.example.com",
						Port:                          587,
						FromAddress:                   "nexus@example.com",
						SslServerIdentityCheckEnabled: falsePtr(),
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButNexusTrustStoreDiffers",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				cr.Spec.ForProvider.NexusTrustStoreEnabled = truePtr()

				return cr
			}(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
					enabled := true

					return &mailschema.MailConfig{
						Enabled:                &enabled,
						Host:                   "smtp.example.com",
						Port:                   587,
						FromAddress:            "nexus@example.com",
						NexusTrustStoreEnabled: falsePtr(),
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "DeletingReturnsAbsent",
			cr: func() *instancev1alpha1.EmailConfiguration {
				cr := newTestEmailConfig()
				now := metav1.Now()
				cr.DeletionTimestamp = &now

				return cr
			}(),
			mockSetup:    nil,
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockEmailConfigurationClient()
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

	e := &external{client: instancemocks.NewMockEmailConfigurationClient()}

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
		cr        *instancev1alpha1.EmailConfiguration
		mockSetup func(*instancemocks.MockEmailConfigurationClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.UpdateEmailConfigurationFn = func(_ context.Context, _ mailschema.MailConfig) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.UpdateEmailConfigurationFn = func(_ context.Context, _ mailschema.MailConfig) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockEmailConfigurationClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			fakeKube := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
			e := &external{client: mc, kube: fakeKube}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCreate_WrongType tests Create with wrong resource type.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockEmailConfigurationClient()}

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
		cr        *instancev1alpha1.EmailConfiguration
		mockSetup func(*instancemocks.MockEmailConfigurationClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.UpdateEmailConfigurationFn = func(_ context.Context, _ mailschema.MailConfig) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateError",
			cr:   newTestEmailConfig(),
			mockSetup: func(mc *instancemocks.MockEmailConfigurationClient) {
				mc.UpdateEmailConfigurationFn = func(_ context.Context, _ mailschema.MailConfig) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockEmailConfigurationClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			fakeKube := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
			e := &external{client: mc, kube: fakeKube}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdate_WrongType tests Update with wrong resource type.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockEmailConfigurationClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests Delete is a no-op for EmailConfiguration.
func TestDelete(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()
	mc := instancemocks.NewMockEmailConfigurationClient()

	e := &external{client: mc}

	_, err := e.Delete(context.Background(), cr)
	if err != nil {
		t.Errorf("Delete() returned unexpected error: %v", err)
	}

	if len(mc.UpdateEmailConfigurationCalls) != 0 {
		t.Error("Delete() should not call UpdateEmailConfiguration")
	}
}

// TestDisconnect tests Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: instancemocks.NewMockEmailConfigurationClient()}

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

	if err.Error() != errNotEmailConfig {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotEmailConfig)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	cr := newTestEmailConfig()
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ClusterProviderConfig",
	})

	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})
	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}

// TestConnect_GetProviderConfigError tests Connect when GetCredentials fails.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	cr := newTestEmailConfig()
	cr.UID = types.UID("test-uid-5678")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})
	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}

// TestConnect_Success tests Connect successfully creates an external client.
func TestConnect_Success(t *testing.T) {
	t.Parallel()

	creds := nexusclient.Credentials{
		URL:      "http://localhost:8081",
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshaling credentials: %v", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nexus-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"credentials": credsJSON,
		},
	}

	pc := &nexusv1alpha1.ClusterProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: nexusv1alpha1.ProviderConfigSpec{
			Credentials: nexusv1alpha1.ProviderCredentials{
				CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
					SecretRef: &xpv2.SecretKeySelector{
						SecretReference: xpv2.SecretReference{
							Name:      "nexus-credentials",
							Namespace: "default",
						},
						Key: "credentials",
					},
				},
				Source: "Secret",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(secret, pc).
		Build()

	cr := newTestEmailConfig()
	cr.UID = types.UID("test-uid-connect-success")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ClusterProviderConfig",
	})

	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ClusterProviderConfigUsage{})
	c := &connector{kube: fakeClient, usage: usage}

	emailClient, connectErr := c.Connect(context.Background(), cr)
	if connectErr != nil {
		t.Fatalf("Connect() unexpected error: %v", connectErr)
	}

	if emailClient == nil {
		t.Error("Connect() returned nil client")
	}
}

// TestCreate_WithPasswordSecret tests Create resolves a password secret.
func TestCreate_WithPasswordSecret(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "nonexistent-secret",
			Namespace: "default",
		},
		Key: "password",
	}

	mc := instancemocks.NewMockEmailConfigurationClient()
	fakeKube := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	e := &external{client: mc, kube: fakeKube}

	_, err := e.Create(context.Background(), cr)
	if err == nil {
		t.Error("Create() should fail when secret does not exist")
	}
}

// TestUpdate_WithPasswordSecret tests Update resolves a password secret.
func TestUpdate_WithPasswordSecret(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "nonexistent-secret",
			Namespace: "default",
		},
		Key: "password",
	}

	mc := instancemocks.NewMockEmailConfigurationClient()
	fakeKube := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	e := &external{client: mc, kube: fakeKube}

	_, err := e.Update(context.Background(), cr)
	if err == nil {
		t.Error("Update() should fail when secret does not exist")
	}
}

// TestObserve_AllFieldsUpToDate verifies all optional fields are checked.
func TestObserve_AllFieldsUpToDate(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()
	user := "smtp-user"
	pfx := "[Nexus]"
	cr.Spec.ForProvider.Username = &user
	cr.Spec.ForProvider.SubjectPrefix = &pfx
	cr.Spec.ForProvider.StartTlsEnabled = truePtr()
	cr.Spec.ForProvider.StartTlsRequired = falsePtr()
	cr.Spec.ForProvider.SslOnConnectEnabled = falsePtr()
	cr.Spec.ForProvider.SslServerIdentityCheckEnabled = truePtr()
	cr.Spec.ForProvider.NexusTrustStoreEnabled = falsePtr()

	mc := instancemocks.NewMockEmailConfigurationClient()
	mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
		enabled := true
		user := "smtp-user"
		pfx := "[Nexus]"
		tls := true
		tlsReq := false
		sslConnect := false
		sslID := true
		trustStore := false

		return &mailschema.MailConfig{
			Enabled:                       &enabled,
			Host:                          "smtp.example.com",
			Port:                          587,
			FromAddress:                   "nexus@example.com",
			Username:                      &user,
			SubjectPrefix:                 &pfx,
			StartTlsEnabled:               &tls,
			StartTlsRequired:              &tlsReq,
			SslOnConnectEnabled:           &sslConnect,
			SslServerIdentityCheckEnabled: &sslID,
			NexusTrustStoreEnabled:        &trustStore,
		}, nil
	}

	e := &external{client: mc}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true")
	}

	if !obs.ResourceUpToDate {
		t.Error("Observe() ResourceUpToDate = false, want true")
	}
}

// TestObserve_EmptyConfig tests Observe with an empty config response from API.
func TestObserve_EmptyConfig(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()

	mc := instancemocks.NewMockEmailConfigurationClient()
	mc.GetEmailConfigurationFn = func(_ context.Context) (*mailschema.MailConfig, error) {
		return &mailschema.MailConfig{}, nil
	}

	e := &external{client: mc}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true for existing config")
	}
}

// TestObserve_DeletingWithTimestamp verifies deletion timestamp causes
// ResourceExists to be false.
func TestObserve_DeletingWithTimestamp(t *testing.T) {
	t.Parallel()

	cr := newTestEmailConfig()
	now := metav1.NewTime(time.Now())
	cr.DeletionTimestamp = &now

	mc := instancemocks.NewMockEmailConfigurationClient()

	e := &external{client: mc}

	obs, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if obs.ResourceExists {
		t.Error("Observe() ResourceExists = true, want false for deleting resource")
	}

	if mc.GetEmailConfigurationCalls != 0 {
		t.Error("Observe() should not call GetEmailConfiguration when deleting")
	}
}
