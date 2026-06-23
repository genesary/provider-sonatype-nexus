package cleanuppolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"
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

// newTestCleanupPolicy returns a minimal CleanupPolicy for tests.
func newTestCleanupPolicy(name, format string) *contentv1alpha1.CleanupPolicy {
	return &contentv1alpha1.CleanupPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: contentv1alpha1.CleanupPolicySpec{
			ForProvider: contentv1alpha1.CleanupPolicyParameters{
				Name:   name,
				Format: format,
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

	tests := []struct {
		name         string
		cr           *contentv1alpha1.CleanupPolicy
		mockSetup    func(*contentmocks.MockCleanupPolicyClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestCleanupPolicy("my-policy", "maven2"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NotFound_NotFoundText",
			cr:   newTestCleanupPolicy("my-policy", "npm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					return nil, errors.New("resource not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "NilPolicyReturned",
			cr:   newTestCleanupPolicy("my-policy", "docker"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					//nolint:nilnil // intentionally testing nil policy with nil error case
					return nil, nil
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestCleanupPolicy("my-policy", "raw"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr:   newTestCleanupPolicy("my-policy", "maven2"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					return &cleanuppolicies.CleanupPolicy{
						Name:   "my-policy",
						Format: cleanuppolicies.RepositoryFormatMaven2,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated",
			cr:   newTestCleanupPolicy("my-policy", "maven2"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
					return &cleanuppolicies.CleanupPolicy{
						Name:   "my-policy",
						Format: cleanuppolicies.RepositoryFormatNpm,
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UsesExternalNameAnnotation",
			cr: func() *contentv1alpha1.CleanupPolicy {
				cr := newTestCleanupPolicy("my-policy", "pypi")
				cr.Annotations = map[string]string{
					"crossplane.io/external-name": "external-name",
				}

				return cr
			}(),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.GetFn = func(name string) (*cleanuppolicies.CleanupPolicy, error) {
					if name != "external-name" {
						return nil, errors.New("wrong name called")
					}

					return &cleanuppolicies.CleanupPolicy{
						Name:   "external-name",
						Format: cleanuppolicies.RepositoryFormatPypi,
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

			mc := contentmocks.NewMockCleanupPolicyClient()
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

	e := &external{client: contentmocks.NewMockCleanupPolicyClient()}

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
		cr        *contentv1alpha1.CleanupPolicy
		mockSetup func(*contentmocks.MockCleanupPolicyClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockCleanupPolicyClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestCleanupPolicy("new-policy", "npm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.CreateFn = func(_ *cleanuppolicies.CleanupPolicy) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockCleanupPolicyClient) {
				t.Helper()

				if len(mc.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateCalls))
				}

				if mc.CreateCalls[0].Name != "new-policy" {
					t.Errorf("wrong policy name: %v", mc.CreateCalls[0].Name)
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestCleanupPolicy("new-policy", "npm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.CreateFn = func(_ *cleanuppolicies.CleanupPolicy) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockCleanupPolicyClient()
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

	e := &external{client: contentmocks.NewMockCleanupPolicyClient()}

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
		cr        *contentv1alpha1.CleanupPolicy
		mockSetup func(*contentmocks.MockCleanupPolicyClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockCleanupPolicyClient)
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestCleanupPolicy("existing-policy", "docker"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.UpdateFn = func(_ *cleanuppolicies.CleanupPolicy) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockCleanupPolicyClient) {
				t.Helper()

				if len(mc.UpdateCalls) != 1 {
					t.Errorf("expected 1 Update call, got %d", len(mc.UpdateCalls))
				}
			},
		},
		{
			name: "UpdateError",
			cr:   newTestCleanupPolicy("existing-policy", "docker"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.UpdateFn = func(_ *cleanuppolicies.CleanupPolicy) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockCleanupPolicyClient()
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

	e := &external{client: contentmocks.NewMockCleanupPolicyClient()}

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
		cr        *contentv1alpha1.CleanupPolicy
		mockSetup func(*contentmocks.MockCleanupPolicyClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockCleanupPolicyClient)
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestCleanupPolicy("old-policy", "helm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockCleanupPolicyClient) {
				t.Helper()

				if len(mc.DeleteCalls) != 1 {
					t.Errorf("expected 1 Delete call, got %d", len(mc.DeleteCalls))
				}

				if mc.DeleteCalls[0] != "old-policy" {
					t.Errorf("wrong name passed to Delete: %v", mc.DeleteCalls[0])
				}
			},
		},
		{
			name: "DeleteNotFound",
			cr:   newTestCleanupPolicy("old-policy", "helm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFoundDoesNotExist",
			cr:   newTestCleanupPolicy("old-policy", "helm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("does not exist")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestCleanupPolicy("old-policy", "helm"),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("server error")
				}
			},
			wantErr: true,
		},
		{
			name: "DeleteUsesExternalName",
			cr: func() *contentv1alpha1.CleanupPolicy {
				cr := newTestCleanupPolicy("k8s-name", "raw")
				cr.Annotations = map[string]string{
					"crossplane.io/external-name": "nexus-policy-name",
				}

				return cr
			}(),
			mockSetup: func(mc *contentmocks.MockCleanupPolicyClient) {
				mc.DeleteFn = func(name string) error {
					if name != "nexus-policy-name" {
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

			mc := contentmocks.NewMockCleanupPolicyClient()
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

	e := &external{client: contentmocks.NewMockCleanupPolicyClient()}

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

	if err.Error() != errNotCleanupPolicy {
		t.Errorf("Connect() error = %q, want %q", err.Error(), errNotCleanupPolicy)
	}
}

// TestConnect_TrackError tests Connect when ProviderConfig tracking fails.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := &contentv1alpha1.CleanupPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: contentv1alpha1.CleanupPolicySpec{
			ForProvider: contentv1alpha1.CleanupPolicyParameters{
				Name:   "test-policy",
				Format: "maven2",
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

	cr := &contentv1alpha1.CleanupPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
			UID:       types.UID("test-uid-1234"),
		},
		Spec: contentv1alpha1.CleanupPolicySpec{
			ForProvider: contentv1alpha1.CleanupPolicyParameters{
				Name:   "test-policy",
				Format: "maven2",
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

	e := &external{client: contentmocks.NewMockCleanupPolicyClient()}

	err := e.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestObserve_UpdatesAtProvider checks that Observe populates AtProvider.
func TestObserve_UpdatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestCleanupPolicy("my-policy", "maven2")

	notes := "some notes"

	mc := contentmocks.NewMockCleanupPolicyClient()
	mc.GetFn = func(_ string) (*cleanuppolicies.CleanupPolicy, error) {
		return &cleanuppolicies.CleanupPolicy{
			Name:   "my-policy",
			Format: cleanuppolicies.RepositoryFormatMaven2,
			Notes:  &notes,
			Retain: 5,
		}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.Name != "my-policy" {
		t.Errorf("AtProvider.Name = %q, want %q", cr.Status.AtProvider.Name, "my-policy")
	}

	if cr.Status.AtProvider.Notes != "some notes" {
		t.Errorf("AtProvider.Notes = %q, want %q", cr.Status.AtProvider.Notes, "some notes")
	}

	if cr.Status.AtProvider.Retain != 5 {
		t.Errorf("AtProvider.Retain = %d, want 5", cr.Status.AtProvider.Retain)
	}
}
