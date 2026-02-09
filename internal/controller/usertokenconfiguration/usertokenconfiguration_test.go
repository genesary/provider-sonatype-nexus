package usertokenconfiguration

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
		cr           *v1alpha1.UserTokenConfiguration
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "ExistsAndUpToDate",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: true,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserTokenConfigFn = func(ctx context.Context) (*security.UserTokenConfiguration, error) {
					return &security.UserTokenConfiguration{
						Enabled: true,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: true,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserTokenConfigFn = func(ctx context.Context) (*security.UserTokenConfiguration, error) {
					return &security.UserTokenConfiguration{
						Enabled: false,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: true,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserTokenConfigFn = func(ctx context.Context) (*security.UserTokenConfiguration, error) {
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
		cr        *v1alpha1.UserTokenConfiguration
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: true,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateUserTokenConfigFn = func(ctx context.Context, config security.UserTokenConfiguration) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: true,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateUserTokenConfigFn = func(ctx context.Context, config security.UserTokenConfiguration) error {
					return errors.New("update error")
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

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.UserTokenConfiguration
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr: &v1alpha1.UserTokenConfiguration{
				ObjectMeta: metav1.ObjectMeta{Name: "user-token-config"},
				Spec: v1alpha1.UserTokenConfigurationSpec{
					ForProvider: v1alpha1.UserTokenConfigurationParameters{
						Enabled: false,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateUserTokenConfigFn = func(ctx context.Context, config security.UserTokenConfiguration) error {
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
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
