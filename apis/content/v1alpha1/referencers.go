package v1alpha1

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
)

// ResolveReferences resolves the blob store, cleanup policies, group members
// and routing rule of a Repository from the managed resources it references,
// by their external name. References and selectors resolve in the
// Repository's own namespace.
func (mg *Repository) ResolveReferences(ctx context.Context, reader client.Reader) error {
	resolver := reference.NewAPIResolver(reader, mg)
	namespace := mg.GetNamespace()
	params := &mg.Spec.ForProvider

	if params.Storage != nil {
		err := mg.resolveBlobStore(ctx, resolver)
		if err != nil {
			return err
		}
	}

	if params.Cleanup != nil {
		err := resolveNames(ctx, resolver, namespace, "spec.forProvider.cleanup.policyNames",
			reference.To{Managed: &CleanupPolicy{}, List: &CleanupPolicyList{}},
			&params.Cleanup.PolicyNames, &params.Cleanup.PolicyNameRefs, params.Cleanup.PolicyNameSelector)
		if err != nil {
			return err
		}
	}

	if params.Group != nil {
		err := resolveNames(ctx, resolver, namespace, "spec.forProvider.group.memberNames",
			reference.To{Managed: &Repository{}, List: &RepositoryList{}},
			&params.Group.MemberNames, &params.Group.MemberNameRefs, params.Group.MemberNameSelector)
		if err != nil {
			return err
		}
	}

	return resolveName(ctx, resolver, namespace, "spec.forProvider.routingRule",
		reference.To{Managed: &RoutingRule{}, List: &RoutingRuleList{}},
		&params.RoutingRule, &params.RoutingRuleRef, params.RoutingRuleSelector)
}

// resolveBlobStore resolves spec.forProvider.storage.blobStoreName, which
// unlike the other names is not a pointer.
func (mg *Repository) resolveBlobStore(ctx context.Context, resolver *reference.APIResolver) error {
	storage := mg.Spec.ForProvider.Storage

	blobStore, err := resolver.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: storage.BlobStoreName,
		Reference:    storage.BlobStoreNameRef,
		Selector:     storage.BlobStoreNameSelector,
		To:           reference.To{Managed: &instancev1alpha1.BlobStore{}, List: &instancev1alpha1.BlobStoreList{}},
		Extract:      reference.ExternalName(),
		Namespace:    mg.GetNamespace(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.storage.blobStoreName")
	}

	storage.BlobStoreName = blobStore.ResolvedValue
	storage.BlobStoreNameRef = blobStore.ResolvedReference

	return nil
}
