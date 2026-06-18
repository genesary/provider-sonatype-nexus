// Package content provides clients for the Nexus content API group.
package content

import (
	"context"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// CleanupPolicyClient defines the interface for cleanup policy operations.
type CleanupPolicyClient interface {
	GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	DeleteCleanupPolicy(ctx context.Context, name string) error
}

// withListFallback wraps nexus.CleanupPolicyService and overrides
// GetCleanupPolicy to fall back to a list-and-filter search when
// the per-name GET returns 404.
// Some Nexus deployments serve cleanup policies only through the list endpoint.
type withListFallback struct {
	nexus.CleanupPolicyService
}

// GetCleanupPolicy tries the individual GET endpoint first; on a not-found
// error it lists all policies and returns the one whose name matches.
func (w *withListFallback) GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error) {
	policy, err := w.CleanupPolicyService.GetCleanupPolicy(ctx, name)
	if err == nil {
		return policy, nil
	}

	if !IsNotFound(err) {
		return nil, err
	}

	all, listErr := w.ListCleanupPolicies(ctx)
	if listErr != nil {
		return nil, err
	}

	for _, p := range all {
		if p != nil && p.Name == name {
			return p, nil
		}
	}

	return nil, err
}

// NewCleanupPolicyClient creates a CleanupPolicyClient from credentials.
func NewCleanupPolicyClient(creds nexus.Credentials) (CleanupPolicyClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &withListFallback{nexusClient.CleanupPolicy()}, nil
}

// GenerateCleanupPolicy builds a Nexus CleanupPolicy object from a CR spec.
func GenerateCleanupPolicy(cr *contentv1alpha1.CleanupPolicy) *cleanuppolicies.CleanupPolicy {
	params := cr.Spec.ForProvider

	policy := &cleanuppolicies.CleanupPolicy{
		Name:                    params.Name,
		Format:                  cleanuppolicies.RepositoryFormat(params.Format),
		Notes:                   params.Notes,
		CriteriaLastBlobUpdated: params.CriteriaLastBlobUpdated,
		CriteriaLastDownloaded:  params.CriteriaLastDownloaded,
		CriteriaAssetRegex:      params.CriteriaAssetRegex,
	}

	if params.CriteriaReleaseType != nil {
		rt := cleanuppolicies.CriteriaReleaseType(*params.CriteriaReleaseType)
		policy.CriteriaReleaseType = &rt
	}

	if params.Retain != nil {
		policy.Retain = *params.Retain
	}

	return policy
}

// IsCleanupPolicyUpToDate reports whether the CR matches the observed policy.
func IsCleanupPolicyUpToDate(cr *contentv1alpha1.CleanupPolicy, observed *cleanuppolicies.CleanupPolicy) bool {
	params := cr.Spec.ForProvider

	if string(observed.Format) != params.Format {
		return false
	}

	if !areCriteriaFieldsEqual(params, observed) {
		return false
	}

	return helpers.IsComparablePtrEqualComparablePtr(params.CriteriaReleaseType, (*string)(observed.CriteriaReleaseType)) &&
		helpers.IsComparablePtrEqualComparable(params.Retain, observed.Retain)
}

// GenerateCleanupPolicyObservation returns the observed policy state.
func GenerateCleanupPolicyObservation(observed *cleanuppolicies.CleanupPolicy) contentv1alpha1.CleanupPolicyObservation {
	if observed == nil {
		return contentv1alpha1.CleanupPolicyObservation{}
	}

	obs := contentv1alpha1.CleanupPolicyObservation{
		Name:   observed.Name,
		Format: string(observed.Format),
		Retain: observed.Retain,
	}

	if observed.Notes != nil {
		obs.Notes = *observed.Notes
	}

	if observed.CriteriaLastBlobUpdated != nil {
		obs.CriteriaLastBlobUpdated = *observed.CriteriaLastBlobUpdated
	}

	if observed.CriteriaLastDownloaded != nil {
		obs.CriteriaLastDownloaded = *observed.CriteriaLastDownloaded
	}

	if observed.CriteriaReleaseType != nil {
		obs.CriteriaReleaseType = string(*observed.CriteriaReleaseType)
	}

	if observed.CriteriaAssetRegex != nil {
		obs.CriteriaAssetRegex = *observed.CriteriaAssetRegex
	}

	return obs
}

// IsNotFound reports whether an error indicates a resource was not found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(strings.ToLower(msg), "does not exist")
}

// areCriteriaFieldsEqual checks string/int pointer criteria fields equality.
func areCriteriaFieldsEqual(params contentv1alpha1.CleanupPolicyParameters, observed *cleanuppolicies.CleanupPolicy) bool {
	return helpers.IsComparablePtrEqualComparablePtr(params.Notes, observed.Notes) &&
		helpers.IsComparablePtrEqualComparablePtr(params.CriteriaLastBlobUpdated, observed.CriteriaLastBlobUpdated) &&
		helpers.IsComparablePtrEqualComparablePtr(params.CriteriaLastDownloaded, observed.CriteriaLastDownloaded) &&
		helpers.IsComparablePtrEqualComparablePtr(params.CriteriaAssetRegex, observed.CriteriaAssetRegex)
}
