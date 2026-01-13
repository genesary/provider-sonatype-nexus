package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// GetCondition of this BlobStore.
func (mg *BlobStore) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this BlobStore.
func (mg *BlobStore) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// GetDeletionPolicy of this BlobStore.
func (mg *BlobStore) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetDeletionPolicy of this BlobStore.
func (mg *BlobStore) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// GetManagementPolicies of this BlobStore.
func (mg *BlobStore) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// SetManagementPolicies of this BlobStore.
func (mg *BlobStore) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// GetProviderConfigReference of this BlobStore.
func (mg *BlobStore) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this BlobStore.
func (mg *BlobStore) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// GetPublishConnectionDetailsTo of this BlobStore.
func (mg *BlobStore) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// SetPublishConnectionDetailsTo of this BlobStore.
func (mg *BlobStore) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// GetWriteConnectionSecretToReference of this BlobStore.
func (mg *BlobStore) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this BlobStore.
func (mg *BlobStore) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// GetCondition of this Repository.
func (mg *Repository) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this Repository.
func (mg *Repository) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// GetDeletionPolicy of this Repository.
func (mg *Repository) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetDeletionPolicy of this Repository.
func (mg *Repository) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// GetManagementPolicies of this Repository.
func (mg *Repository) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// SetManagementPolicies of this Repository.
func (mg *Repository) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// GetProviderConfigReference of this Repository.
func (mg *Repository) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this Repository.
func (mg *Repository) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// GetPublishConnectionDetailsTo of this Repository.
func (mg *Repository) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// SetPublishConnectionDetailsTo of this Repository.
func (mg *Repository) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// GetWriteConnectionSecretToReference of this Repository.
func (mg *Repository) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this Repository.
func (mg *Repository) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// GetCondition of this User.
func (mg *User) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this User.
func (mg *User) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// GetDeletionPolicy of this User.
func (mg *User) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetDeletionPolicy of this User.
func (mg *User) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// GetManagementPolicies of this User.
func (mg *User) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// SetManagementPolicies of this User.
func (mg *User) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// GetProviderConfigReference of this User.
func (mg *User) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this User.
func (mg *User) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// GetPublishConnectionDetailsTo of this User.
func (mg *User) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// SetPublishConnectionDetailsTo of this User.
func (mg *User) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// GetWriteConnectionSecretToReference of this User.
func (mg *User) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this User.
func (mg *User) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// GetCondition of this Role.
func (mg *Role) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this Role.
func (mg *Role) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// GetDeletionPolicy of this Role.
func (mg *Role) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetDeletionPolicy of this Role.
func (mg *Role) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// GetManagementPolicies of this Role.
func (mg *Role) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// SetManagementPolicies of this Role.
func (mg *Role) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// GetProviderConfigReference of this Role.
func (mg *Role) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this Role.
func (mg *Role) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// GetPublishConnectionDetailsTo of this Role.
func (mg *Role) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// SetPublishConnectionDetailsTo of this Role.
func (mg *Role) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// GetWriteConnectionSecretToReference of this Role.
func (mg *Role) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this Role.
func (mg *Role) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

// ProviderConfigUsage methods

// GetProviderConfigReference of this ProviderConfigUsage.
func (p *ProviderConfigUsage) GetProviderConfigReference() xpv1.Reference {
	return p.ProviderConfigUsage.ProviderConfigReference
}

// SetProviderConfigReference of this ProviderConfigUsage.
func (p *ProviderConfigUsage) SetProviderConfigReference(r xpv1.Reference) {
	p.ProviderConfigUsage.ProviderConfigReference = r
}

// GetResourceReference of this ProviderConfigUsage.
func (p *ProviderConfigUsage) GetResourceReference() xpv1.TypedReference {
	return p.ProviderConfigUsage.ResourceReference
}

// SetResourceReference of this ProviderConfigUsage.
func (p *ProviderConfigUsage) SetResourceReference(r xpv1.TypedReference) {
	p.ProviderConfigUsage.ResourceReference = r
}
