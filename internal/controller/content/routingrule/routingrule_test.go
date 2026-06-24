package routingrule

import (
	"context"
	"errors"
	"testing"

	nexusschema "github.com/datadrivers/go-nexus-client/nexus3/schema"
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

// newTestRoutingRule returns a minimal RoutingRule CR for tests.
func newTestRoutingRule(name, mode string, matchers []string) *contentv1alpha1.RoutingRule {
	return &contentv1alpha1.RoutingRule{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: contentv1alpha1.RoutingRuleSpec{
			ForProvider: contentv1alpha1.RoutingRuleParameters{
				Name:     name,
				Mode:     mode,
				Matchers: matchers,
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

// TestObserve tests the Observe method across several scenarios.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cr           *contentv1alpha1.RoutingRule
		mockSetup    func(*contentmocks.MockRoutingRuleClient)
		wantExists   bool
		wantUpToDate bool
		wantErr      bool
	}{
		{
			name: "NotFound_404",
			cr:   newTestRoutingRule("block-snaps", "BLOCK", []string{".*-SNAPSHOT.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
					return nil, errors.New("404 not found")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "GetError",
			cr:   newTestRoutingRule("block-snaps", "BLOCK", []string{".*-SNAPSHOT.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
					return nil, errors.New("connection refused")
				}
			},
			wantExists:   false,
			wantUpToDate: false,
			wantErr:      true,
		},
		{
			name: "ExistsAndUpToDate",
			cr: func() *contentv1alpha1.RoutingRule {
				rr := newTestRoutingRule("block-snaps", "BLOCK", []string{".*-SNAPSHOT.*"})
				desc := "block snapshots"
				rr.Spec.ForProvider.Description = &desc

				return rr
			}(),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
					return &nexusschema.RoutingRule{
						Name:        "block-snaps",
						Mode:        "BLOCK",
						Matchers:    []string{".*-SNAPSHOT.*"},
						Description: "block snapshots",
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: true,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated_ModeMismatch",
			cr:   newTestRoutingRule("allow-releases", "ALLOW", []string{".*-RELEASE.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
					return &nexusschema.RoutingRule{
						Name:     "allow-releases",
						Mode:     "BLOCK",
						Matchers: []string{".*-RELEASE.*"},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "ExistsButOutdated_MatchersMismatch",
			cr:   newTestRoutingRule("my-rule", "BLOCK", []string{".*-SNAPSHOT.*", ".*-alpha.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
					return &nexusschema.RoutingRule{
						Name:     "my-rule",
						Mode:     "BLOCK",
						Matchers: []string{".*-SNAPSHOT.*"},
					}, nil
				}
			},
			wantExists:   true,
			wantUpToDate: false,
			wantErr:      false,
		},
		{
			name: "UsesExternalNameAnnotation",
			cr: func() *contentv1alpha1.RoutingRule {
				rr := newTestRoutingRule("k8s-name", "BLOCK", []string{".*"})
				rr.Annotations = map[string]string{
					"crossplane.io/external-name": "nexus-rule-name",
				}

				return rr
			}(),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.GetFn = func(name string) (*nexusschema.RoutingRule, error) {
					if name != "nexus-rule-name" {
						return nil, errors.New("wrong name called: " + name)
					}

					return &nexusschema.RoutingRule{
						Name:     "nexus-rule-name",
						Mode:     "BLOCK",
						Matchers: []string{".*"},
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

			mc := contentmocks.NewMockRoutingRuleClient()
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

	e := &external{client: contentmocks.NewMockRoutingRuleClient()}

	_, err := e.Observe(context.Background(), nil)
	if err == nil {
		t.Error("Observe() with nil managed resource should return error")
	}
}

// TestObserve_UpdatesAtProvider checks that Observe populates AtProvider.
func TestObserve_UpdatesAtProvider(t *testing.T) {
	t.Parallel()

	cr := newTestRoutingRule("my-rule", "BLOCK", []string{".*-SNAPSHOT.*"})

	mc := contentmocks.NewMockRoutingRuleClient()
	mc.GetFn = func(_ string) (*nexusschema.RoutingRule, error) {
		return &nexusschema.RoutingRule{
			Name:        "my-rule",
			Description: "block snapshots",
			Mode:        "BLOCK",
			Matchers:    []string{".*-SNAPSHOT.*"},
		}, nil
	}

	e := &external{client: mc}

	_, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatalf("Observe() unexpected error: %v", err)
	}

	if cr.Status.AtProvider.Name != "my-rule" {
		t.Errorf("AtProvider.Name = %q, want %q", cr.Status.AtProvider.Name, "my-rule")
	}

	if cr.Status.AtProvider.Description != "block snapshots" {
		t.Errorf("AtProvider.Description = %q, want %q", cr.Status.AtProvider.Description, "block snapshots")
	}

	if cr.Status.AtProvider.Mode != "BLOCK" {
		t.Errorf("AtProvider.Mode = %q, want BLOCK", cr.Status.AtProvider.Mode)
	}

	if len(cr.Status.AtProvider.Matchers) != 1 || cr.Status.AtProvider.Matchers[0] != ".*-SNAPSHOT.*" {
		t.Errorf("AtProvider.Matchers = %v, want [.*-SNAPSHOT.*]", cr.Status.AtProvider.Matchers)
	}
}

// TestCreate tests the Create method across several scenarios.
func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.RoutingRule
		mockSetup func(*contentmocks.MockRoutingRuleClient)
		wantErr   bool
		validate  func(*testing.T, *contentmocks.MockRoutingRuleClient)
	}{
		{
			name: "CreateSuccess",
			cr:   newTestRoutingRule("new-rule", "BLOCK", []string{".*-SNAPSHOT.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.CreateFn = func(_ *nexusschema.RoutingRule) error {
					return nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, mc *contentmocks.MockRoutingRuleClient) {
				t.Helper()

				if len(mc.CreateCalls) != 1 {
					t.Errorf("expected 1 Create call, got %d", len(mc.CreateCalls))
				}
			},
		},
		{
			name: "CreateError",
			cr:   newTestRoutingRule("new-rule", "BLOCK", []string{".*-SNAPSHOT.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.CreateFn = func(_ *nexusschema.RoutingRule) error {
					return errors.New("create failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockRoutingRuleClient()
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

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.RoutingRule
		mockSetup func(*contentmocks.MockRoutingRuleClient)
		wantErr   bool
	}{
		{
			name: "UpdateSuccess",
			cr:   newTestRoutingRule("existing-rule", "ALLOW", []string{".*-RELEASE.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.UpdateFn = func(_ *nexusschema.RoutingRule) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "UpdateError",
			cr:   newTestRoutingRule("existing-rule", "ALLOW", []string{".*-RELEASE.*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.UpdateFn = func(_ *nexusschema.RoutingRule) error {
					return errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockRoutingRuleClient()
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

// TestDelete tests the Delete method.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cr        *contentv1alpha1.RoutingRule
		mockSetup func(*contentmocks.MockRoutingRuleClient)
		wantErr   bool
	}{
		{
			name: "DeleteSuccess",
			cr:   newTestRoutingRule("old-rule", "BLOCK", []string{".*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.DeleteFn = func(_ string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteNotFound_Ignored",
			cr:   newTestRoutingRule("old-rule", "BLOCK", []string{".*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("404 not found")
				}
			},
			wantErr: false,
		},
		{
			name: "DeleteError",
			cr:   newTestRoutingRule("old-rule", "BLOCK", []string{".*"}),
			mockSetup: func(mc *contentmocks.MockRoutingRuleClient) {
				mc.DeleteFn = func(_ string) error {
					return errors.New("connection refused")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := contentmocks.NewMockRoutingRuleClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mc)
			}

			e := &external{client: mc}
			_, err := e.Delete(context.Background(), tt.cr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConnect_GetProviderConfigError tests that Connect fails gracefully when
// the ProviderConfig is missing.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	s := newTestScheme(t)
	kube := fake.NewClientBuilder().WithScheme(s).Build()

	c := &connector{
		kube:  kube,
		usage: resource.NewProviderConfigUsageTracker(kube, &nexusv1alpha1.ProviderConfigUsage{}),
	}

	rr := newTestRoutingRule("my-rule", "BLOCK", []string{".*"})
	rr.Spec.ProviderConfigReference = &xpv2.ProviderConfigReference{Name: "missing"}

	_, err := c.Connect(context.Background(), rr)
	if err == nil {
		t.Error("Connect() expected error when ProviderConfig is missing, got nil")
	}
}

// TestConnect_TrackError tests that Connect fails gracefully when usage
// tracking fails (ProviderConfig exists but usage tracking fails).
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	s := newTestScheme(t)
	pc := &nexusv1alpha1.ProviderConfig{}
	pc.Name = "default"
	kube := fake.NewClientBuilder().WithScheme(s).WithObjects(pc).Build()

	c := &connector{
		kube: kube,
		usage: resource.NewProviderConfigUsageTracker(
			kube,
			&nexusv1alpha1.ProviderConfigUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "usage",
					Namespace: "default",
					UID:       types.UID("fake-uid"),
				},
			},
		),
	}

	rr := newTestRoutingRule("my-rule", "BLOCK", []string{".*"})
	rr.Spec.ProviderConfigReference = &xpv2.ProviderConfigReference{Name: "default"}

	_, err := c.Connect(context.Background(), rr)
	if err == nil {
		t.Error("Connect() expected error due to usage tracking failure, got nil")
	}
}
