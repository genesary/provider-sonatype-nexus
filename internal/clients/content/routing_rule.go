package content

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// RoutingRuleClient defines the interface for routing rule operations.
type RoutingRuleClient interface {
	Get(ctx context.Context, name string) (*v395.RoutingRuleXO, error)
	Create(ctx context.Context, rule *v395.RoutingRuleXO) error
	Update(ctx context.Context, rule *v395.RoutingRuleXO) error
	Delete(ctx context.Context, name string) error
}

// routingRuleClient implements RoutingRuleClient over the Nexus API.
type routingRuleClient struct {
	api *v395.RoutingRulesAPIService
}

// NewRoutingRuleClient creates a RoutingRuleClient from credentials.
func NewRoutingRuleClient(creds nexus.Credentials) (RoutingRuleClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &routingRuleClient{api: nc.RoutingRulesAPI}, nil
}

// Get returns the routing rule with the given name.
func (c *routingRuleClient) Get(ctx context.Context, name string) (*v395.RoutingRuleXO, error) {
	rule, resp, err := c.api.GetRoutingRules(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return rule, nil
}

// Create creates a routing rule.
func (c *routingRuleClient) Create(ctx context.Context, rule *v395.RoutingRuleXO) error {
	return nexus.APIError(c.api.CreateRoutingRules(ctx).RoutingRuleXO(*rule).Execute())
}

// Update updates the routing rule named by rule.
func (c *routingRuleClient) Update(ctx context.Context, rule *v395.RoutingRuleXO) error {
	return nexus.APIError(c.api.UpdateRoutingRules(ctx, rule.GetName()).RoutingRuleXO(*rule).Execute())
}

// Delete deletes the routing rule with the given name.
func (c *routingRuleClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteRoutingRules(ctx, name).Execute())
}

// GenerateRoutingRule builds a Nexus RoutingRule from a CR spec.
func GenerateRoutingRule(cr *contentv1alpha1.RoutingRule) *v395.RoutingRuleXO {
	params := cr.Spec.ForProvider

	return &v395.RoutingRuleXO{
		Name:        new(params.Name),
		Mode:        new(params.Mode),
		Matchers:    params.Matchers,
		Description: new(ptr.Deref(params.Description, "")),
	}
}

// IsRoutingRuleUpToDate reports whether the CR spec matches the observed rule.
//
// A spec without a description asks for a rule without one: GenerateRoutingRule
// submits an empty description in that case, and Nexus stores it verbatim, so
// dropping the description from the spec has to be reported as drift.
func IsRoutingRuleUpToDate(params *contentv1alpha1.RoutingRuleParameters, observed *contentv1alpha1.RoutingRuleObservation) bool {
	return params.Mode == observed.Mode &&
		helpers.AreStringSlicesEqual(params.Matchers, observed.Matchers) &&
		ptr.Deref(params.Description, "") == observed.Description
}

// GenerateRoutingRuleObservation converts an observed Nexus routing rule into
// a CRD observation type.
func GenerateRoutingRuleObservation(observed *v395.RoutingRuleXO) contentv1alpha1.RoutingRuleObservation {
	if observed == nil {
		return contentv1alpha1.RoutingRuleObservation{}
	}

	return contentv1alpha1.RoutingRuleObservation{
		Name:        observed.GetName(),
		Description: observed.GetDescription(),
		Mode:        observed.GetMode(),
		Matchers:    observed.Matchers,
	}
}
