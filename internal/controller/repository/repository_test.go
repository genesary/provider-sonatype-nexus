package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane-contrib/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/crossplane-contrib/provider-sonatype-nexus/test/mocks"
)

func TestRepositoryObserve(t *testing.T) {
	tests := []struct {
		name         string
		cr           *v1alpha1.Repository
		mockSetup    func(*mocks.MockClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "MavenHostedNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-releases"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetMavenHostedFn = func(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "MavenHostedExistsAndUpToDate",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-releases"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetMavenHostedFn = func(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
					return &repository.MavenHostedRepository{
						Name:   "maven-releases",
						Online: true,
						Maven: repository.Maven{
							VersionPolicy: repository.MavenVersionPolicyRelease,
							LayoutPolicy:  repository.MavenLayoutPolicyStrict,
						},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "MavenProxyNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-central"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-central",
						Format: "maven2",
						Type:   "proxy",
						Proxy: &v1alpha1.ProxyConfig{
							RemoteURL: "https://repo1.maven.org/maven2/",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetMavenProxyFn = func(ctx context.Context, name string) (*repository.MavenProxyRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "MavenGroupNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-public"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-public",
						Format: "maven2",
						Type:   "group",
						Group: &v1alpha1.GroupConfig{
							MemberNames: []string{"maven-releases", "maven-central"},
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetMavenGroupFn = func(ctx context.Context, name string) (*repository.MavenGroupRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "DockerHostedNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "docker-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "docker-hosted",
						Format: "docker",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetDockerHostedFn = func(ctx context.Context, name string) (*repository.DockerHostedRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NpmHostedNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "npm-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "npm-hosted",
						Format: "npm",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetNpmHostedFn = func(ctx context.Context, name string) (*repository.NpmHostedRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "RawHostedNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "raw-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "raw-hosted",
						Format: "raw",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.GetRawHostedFn = func(ctx context.Context, name string) (*repository.RawHostedRepository, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UnsupportedFormat",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "unsupported-repo"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "unsupported-repo",
						Format: "unsupported",
						Type:   "hosted",
					},
				},
			},
			mockSetup:    func(mc *mocks.MockClient) {},
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

func TestRepositoryCreate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.Repository
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "CreateMavenHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-releases"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateMavenHostedFn = func(ctx context.Context, repo repository.MavenHostedRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateMavenProxy",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-central"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-central",
						Format: "maven2",
						Type:   "proxy",
						Proxy: &v1alpha1.ProxyConfig{
							RemoteURL: "https://repo1.maven.org/maven2/",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateMavenProxyFn = func(ctx context.Context, repo repository.MavenProxyRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateMavenGroup",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-public"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-public",
						Format: "maven2",
						Type:   "group",
						Group: &v1alpha1.GroupConfig{
							MemberNames: []string{"maven-releases"},
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateMavenGroupFn = func(ctx context.Context, repo repository.MavenGroupRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateDockerHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "docker-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "docker-hosted",
						Format: "docker",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateDockerHostedFn = func(ctx context.Context, repo repository.DockerHostedRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateNpmHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "npm-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "npm-hosted",
						Format: "npm",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateNpmHostedFn = func(ctx context.Context, repo repository.NpmHostedRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateRawHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "raw-hosted"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "raw-hosted",
						Format: "raw",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateRawHostedFn = func(ctx context.Context, repo repository.RawHostedRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "CreateError",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "maven-releases"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.CreateMavenHostedFn = func(ctx context.Context, repo repository.MavenHostedRepository) error {
					return errors.New("create error")
				}
			},
			wantErr: true,
		},
		{
			name: "UnsupportedFormat",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{Name: "unsupported-repo"},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "unsupported-repo",
						Format: "unsupported",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {},
			wantErr:   true,
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

func TestRepositoryUpdate(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.Repository
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "UpdateMavenHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-releases",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-releases",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.UpdateMavenHostedFn = func(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateMavenProxy",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-central",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-central",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-central",
						Format: "maven2",
						Type:   "proxy",
						Proxy: &v1alpha1.ProxyConfig{
							RemoteURL: "https://repo1.maven.org/maven2/",
						},
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.UpdateMavenProxyFn = func(ctx context.Context, name string, repo repository.MavenProxyRepository) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateError",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-releases",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-releases",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.UpdateMavenHostedFn = func(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
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

func TestRepositoryDelete(t *testing.T) {
	tests := []struct {
		name      string
		cr        *v1alpha1.Repository
		mockSetup func(*mocks.MockClient)
		wantErr   bool
	}{
		{
			name: "DeleteMavenHosted",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-releases",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-releases",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.DeleteMavenHostedFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteMavenProxy",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-central",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-central",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-central",
						Format: "maven2",
						Type:   "proxy",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.DeleteMavenProxyFn = func(ctx context.Context, name string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-releases",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-releases",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.DeleteMavenHostedFn = func(ctx context.Context, name string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false, // Not found is not an error for delete
		},
		{
			name: "DeleteError",
			cr: &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: "maven-releases",
					Annotations: map[string]string{
						"crossplane.io/external-name": "maven-releases",
					},
				},
				Spec: v1alpha1.RepositorySpec{
					ForProvider: v1alpha1.RepositoryParameters{
						Name:   "maven-releases",
						Format: "maven2",
						Type:   "hosted",
					},
				},
			},
			mockSetup: func(mc *mocks.MockClient) {
				mc.MockRepository.DeleteMavenHostedFn = func(ctx context.Context, name string) error {
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

func TestRepositoryIsNotFound(t *testing.T) {
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
			err:  errors.New("resource does not exist"),
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

func TestStringSlicesEqual(t *testing.T) {
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
			if got := stringSlicesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("stringSlicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
