package privilege

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/test/mocks"
)

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	domain := "domain"

	tests := []struct {
		name         string
		cr           *v1alpha1.Privilege
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "ApplicationNotFound",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:    "test-priv",
						Type:    "application",
						Domain:  &domain,
						Actions: []string{"read"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetPrivilegeFn = func(ctx context.Context, name string) (*security.Privilege, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ApplicationExistsAndUpToDate",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:    "test-priv",
						Type:    "application",
						Domain:  &domain,
						Actions: []string{"read"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetPrivilegeFn = func(ctx context.Context, name string) (*security.Privilege, error) {
					return &security.Privilege{
						Name:    "test-priv",
						Type:    "application",
						Domain:  "domain",
						Actions: []string{"read"},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name: "test-priv",
						Type: "application",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetPrivilegeFn = func(ctx context.Context, name string) (*security.Privilege, error) {
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
			t.Parallel()

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

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	domain := "domain"
	format := "maven2"
	repo := "my-repo"
	pattern := "nexus:*"

	tests := []struct {
		name      string
		cr        *v1alpha1.Privilege
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateApplicationSuccess",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:    "test-priv",
						Type:    "application",
						Domain:  &domain,
						Actions: []string{"read"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreatePrivilegeApplicationFn = func(ctx context.Context, p security.PrivilegeApplication) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateRepositoryViewSuccess",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:       "test-priv",
						Type:       "repository-view",
						Format:     &format,
						Repository: &repo,
						Actions:    []string{"read"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreatePrivilegeRepositoryViewFn = func(ctx context.Context, p security.PrivilegeRepositoryView) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateWildcardSuccess",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:    "test-priv",
						Type:    "wildcard",
						Pattern: &pattern,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreatePrivilegeWildcardFn = func(ctx context.Context, p security.PrivilegeWildcard) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{Name: "test-priv"},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name:    "test-priv",
						Type:    "application",
						Domain:  &domain,
						Actions: []string{"read"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreatePrivilegeApplicationFn = func(ctx context.Context, p security.PrivilegeApplication) error {
					return errors.New("create error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.Privilege
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-priv",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-priv",
					},
				},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name: "test-priv",
						Type: "application",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeletePrivilegeFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: &v1alpha1.Privilege{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-priv",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-priv",
					},
				},
				Spec: v1alpha1.PrivilegeSpec{
					ForProvider: v1alpha1.PrivilegeParameters{
						Name: "test-priv",
						Type: "application",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeletePrivilegeFn = func(ctx context.Context, name string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := mocks.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
