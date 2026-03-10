// Package v1alpha1 contains the v1alpha1 group Sample resources of the Nexus provider.
// +kubebuilder:object:generate=true
// +groupName=nexus.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "nexus.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// BlobStore type metadata.
var (
	BlobStoreKind             = reflect.TypeOf(BlobStore{}).Name()
	BlobStoreGroupKind        = schema.GroupKind{Group: Group, Kind: BlobStoreKind}.String()
	BlobStoreKindAPIVersion   = BlobStoreKind + "." + SchemeGroupVersion.String()
	BlobStoreGroupVersionKind = SchemeGroupVersion.WithKind(BlobStoreKind)
)

// Repository type metadata.
var (
	RepositoryKind             = reflect.TypeOf(Repository{}).Name()
	RepositoryGroupKind        = schema.GroupKind{Group: Group, Kind: RepositoryKind}.String()
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)

// User type metadata.
var (
	UserKind             = reflect.TypeOf(User{}).Name()
	UserGroupKind        = schema.GroupKind{Group: Group, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

// Role type metadata.
var (
	RoleKind             = reflect.TypeOf(Role{}).Name()
	RoleGroupKind        = schema.GroupKind{Group: Group, Kind: RoleKind}.String()
	RoleKindAPIVersion   = RoleKind + "." + SchemeGroupVersion.String()
	RoleGroupVersionKind = SchemeGroupVersion.WithKind(RoleKind)
)

// SecurityRealm type metadata.
var (
	SecurityRealmKind             = reflect.TypeOf(SecurityRealm{}).Name()
	SecurityRealmGroupKind        = schema.GroupKind{Group: Group, Kind: SecurityRealmKind}.String()
	SecurityRealmKindAPIVersion   = SecurityRealmKind + "." + SchemeGroupVersion.String()
	SecurityRealmGroupVersionKind = SchemeGroupVersion.WithKind(SecurityRealmKind)
)

// ContentSelector type metadata.
var (
	ContentSelectorKind             = reflect.TypeOf(ContentSelector{}).Name()
	ContentSelectorGroupKind        = schema.GroupKind{Group: Group, Kind: ContentSelectorKind}.String()
	ContentSelectorKindAPIVersion   = ContentSelectorKind + "." + SchemeGroupVersion.String()
	ContentSelectorGroupVersionKind = SchemeGroupVersion.WithKind(ContentSelectorKind)
)

// Privilege type metadata.
var (
	PrivilegeKind             = reflect.TypeOf(Privilege{}).Name()
	PrivilegeGroupKind        = schema.GroupKind{Group: Group, Kind: PrivilegeKind}.String()
	PrivilegeKindAPIVersion   = PrivilegeKind + "." + SchemeGroupVersion.String()
	PrivilegeGroupVersionKind = SchemeGroupVersion.WithKind(PrivilegeKind)
)

// AnonymousAccess type metadata.
var (
	AnonymousAccessKind             = reflect.TypeOf(AnonymousAccess{}).Name()
	AnonymousAccessGroupKind        = schema.GroupKind{Group: Group, Kind: AnonymousAccessKind}.String()
	AnonymousAccessKindAPIVersion   = AnonymousAccessKind + "." + SchemeGroupVersion.String()
	AnonymousAccessGroupVersionKind = SchemeGroupVersion.WithKind(AnonymousAccessKind)
)

// SAML type metadata.
var (
	SAMLKind             = reflect.TypeOf(SAML{}).Name()
	SAMLGroupKind        = schema.GroupKind{Group: Group, Kind: SAMLKind}.String()
	SAMLKindAPIVersion   = SAMLKind + "." + SchemeGroupVersion.String()
	SAMLGroupVersionKind = SchemeGroupVersion.WithKind(SAMLKind)
)

// LDAP type metadata.
var (
	LDAPKind             = reflect.TypeOf(LDAP{}).Name()
	LDAPGroupKind        = schema.GroupKind{Group: Group, Kind: LDAPKind}.String()
	LDAPKindAPIVersion   = LDAPKind + "." + SchemeGroupVersion.String()
	LDAPGroupVersionKind = SchemeGroupVersion.WithKind(LDAPKind)
)

// UserTokenConfiguration type metadata.
var (
	UserTokenConfigurationKind             = reflect.TypeOf(UserTokenConfiguration{}).Name()
	UserTokenConfigurationGroupKind        = schema.GroupKind{Group: Group, Kind: UserTokenConfigurationKind}.String()
	UserTokenConfigurationKindAPIVersion   = UserTokenConfigurationKind + "." + SchemeGroupVersion.String()
	UserTokenConfigurationGroupVersionKind = SchemeGroupVersion.WithKind(UserTokenConfigurationKind)
)

// License type metadata.
var (
	LicenseKind             = reflect.TypeOf(License{}).Name()
	LicenseGroupKind        = schema.GroupKind{Group: Group, Kind: LicenseKind}.String()
	LicenseKindAPIVersion   = LicenseKind + "." + SchemeGroupVersion.String()
	LicenseGroupVersionKind = SchemeGroupVersion.WithKind(LicenseKind)
)
