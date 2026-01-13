package user

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/test/mocks"
)

func TestUserObserve(t *testing.T) {
	activeStatus := "active"

	tests := []struct {
		name         string
		cr           *v1alpha1.User
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "UserNotFound",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "test-user"},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserFn = func(ctx context.Context, id string) (*security.User, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UserExistsAndUpToDate",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "test-user"},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Status:       &activeStatus,
						Roles:        []string{"nx-admin"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserFn = func(ctx context.Context, id string) (*security.User, error) {
					return &security.User{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Status:       "active",
						Roles:        []string{"nx-admin"},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "UserExistsButOutdated",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "test-user"},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Roles:        []string{"nx-admin", "nx-developer"},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserFn = func(ctx context.Context, id string) (*security.User, error) {
					return &security.User{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Roles:        []string{"nx-admin"}, // Different roles
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetUserError",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "test-user"},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetUserFn = func(ctx context.Context, id string) (*security.User, error) {
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

			e := &external{client: mc, kube: nil}
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

func TestUserDelete(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.User
		mockSetup func(*mocks.MockClient)
		wantErr   bool
		validate  func(*testing.T, *mocks.MockClient)
	}{
		{
			name: "DeleteUserSuccess",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					Annotations: map[string]string{
						"crossplane.io/external-name": "testuser",
					},
				},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID: "testuser",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteUserFn = func(ctx context.Context, id string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *mocks.MockClient) {
				if len(mc.MockSecurity.DeleteUserCalls) != 1 {
					t.Errorf("Expected 1 DeleteUser call, got %d", len(mc.MockSecurity.DeleteUserCalls))
				}
				if mc.MockSecurity.DeleteUserCalls[0] != "testuser" {
					t.Errorf("DeleteUser called with wrong ID: %s", mc.MockSecurity.DeleteUserCalls[0])
				}
			},
		},
		{
			name: "DeleteUserNotFound",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					Annotations: map[string]string{
						"crossplane.io/external-name": "testuser",
					},
				},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID: "testuser",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteUserFn = func(ctx context.Context, id string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false, // Not found is not an error for delete
		},
		{
			name: "DeleteUserError",
			cr: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					Annotations: map[string]string{
						"crossplane.io/external-name": "testuser",
					},
				},
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID: "testuser",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteUserFn = func(ctx context.Context, id string) error {
					return errors.New("connection error")
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

			e := &external{client: mc, kube: nil}
			err := e.Delete(context.Background(), tt.cr)

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

func TestGenerateUser(t *testing.T) {
	activeStatus := "active"
	defaultSource := "default"

	tests := []struct {
		name     string
		cr       *v1alpha1.User
		password string
		want     security.User
	}{
		{
			name: "BasicUser",
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Roles:        []string{"nx-admin"},
					},
				},
			},
			password: "password123",
			want: security.User{
				UserID:       "testuser",
				FirstName:    "Test",
				LastName:     "User",
				EmailAddress: "test@example.com",
				Password:     "password123",
				Status:       "active",
				Source:       "default",
				Roles:        []string{"nx-admin"},
			},
		},
		{
			name: "UserWithCustomStatus",
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Status:       &activeStatus,
						Source:       &defaultSource,
						Roles:        []string{"nx-admin", "nx-developer"},
					},
				},
			},
			password: "",
			want: security.User{
				UserID:       "testuser",
				FirstName:    "Test",
				LastName:     "User",
				EmailAddress: "test@example.com",
				Password:     "",
				Status:       "active",
				Source:       "default",
				Roles:        []string{"nx-admin", "nx-developer"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateUser(tt.cr, tt.password)
			if got.UserID != tt.want.UserID {
				t.Errorf("generateUser() UserID = %v, want %v", got.UserID, tt.want.UserID)
			}
			if got.FirstName != tt.want.FirstName {
				t.Errorf("generateUser() FirstName = %v, want %v", got.FirstName, tt.want.FirstName)
			}
			if got.LastName != tt.want.LastName {
				t.Errorf("generateUser() LastName = %v, want %v", got.LastName, tt.want.LastName)
			}
			if got.EmailAddress != tt.want.EmailAddress {
				t.Errorf("generateUser() EmailAddress = %v, want %v", got.EmailAddress, tt.want.EmailAddress)
			}
			if got.Password != tt.want.Password {
				t.Errorf("generateUser() Password = %v, want %v", got.Password, tt.want.Password)
			}
			if got.Status != tt.want.Status {
				t.Errorf("generateUser() Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.Source != tt.want.Source {
				t.Errorf("generateUser() Source = %v, want %v", got.Source, tt.want.Source)
			}
		})
	}
}

func TestIsUserUpToDate(t *testing.T) {
	activeStatus := "active"

	tests := []struct {
		name string
		cr   *v1alpha1.User
		user *security.User
		want bool
	}{
		{
			name: "UpToDate",
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Status:       &activeStatus,
						Roles:        []string{"nx-admin"},
					},
				},
			},
			user: &security.User{
				UserID:       "testuser",
				FirstName:    "Test",
				LastName:     "User",
				EmailAddress: "test@example.com",
				Status:       "active",
				Roles:        []string{"nx-admin"},
			},
			want: true,
		},
		{
			name: "DifferentFirstName",
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
					},
				},
			},
			user: &security.User{
				UserID:       "testuser",
				FirstName:    "Different",
				LastName:     "User",
				EmailAddress: "test@example.com",
			},
			want: false,
		},
		{
			name: "DifferentRoles",
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: v1alpha1.UserParameters{
						UserID:       "testuser",
						FirstName:    "Test",
						LastName:     "User",
						EmailAddress: "test@example.com",
						Roles:        []string{"nx-admin", "nx-developer"},
					},
				},
			},
			user: &security.User{
				UserID:       "testuser",
				FirstName:    "Test",
				LastName:     "User",
				EmailAddress: "test@example.com",
				Roles:        []string{"nx-admin"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUserUpToDate(tt.cr, tt.user); got != tt.want {
				t.Errorf("isUserUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserIsNotFound(t *testing.T) {
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
			name: "OtherError",
			err:  errors.New("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFound(tt.err); got != tt.want {
				t.Errorf("isNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
