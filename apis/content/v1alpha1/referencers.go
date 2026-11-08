package v1alpha1

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/apis/internal/resolve"
)

// ResolveReferences resolves the blob store, cleanup policies, group members
// and routing rule of a Repository from the managed resources it references,
// by their external name. References and selectors resolve in the
// Repository's own namespace, and a group never resolves to itself.
func (mg *Repository) ResolveReferences(ctx context.Context, reader client.Reader) error {
	resolver := reference.NewAPIResolver(reader, mg)
	namespace := mg.GetNamespace()
	params := &mg.Spec.ForProvider

	if storage := params.Storage; storage != nil {
		err := resolve.Name(ctx, resolver, namespace, "spec.forProvider.storage.blobStoreName",
			reference.To{Managed: &instancev1alpha1.BlobStore{}, List: &instancev1alpha1.BlobStoreList{}},
			&storage.BlobStoreName, &storage.BlobStoreNameRef, storage.BlobStoreNameSelector)
		if err != nil {
			return err
		}
	}

	if cleanup := params.Cleanup; cleanup != nil {
		err := resolve.Names(ctx, resolver, namespace, "", "spec.forProvider.cleanup.policyNames",
			reference.To{Managed: &CleanupPolicy{}, List: &CleanupPolicyList{}},
			&cleanup.PolicyNames, &cleanup.PolicyNameRefs, cleanup.PolicyNameSelector)
		if err != nil {
			return err
		}
	}

	if group := params.Group; group != nil {
		err := resolve.Names(ctx, resolver, namespace, mg.GetName(), "spec.forProvider.group.memberNames",
			reference.To{Managed: &Repository{}, List: &RepositoryList{}},
			&group.MemberNames, &group.MemberNameRefs, group.MemberNameSelector)
		if err != nil {
			return err
		}
	}

	return resolve.OptionalName(ctx, resolver, namespace, "spec.forProvider.routingRule",
		reference.To{Managed: &RoutingRule{}, List: &RoutingRuleList{}},
		&params.RoutingRule, &params.RoutingRuleRef, params.RoutingRuleSelector)
}
