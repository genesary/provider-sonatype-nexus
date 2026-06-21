package content

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// mockScriptService is a test double for nexus.ScriptService.
type mockScriptService struct {
	getScriptFn    func(ctx context.Context, name string) (*schema.Script, error)
	listScriptsFn  func(ctx context.Context) ([]schema.Script, error)
	createScriptFn func(ctx context.Context, script *schema.Script) error
	updateScriptFn func(ctx context.Context, script *schema.Script) error
	deleteScriptFn func(ctx context.Context, name string) error
}

// GetScript implements nexus.ScriptService.
func (m *mockScriptService) GetScript(ctx context.Context, name string) (*schema.Script, error) {
	if m.getScriptFn != nil {
		return m.getScriptFn(ctx, name)
	}

	return nil, errors.New("not configured")
}

// ListScripts implements nexus.ScriptService.
func (m *mockScriptService) ListScripts(ctx context.Context) ([]schema.Script, error) {
	if m.listScriptsFn != nil {
		return m.listScriptsFn(ctx)
	}

	return nil, errors.New("not configured")
}

// CreateScript implements nexus.ScriptService.
func (m *mockScriptService) CreateScript(ctx context.Context, script *schema.Script) error {
	if m.createScriptFn != nil {
		return m.createScriptFn(ctx, script)
	}

	return errors.New("not configured")
}

// UpdateScript implements nexus.ScriptService.
func (m *mockScriptService) UpdateScript(ctx context.Context, script *schema.Script) error {
	if m.updateScriptFn != nil {
		return m.updateScriptFn(ctx, script)
	}

	return errors.New("not configured")
}

// DeleteScript implements nexus.ScriptService.
func (m *mockScriptService) DeleteScript(ctx context.Context, name string) error {
	if m.deleteScriptFn != nil {
		return m.deleteScriptFn(ctx, name)
	}

	return errors.New("not configured")
}

// TestGetScript_DirectHit verifies the direct GET path succeeds.
func TestGetScript_DirectHit(t *testing.T) {
	t.Parallel()

	svc := &mockScriptService{
		getScriptFn: func(_ context.Context, name string) (*schema.Script, error) {
			return &schema.Script{Name: name, Type: "groovy", Content: "log.info('hi')"}, nil
		},
	}

	c := &scriptClientImpl{ScriptService: svc}

	s, err := c.GetScript(context.Background(), "my-script")
	if err != nil {
		t.Fatalf("GetScript() unexpected error: %v", err)
	}

	if s.Name != "my-script" {
		t.Errorf("Name = %q, want %q", s.Name, "my-script")
	}
}

// TestGetScript_FallbackToList verifies fallback to list when GET returns 404.
func TestGetScript_FallbackToList(t *testing.T) {
	t.Parallel()

	svc := &mockScriptService{
		getScriptFn: func(_ context.Context, _ string) (*schema.Script, error) {
			return nil, errors.New("404 not found")
		},
		listScriptsFn: func(_ context.Context) ([]schema.Script, error) {
			return []schema.Script{
				{Name: "other-script", Type: "groovy", Content: "log.info('other')"},
				{Name: "my-script", Type: "groovy", Content: "log.info('found')"},
			}, nil
		},
	}

	c := &scriptClientImpl{ScriptService: svc}

	s, err := c.GetScript(context.Background(), "my-script")
	if err != nil {
		t.Fatalf("GetScript() unexpected error: %v", err)
	}

	if s.Name != "my-script" {
		t.Errorf("Name = %q, want %q", s.Name, "my-script")
	}
}

// TestGetScript_FallbackToList_NotFound verifies nil + original error when
// the script is absent from the list fallback.
func TestGetScript_FallbackToList_NotFound(t *testing.T) {
	t.Parallel()

	svc := &mockScriptService{
		getScriptFn: func(_ context.Context, _ string) (*schema.Script, error) {
			return nil, errors.New("404 not found")
		},
		listScriptsFn: func(_ context.Context) ([]schema.Script, error) {
			return []schema.Script{
				{Name: "other-script", Type: "groovy", Content: "log.info('other')"},
			}, nil
		},
	}

	c := &scriptClientImpl{ScriptService: svc}

	s, err := c.GetScript(context.Background(), "my-script")
	if err == nil {
		t.Fatal("GetScript() expected error but got nil")
	}

	if s != nil {
		t.Errorf("GetScript() should return nil when not found in list")
	}
}

// TestGetScript_NonNotFoundError verifies non-404 errors are returned
// immediately without attempting a list fallback.
func TestGetScript_NonNotFoundError(t *testing.T) {
	t.Parallel()

	svc := &mockScriptService{
		getScriptFn: func(_ context.Context, _ string) (*schema.Script, error) {
			return nil, errors.New("connection refused")
		},
	}

	c := &scriptClientImpl{ScriptService: svc}

	_, err := c.GetScript(context.Background(), "my-script")
	if err == nil {
		t.Fatal("GetScript() expected error but got nil")
	}
}

// sentinel used to verify the original error is returned on list failure.
var errOriginal = errors.New("404 not found original")

// TestGetScript_ListError_ReturnsOriginalError verifies that the original
// GET error is returned when the fallback list call also fails.
func TestGetScript_ListError_ReturnsOriginalError(t *testing.T) {
	t.Parallel()

	svc := &mockScriptService{
		getScriptFn: func(_ context.Context, _ string) (*schema.Script, error) {
			return nil, errOriginal
		},
		listScriptsFn: func(_ context.Context) ([]schema.Script, error) {
			return nil, errors.New("list failed")
		},
	}

	c := &scriptClientImpl{ScriptService: svc}

	_, err := c.GetScript(context.Background(), "my-script")
	if !errors.Is(err, errOriginal) {
		t.Errorf("GetScript() should return original error when list fails, got: %v", err)
	}
}

// TestGenerateScript verifies the CR parameters are mapped correctly.
func TestGenerateScript(t *testing.T) {
	t.Parallel()

	cr := &contentv1alpha1.Script{
		Spec: contentv1alpha1.ScriptSpec{
			ForProvider: contentv1alpha1.ScriptParameters{
				Name:    "my-script",
				Type:    "groovy",
				Content: "log.info('hello')",
			},
		},
	}

	s := GenerateScript(cr)

	if s.Name != "my-script" {
		t.Errorf("Name = %q, want %q", s.Name, "my-script")
	}

	if s.Type != "groovy" {
		t.Errorf("Type = %q, want %q", s.Type, "groovy")
	}

	if s.Content != "log.info('hello')" {
		t.Errorf("Content = %q, want %q", s.Content, "log.info('hello')")
	}
}

// TestIsScriptUpToDate verifies the up-to-date comparison logic.
func TestIsScriptUpToDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cr       *contentv1alpha1.Script
		observed *schema.Script
		want     bool
	}{
		{
			name: "identical",
			cr: &contentv1alpha1.Script{
				Spec: contentv1alpha1.ScriptSpec{
					ForProvider: contentv1alpha1.ScriptParameters{
						Name:    "s",
						Type:    "groovy",
						Content: "log.info('hi')",
					},
				},
			},
			observed: &schema.Script{Name: "s", Type: "groovy", Content: "log.info('hi')"},
			want:     true,
		},
		{
			name: "content differs",
			cr: &contentv1alpha1.Script{
				Spec: contentv1alpha1.ScriptSpec{
					ForProvider: contentv1alpha1.ScriptParameters{
						Type:    "groovy",
						Content: "log.info('new')",
					},
				},
			},
			observed: &schema.Script{Type: "groovy", Content: "log.info('old')"},
			want:     false,
		},
		{
			name: "type differs",
			cr: &contentv1alpha1.Script{
				Spec: contentv1alpha1.ScriptSpec{
					ForProvider: contentv1alpha1.ScriptParameters{
						Type:    "groovy2",
						Content: "log.info('hi')",
					},
				},
			},
			observed: &schema.Script{Type: "groovy", Content: "log.info('hi')"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsScriptUpToDate(tt.cr, tt.observed)
			if got != tt.want {
				t.Errorf("IsScriptUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateScriptObservation verifies the observation conversion.
func TestGenerateScriptObservation(t *testing.T) {
	t.Parallel()

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()

		obs := GenerateScriptObservation(nil)
		if obs.Name != "" || obs.Type != "" || obs.Content != "" {
			t.Errorf("expected empty observation, got %+v", obs)
		}
	})

	t.Run("full input", func(t *testing.T) {
		t.Parallel()

		s := &schema.Script{Name: "s", Type: "groovy", Content: "log.info('hi')"}
		obs := GenerateScriptObservation(s)

		if obs.Name != "s" {
			t.Errorf("Name = %q, want %q", obs.Name, "s")
		}

		if obs.Type != "groovy" {
			t.Errorf("Type = %q, want %q", obs.Type, "groovy")
		}

		if obs.Content != "log.info('hi')" {
			t.Errorf("Content = %q, want %q", obs.Content, "log.info('hi')")
		}
	})
}

// TestNewScriptClient verifies a client is returned for valid credentials.
func TestNewScriptClient(t *testing.T) {
	t.Parallel()

	t.Run("returns client on valid credentials", func(t *testing.T) {
		t.Parallel()

		creds := nexus.Credentials{
			URL:      "http://localhost:8081",
			Username: "admin",
			Password: "admin123",
		}

		c, err := NewScriptClient(creds)
		if err != nil {
			t.Fatalf("NewScriptClient() unexpected error: %v", err)
		}

		if c == nil {
			t.Error("NewScriptClient() returned nil client")
		}
	})
}

// TestIsForbidden verifies 403/forbidden/access-denied error detection.
func TestIsForbidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"403 number", errors.New("403 forbidden"), true},
		{"forbidden text", errors.New("FORBIDDEN"), true},
		{"access denied", errors.New("Access Denied"), true},
		{"not found", errors.New("404 not found"), false},
		{"random error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsForbidden(tt.err)
			if got != tt.want {
				t.Errorf("IsForbidden(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
