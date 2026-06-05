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
	BlobStoreKind             = reflect.TypeFor[BlobStore]().Name()
	BlobStoreGroupKind        = schema.GroupKind{Group: Group, Kind: BlobStoreKind}.String()
	BlobStoreKindAPIVersion   = BlobStoreKind + "." + SchemeGroupVersion.String()
	BlobStoreGroupVersionKind = SchemeGroupVersion.WithKind(BlobStoreKind)
)

// Repository type metadata.
var (
	RepositoryKind             = reflect.TypeFor[Repository]().Name()
	RepositoryGroupKind        = schema.GroupKind{Group: Group, Kind: RepositoryKind}.String()
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)

// User type metadata.
var (
	UserKind             = reflect.TypeFor[User]().Name()
	UserGroupKind        = schema.GroupKind{Group: Group, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

// Privilege type metadata.
var (
	PrivilegeKind             = reflect.TypeFor[Privilege]().Name()
	PrivilegeGroupKind        = schema.GroupKind{Group: Group, Kind: PrivilegeKind}.String()
	PrivilegeKindAPIVersion   = PrivilegeKind + "." + SchemeGroupVersion.String()
	PrivilegeGroupVersionKind = SchemeGroupVersion.WithKind(PrivilegeKind)
)

// SAML type metadata.
var (
	SAMLKind             = reflect.TypeFor[SAML]().Name()
	SAMLGroupKind        = schema.GroupKind{Group: Group, Kind: SAMLKind}.String()
	SAMLKindAPIVersion   = SAMLKind + "." + SchemeGroupVersion.String()
	SAMLGroupVersionKind = SchemeGroupVersion.WithKind(SAMLKind)
)

// LDAP type metadata.
var (
	LDAPKind             = reflect.TypeFor[LDAP]().Name()
	LDAPGroupKind        = schema.GroupKind{Group: Group, Kind: LDAPKind}.String()
	LDAPKindAPIVersion   = LDAPKind + "." + SchemeGroupVersion.String()
	LDAPGroupVersionKind = SchemeGroupVersion.WithKind(LDAPKind)
)

// SecuritySSLTruststore type metadata.
var (
	SecuritySSLTruststoreKind             = reflect.TypeFor[SecuritySSLTruststore]().Name()
	SecuritySSLTruststoreGroupKind        = schema.GroupKind{Group: Group, Kind: SecuritySSLTruststoreKind}.String()
	SecuritySSLTruststoreKindAPIVersion   = SecuritySSLTruststoreKind + "." + SchemeGroupVersion.String()
	SecuritySSLTruststoreGroupVersionKind = SchemeGroupVersion.WithKind(SecuritySSLTruststoreKind)
)

// ProviderConfig type metadata.
var (
	ProviderConfigKind             = reflect.TypeFor[ProviderConfig]().Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigKind)
)

// ProviderConfigUsage type metadata.
var (
	ProviderConfigUsageKind             = reflect.TypeFor[ProviderConfigUsage]().Name()
	ProviderConfigUsageGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageKind)

	ProviderConfigUsageListKind             = reflect.TypeFor[ProviderConfigUsageList]().Name()
	ProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageListKind)
)

// ClusterProviderConfig type metadata.
var (
	ClusterProviderConfigKind             = reflect.TypeFor[ClusterProviderConfig]().Name()
	ClusterProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterProviderConfigKind}.String()
	ClusterProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigKind)
)

// ClusterProviderConfigUsage type metadata.
var (
	ClusterProviderConfigUsageKind             = reflect.TypeFor[ClusterProviderConfigUsage]().Name()
	ClusterProviderConfigUsageGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigUsageKind)

	ClusterProviderConfigUsageListKind             = reflect.TypeFor[ClusterProviderConfigUsageList]().Name()
	ClusterProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(ClusterProviderConfigUsageListKind)
)
