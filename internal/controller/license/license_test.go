package license

import (
	"context"
	"errors"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/test/mocks"
)

func newLicenseCR() *v1alpha1.License {
	return &v1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nexus-license",
			Annotations: map[string]string{
				annotationContentHash: computeHash([]byte("license-data")),
			},
		},
		Spec: v1alpha1.LicenseSpec{
			ForProvider: v1alpha1.LicenseParameters{
				LicenseSecretRef: xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{
						Name:      "license-secret",
						Namespace: "default",
					},
					Key: "license.lic",
				},
			},
		},
	}
}

func newLicenseSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "license-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"license.lic": []byte("license-data"),
		},
	}
}

func newFakeKube(objs ...runtime.Object) *external {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	clientObjs := make([]runtime.Object, len(objs))
	copy(clientObjs, objs)

	mc := mocks.NewMockClient()
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(clientObjs...).Build()
	return &external{client: mc, kube: kubeClient}
}

func TestObserve(t *testing.T) {
	tests := []struct {
		name         string
		cr           *v1alpha1.License
		secrets      []runtime.Object
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:    "NoLicenseInstalled",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.GetLicenseFn = func(ctx context.Context) (*nexus.LicenseDetails, error) {
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:    "LicenseExistsAndUpToDate",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.GetLicenseFn = func(ctx context.Context) (*nexus.LicenseDetails, error) {
					return &nexus.LicenseDetails{
						LicenseType:    "PRO",
						ExpirationDate: "2025-12-31",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "LicenseExistsButSecretChanged",
			cr: func() *v1alpha1.License {
				cr := newLicenseCR()
				cr.Annotations[annotationContentHash] = "old-hash-value"
				return cr
			}(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.GetLicenseFn = func(ctx context.Context) (*nexus.LicenseDetails, error) {
					return &nexus.LicenseDetails{
						LicenseType: "PRO",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:    "APIError",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.GetLicenseFn = func(ctx context.Context) (*nexus.LicenseDetails, error) {
					return nil, errors.New("connection error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name:    "NotFoundError",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.GetLicenseFn = func(ctx context.Context) (*nexus.LicenseDetails, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newFakeKube(tt.secrets...)
			mc := e.client.(*mocks.MockClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

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

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.License
		secrets   []runtime.Object
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name:    "CreateSuccess",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.InstallLicenseFn = func(ctx context.Context, licenseBytes []byte) (*nexus.LicenseDetails, error) {
					return &nexus.LicenseDetails{LicenseType: "PRO"}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "CreateAPIError",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.InstallLicenseFn = func(ctx context.Context, licenseBytes []byte) (*nexus.LicenseDetails, error) {
					return nil, errors.New("install error")
				}
			},
			wantErr: true,
		},
		{
			name: "SecretNotFound",
			cr:   newLicenseCR(),
			// no secrets provided
			secrets: []runtime.Object{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newFakeKube(tt.secrets...)
			mc := e.client.(*mocks.MockClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.License
		secrets   []runtime.Object
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name:    "UpdateSuccess",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.InstallLicenseFn = func(ctx context.Context, licenseBytes []byte) (*nexus.LicenseDetails, error) {
					return &nexus.LicenseDetails{LicenseType: "PRO"}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "UpdateAPIError",
			cr:      newLicenseCR(),
			secrets: []runtime.Object{newLicenseSecret()},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.InstallLicenseFn = func(ctx context.Context, licenseBytes []byte) (*nexus.LicenseDetails, error) {
					return nil, errors.New("install error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newFakeKube(tt.secrets...)
			mc := e.client.(*mocks.MockClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.License
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr:   newLicenseCR(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.DeleteLicenseFn = func(ctx context.Context) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr:   newLicenseCR(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.DeleteLicenseFn = func(ctx context.Context) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteAPIError",
			cr:   newLicenseCR(),
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockLicense.DeleteLicenseFn = func(ctx context.Context) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newFakeKube(newLicenseSecret())
			mc := e.client.(*mocks.MockClient)
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
