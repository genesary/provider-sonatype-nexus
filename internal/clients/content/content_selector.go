package content

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// ContentSelectorClient defines the interface for content selector operations.
type ContentSelectorClient interface {
	GetContentSelector(ctx context.Context, name string) (*security.ContentSelector, error)
	CreateContentSelector(ctx context.Context, cs security.ContentSelector) error
	UpdateContentSelector(ctx context.Context, name string, cs security.ContentSelector) error
	DeleteContentSelector(ctx context.Context, name string) error
}

// NewContentSelectorClient creates a ContentSelectorClient from credentials.
func NewContentSelectorClient(creds nexus.Credentials) (ContentSelectorClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nexusClient.Security(), nil
}

// GenerateContentSelector builds a Nexus ContentSelector from a CR spec.
func GenerateContentSelector(cr *contentv1alpha1.ContentSelector) security.ContentSelector {
	params := cr.Spec.ForProvider

	csData := security.ContentSelector{
		Name:       params.Name,
		Expression: params.Expression,
	}

	if params.Description != nil {
		csData.Description = *params.Description
	}

	return csData
}

// IsContentSelectorUpToDate reports whether the CR is up to date.
func IsContentSelectorUpToDate(contentSel *contentv1alpha1.ContentSelector, observed *security.ContentSelector) bool {
	if contentSel.Spec.ForProvider.Expression != observed.Expression {
		return false
	}

	if contentSel.Spec.ForProvider.Description != nil &&
		*contentSel.Spec.ForProvider.Description != observed.Description {
		return false
	}

	return true
}

// GenerateContentSelectorObservation returns the observed selector state.
func GenerateContentSelectorObservation(observed *security.ContentSelector) contentv1alpha1.ContentSelectorObservation {
	if observed == nil {
		return contentv1alpha1.ContentSelectorObservation{}
	}

	return contentv1alpha1.ContentSelectorObservation{
		Name:        observed.Name,
		Description: observed.Description,
		Expression:  observed.Expression,
	}
}
