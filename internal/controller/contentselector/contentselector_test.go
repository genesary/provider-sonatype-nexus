package contentselector

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
	description := "Test selector"

	tests := []struct {
		name         string
		cr           *v1alpha1.ContentSelector
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'maven2'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetContentSelectorFn = func(ctx context.Context, name string) (*security.ContentSelector, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsAndUpToDate",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:        "test-selector",
						Expression:  "format == 'maven2'",
						Description: &description,
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetContentSelectorFn = func(ctx context.Context, name string) (*security.ContentSelector, error) {
					return &security.ContentSelector{
						Name:        "test-selector",
						Expression:  "format == 'maven2'",
						Description: description,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'docker'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetContentSelectorFn = func(ctx context.Context, name string) (*security.ContentSelector, error) {
					return &security.ContentSelector{
						Name:       "test-selector",
						Expression: "format == 'maven2'",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'maven2'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.GetContentSelectorFn = func(ctx context.Context, name string) (*security.ContentSelector, error) {
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
		cr        *v1alpha1.ContentSelector
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateSuccess",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'maven2'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateContentSelectorFn = func(ctx context.Context, cs security.ContentSelector) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{Name: "test-selector"},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'maven2'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.CreateContentSelectorFn = func(ctx context.Context, cs security.ContentSelector) error {
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

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.ContentSelector
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-selector",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-selector",
					},
				},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'docker'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateContentSelectorFn = func(ctx context.Context, name string, cs security.ContentSelector) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateError",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-selector",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-selector",
					},
				},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name:       "test-selector",
						Expression: "format == 'docker'",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.UpdateContentSelectorFn = func(ctx context.Context, name string, cs security.ContentSelector) error {
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
		cr        *v1alpha1.ContentSelector
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-selector",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-selector",
					},
				},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name: "test-selector",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteContentSelectorFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-selector",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-selector",
					},
				},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name: "test-selector",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteContentSelectorFn = func(ctx context.Context, name string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr: &v1alpha1.ContentSelector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-selector",
					Annotations: map[string]string{
						"crossplane.io/external-name": "test-selector",
					},
				},
				Spec: v1alpha1.ContentSelectorSpec{
					ForProvider: v1alpha1.ContentSelectorParameters{
						Name: "test-selector",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockSecurity.DeleteContentSelectorFn = func(ctx context.Context, name string) error {
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

			e := &external{client: mc}
			err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"NilError", nil, false},
		{"404Error", errors.New("404 not found"), true},
		{"NotFoundError", errors.New("resource not found"), true},
		{"DoesNotExistError", errors.New("resource does not exist"), true},
		{"OtherError", errors.New("connection timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFound(tt.err); got != tt.want {
				t.Errorf("isNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
