package ldap

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	corev1 "k8s.io/api/core/v1"
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

// newTestLDAP returns a minimal LDAP CR for tests.
func newTestLDAP(name, host string) *iamv1alpha1.LDAP {
	return &iamv1alpha1.LDAP{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: iamv1alpha1.LDAPSpec{
			ForProvider: iamv1alpha1.LDAPParameters{
				Name:       name,
				Protocol:   "ldap",
				Host:       host,
				Port:       389,
				SearchBase: "dc=example,dc=com",
				AuthScheme: "none",
				UserBaseDN: "ou=people,dc=example,dc=com",
			},
		},
	}
}

// newTestScheme registers iam, nexus v1alpha1, and corev1 types.
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

	err = corev1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(corev1) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *iamv1alpha1.LDAP
		mockSetup    func(*iammocks.MockLDAPClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.GetFn = func(_ string) (*security.LDAP, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.GetFn = func(_ string) (*security.LDAP, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "NilLDAPReturned",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.GetFn = func(_ string) (*security.LDAP, error) {
					//nolint:nilnil // intentionally testing nil LDAP with nil error
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.GetFn = func(_ string) (*security.LDAP, error) {
					return &security.LDAP{
						Name:       "corp-ldap",
						Protocol:   "ldap",
						Host:       "ldap.example.com",
						Port:       389,
						SearchBase: "dc=example,dc=com",
						AuthSchema: "none",
						UserBaseDN: "ou=people,dc=example,dc=com",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr:   newTestLDAP("corp-ldap", "ldap-new.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.GetFn = func(_ string) (*security.LDAP, error) {
					return &security.LDAP{
						Name:       "corp-ldap",
						Protocol:   "ldap",
						Host:       "ldap-old.example.com",
						Port:       389,
						SearchBase: "dc=example,dc=com",
						AuthSchema: "none",
						UserBaseDN: "ou=people,dc=example,dc=com",
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

			mc := iammocks.NewMockLDAPClient()
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

	e := &external{client: iammocks.NewMockLDAPClient()}

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
		cr        *iamv1alpha1.LDAP
		mockSetup func(*iammocks.MockLDAPClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockLDAPClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.CreateFn = func(_ security.LDAP) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockLDAPClient) {
				t.Helper()

				if len(mc.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateCalls))
				}

				if mc.CreateCalls[0].Host != "ldap.example.com" {
					t.Errorf("wrong host: %v", mc.CreateCalls[0].Host)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.CreateFn = func(_ security.LDAP) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockLDAPClient()
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

	e := &external{client: iammocks.NewMockLDAPClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

// TestCreate_WithPassword tests Create when an auth password secret ref is set.
func TestCreate_WithPassword(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("s3cr3t"),
		},
	}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	mc := iammocks.NewMockLDAPClient()
	mc.CreateFn = func(ldap security.LDAP) error {
		if ldap.AuthPassword != "s3cr3t" {
			return errors.New("unexpected password value")
		}

		return nil
	}

	cr := newTestLDAP("corp-ldap", "ldap.example.com")
	cr.Spec.ForProvider.AuthPasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{
			Name:      "ldap-secret",
			Namespace: "default",
		},
		Key: "password",
	}

	e := &external{client: mc, kube: kubeClient}

	_, err := e.Create(context.Background(), cr)
	if err != nil {
		t.Errorf("Create() with password returned unexpected error: %v", err)
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *iamv1alpha1.LDAP
		mockSetup func(*iammocks.MockLDAPClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockLDAPClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.UpdateFn = func(_ string, _ security.LDAP) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockLDAPClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.UpdateFn = func(_ string, _ security.LDAP) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockLDAPClient()
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

	e := &external{client: iammocks.NewMockLDAPClient()}

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
		cr        *iamv1alpha1.LDAP
		mockSetup func(*iammocks.MockLDAPClient)
		wantErr   bool
		validate  func(*testing.T, *iammocks.MockLDAPClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *iammocks.MockLDAPClient) {
				t.Helper()

				if len(mc.DeleteCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeleteCalls))
				}

				if mc.DeleteCalls[0] != "corp-ldap" {
					t.Errorf("wrong name passed to Delete: %v", mc.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestLDAP("corp-ldap", "ldap.example.com"),
			mockSetup: func(mc *iammocks.MockLDAPClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := iammocks.NewMockLDAPClient()
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

	e := &external{client: iammocks.NewMockLDAPClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: iammocks.NewMockLDAPClient()}

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

	if err.Error() != errNotLDAP {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotLDAP)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestLDAP("corp-ldap", "ldap.example.com")
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

	cr := newTestLDAP("corp-ldap", "ldap.example.com")
	cr.UID = types.UID("test-uid-1234")
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
