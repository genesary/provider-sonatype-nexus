package script

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	pkgerrors "github.com/pkg/errors"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	instanceclient "github.com/genesary/provider-sonatype-nexus/internal/clients/instance"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	instancemocks "github.com/genesary/provider-sonatype-nexus/test/mocks/instance"
)

const (
	// testScriptName is the Nexus name of the test script.
	testScriptName = "create-local-user"
	// testContent is the source of the test script.
	testContent = "return 'hello'"
)

// errBoom is a generic failure returned by mocks.
var errBoom = errors.New("boom")

// newTestScript returns a Script named testScriptName with testContent.
func newTestScript() *instancev1alpha1.Script {
	return &instancev1alpha1.Script{
		ObjectMeta: metav1.ObjectMeta{Name: "my-script", Namespace: "default"},
		Spec: instancev1alpha1.ScriptSpec{
			ForProvider: instancev1alpha1.ScriptParameters{
				Name:    testScriptName,
				Type:    "groovy",
				Content: testContent,
			},
		},
	}
}

// newObservedScript returns the Nexus script matching newTestScript.
func newObservedScript() *v395.ScriptXO {
	return &v395.ScriptXO{Name: testScriptName, Type: "groovy", Content: testContent}
}

// newTestScheme returns a runtime.Scheme with all required types registered.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()

	err := instancev1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(instance) failed: %v", err)
	}

	err = nexusv1alpha1.AddToScheme(s)
	if err != nil {
		t.Fatalf("AddToScheme(nexus) failed: %v", err)
	}

	return s
}

// TestObserve covers the Observe method across all meaningful branches.
func TestObserve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		externalName string
		observed     *v395.ScriptXO
		getErr       error
		wantGetName  string
		wantExists   bool
		wantUpToDate bool
		wantReady    bool
		wantErr      bool
	}{
		{
			name:        "NotFound",
			getErr:      pkgerrors.Wrap(nexus.ErrNotFound, "404"),
			wantGetName: testScriptName,
		},
		{
			name:        "GetError",
			getErr:      errBoom,
			wantGetName: testScriptName,
			wantErr:     true,
		},
		{
			name:         "UpToDate",
			observed:     newObservedScript(),
			wantGetName:  testScriptName,
			wantExists:   true,
			wantUpToDate: true,
			wantReady:    true,
		},
		{
			name:        "ContentDrift",
			observed:    &v395.ScriptXO{Name: testScriptName, Type: "groovy", Content: "return 'changed'"},
			wantGetName: testScriptName,
			wantExists:  true,
			wantReady:   true,
		},
		{
			// Reconciling cannot change the type, so the script is not ready.
			name:        "TypeDrift",
			observed:    &v395.ScriptXO{Name: testScriptName, Type: "javascript", Content: testContent},
			wantGetName: testScriptName,
			wantExists:  true,
		},
		{
			// The Groovy engine answers to both spellings.
			name:         "TypeCaseIgnored",
			observed:     &v395.ScriptXO{Name: testScriptName, Type: "Groovy", Content: testContent},
			wantGetName:  testScriptName,
			wantExists:   true,
			wantUpToDate: true,
			wantReady:    true,
		},
		{
			// An explicit external name wins over the spec name.
			name:         "ExternalNameWins",
			externalName: "imported",
			observed:     &v395.ScriptXO{Name: "imported", Type: "groovy", Content: testContent},
			wantGetName:  "imported",
			wantExists:   true,
			wantUpToDate: true,
			wantReady:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := newTestScript()
			if tt.externalName != "" {
				meta.SetExternalName(cr, tt.externalName)
			}

			mc := instancemocks.NewMockScriptClient()
			mc.GetFn = func(_ string) (*v395.ScriptXO, error) {
				if tt.getErr != nil {
					return nil, tt.getErr
				}

				return tt.observed, nil
			}

			obs, err := (&external{client: mc}).Observe(context.Background(), cr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Observe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if obs.ResourceExists != tt.wantExists {
				t.Errorf("Observe() ResourceExists = %v, want %v", obs.ResourceExists, tt.wantExists)
			}

			if obs.ResourceUpToDate != tt.wantUpToDate {
				t.Errorf("Observe() ResourceUpToDate = %v, want %v", obs.ResourceUpToDate, tt.wantUpToDate)
			}

			if len(mc.GetCalls) != 1 || mc.GetCalls[0] != tt.wantGetName {
				t.Errorf("Observe() read %v, want [%s]", mc.GetCalls, tt.wantGetName)
			}

			if tt.observed != nil {
				checkObservedStatus(t, cr, tt.observed, tt.wantReady)
			}
		})
	}
}

// checkObservedStatus checks the status Observe records for a found script.
func checkObservedStatus(t *testing.T, cr *instancev1alpha1.Script, observed *v395.ScriptXO, wantReady bool) {
	t.Helper()

	want := instanceclient.GenerateScriptObservation(observed)
	if cr.Status.AtProvider != want {
		t.Errorf("Observe() AtProvider = %+v, want %+v", cr.Status.AtProvider, want)
	}

	wantStatus := corev1.ConditionFalse
	if wantReady {
		wantStatus = corev1.ConditionTrue
	}

	if got := cr.GetCondition(xpv2.TypeReady).Status; got != wantStatus {
		t.Errorf("Observe() Ready = %v, want %v", got, wantStatus)
	}
}

// TestCreate covers storing the script.
func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockScriptClient()
		mc.CreateFn = func(_ v395.ScriptXO) error { return nil }

		cr := newTestScript()
		cr.Spec.ForProvider.Type = ""

		_, err := (&external{client: mc}).Create(context.Background(), cr)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if len(mc.CreateCalls) != 1 {
			t.Fatalf("expected 1 Create call, got %d", len(mc.CreateCalls))
		}

		got := mc.CreateCalls[0]
		if got.Name != testScriptName || got.Type != instanceclient.DefaultScriptType || got.Content != testContent {
			t.Errorf("Create() sent %+v", got)
		}
	})

	t.Run("ScriptingDisabled", func(t *testing.T) {
		t.Parallel()

		mc := instancemocks.NewMockScriptClient()
		mc.CreateFn = func(_ v395.ScriptXO) error {
			return pkgerrors.Wrap(instanceclient.ErrScriptingDisabled, "410 Gone")
		}

		_, err := (&external{client: mc}).Create(context.Background(), newTestScript())
		if !errors.Is(err, instanceclient.ErrScriptingDisabled) {
			t.Errorf("Create() error = %v, want ErrScriptingDisabled", err)
		}
	})
}

// TestUpdate covers replacing the script.
func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		externalName string
		observedType string
		observedSHA  string
		updateErr    error
		wantName     string
		wantCalls    int
		wantErrIs    error
		wantErr      bool
	}{
		{
			name:         "Success",
			observedType: "groovy",
			wantName:     testScriptName,
			wantCalls:    1,
		},
		{
			// Status is empty when Update runs without an observation.
			name:      "NoObservedType",
			wantName:  testScriptName,
			wantCalls: 1,
		},
		{
			// Nexus refuses a body naming another script than its path.
			name:         "BodyNamesExternalName",
			externalName: "imported",
			observedType: "groovy",
			wantName:     "imported",
			wantCalls:    1,
		},
		{
			name:         "Error",
			observedType: "groovy",
			updateErr:    errBoom,
			wantName:     testScriptName,
			wantCalls:    1,
			wantErr:      true,
		},
		{
			// Nexus ignores the type of an update: the source is still
			// written, then the type mismatch is reported.
			name:         "TypeMismatchStillUpdatesContent",
			observedType: "javascript",
			wantName:     testScriptName,
			wantCalls:    1,
			wantErrIs:    instanceclient.ErrScriptTypeChanged,
			wantErr:      true,
		},
		{
			// Once the source matches, only the mismatch is reported.
			name:         "TypeMismatchContentInSync",
			observedType: "javascript",
			observedSHA:  instanceclient.ScriptContentSHA256(testContent),
			wantCalls:    0,
			wantErrIs:    instanceclient.ErrScriptTypeChanged,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := newTestScript()
			cr.Status.AtProvider.Type = tt.observedType
			cr.Status.AtProvider.ContentSHA256 = tt.observedSHA

			if tt.externalName != "" {
				meta.SetExternalName(cr, tt.externalName)
			}

			var path string

			mc := instancemocks.NewMockScriptClient()
			mc.UpdateFn = func(name string, _ v395.ScriptXO) error {
				path = name

				return tt.updateErr
			}

			_, err := (&external{client: mc}).Update(context.Background(), cr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErrIs)
			}

			if len(mc.UpdateCalls) != tt.wantCalls {
				t.Fatalf("expected %d Update calls, got %d", tt.wantCalls, len(mc.UpdateCalls))
			}

			if tt.wantCalls == 0 {
				return
			}

			got := mc.UpdateCalls[0]
			if path != tt.wantName || got.Name != tt.wantName || got.Content != testContent || got.Type != "groovy" {
				t.Errorf("Update() sent %+v to %q, want name %q", got, path, tt.wantName)
			}
		})
	}
}

// TestDelete covers deleting the script.
func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		deleteErr error
		wantErr   bool
	}{
		{name: "Success"},
		{name: "AlreadyGone", deleteErr: pkgerrors.Wrap(nexus.ErrNotFound, "404")},
		{name: "Error", deleteErr: errBoom, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := instancemocks.NewMockScriptClient()
			mc.DeleteFn = func(_ string) error { return tt.deleteErr }

			_, err := (&external{client: mc}).Delete(context.Background(), newTestScript())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(mc.DeleteCalls) != 1 || mc.DeleteCalls[0] != testScriptName {
				t.Errorf("Delete() deleted %v, want [%s]", mc.DeleteCalls, testScriptName)
			}
		})
	}
}

// TestDisconnect verifies Disconnect is a no-op.
func TestDisconnect(t *testing.T) {
	t.Parallel()

	err := (&external{client: instancemocks.NewMockScriptClient()}).Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect() returned unexpected error: %v", err)
	}
}

// TestConnect_TrackError verifies Connect fails without ProviderConfig Kind.
func TestConnect_TrackError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestScript()
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default"})

	_, err := (&connector{kube: fakeClient, usage: usage}).Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail when ProviderConfig ref Kind is missing")
	}
}

// TestConnect_GetProviderConfigError verifies Connect needs ProviderConfig.
func TestConnect_GetProviderConfigError(t *testing.T) {
	t.Parallel()

	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()
	usage := resource.NewProviderConfigUsageTracker(fakeClient, &nexusv1alpha1.ProviderConfigUsage{})

	cr := newTestScript()
	cr.UID = types.UID("test-uid-1234")
	cr.SetProviderConfigReference(&xpv2.ProviderConfigReference{Name: "default", Kind: "ProviderConfig"})

	_, err := (&connector{kube: fakeClient, usage: usage}).Connect(context.Background(), cr)
	if err == nil {
		t.Error("Connect() should fail without ProviderConfig in store")
	}
}
