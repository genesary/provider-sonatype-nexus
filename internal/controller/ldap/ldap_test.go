package ldap

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
		cr           *v1alpha1.LDAP
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ldap"},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name:     "test-ldap",
						Protocol: "ldap",
						Host:     "ldap.example.com",
						Port:     389,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetLDAPFn = func(ctx context.Context, name string) (*security.LDAP, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ldap"},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name:     "test-ldap",
						Protocol: "ldap",
						Host:     "ldap.example.com",
						Port:     389,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetLDAPFn = func(ctx context.Context, name string) (*security.LDAP, error) {
					return &security.LDAP{
						Name:     "test-ldap",
						Protocol: "ldap",
						Host:     "ldap.example.com",
						Port:     389,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ldap"},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name: "test-ldap",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetLDAPFn = func(ctx context.Context, name string) (*security.LDAP, error) {
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
		cr        *v1alpha1.LDAP
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ldap"},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name:     "test-ldap",
						Protocol: "ldap",
						Host:     "ldap.example.com",
						Port:     389,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateLDAPFn = func(ctx context.Context, ldap security.LDAP) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ldap"},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name: "test-ldap",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateLDAPFn = func(ctx context.Context, ldap security.LDAP) error {
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
		cr        *v1alpha1.LDAP
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ldap",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-ldap",
					},
				},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name: "test-ldap",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteLDAPFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: &v1alpha1.LDAP{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ldap",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-ldap",
					},
				},
				Spec: v1alpha1.LDAPSpec{
					ForProvider: v1alpha1.LDAPParameters{
						Name: "test-ldap",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteLDAPFn = func(ctx context.Context, name string) error {
					return errors.New("404 not found")
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
