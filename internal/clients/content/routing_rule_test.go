package content

import (
	"testing"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

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

	if got.GetName() != "my-rule" {
		t.Errorf("Name = %q, want %q", got.GetName(), "my-rule")
	}

	if got.GetMode() != "BLOCK" {
		t.Errorf("Mode = %q, want BLOCK", got.GetMode())
	}

	if len(got.Matchers) != 1 || got.Matchers[0] != ".*-SNAPSHOT.*" {
		t.Errorf("Matchers = %v, want [.*-SNAPSHOT.*]", got.Matchers)
	}

	if got.GetDescription() != "" {
		t.Errorf("Description = %q, want empty", got.GetDescription())
	}
}

// TestGenerateRoutingRule_WithDescription tests that
// GenerateRoutingRule correctly converts a CR to a Nexus RoutingRule
// when a description is provided.
func TestGenerateRoutingRule_WithDescription(t *testing.T) {
	t.Parallel()

	cr := newCR("my-rule", "ALLOW", []string{".*"}, new("allow all"))
	got := GenerateRoutingRule(cr)

	if got.GetDescription() != "allow all" {
		t.Errorf("Description = %q, want %q", got.GetDescription(), "allow all")
	}

	if got.GetMode() != "ALLOW" {
		t.Errorf("Mode = %q, want ALLOW", got.GetMode())
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

// TestIsRoutingRuleUpToDate_NilDescriptionDropped tests that
// IsRoutingRuleUpToDate reports drift when the CR spec sets no description but
// the observed rule still carries one: GenerateRoutingRule submits an empty
// description in that case, so the rule is not in the state the spec asks for.
func TestIsRoutingRuleUpToDate_NilDescriptionDropped(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*"}, nil)
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:        "r",
		Mode:        "BLOCK",
		Matchers:    []string{".*"},
		Description: "some description",
	}

	if IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("a description dropped from the spec should be reported as drift")
	}
}

// TestIsRoutingRuleUpToDate_NilDescriptionMatchesEmpty tests that
// IsRoutingRuleUpToDate reports no drift when neither the CR spec nor the
// observed rule carries a description.
func TestIsRoutingRuleUpToDate_NilDescriptionMatchesEmpty(t *testing.T) {
	t.Parallel()

	cr := newCR("r", "BLOCK", []string{".*"}, nil)
	observed := &contentv1alpha1.RoutingRuleObservation{
		Name:     "r",
		Mode:     "BLOCK",
		Matchers: []string{".*"},
	}

	if !IsRoutingRuleUpToDate(&cr.Spec.ForProvider, observed) {
		t.Error("an unset description matching an empty one should not be drift")
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

	rule := &v395.RoutingRuleXO{
		Name:        new("my-rule"),
		Description: new("some desc"),
		Mode:        new("BLOCK"),
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

// ---- IsContentSelectorUpToDate ----

// TestIsContentSelectorUpToDate_DescriptionDropped tests that removing the
// description from the spec is reported as drift: GenerateContentSelector
// submits an empty description, which is how Nexus is told to clear it.
func TestIsContentSelectorUpToDate_DescriptionDropped(t *testing.T) {
	t.Parallel()

	cr := &contentv1alpha1.ContentSelector{}
	cr.Spec.ForProvider = contentv1alpha1.ContentSelectorParameters{
		Name:       "selector",
		Expression: `format == "maven2"`,
	}

	observed := &v395.ContentSelectorApiResponse{
		Name:        new("selector"),
		Expression:  new(`format == "maven2"`),
		Description: new("leftover"),
	}

	if IsContentSelectorUpToDate(cr, observed) {
		t.Error("IsContentSelectorUpToDate() = true, want false when the description was dropped")
	}

	observed.Description = new("")

	if !IsContentSelectorUpToDate(cr, observed) {
		t.Error("IsContentSelectorUpToDate() = false, want true once the description is gone")
	}
}
