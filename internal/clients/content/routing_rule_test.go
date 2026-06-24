package content

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	nexus3 "github.com/datadrivers/go-nexus-client/nexus3"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	nexusschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// newCR is a minimal RoutingRule CR for use in unit tests.
func newCR(name, mode string, matchers []string, desc *string) *contentv1alpha1.RoutingRule {
	cr := &contentv1alpha1.RoutingRule{}
	cr.Spec.ForProvider = contentv1alpha1.RoutingRuleParameters{
		Name:        name,
		Mode:        mode,
		Matchers:    matchers,
		Description: desc,
	}

	return cr
}

// ---- GenerateRoutingRule ----

// TestGenerateRoutingRule_Basic tests that GenerateRoutingRule correctly
// converts a CR to a Nexus RoutingRule.
func TestGenerateRoutingRule_Basic(t *testing.T) {
	t.Parallel()

	cr := newCR("my-rule", "BLOCK", []string{".*-SNAPSHOT.*"}, nil)
	got := GenerateRoutingRule(cr)

	if got.Name != "my-rule" {
		t.Errorf("Name = %q, want %q", got.Name, "my-rule")
	}

	if got.Mode != "BLOCK" {
		t.Errorf("Mode = %q, want BLOCK", got.Mode)
	}

	if len(got.Matchers) != 1 || got.Matchers[0] != ".*-SNAPSHOT.*" {
		t.Errorf("Matchers = %v, want [.*-SNAPSHOT.*]", got.Matchers)
	}

	if got.Description != "" {
		t.Errorf("Description = %q, want empty", got.Description)
	}
}

// TestGenerateRoutingRule_WithDescription tests that
// GenerateRoutingRule correctly converts a CR to a Nexus RoutingRule
// when a description is provided.
func TestGenerateRoutingRule_WithDescription(t *testing.T) {
	t.Parallel()

	cr := newCR("my-rule", "ALLOW", []string{".*"}, new("allow all"))
	got := GenerateRoutingRule(cr)

	if got.Description != "allow all" {
		t.Errorf("Description = %q, want %q", got.Description, "allow all")
	}

	if got.Mode != nexusschema.RoutingRuleModeAllow {
		t.Errorf("Mode = %q, want ALLOW", got.Mode)
	}
}

// ---- IsRoutingRuleUpToDate ----

// TestIsRoutingRuleUpToDate_AllMatch tests that IsRoutingRuleUpToDate
// returns true when the CR spec and observed rule match in all fields.
func TestIsRoutingRuleUpToDate_AllMatch(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*-SNAPSHOT.*"}, new("my desc"))
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:        "r",
		Mode:        "BLOCK",
		Matchers:    []string{".*-SNAPSHOT.*"},
		Description: "my desc",
	}

	if !IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("expected up to date")
	}
}

// TestIsRoutingRuleUpToDate_ModeMismatch tests that IsRoutingRuleUpToDate
// returns false when the CR spec and observed rule have different modes.
func TestIsRoutingRuleUpToDate_ModeMismatch(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "ALLOW", []string{".*"}, nil)
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:     "r",
		Mode:     "BLOCK",
		Matchers: []string{".*"},
	}

	if IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("expected NOT up to date (mode mismatch)")
	}
}

// TestIsRoutingRuleUpToDate_MatchersMismatch tests that IsRoutingRuleUpToDate
// returns false when the CR spec and observed rule have different matchers.
func TestIsRoutingRuleUpToDate_MatchersMismatch(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*-SNAPSHOT.*", ".*-alpha.*"}, nil)
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:     "r",
		Mode:     "BLOCK",
		Matchers: []string{".*-SNAPSHOT.*"},
	}

	if IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("expected NOT up to date (matchers mismatch)")
	}
}

// TestIsRoutingRuleUpToDate_DescriptionMismatch tests that
// IsRoutingRuleUpToDate returns false when the CR spec
// and observed rule have different descriptions.
func TestIsRoutingRuleUpToDate_DescriptionMismatch(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*"}, new("new desc"))
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:        "r",
		Mode:        "BLOCK",
		Matchers:    []string{".*"},
		Description: "old desc",
	}

	if IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("expected NOT up to date (description mismatch)")
	}
}

// TestIsRoutingRuleUpToDate_NilDescriptionIgnored tests that
// IsRoutingRuleUpToDate ignores a nil description in the CR spec
// when comparing to the observed rule.
func TestIsRoutingRuleUpToDate_NilDescriptionIgnored(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*"}, nil)
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:        "r",
		Mode:        "BLOCK",
		Matchers:    []string{".*"},
		Description: "some description",
	}

	if !IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("nil description in CR should not cause out-of-date")
	}
}

// ---- GenerateRoutingRuleObservation ----

// TestGenerateRoutingRuleObservation_Nil tests that
// GenerateRoutingRuleObservation returns an empty observation when given
// a nil observed rule.
func TestGenerateRoutingRuleObservation_Nil(t *testing.T) {
	t.Parallel()

	obs := GenerateRoutingRuleObservation(nil)

	if obs.Name != "" || obs.Mode != "" || obs.Matchers != nil {
		t.Errorf("expected empty observation, got %+v", obs)
	}
}

// TestGenerateRoutingRuleObservation_Full tests that
// GenerateRoutingRuleObservation correctly converts a full observed rule
// into a CRD observation.
func TestGenerateRoutingRuleObservation_Full(t *testing.T) {
	t.Parallel()

	rule := &nexusschema.RoutingRule{
		Name:        "my-rule",
		Description: "some desc",
		Mode:        "BLOCK",
		Matchers:    []string{".*-SNAPSHOT.*"},
	}

	obs := GenerateRoutingRuleObservation(rule)

	if obs.Name != "my-rule" {
		t.Errorf("Name = %q, want %q", obs.Name, "my-rule")
	}

	if obs.Description != "some desc" {
		t.Errorf("Description = %q, want %q", obs.Description, "some desc")
	}

	if obs.Mode != "BLOCK" {
		t.Errorf("Mode = %q, want BLOCK", obs.Mode)
	}

	if len(obs.Matchers) != 1 || obs.Matchers[0] != ".*-SNAPSHOT.*" {
		t.Errorf("Matchers = %v, want [.*-SNAPSHOT.*]", obs.Matchers)
	}
}

// ---- routingRuleWithListFallback.Get ----

// newTestNexusClient creates a *nexus3.NexusClient pointed at
// a test HTTP server.
func newTestNexusClient(t *testing.T, handler http.Handler) *nexus3.NexusClient {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := client.Config{URL: srv.URL, Username: "u", Password: "p"}

	nc := nexus3.NewClient(cfg)
	if nc == nil {
		t.Fatal("nexus3.NewClient returned nil")
	}

	return nc
}

// TestRoutingRuleWithListFallback_Get_DirectSuccess tests that
// routingRuleWithListFallback.Get returns the rule directly when the GET
// request succeeds.
func TestRoutingRuleWithListFallback_Get_DirectSuccess(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"my-rule","mode":"BLOCK","matchers":[".*"]}`))
	})

	nc := newTestNexusClient(t, handler)
	svc := &routingRuleWithListFallback{nc.RoutingRule}

	got, err := svc.Get("my-rule")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if got == nil || got.Name != "my-rule" {
		t.Errorf("Get() = %v, want name=my-rule", got)
	}
}

// TestRoutingRuleWithListFallback_Get_ListFallback tests that
// routingRuleWithListFallback.Get falls back to listing all rules when the
// GET request returns a 404, and returns the matching rule from the list.
func TestRoutingRuleWithListFallback_Get_ListFallback(t *testing.T) {
	t.Parallel()

	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/service/rest/v1/routing-rules/target-rule" {
			// Return 404 to trigger the list fallback.
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"other-rule","mode":"BLOCK","matchers":[".*"]},{"name":"target-rule","mode":"ALLOW","matchers":[".*-release.*"]}]`))
	})

	nc := newTestNexusClient(t, handler)
	svc := &routingRuleWithListFallback{nc.RoutingRule}

	got, err := svc.Get("target-rule")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if got == nil || got.Name != "target-rule" {
		t.Errorf("Get() = %v, want name=target-rule", got)
	}

	if calls < 2 {
		t.Errorf("expected at least 2 HTTP calls (GET then LIST), got %d", calls)
	}
}

// TestRoutingRuleWithListFallback_Get_NotFoundAfterFallback tests that
// routingRuleWithListFallback.Get returns an error when the rule is not found
// after falling back to listing all rules.
func TestRoutingRuleWithListFallback_Get_NotFoundAfterFallback(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/service/rest/v1/routing-rules/missing" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"other-rule","mode":"BLOCK","matchers":[".*"]}]`))
	})

	nc := newTestNexusClient(t, handler)
	svc := &routingRuleWithListFallback{nc.RoutingRule}

	got, err := svc.Get("missing")

	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}

	if err == nil {
		t.Error("Get() expected error for missing rule, got nil")
	}
}

// TestRoutingRuleWithListFallback_Get_NonNotFoundError tests that
// routingRuleWithListFallback.Get returns an error when the GET request
// returns a non-404 error.
func TestRoutingRuleWithListFallback_Get_NonNotFoundError(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	nc := newTestNexusClient(t, handler)
	svc := &routingRuleWithListFallback{nc.RoutingRule}

	_, err := svc.Get("any-rule")
	if err == nil {
		t.Error("Get() expected error on server error, got nil")
	}

	if errors.Is(err, nil) {
		t.Error("expected non-nil error")
	}
}
