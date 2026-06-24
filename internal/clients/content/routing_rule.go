package content

import (
	nexusschema "github.com/datadrivers/go-nexus-client/nexus3/schema"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// RoutingRuleClient defines the interface for routing rule operations.
type RoutingRuleClient interface {
	Get(name string) (*nexusschema.RoutingRule, error)
	Create(rule *nexusschema.RoutingRule) error
	Update(rule *nexusschema.RoutingRule) error
	Delete(name string) error
}

// NewRoutingRuleClient creates a RoutingRuleClient from credentials.
func NewRoutingRuleClient(creds nexus.Credentials) (RoutingRuleClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.RoutingRule, nil
}

// GenerateRoutingRule builds a Nexus RoutingRule from a CR spec.
func GenerateRoutingRule(cr *contentv1alpha1.RoutingRule) *nexusschema.RoutingRule {
	params := cr.Spec.ForProvider

	rule := &nexusschema.RoutingRule{
		Name:     params.Name,
		Mode:     nexusschema.RoutingRuleMode(params.Mode),
		Matchers: params.Matchers,
	}

	if params.Description != nil {
		rule.Description = *params.Description
	}

	return rule
}

// IsRoutingRuleUpToDate reports whether the CR spec matches the observed rule.
func IsRoutingRuleUpToDate(params *contentv1alpha1.RoutingRuleParameters, observed *contentv1alpha1.RoutingRuleObservation) bool {
	return params.Mode == observed.Mode &&
		helpers.AreStringSlicesEqual(params.Matchers, observed.Matchers) &&
		helpers.IsComparablePtrEqualComparable(params.Description, observed.Description)
}

// GenerateRoutingRuleObservation converts an observed Nexus routing rule into
// a CRD observation type.
func GenerateRoutingRuleObservation(observed *nexusschema.RoutingRule) contentv1alpha1.RoutingRuleObservation {
	if observed == nil {
		return contentv1alpha1.RoutingRuleObservation{}
	}

	return contentv1alpha1.RoutingRuleObservation{
		Name:        observed.Name,
		Description: observed.Description,
		Mode:        string(observed.Mode),
		Matchers:    observed.Matchers,
	}
}
