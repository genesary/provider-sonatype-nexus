package repository

import (
	"context"
	"errors"
	"testing"

	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// mockFormatHandler implements FormatHandler for tests, returning
// configured values.
type mockFormatHandler struct {
	exists    bool
	upToDate  bool
	createErr error
	updateErr error
	deleteErr error
}

// Observe implements FormatHandler.
func (m *mockFormatHandler) Observe(_ context.Context, _ *pkgrepository.RepositoryService, _, _ string, _ *repositoryv1alpha1.Repository) (bool, bool) {
	return m.exists, m.upToDate
}

// Create implements FormatHandler.
func (m *mockFormatHandler) Create(_ context.Context, _ *pkgrepository.RepositoryService, _ *repositoryv1alpha1.Repository, _ string) error {
	return m.createErr
}

// Update implements FormatHandler.
func (m *mockFormatHandler) Update(_ context.Context, _ *pkgrepository.RepositoryService, _ string, _ *repositoryv1alpha1.Repository, _ string) error {
	return m.updateErr
}

// Delete implements FormatHandler.
func (m *mockFormatHandler) Delete(_ context.Context, _ *pkgrepository.RepositoryService, _, _ string) error {
	return m.deleteErr
}

// SupportedTypes implements FormatHandler.
func (m *mockFormatHandler) SupportedTypes() []string {
	return []string{"hosted", "proxy", "group"}
}

// newTestExternal creates a new external object for testing,
// using the provided FormatHandler.
func newTestExternal(handler FormatHandler) *external {
	return &external{
		getHandler: func(_ string) FormatHandler {
			return handler
		},
	}
}

// newTestExternalUnsupported creates a new external object for testing,
// returning nil for unsupported formats.
func newTestExternalUnsupported() *external {
	return &external{
		getHandler: func(_ string) FormatHandler {
			return nil
		},
	}
}

// newTestRepo creates a new Repository object for testing.
func newTestRepo(name, format, repoType string) *repositoryv1alpha1.Repository {
	return &repositoryv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"crossplane.io/external-name": name,
			},
		},
		Spec: repositoryv1alpha1.RepositorySpec{
			ForProvider: repositoryv1alpha1.RepositoryParameters{
				Name:   name,
				Format: format,
				Type:   repoType,
			},
		},
	}
}

// TestRepositoryObserve tests the Observe method for repositories.
func TestRepositoryObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *repositoryv1alpha1.Repository
		handler      FormatHandler
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:         "MavenHostedNotFound",
			cr:           newTestRepo("maven-releases", "maven2", "hosted"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:         "MavenHostedExistsAndUpToDate",
			cr:           newTestRepo("maven-releases", "maven2", "hosted"),
			handler:      &mockFormatHandler{exists: true, upToDate: true},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name:         "MavenProxyNotFound",
			cr:           newTestRepo("maven-central", "maven2", "proxy"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:         "MavenGroupNotFound",
			cr:           newTestRepo("maven-public", "maven2", "group"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:         "DockerHostedNotFound",
			cr:           newTestRepo("docker-hosted", "docker", "hosted"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:         "NpmHostedNotFound",
			cr:           newTestRepo("npm-hosted", "npm", "hosted"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:         "RawHostedNotFound",
			cr:           newTestRepo("raw-hosted", "raw", "hosted"),
			handler:      &mockFormatHandler{exists: false, upToDate: false},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", "hosted"),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var e *external
			if tt.handler == nil {
				e = newTestExternalUnsupported()
			} else {
				e = newTestExternal(tt.handler)
			}

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

// TestRepositoryCreate tests the Create method for repositories.
func TestRepositoryCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cr      *repositoryv1alpha1.Repository
		handler FormatHandler
		wantErr bool
	}{
		{
			name:    "CreateMavenHosted",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateMavenProxy",
			cr:      newTestRepo("maven-central", "maven2", "proxy"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateMavenGroup",
			cr:      newTestRepo("maven-public", "maven2", "group"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateDockerHosted",
			cr:      newTestRepo("docker-hosted", "docker", "hosted"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateNpmHosted",
			cr:      newTestRepo("npm-hosted", "npm", "hosted"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateRawHosted",
			cr:      newTestRepo("raw-hosted", "raw", "hosted"),
			handler: &mockFormatHandler{createErr: nil},
			wantErr: false,
		},
		{
			name:    "CreateError",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{createErr: errors.New("create error")},
			wantErr: true,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", "hosted"),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var e *external
			if tt.handler == nil {
				e = newTestExternalUnsupported()
			} else {
				e = newTestExternal(tt.handler)
			}

			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRepositoryUpdate tests the Update method for repositories.
func TestRepositoryUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cr      *repositoryv1alpha1.Repository
		handler FormatHandler
		wantErr bool
	}{
		{
			name:    "UpdateMavenHosted",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{updateErr: nil},
			wantErr: false,
		},
		{
			name:    "UpdateMavenProxy",
			cr:      newTestRepo("maven-central", "maven2", "proxy"),
			handler: &mockFormatHandler{updateErr: nil},
			wantErr: false,
		},
		{
			name:    "UpdateError",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{updateErr: errors.New("update error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var e *external
			if tt.handler == nil {
				e = newTestExternalUnsupported()
			} else {
				e = newTestExternal(tt.handler)
			}

			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRepositoryDelete tests the Delete method for repositories.
func TestRepositoryDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cr      *repositoryv1alpha1.Repository
		handler FormatHandler
		wantErr bool
	}{
		{
			name:    "DeleteMavenHosted",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{deleteErr: nil},
			wantErr: false,
		},
		{
			name:    "DeleteMavenProxy",
			cr:      newTestRepo("maven-central", "maven2", "proxy"),
			handler: &mockFormatHandler{deleteErr: nil},
			wantErr: false,
		},
		{
			name:    "DeleteNotFound",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{deleteErr: errors.New("404 not found")},
			wantErr: false,
		},
		{
			name:    "DeleteError",
			cr:      newTestRepo("maven-releases", "maven2", "hosted"),
			handler: &mockFormatHandler{deleteErr: errors.New("connection error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var e *external
			if tt.handler == nil {
				e = newTestExternalUnsupported()
			} else {
				e = newTestExternal(tt.handler)
			}

			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRepositoryIsNotFound tests the helpers.IsNotFound function.
func TestRepositoryIsNotFound(t *testing.T) {
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
			t.Parallel()

			if got := helpers.IsNotFound(tt.err); got != tt.want {
				t.Errorf("helpers.IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStringSlicesEqual tests helpers.AreStringSlicesEqual.
func TestStringSlicesEqual(t *testing.T) {
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

			if got := helpers.AreStringSlicesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("helpers.AreStringSlicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
