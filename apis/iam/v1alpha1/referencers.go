package v1alpha1

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"

	contentv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// ResolveReferences resolves the privileges and roles of a Role from the
// Privilege and Role managed resources it references, by their external
// name. References and selectors resolve in the Role's own namespace.
func (mg *Role) ResolveReferences(ctx context.Context, reader client.Reader) error {
	resolver := reference.NewAPIResolver(reader, mg)
	params := &mg.Spec.ForProvider

	err := resolveNames(ctx, resolver, mg.GetNamespace(), "spec.forProvider.privileges",
		reference.To{Managed: &Privilege{}, List: &PrivilegeList{}},
		&params.Privileges, &params.PrivilegeRefs, params.PrivilegeSelector)
	if err != nil {
		return err
	}

	return resolveNames(ctx, resolver, mg.GetNamespace(), "spec.forProvider.roles",
		reference.To{Managed: &Role{}, List: &RoleList{}},
		&params.Roles, &params.RoleRefs, params.RoleSelector)
}

// ResolveReferences resolves the roles of a User from the Role managed
// resources it references, by their external name. References and selectors
// resolve in the User's own namespace.
func (mg *User) ResolveReferences(ctx context.Context, reader client.Reader) error {
	params := &mg.Spec.ForProvider

	return resolveNames(ctx, reference.NewAPIResolver(reader, mg), mg.GetNamespace(), "spec.forProvider.roles",
		reference.To{Managed: &Role{}, List: &RoleList{}},
		&params.Roles, &params.RoleRefs, params.RoleSelector)
}

// ResolveReferences resolves the repository and content selector of a
// Privilege from the Repository and ContentSelector managed resources it
// references, by their external name. References and selectors resolve in the
// Privilege's own namespace.
func (mg *Privilege) ResolveReferences(ctx context.Context, reader client.Reader) error {
	resolver := reference.NewAPIResolver(reader, mg)
	params := &mg.Spec.ForProvider

	err := resolveName(ctx, resolver, mg.GetNamespace(), "spec.forProvider.repository",
		reference.To{Managed: &contentv1alpha1.Repository{}, List: &contentv1alpha1.RepositoryList{}},
		&params.Repository, &params.RepositoryRef, params.RepositorySelector)
	if err != nil {
		return err
	}

	return resolveName(ctx, resolver, mg.GetNamespace(), "spec.forProvider.contentSelector",
		reference.To{Managed: &contentv1alpha1.ContentSelector{}, List: &contentv1alpha1.ContentSelectorList{}},
		&params.ContentSelector, &params.ContentSelectorRef, params.ContentSelectorSelector)
}
