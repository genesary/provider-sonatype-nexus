// Package content provides clients for the Nexus content API group.
package content

import (
	"context"
	"strings"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// CleanupPolicyClient defines the interface for cleanup policy operations.
type CleanupPolicyClient interface {
	GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	DeleteCleanupPolicy(ctx context.Context, name string) error
}

// NewCleanupPolicyClient creates a CleanupPolicyClient from credentials.
func NewCleanupPolicyClient(creds nexus.Credentials) (CleanupPolicyClient, error) {
	nexusClient, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return nexusClient.CleanupPolicy(), nil
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

	return isReleaseTypeEqual(params.CriteriaReleaseType, observed.CriteriaReleaseType) &&
		isRetainEqual(params.Retain, observed.Retain)
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
	return stringPtrEqual(params.Notes, observed.Notes) &&
		intPtrEqual(params.CriteriaLastBlobUpdated, observed.CriteriaLastBlobUpdated) &&
		intPtrEqual(params.CriteriaLastDownloaded, observed.CriteriaLastDownloaded) &&
		stringPtrEqual(params.CriteriaAssetRegex, observed.CriteriaAssetRegex)
}

// isReleaseTypeEqual compares the desired and observed release type values.
func isReleaseTypeEqual(desired *string, observed *cleanuppolicies.CriteriaReleaseType) bool {
	if desired == nil {
		return observed == nil
	}

	if observed == nil {
		return false
	}

	return string(*observed) == *desired
}

// isRetainEqual compares the desired and observed retain values.
func isRetainEqual(desired *int, observedRetain int) bool {
	desiredVal := 0
	if desired != nil {
		desiredVal = *desired
	}

	return observedRetain == desiredVal
}

// stringPtrEqual returns true when both are nil or point to equal values.
func stringPtrEqual(first, second *string) bool {
	if first == nil && second == nil {
		return true
	}

	if first == nil || second == nil {
		return false
	}

	return *first == *second
}

// intPtrEqual returns true when both are nil or point to equal values.
func intPtrEqual(first, second *int) bool {
	if first == nil && second == nil {
		return true
	}

	if first == nil || second == nil {
		return false
	}

	return *first == *second
}
