package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	pkgrepository "github.com/datadrivers/go-nexus-client/nexus3/pkg/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// mockHandler implements formatHandler, returning configured values.
type mockHandler struct {
	exists     bool
	upToDate   bool
	fields     map[string]any
	observeErr error
	createErr  error
	updateErr  error
	deleteErr  error

	// observedName records the repository name the controller looked up.
	observedName string
}

// Observe implements formatHandler.
func (m *mockHandler) Observe(_ *pkgrepository.RepositoryService, name, _ string, _ desiredRepo) (repoState, error) {
	m.observedName = name

	return repoState{exists: m.exists, upToDate: m.upToDate, fields: m.fields}, m.observeErr
}

// Create implements formatHandler.
func (m *mockHandler) Create(_ *pkgrepository.RepositoryService, _ string, _ desiredRepo) error {
	return m.createErr
}

// Update implements formatHandler.
func (m *mockHandler) Update(_ *pkgrepository.RepositoryService, name, _ string, _ desiredRepo) error {
	m.observedName = name

	return m.updateErr
}

// Delete implements formatHandler.
func (m *mockHandler) Delete(_ *pkgrepository.RepositoryService, name, _ string) error {
	m.observedName = name

	return m.deleteErr
}

// SupportedTypes implements formatHandler.
func (m *mockHandler) SupportedTypes() []string {
	return []string{repoTypeHosted, repoTypeProxy, repoTypeGroup}
}

// newTestExternal creates an external client backed by the given handler. A
// nil handler stands for an unregistered format.
func newTestExternal(handler formatHandler) *external {
	return &external{
		baseURL: "https://nexus.example.com",
		getHandler: func(_ string) formatHandler {
			if handler == nil {
				return nil
			}

			return handler
		},
	}
}

// newTestRepo creates a Repository whose Kubernetes name matches the Nexus
// repository name.
func newTestRepo(name, format, repoType string) *repositoryv1alpha1.Repository {
	return newTestRepoNamed(name, name, format, repoType)
}

// newTestRepoNamed creates a Repository whose Kubernetes name and Nexus
// repository name may differ.
func newTestRepoNamed(objectName, repoName, format, repoType string) *repositoryv1alpha1.Repository {
	return &repositoryv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectName,
			Annotations: map[string]string{
				"crossplane.io/external-name": objectName,
			},
		},
		Spec: repositoryv1alpha1.RepositorySpec{
			ForProvider: repositoryv1alpha1.RepositoryParameters{
				Name:   repoName,
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
		handler      *mockHandler
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name:    "NotFound",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{exists: false},
		},
		{
			name:         "ExistsAndUpToDate",
			cr:           newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler:      &mockHandler{exists: true, upToDate: true},
			wantExists:   true,
			wantUpToDate: true,
		},
		{
			name:       "ExistsButDrifted",
			cr:         newTestRepo("apt-hosted", "apt", repoTypeHosted),
			handler:    &mockHandler{exists: true, upToDate: false},
			wantExists: true,
		},
		{
			// A failure to read the repository must not be reported as a
			// missing repository: Crossplane would try to create a repository
			// that already exists.
			name:    "ReadFailureIsAnError",
			cr:      newTestRepo("cargo-hosted", "cargo", repoTypeHosted),
			handler: &mockHandler{observeErr: errors.New("connection refused")},
			wantErr: true,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", repoTypeHosted),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handler formatHandler
			if tt.handler != nil {
				handler = tt.handler
			}

			obs, err := newTestExternal(handler).Observe(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
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

// TestRepositoryObserveUsesSpecName checks that the Nexus repository name comes
// from spec.forProvider.name rather than from the external-name annotation
// crossplane-runtime seeds from metadata.name.
func TestRepositoryObserveUsesSpecName(t *testing.T) {
	t.Parallel()

	cr := newTestRepoNamed("my-apt-repo", "apt-hosted", "apt", repoTypeHosted)
	handler := &mockHandler{exists: true, upToDate: true, fields: map[string]any{"name": "apt-hosted"}}

	obs, err := newTestExternal(handler).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if !obs.ResourceExists {
		t.Error("Observe() ResourceExists = false, want true")
	}

	if handler.observedName != "apt-hosted" {
		t.Errorf("Observe() looked up %q, want %q", handler.observedName, "apt-hosted")
	}

	if got := meta.GetExternalName(cr); got != "apt-hosted" {
		t.Errorf("external-name = %q, want %q", got, "apt-hosted")
	}

	wantURL := "https://nexus.example.com/repository/apt-hosted"
	if cr.Status.AtProvider.URL == nil || *cr.Status.AtProvider.URL != wantURL {
		t.Errorf("status.atProvider.url = %v, want %q", cr.Status.AtProvider.URL, wantURL)
	}

	if cr.Status.AtProvider.Name != "apt-hosted" {
		t.Errorf("status.atProvider.name = %q, want %q", cr.Status.AtProvider.Name, "apt-hosted")
	}
}

// TestRepositoryObserveDoesNotReportAtProviderWhenAbsent checks that a missing
// repository leaves the observed state alone rather than reporting a blank
// one as if it had been read from Nexus.
func TestRepositoryObserveDoesNotReportAtProviderWhenAbsent(t *testing.T) {
	t.Parallel()

	cr := newTestRepo("apt-hosted", "apt", repoTypeHosted)
	cr.Status.AtProvider.Name = "stale"

	obs, err := newTestExternal(&mockHandler{exists: false}).Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if obs.ResourceExists {
		t.Error("Observe() ResourceExists = true, want false")
	}

	if cr.Status.AtProvider.URL != nil {
		t.Errorf("status.atProvider.url = %v, want nil for a repository that does not exist", cr.Status.AtProvider.URL)
	}
}

// TestRepositoryCreate tests the Create method for repositories.
func TestRepositoryCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cr      *repositoryv1alpha1.Repository
		handler *mockHandler
		wantErr bool
	}{
		{
			name:    "Created",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{},
		},
		{
			name:    "CreateError",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{createErr: errors.New("create error")},
			wantErr: true,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", repoTypeHosted),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handler formatHandler
			if tt.handler != nil {
				handler = tt.handler
			}

			_, err := newTestExternal(handler).Create(context.Background(), tt.cr)
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
		handler *mockHandler
		wantErr bool
	}{
		{
			name:    "Updated",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{},
		},
		{
			name:    "UpdateError",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{updateErr: errors.New("update error")},
			wantErr: true,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", repoTypeHosted),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handler formatHandler
			if tt.handler != nil {
				handler = tt.handler
			}

			_, err := newTestExternal(handler).Update(context.Background(), tt.cr)
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
		handler *mockHandler
		wantErr bool
	}{
		{
			name:    "Deleted",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{},
		},
		{
			name:    "DeleteNotFoundIsIgnored",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{deleteErr: errors.New("404 not found")},
		},
		{
			name:    "DeleteError",
			cr:      newTestRepo("maven-releases", "maven2", repoTypeHosted),
			handler: &mockHandler{deleteErr: errors.New("connection error")},
			wantErr: true,
		},
		{
			name:    "UnsupportedFormat",
			cr:      newTestRepo("unsupported-repo", "unsupported", repoTypeHosted),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var handler formatHandler
			if tt.handler != nil {
				handler = tt.handler
			}

			_, err := newTestExternal(handler).Delete(context.Background(), tt.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
