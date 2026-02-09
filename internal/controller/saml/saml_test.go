package saml

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/test/mocks"
)

func TestObserve(t *testing.T) {
	tests := []struct {
		name         string
		cr           *v1alpha1.SAML
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "ExistsAndUpToDate",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata:    "metadata-xml",
						EntityId:       "nexus-entity",
						UsernameAttribute: "username",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetSAMLFn = func(ctx context.Context) (*security.SAML, error) {
					return &security.SAML{
						IdpMetadata:       "metadata-xml",
						EntityId:          "nexus-entity",
						UsernameAttribute: "username",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "NotFound",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata: "metadata-xml",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetSAMLFn = func(ctx context.Context) (*security.SAML, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata: "metadata-xml",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetSAMLFn = func(ctx context.Context) (*security.SAML, error) {
					return nil, errors.New("connection error")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockClient()
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

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.SAML
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata:    "metadata-xml",
						EntityId:       "nexus-entity",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateSAMLFn = func(ctx context.Context, saml security.SAML) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata: "metadata-xml",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateSAMLFn = func(ctx context.Context, saml security.SAML) error {
					return errors.New("create error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.SAML
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: &v1alpha1.SAML{
				ObjectMeta: metav1.ObjectMeta{Name: "saml"},
				Spec: v1alpha1.SAMLSpec{
					ForProvider: v1alpha1.SAMLParameters{
						IdpMetadata: "metadata-xml",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteSAMLFn = func(ctx context.Context) error {
					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
