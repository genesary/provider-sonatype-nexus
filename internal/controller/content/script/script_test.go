package script

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema"
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

// newTestScript creates a Script CR for use in tests.
func newTestScript(name, scriptType, content string) *contentv1alpha1.Script {
	return &contentv1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: contentv1alpha1.ScriptSpec{
			ForProvider: contentv1alpha1.ScriptParameters{
				Name:    name,
				Type:    scriptType,
				Content: content,
			},
		},
	}
}

// newTestScheme creates a runtime.Scheme with the required API groups.
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

// TestObserve tests the Observe method of the external client.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *contentv1alpha1.Script
		mockSetup    func(*contentmocks.MockScriptClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NotFound_resource_not_found",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return nil, errors.New("resource not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NilReturned",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return nil, nil //nolint:nilnil // intentionally testing nil script with nil error case
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError_connection",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "Forbidden_scripting_disabled",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return nil, errors.New("403 forbidden")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestScript("my-script", "groovy", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return &schema.Script{
						Name:    "my-script",
						Type:    "groovy",
						Content: "log.info('hello')",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButContentChanged",
			cr:   newTestScript("my-script", "groovy", "log.info('new content')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return &schema.Script{
						Name:    "my-script",
						Type:    "groovy",
						Content: "log.info('old content')",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButTypeChanged",
			cr:   newTestScript("my-script", "groovy2", "log.info('hello')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
					return &schema.Script{
						Name:    "my-script",
						Type:    "groovy",
						Content: "log.info('hello')",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UsesExternalNameAnnotation",
			cr: func() *contentv1alpha1.Script {
				cr := newTestScript("my-script", "groovy", "log.info('hello')")
				cr.Annotations = map[string]string{
					"crossplane.io/external-name": "external-name",
				}

				return cr
			}(),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.GetScriptFn = func(_ context.Context, name string) (*schema.Script, error) {
					if name != "external-name" {
						return nil, errors.New("wrong name: " + name)
					}

					return &schema.Script{
						Name:    "external-name",
						Type:    "groovy",
						Content: "log.info('hello')",
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

			m := contentmocks.NewMockScriptClient()
			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			e := &external{client: m}
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

// TestObserve_WrongType verifies that Observe rejects non-Script resources.
func TestObserve_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockScriptClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

// TestObserve_UpdatesAtProvider verifies AtProvider is populated from Nexus.
func TestObserve_UpdatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestScript("my-script", "groovy", "log.info('hello')")

	m := contentmocks.NewMockScriptClient()
	m.GetScriptFn = func(_ context.Context, _ string) (*schema.Script, error) {
		return &schema.Script{
			Name:    "my-script",
			Type:    "groovy",
			Content: "log.info('hello')",
		}, nil
	}

	e := &external{client: m}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.Name != "my-script" {
		t.Errorf("AtProvider.Name = %q, want %q", cr.Status.AtProvider.Name, "my-script")
	}

	if cr.Status.AtProvider.Type != "groovy" {
		t.Errorf("AtProvider.Type = %q, want %q", cr.Status.AtProvider.Type, "groovy")
	}

	if cr.Status.AtProvider.Content != "log.info('hello')" {
		t.Errorf("AtProvider.Content = %q, want %q", cr.Status.AtProvider.Content, "log.info('hello')")
	}
}

// TestCreate tests the Create method of the external client.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.Script
		mockSetup func(*contentmocks.MockScriptClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockScriptClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestScript("new-script", "groovy", "log.info('hi')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.CreateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, m *contentmocks.MockScriptClient) {
				t.Helper()

				if len(m.CreateScriptCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(m.CreateScriptCalls))
				}

				if m.CreateScriptCalls[0].Name != "new-script" {
					t.Errorf("wrong script name: %v", m.CreateScriptCalls[0].Name)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestScript("new-script", "groovy", "log.info('hi')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.CreateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
		{
			name: "CreateForbidden_ScriptingDisabled",
			cr:   newTestScript("new-script", "groovy", "log.info('hi')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.CreateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return errors.New("403 forbidden")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := contentmocks.NewMockScriptClient()
			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			e := &external{client: m}
			_, err := e.Create(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, m)
			}
		})
	}
}

// TestCreate_WrongType verifies that Create rejects non-Script resources.
func TestCreate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockScriptClient()}

	_, err := e.Create(context.Background(), nil)
	if err == nil {
		t.Error("Create() with nil managed resource should return error")
	}
}

// TestUpdate tests the Update method of the external client.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.Script
		mockSetup func(*contentmocks.MockScriptClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockScriptClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestScript("existing-script", "groovy", "log.info('updated')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.UpdateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, m *contentmocks.MockScriptClient) {
				t.Helper()

				if len(m.UpdateScriptCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(m.UpdateScriptCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestScript("existing-script", "groovy", "log.info('updated')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.UpdateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
		{
			name: "UpdateForbidden_ScriptingDisabled",
			cr:   newTestScript("existing-script", "groovy", "log.info('updated')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.UpdateScriptFn = func(_ context.Context, _ *schema.Script) error {
					return errors.New("403 forbidden")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := contentmocks.NewMockScriptClient()
			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			e := &external{client: m}
			_, err := e.Update(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, m)
			}
		})
	}
}

// TestUpdate_WrongType verifies that Update rejects non-Script resources.
func TestUpdate_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockScriptClient()}

	_, err := e.Update(context.Background(), nil)
	if err == nil {
		t.Error("Update() with nil managed resource should return error")
	}
}

// TestDelete tests the Delete method of the external client.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.Script
		mockSetup func(*contentmocks.MockScriptClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockScriptClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestScript("old-script", "groovy", "log.info('bye')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, _ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, m *contentmocks.MockScriptClient) {
				t.Helper()

				if len(m.DeleteScriptCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(m.DeleteScriptCalls))
				}

				if m.DeleteScriptCalls[0] != "old-script" {
					t.Errorf("wrong name passed to Delete: %v", m.DeleteScriptCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestScript("old-script", "groovy", "log.info('bye')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, _ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFoundDoesNotExist",
			cr:   newTestScript("old-script", "groovy", "log.info('bye')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, _ string) error {
					return errors.New("does not exist")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestScript("old-script", "groovy", "log.info('bye')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, _ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
		{
			name: "DeleteForbidden_ScriptingDisabled",
			cr:   newTestScript("old-script", "groovy", "log.info('bye')"),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, _ string) error {
					return errors.New("403 forbidden")
				}
			},
			wantErr: true,
		},
		{
			name: "DeleteUsesExternalName",
			cr: func() *contentv1alpha1.Script {
				cr := newTestScript("k8s-name", "groovy", "log.info('hi')")
				cr.Annotations = map[string]string{
					"crossplane.io/external-name": "nexus-script-name",
				}

				return cr
			}(),
			mockSetup: func(m *contentmocks.MockScriptClient) {
				m.DeleteScriptFn = func(_ context.Context, name string) error {
					if name != "nexus-script-name" {
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

			m := contentmocks.NewMockScriptClient()
			if tt.mockSetup != nil {
				tt.mockSetup(m)
			}

			e := &external{client: m}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, m)
			}
		})
	}
}

// TestDelete_WrongType verifies that Delete rejects non-Script resources.
func TestDelete_WrongType(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockScriptClient()}

	_, err := e.Delete(context.Background(), nil)
	if err == nil {
		t.Error("Delete() with nil managed resource should return error")
	}
}

// TestConnect_WrongType verifies that Connect rejects non-Script resources.
func TestConnect_WrongType(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), nil)
	if err == nil {
		t.Error("Connect() with nil managed resource should return error")
	}

	if err.Error() != errNotScript {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotScript)
	}
}

// nonModernManaged is a Script wrapper that breaks the ModernManaged assertion.
type nonModernManaged struct {
	resource.Managed
}

// DeepCopyObject satisfies runtime.Object for nonModernManaged.
func (n *nonModernManaged) DeepCopyObject() runtime.Object { return n }

// TestConnect_NotModernManaged verifies Connect rejects non-ModernManaged.
func TestConnect_NotModernManaged(t *testing.T) {
	t.Parallel()

	c := &connector{}

	_, err := c.Connect(context.Background(), &nonModernManaged{})
	if err == nil {
		t.Error("Connect() should fail when managed resource is not a Script")
	}
}

// TestConnect_TrackError verifies Connect fails when tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &contentv1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-script",
			Namespace: "default",
		},
		Spec: contentv1alpha1.ScriptSpec{
			ForProvider: contentv1alpha1.ScriptParameters{
				Name:    "test-script",
				Type:    "groovy",
				Content: "log.info('hello')",
			},
		},
	}
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	c := &connector{kube: fakeClient, usage: usage}

	_, err := c.Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError verifies Connect fails when PC is absent.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &contentv1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-script",
			Namespace: "default",
			UID:       types.UID("test-uid-1234"),
		},
		Spec: contentv1alpha1.ScriptSpec{
			ForProvider: contentv1alpha1.ScriptParameters{
				Name:    "test-script",
				Type:    "groovy",
				Content: "log.info('hello')",
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

// TestDisconnect verifies Disconnect is always a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	e := &external{client: contentmocks.NewMockScriptClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}
