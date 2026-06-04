package role

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/test/mocks"
)

// TestRoleObserve tests the Observe method for roles.
func TestRoleObserve(t *testing.T) {
	t.Parallel()

	testDescription := "Test role description"

	tests := []struct {
		name         string
		cr           *v1alpha1.Role
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "RoleNotFound",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:   "test-role",
						Name: "Test Role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetRoleFn = func(ctx context.Context, id string) (*security.Role, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "RoleExistsAndUpToDate",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:          "test-role",
						Name:        "Test Role",
						Description: &testDescription,
						Privileges:  []string{"nx-repository-view-*-*-*"},
						Roles:       []string{},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetRoleFn = func(ctx context.Context, id string) (*security.Role, error) {
					return &security.Role{
						ID:          "test-role",
						Name:        "Test Role",
						Description: testDescription,
						Privileges:  []string{"nx-repository-view-*-*-*"},
						Roles:       []string{},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "RoleExistsButOutdated",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:         "test-role",
						Name:       "Test Role",
						Privileges: []string{"nx-repository-view-*-*-*", "nx-admin"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetRoleFn = func(ctx context.Context, id string) (*security.Role, error) {
					return &security.Role{
						ID:         "test-role",
						Name:       "Test Role",
						Privileges: []string{"nx-repository-view-*-*-*"}, // Different privileges
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetRoleError",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:   "test-role",
						Name: "Test Role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetRoleFn = func(ctx context.Context, id string) (*security.Role, error) {
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

			if !tt.wantErr {
				if obs.ResourceExists != tt.wantExists {
					t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
				}

				if obs.ResourceUpToDate != tt.wantUpToDate {
					t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
				}
			}
		})
	}
}

// TestRoleCreate tests the Create method for roles.
func TestRoleCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.Role
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "CreateRoleSuccess",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:         "test-role",
						Name:       "Test Role",
						Privileges: []string{"nx-repository-view-*-*-*"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateRoleFn = func(ctx context.Context, role security.Role) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockSecurity.CreateRoleCalls) != 1 {
					t.Errorf("Expected 1 CreateRole call, got %d", len(mc.MockSecurity.CreateRoleCalls))
				}

				if mc.MockSecurity.CreateRoleCalls[0].ID != "test-role" {
					t.Errorf("CreateRole called with wrong ID: %s", mc.MockSecurity.CreateRoleCalls[0].ID)
				}
			},
		},
		{
			name: "CreateRoleError",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:   "test-role",
						Name: "Test Role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateRoleFn = func(ctx context.Context, role security.Role) error {
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

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestRoleUpdate tests the Update method for roles.
func TestRoleUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.Role
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "UpdateRoleSuccess",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-role",
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:         "test-role",
						Name:       "Test Role Updated",
						Privileges: []string{"nx-repository-view-*-*-*"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateRoleFn = func(ctx context.Context, id string, role security.Role) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockSecurity.UpdateRoleCalls) != 1 {
					t.Errorf("Expected 1 UpdateRole call, got %d", len(mc.MockSecurity.UpdateRoleCalls))
				}

				if mc.MockSecurity.UpdateRoleCalls[0].ID != "test-role" {
					t.Errorf("UpdateRole called with wrong ID: %s", mc.MockSecurity.UpdateRoleCalls[0].ID)
				}
			},
		},
		{
			name: "UpdateRoleError",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-role",
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:   "test-role",
						Name: "Test Role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateRoleFn = func(ctx context.Context, id string, role security.Role) error {
					return errors.New("update error")
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
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestRoleDelete tests the Delete method for roles.
func TestRoleDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *v1alpha1.Role
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "DeleteRoleSuccess",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-role",
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID: "test-role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteRoleFn = func(ctx context.Context, id string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				t.Helper()

				if len(mc.MockSecurity.DeleteRoleCalls) != 1 {
					t.Errorf("Expected 1 DeleteRole call, got %d", len(mc.MockSecurity.DeleteRoleCalls))
				}

				if mc.MockSecurity.DeleteRoleCalls[0] != "test-role" {
					t.Errorf("DeleteRole called with wrong ID: %s", mc.MockSecurity.DeleteRoleCalls[0])
				}
			},
		},
		{
			name: "DeleteRoleNotFound",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-role",
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID: "test-role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteRoleFn = func(ctx context.Context, id string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false, // Not found is not an error for delete
		},
		{
			name: "DeleteRoleError",
			cr: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-role",
					},
				},
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID: "test-role",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteRoleFn = func(ctx context.Context, id string) error {
					return errors.New("connection error")
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
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, mc)
			}
		})
	}
}

// TestGenerateRole tests the generateRole function.
func TestGenerateRole(t *testing.T) {
	t.Parallel()

	testDescription := "Test role description"

	tests := []struct {
		name string
		cr   *v1alpha1.Role
		want security.Role
	}{
		{
			name: "BasicRole",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:         "test-role",
						Name:       "Test Role",
						Privileges: []string{"nx-repository-view-*-*-*"},
						Roles:      []string{"nx-admin"},
					},
				},
			},
			want: security.Role{
				ID:         "test-role",
				Name:       "Test Role",
				Privileges: []string{"nx-repository-view-*-*-*"},
				Roles:      []string{"nx-admin"},
			},
		},
		{
			name: "RoleWithDescription",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:          "test-role",
						Name:        "Test Role",
						Description: &testDescription,
						Privileges:  []string{"nx-repository-view-*-*-*"},
					},
				},
			},
			want: security.Role{
				ID:          "test-role",
				Name:        "Test Role",
				Description: testDescription,
				Privileges:  []string{"nx-repository-view-*-*-*"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateRole(tt.cr)
			if got.ID != tt.want.ID {
				t.Errorf("generateRole() ID = %v, want %v", got.ID, tt.want.ID)
			}

			if got.Name != tt.want.Name {
				t.Errorf("generateRole() Name = %v, want %v", got.Name, tt.want.Name)
			}

			if got.Description != tt.want.Description {
				t.Errorf("generateRole() Description = %v, want %v", got.Description, tt.want.Description)
			}
		})
	}
}

// TestIsRoleUpToDate tests the isRoleUpToDate function.
func TestIsRoleUpToDate(t *testing.T) {
	t.Parallel()

	testDescription := "Test description"

	tests := []struct {
		name string
		cr   *v1alpha1.Role
		role *security.Role
		want bool
	}{
		{
			name: "UpToDate",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:          "test-role",
						Name:        "Test Role",
						Description: &testDescription,
						Privileges:  []string{"nx-admin"},
						Roles:       []string{},
					},
				},
			},
			role: &security.Role{
				ID:          "test-role",
				Name:        "Test Role",
				Description: testDescription,
				Privileges:  []string{"nx-admin"},
				Roles:       []string{},
			},
			want: true,
		},
		{
			name: "DifferentName",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:   "test-role",
						Name: "Test Role",
					},
				},
			},
			role: &security.Role{
				ID:   "test-role",
				Name: "Different Name",
			},
			want: false,
		},
		{
			name: "DifferentPrivileges",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:         "test-role",
						Name:       "Test Role",
						Privileges: []string{"nx-admin", "nx-developer"},
					},
				},
			},
			role: &security.Role{
				ID:         "test-role",
				Name:       "Test Role",
				Privileges: []string{"nx-admin"},
			},
			want: false,
		},
		{
			name: "DifferentRoles",
			cr: &v1alpha1.Role{
				Spec: v1alpha1.RoleSpec{
					ForProvider: v1alpha1.RoleParameters{
						ID:    "test-role",
						Name:  "Test Role",
						Roles: []string{"role1", "role2"},
					},
				},
			},
			role: &security.Role{
				ID:    "test-role",
				Name:  "Test Role",
				Roles: []string{"role1"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isRoleUpToDate(tt.cr, tt.role); got != tt.want {
				t.Errorf("isRoleUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRoleIsNotFound tests the isNotFound function for roles.
func TestRoleIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NilError",
			err:  nil,
			want: false,
		},
		{
			name: "404Error",
			err:  errors.New("404 not found"),
			want: true,
		},
		{
			name: "NotFoundError",
			err:  errors.New("resource not found"),
			want: true,
		},
		{
			name: "DoesNotExistError",
			err:  errors.New("role does not exist"),
			want: true,
		},
		{
			name: "OtherError",
			err:  errors.New("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isNotFound(tt.err); got != tt.want {
				t.Errorf("isNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRoleStringSlicesEqual tests the stringSlicesEqual function for roles.
func TestRoleStringSlicesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{
			name: "BothEmpty",
			a:    []string{},
			b:    []string{},
			want: true,
		},
		{
			name: "BothNil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "EqualSlices",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: true,
		},
		{
			name: "DifferentLength",
			a:    []string{"a", "b"},
			b:    []string{"a", "b", "c"},
			want: false,
		},
		{
			name: "DifferentContent",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "x", "c"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := stringSlicesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("stringSlicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
