package content

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// ContentSelectorClient defines the interface for content selector operations.
type ContentSelectorClient interface {
	Get(ctx context.Context, name string) (*v395.ContentSelectorApiResponse, error)
	Create(ctx context.Context, cs v395.ContentSelectorApiCreateRequest) error
	Update(ctx context.Context, name string, cs v395.ContentSelectorApiUpdateRequest) error
	Delete(ctx context.Context, name string) error
}

// contentSelectorClient implements ContentSelectorClient over the Nexus API.
type contentSelectorClient struct {
	api *v395.ContentSelectorsAPIService
}

// NewContentSelectorClient creates a ContentSelectorClient from credentials.
func NewContentSelectorClient(creds nexus.Credentials) (ContentSelectorClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &contentSelectorClient{api: nc.ContentSelectorsAPI}, nil
}

// Get returns the content selector with the given name.
func (c *contentSelectorClient) Get(ctx context.Context, name string) (*v395.ContentSelectorApiResponse, error) {
	selector, resp, err := c.api.GetSecurityContentSelectors(ctx, name).Execute()
	if err != nil {
		return nil, nexus.APIError(resp, err)
	}

	return selector, nil
}

// Create creates a content selector.
func (c *contentSelectorClient) Create(ctx context.Context, cs v395.ContentSelectorApiCreateRequest) error {
	return nexus.APIError(c.api.CreateSecurityContentSelectors(ctx).ContentSelectorApiCreateRequest(cs).Execute())
}

// Update updates the content selector with the given name.
func (c *contentSelectorClient) Update(ctx context.Context, name string, cs v395.ContentSelectorApiUpdateRequest) error {
	return nexus.APIError(c.api.UpdateSecurityContentSelectors(ctx, name).ContentSelectorApiUpdateRequest(cs).Execute())
}

// Delete deletes the content selector with the given name.
func (c *contentSelectorClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteSecurityContentSelectors(ctx, name).Execute())
}

// GenerateContentSelector builds a Nexus content selector creation request
// from a CR spec.
//
// The description is always submitted, empty when the spec sets none, so that
// removing it from the spec removes it from Nexus too.
func GenerateContentSelector(cr *contentv1alpha1.ContentSelector) v395.ContentSelectorApiCreateRequest {
	params := cr.Spec.ForProvider

	return v395.ContentSelectorApiCreateRequest{
		Name:        params.Name,
		Expression:  params.Expression,
		Description: new(ptr.Deref(params.Description, "")),
	}
}

// GenerateContentSelectorUpdate builds a Nexus content selector update request
// from a CR spec.
func GenerateContentSelectorUpdate(cr *contentv1alpha1.ContentSelector) v395.ContentSelectorApiUpdateRequest {
	create := GenerateContentSelector(cr)

	return v395.ContentSelectorApiUpdateRequest{
		Expression:  create.Expression,
		Description: create.Description,
	}
}

// IsContentSelectorUpToDate reports whether the CR is up to date.
//
// A spec without a description asks for a selector without one:
// GenerateContentSelector submits an empty description in that case, and Nexus
// stores it verbatim, so dropping the description from the spec has to be
// reported as drift.
func IsContentSelectorUpToDate(contentSel *contentv1alpha1.ContentSelector, observed *v395.ContentSelectorApiResponse) bool {
	if contentSel.Spec.ForProvider.Expression != observed.GetExpression() {
		return false
	}

	return ptr.Deref(contentSel.Spec.ForProvider.Description, "") == observed.GetDescription()
}

// GenerateContentSelectorObservation returns the observed selector state.
func GenerateContentSelectorObservation(observed *v395.ContentSelectorApiResponse) contentv1alpha1.ContentSelectorObservation {
	if observed == nil {
		return contentv1alpha1.ContentSelectorObservation{}
	}

	return contentv1alpha1.ContentSelectorObservation{
		Name:        observed.GetName(),
		Description: observed.GetDescription(),
		Expression:  observed.GetExpression(),
	}
}
