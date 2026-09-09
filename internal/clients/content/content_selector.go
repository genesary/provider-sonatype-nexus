package content

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"k8s.io/utils/ptr"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// ContentSelectorClient defines the interface for content selector operations.
type ContentSelectorClient interface {
	Get(name string) (*security.ContentSelector, error)
	Create(cs security.ContentSelector) error
	Update(name string, cs security.ContentSelector) error
	Delete(name string) error
}

// NewContentSelectorClient creates a ContentSelectorClient from credentials.
func NewContentSelectorClient(creds nexus.Credentials) (ContentSelectorClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nc.Security.ContentSelector, nil
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
//
// A spec without a description asks for a selector without one:
// GenerateContentSelector submits an empty description in that case, and Nexus
// stores it verbatim, so dropping the description from the spec has to be
// reported as drift.
func IsContentSelectorUpToDate(contentSel *contentv1alpha1.ContentSelector, observed *security.ContentSelector) bool {
	if contentSel.Spec.ForProvider.Expression != observed.Expression {
		return false
	}

	return ptr.Deref(contentSel.Spec.ForProvider.Description, "") == observed.Description
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
