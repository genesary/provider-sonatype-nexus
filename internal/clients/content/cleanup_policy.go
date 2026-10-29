// Package content provides clients for the Nexus content API group.
package content

import (
	"context"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// CleanupPolicy is the Nexus representation of a cleanup policy.
type CleanupPolicy = v395.CleanupPolicyResourceXO

// CleanupPolicyClient defines the interface for cleanup policy operations.
type CleanupPolicyClient interface {
	Get(ctx context.Context, name string) (*CleanupPolicy, error)
	Create(ctx context.Context, policy *CleanupPolicy) error
	Update(ctx context.Context, policy *CleanupPolicy) error
	Delete(ctx context.Context, name string) error
}

// cleanupPolicyClient implements CleanupPolicyClient over the Nexus API.
type cleanupPolicyClient struct {
	api *v395.CleanupPoliciesAPIService
}

// NewCleanupPolicyClient creates a CleanupPolicyClient from credentials.
func NewCleanupPolicyClient(creds nexus.Credentials) (CleanupPolicyClient, error) {
	nc, err := nexus.NewClient(creds)
	if err != nil {
		return nil, err
	}

	return &cleanupPolicyClient{api: nc.CleanupPoliciesAPI}, nil
}

// Get tries the individual GET endpoint first; on a not-found error it lists
// all policies and returns the one whose name matches. Some Nexus deployments
// serve cleanup policies only through the list endpoint.
func (c *cleanupPolicyClient) Get(ctx context.Context, name string) (*CleanupPolicy, error) {
	policy, resp, err := c.api.GetCleanupPolicies(ctx, name).Execute()

	err = nexus.APIError(resp, err)
	if err == nil {
		return policy, nil
	}

	if !nexus.IsNotFound(err) {
		return nil, err
	}

	all, _, listErr := c.api.ListCleanupPolicies(ctx).Execute()
	if listErr != nil {
		return nil, err
	}

	for i := range all {
		if all[i].Name == name {
			return &all[i], nil
		}
	}

	return nil, err
}

// Create creates a cleanup policy.
func (c *cleanupPolicyClient) Create(ctx context.Context, policy *CleanupPolicy) error {
	return nexus.APIError(c.api.CreateCleanupPolicies(ctx).CleanupPolicyResourceXO(*policy).Execute())
}

// Update updates the cleanup policy named by policy.
func (c *cleanupPolicyClient) Update(ctx context.Context, policy *CleanupPolicy) error {
	return nexus.APIError(c.api.UpdateCleanupPolicies(ctx, policy.Name).CleanupPolicyResourceXO(*policy).Execute())
}

// Delete deletes the cleanup policy with the given name.
func (c *cleanupPolicyClient) Delete(ctx context.Context, name string) error {
	return nexus.APIError(c.api.DeleteCleanupPolicies(ctx, name).Execute())
}

// GenerateCleanupPolicy builds a Nexus CleanupPolicy object from a CR spec.
func GenerateCleanupPolicy(cr *contentv1alpha1.CleanupPolicy) *CleanupPolicy {
	params := cr.Spec.ForProvider

	policy := &CleanupPolicy{
		Name:                    params.Name,
		Format:                  params.Format,
		Notes:                   params.Notes,
		CriteriaLastBlobUpdated: toInt64Ptr(params.CriteriaLastBlobUpdated),
		CriteriaLastDownloaded:  toInt64Ptr(params.CriteriaLastDownloaded),
		CriteriaReleaseType:     params.CriteriaReleaseType,
		CriteriaAssetRegex:      params.CriteriaAssetRegex,
	}

	if params.Retain != nil {
		policy.Retain = new(helpers.ClampInt32(*params.Retain))
	}

	return policy
}

// IsCleanupPolicyUpToDate reports whether the CR matches the observed policy.
func IsCleanupPolicyUpToDate(cr *contentv1alpha1.CleanupPolicy, observed *CleanupPolicy) bool {
	params := cr.Spec.ForProvider

	if observed.Format != params.Format {
		return false
	}

	if !areCriteriaFieldsEqual(params, observed) {
		return false
	}

	return helpers.IsComparablePtrEqualComparablePtr(params.CriteriaReleaseType, observed.CriteriaReleaseType) &&
		helpers.IsComparablePtrEqualComparable(params.Retain, int(observed.GetRetain()))
}

// GenerateCleanupPolicyObservation returns the observed policy state.
func GenerateCleanupPolicyObservation(observed *CleanupPolicy) contentv1alpha1.CleanupPolicyObservation {
	if observed == nil {
		return contentv1alpha1.CleanupPolicyObservation{}
	}

	return contentv1alpha1.CleanupPolicyObservation{
		Name:                    observed.Name,
		Format:                  observed.Format,
		Retain:                  int(observed.GetRetain()),
		Notes:                   observed.GetNotes(),
		CriteriaLastBlobUpdated: int(observed.GetCriteriaLastBlobUpdated()),
		CriteriaLastDownloaded:  int(observed.GetCriteriaLastDownloaded()),
		CriteriaReleaseType:     observed.GetCriteriaReleaseType(),
		CriteriaAssetRegex:      observed.GetCriteriaAssetRegex(),
	}
}

// areCriteriaFieldsEqual reports whether the notes and criteria fields in the
// CR spec match the observed policy.
func areCriteriaFieldsEqual(params contentv1alpha1.CleanupPolicyParameters, observed *CleanupPolicy) bool {
	return helpers.IsComparablePtrEqualComparablePtr(params.Notes, observed.Notes) &&
		helpers.IsComparablePtrEqualComparablePtr(toInt64Ptr(params.CriteriaLastBlobUpdated), observed.CriteriaLastBlobUpdated) &&
		helpers.IsComparablePtrEqualComparablePtr(toInt64Ptr(params.CriteriaLastDownloaded), observed.CriteriaLastDownloaded) &&
		helpers.IsComparablePtrEqualComparablePtr(params.CriteriaAssetRegex, observed.CriteriaAssetRegex)
}

// toInt64Ptr widens an optional int, keeping nil as nil.
func toInt64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}

	return new(int64(*value))
}
