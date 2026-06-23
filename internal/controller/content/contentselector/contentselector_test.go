package contentselector

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	contentmocks "github.com/genesary/provider-sonatype-nexus/test/mocks/content"
)

// newTestContentSelector returns a minimal ContentSelector for tests.
func newTestContentSelector(name, expr string) *contentv1alpha1.ContentSelector {
	return &contentv1alpha1.ContentSelector{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: contentv1alpha1.ContentSelectorSpec{
			ForProvider: contentv1alpha1.ContentSelectorParameters{
				Name:       name,
				Expression: expr,
			},
		},
	}
}

// newTestScheme registers content and nexus v1alpha1 types in a new scheme.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := contentv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(content) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestObserve tests the Observe method.
func TestObserve(t *testing.T) {
	t.Parallel()

	description := "Test selector"

	tests := []struct {
		name         string
		cr           *contentv1alpha1.ContentSelector
		mockSetup    func(*contentmocks.MockContentSelectorClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestContentSelector("my-selector", "format == 'maven2'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(_ string) (*security.ContentSelector, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NilReturned",
			cr:   newTestContentSelector("my-selector", "format == 'maven2'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(_ string) (*security.ContentSelector, error) {
					//nolint:nilnil // intentionally testing nil with nil error
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestContentSelector("my-selector", "format == 'maven2'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(_ string) (*security.ContentSelector, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr: func() *contentv1alpha1.ContentSelector {
				cs := newTestContentSelector("my-selector", "format == 'maven2'")
				cs.Spec.ForProvider.Description = &description

				return cs
			}(),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(_ string) (*security.ContentSelector, error) {
					return &security.ContentSelector{
						Name:        "my-selector",
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
			cr:   newTestContentSelector("my-selector", "format == 'docker'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(_ string) (*security.ContentSelector, error) {
					return &security.ContentSelector{
						Name:       "my-selector",
						Expression: "format == 'maven2'",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UsesExternalNameAnnotation",
			cr: func() *contentv1alpha1.ContentSelector {
				cs := newTestContentSelector("my-selector", "format == 'npm'")
				cs.Annotations = map[string]string{
					"crossplane.io/external-name": "external-name",
				}

				return cs
			}(),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.GetFn = func(name string) (*security.ContentSelector, error) {
					if name != "external-name" {
						return nil, errors.New("wrong name called")
					}

					return &security.ContentSelector{
						Name:       "external-name",
						Expression: "format == 'npm'",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockContentSelectorClient()
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

// TestObserve_WrongType tests Observe with wrong resource type.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockContentSelectorClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.ContentSelector
		mockSetup func(*contentmocks.MockContentSelectorClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockContentSelectorClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestContentSelector("new-selector", "format == 'helm'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.CreateFn = func(_ security.ContentSelector) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockContentSelectorClient) {
				t.Helper()

				if len(mc.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateCalls))
				}

				if mc.CreateCalls[0].Name != "new-selector" {
					t.Errorf("wrong selector name: %v", mc.CreateCalls[0].Name)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestContentSelector("new-selector", "format == 'helm'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.CreateFn = func(_ security.ContentSelector) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockContentSelectorClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestCreate_WrongType tests Create with wrong resource type.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockContentSelectorClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.ContentSelector
		mockSetup func(*contentmocks.MockContentSelectorClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockContentSelectorClient)
	}{
		{
			name: "UpdateSuccess",
			cr: func() *contentv1alpha1.ContentSelector {
				cs := newTestContentSelector("existing-selector", "format == 'raw'")
				cs.Annotations = map[string]string{
					"crossplane.io/external-name": "existing-selector",
				}

				return cs
			}(),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.UpdateFn = func(_ string, _ security.ContentSelector) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockContentSelectorClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestContentSelector("existing-selector", "format == 'raw'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.UpdateFn = func(_ string, _ security.ContentSelector) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockContentSelectorClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestUpdate_WrongType tests Update with wrong resource type.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockContentSelectorClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.ContentSelector
		mockSetup func(*contentmocks.MockContentSelectorClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockContentSelectorClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestContentSelector("old-selector", "format == 'pypi'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockContentSelectorClient) {
				t.Helper()

				if len(mc.DeleteCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeleteCalls))
				}

				if mc.DeleteCalls[0] != "old-selector" {
					t.Errorf("wrong name passed to Delete: %v", mc.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestContentSelector("old-selector", "format == 'pypi'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestContentSelector("old-selector", "format == 'pypi'"),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
		{
			name: "DeleteUsesExternalName",
			cr: func() *contentv1alpha1.ContentSelector {
				cs := newTestContentSelector("k8s-name", "format == 'go'")
				cs.Annotations = map[string]string{
					"crossplane.io/external-name": "nexus-selector-name",
				}

				return cs
			}(),
			mockSetup: func(mc *contentmocks.MockContentSelectorClient) {
				mc.DeleteFn = func(name string) error {
					if name != "nexus-selector-name" {
						return errors.New("wrong name: " + name)
					}

					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockContentSelectorClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, mc)
			}
		})
	}
}

// TestDelete_WrongType tests Delete with wrong resource type.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockContentSelectorClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestConnect_WrongType tests Connect with wrong resource type.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotContentSelector {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotContentSelector)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &contentv1alpha1.ContentSelector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-selector",
			Namespace: "default",
		},
		Spec: contentv1alpha1.ContentSelectorSpec{
			ForProvider: contentv1alpha1.ContentSelectorParameters{
				Name:       "test-selector",
				Expression: "format == 'maven2'",
			},
		},
	}
	// Kind is empty → Track will fail early with missingRefError.
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError tests ProviderConfig get failure.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &contentv1alpha1.ContentSelector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-selector",
			Namespace: "default",
			UID:       types.UID("test-uid-1234"),
		},
		Spec: contentv1alpha1.ContentSelectorSpec{
			ForProvider: contentv1alpha1.ContentSelectorParameters{
				Name:       "test-selector",
				Expression: "format == 'maven2'",
			},
		},
	}
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{
		Name: "default",
		Kind: "ProviderConfig",
	})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}

// TestDisconnect tests the Disconnect method.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockContentSelectorClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestObserve_UpdatesAtProvider checks that Observe populates AtProvider.
func TestObserve_UpdatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestContentSelector("my-selector", "format == 'docker'")

	mc := contentmocks.NewMockContentSelectorClient()
	mc.GetFn = func(_ string) (*security.ContentSelector, error) {
		return &security.ContentSelector{
			Name:        "my-selector",
			Expression:  "format == 'docker'",
			Description: "my description",
		}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.Name != "my-selector" {
		t.Errorf("AtProvider.Name = %q, want %q", cr.Status.AtProvider.Name, "my-selector")
	}

	if cr.Status.AtProvider.Description != "my description" {
		t.Errorf("AtProvider.Description = %q, want %q", cr.Status.AtProvider.Description, "my description")
	}

	if cr.Status.AtProvider.Expression != "format == 'docker'" {
		t.Errorf("AtProvider.Expression = %q, want format == 'docker'", cr.Status.AtProvider.Expression)
	}
}
