package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RepositoryParameters defines the desired state of a Repository.
type RepositoryParameters struct {
	// Name of the repository.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Format of the repository (maven2, npm, docker, raw, etc.).
	// +kubebuilder:validation:Enum=maven2;npm;docker;raw;nuget;pypi;rubygems;yum;apt;helm;go;r;conan;conda;cocoapods;bower;gitlfs;p2;cargo
	// +kubebuilder:validation:Required
	Format string `json:"format"`

	// Type of the repository (hosted, proxy, group).
	// +kubebuilder:validation:Enum=hosted;proxy;group
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Online determines if the repository is online.
	// +kubebuilder:default=true
	// +kubebuilder:validation:Optional
	Online *bool `json:"online,omitempty"`

	// Storage configuration.
	// +kubebuilder:validation:Optional
	Storage *RepositoryStorage `json:"storage,omitempty"`

	// Cleanup configuration.
	// +kubebuilder:validation:Optional
	Cleanup *RepositoryCleanup `json:"cleanup,omitempty"`

	// Maven specific configuration.
	// +kubebuilder:validation:Optional
	Maven *MavenConfig `json:"maven,omitempty"`

	// Docker specific configuration.
	// +kubebuilder:validation:Optional
	Docker *DockerConfig `json:"docker,omitempty"`

	// DockerProxy specific configuration for Docker proxy repositories.
	// +kubebuilder:validation:Optional
	DockerProxy *DockerProxyConfig `json:"dockerProxy,omitempty"`

	// Proxy configuration for proxy repositories.
	// +kubebuilder:validation:Optional
	Proxy *ProxyConfig `json:"proxy,omitempty"`

	// NegativeCache configuration for proxy repositories.
	// +kubebuilder:validation:Optional
	NegativeCache *NegativeCacheConfig `json:"negativeCache,omitempty"`

	// HTTPClient configuration for proxy repositories.
	// +kubebuilder:validation:Optional
	HTTPClient *HTTPClientConfig `json:"httpClient,omitempty"`

	// Group configuration for group repositories.
	// +kubebuilder:validation:Optional
	Group *GroupConfig `json:"group,omitempty"`

	// Npm specific configuration.
	// +kubebuilder:validation:Optional
	Npm *NpmConfig `json:"npm,omitempty"`

	// Apt specific configuration.
	// +kubebuilder:validation:Optional
	Apt *AptConfig `json:"apt,omitempty"`

	// AptSigning configuration for APT hosted repositories.
	// +kubebuilder:validation:Optional
	AptSigning *AptSigningConfig `json:"aptSigning,omitempty"`

	// Yum specific configuration.
	// +kubebuilder:validation:Optional
	Yum *YumConfig `json:"yum,omitempty"`

	// YumSigning configuration for Yum repositories.
	// +kubebuilder:validation:Optional
	YumSigning *YumSigningConfig `json:"yumSigning,omitempty"`

	// NugetProxy specific configuration for NuGet proxy repositories.
	// +kubebuilder:validation:Optional
	NugetProxy *NugetProxyConfig `json:"nugetProxy,omitempty"`

	// Bower specific configuration.
	// +kubebuilder:validation:Optional
	Bower *BowerConfig `json:"bower,omitempty"`

	// Cargo specific configuration.
	// +kubebuilder:validation:Optional
	Cargo *CargoConfig `json:"cargo,omitempty"`

	// RoutingRule is the name of the routing rule for proxy repositories.
	// +kubebuilder:validation:Optional
	RoutingRule *string `json:"routingRule,omitempty"`
}

// RepositoryStorage defines storage configuration for a repository.
type RepositoryStorage struct {
	// BlobStoreName is the name of the blob store to use.
	// +kubebuilder:validation:Required
	BlobStoreName string `json:"blobStoreName"`

	// StrictContentTypeValidation enables strict content type validation.
	// +kubebuilder:default=true
	// +kubebuilder:validation:Optional
	StrictContentTypeValidation *bool `json:"strictContentTypeValidation,omitempty"`

	// WritePolicy for hosted repositories.
	// +kubebuilder:validation:Enum=ALLOW;ALLOW_ONCE;DENY
	// +kubebuilder:validation:Optional
	WritePolicy *string `json:"writePolicy,omitempty"`
}

// RepositoryCleanup defines cleanup configuration for a repository.
type RepositoryCleanup struct {
	// PolicyNames is a list of cleanup policy names.
	// +kubebuilder:validation:Optional
	PolicyNames []string `json:"policyNames,omitempty"`
}

// MavenConfig defines Maven specific configuration.
type MavenConfig struct {
	// VersionPolicy for Maven repositories.
	// +kubebuilder:validation:Enum=RELEASE;SNAPSHOT;MIXED
	// +kubebuilder:validation:Optional
	VersionPolicy *string `json:"versionPolicy,omitempty"`

	// LayoutPolicy for Maven repositories.
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE
	// +kubebuilder:validation:Optional
	LayoutPolicy *string `json:"layoutPolicy,omitempty"`

	// ContentDisposition for Maven repositories.
	// +kubebuilder:validation:Enum=INLINE;ATTACHMENT
	// +kubebuilder:validation:Optional
	ContentDisposition *string `json:"contentDisposition,omitempty"`
}

// DockerConfig defines Docker specific configuration.
type DockerConfig struct {
	// V1Enabled allows clients to use V1 Docker registry API.
	// +kubebuilder:validation:Optional
	V1Enabled *bool `json:"v1Enabled,omitempty"`

	// ForceBasicAuth forces basic authentication.
	// +kubebuilder:validation:Optional
	ForceBasicAuth *bool `json:"forceBasicAuth,omitempty"`

	// HTTPPort for Docker registry.
	// +kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// HTTPSPort for Docker registry.
	// +kubebuilder:validation:Optional
	HTTPSPort *int32 `json:"httpsPort,omitempty"`

	// Subdomain for Docker registry.
	// +kubebuilder:validation:Optional
	Subdomain *string `json:"subdomain,omitempty"`
}

// ProxyConfig defines proxy configuration for proxy repositories.
type ProxyConfig struct {
	// RemoteURL is the URL of the remote repository.
	// +kubebuilder:validation:Required
	RemoteURL string `json:"remoteUrl"`

	// ContentMaxAge is the maximum age of cached content in minutes.
	// +kubebuilder:default=1440
	// +kubebuilder:validation:Optional
	ContentMaxAge *int32 `json:"contentMaxAge,omitempty"`

	// MetadataMaxAge is the maximum age of cached metadata in minutes.
	// +kubebuilder:default=1440
	// +kubebuilder:validation:Optional
	MetadataMaxAge *int32 `json:"metadataMaxAge,omitempty"`
}

// NegativeCacheConfig defines negative cache configuration.
type NegativeCacheConfig struct {
	// Enabled determines if negative caching is enabled.
	// +kubebuilder:default=true
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty"`

	// TimeToLive is the time to live for negative cache entries in minutes.
	// +kubebuilder:default=1440
	// +kubebuilder:validation:Optional
	TimeToLive *int32 `json:"timeToLive,omitempty"`
}

// HTTPClientConfig defines HTTP client configuration for proxy repositories.
type HTTPClientConfig struct {
	// Blocked determines if the repository is blocked.
	// +kubebuilder:validation:Optional
	Blocked *bool `json:"blocked,omitempty"`

	// AutoBlock determines if the repository is auto-blocked.
	// +kubebuilder:default=true
	// +kubebuilder:validation:Optional
	AutoBlock *bool `json:"autoBlock,omitempty"`

	// Connection configuration.
	// +kubebuilder:validation:Optional
	Connection *HTTPClientConnection `json:"connection,omitempty"`

	// Authentication configuration.
	// +kubebuilder:validation:Optional
	Authentication *HTTPClientAuthentication `json:"authentication,omitempty"`
}

// HTTPClientConnection defines HTTP client connection configuration.
type HTTPClientConnection struct {
	// Retries is the number of retries.
	// +kubebuilder:validation:Optional
	Retries *int32 `json:"retries,omitempty"`

	// UserAgentSuffix is a custom user agent suffix.
	// +kubebuilder:validation:Optional
	UserAgentSuffix *string `json:"userAgentSuffix,omitempty"`

	// Timeout is the connection timeout in seconds.
	// +kubebuilder:validation:Optional
	Timeout *int32 `json:"timeout,omitempty"`

	// EnableCircularRedirects enables circular redirects.
	// +kubebuilder:validation:Optional
	EnableCircularRedirects *bool `json:"enableCircularRedirects,omitempty"`

	// EnableCookies enables cookies.
	// +kubebuilder:validation:Optional
	EnableCookies *bool `json:"enableCookies,omitempty"`

	// UseTrustStore uses the trust store.
	// +kubebuilder:validation:Optional
	UseTrustStore *bool `json:"useTrustStore,omitempty"`
}

// HTTPClientAuthentication defines HTTP client authentication configuration.
type HTTPClientAuthentication struct {
	// Type of authentication.
	// +kubebuilder:validation:Enum=username;ntlm
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`

	// Username for authentication.
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef is a reference to a secret containing the password.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// NTLMHost is the NTLM host.
	// +kubebuilder:validation:Optional
	NTLMHost *string `json:"ntlmHost,omitempty"`

	// NTLMDomain is the NTLM domain.
	// +kubebuilder:validation:Optional
	NTLMDomain *string `json:"ntlmDomain,omitempty"`
}

// GroupConfig defines group configuration for group repositories.
type GroupConfig struct {
	// MemberNames is a list of member repository names.
	// +kubebuilder:validation:Optional
	MemberNames []string `json:"memberNames,omitempty"`

	// WritableMember is the writable member of the group (for Docker and Maven).
	// +kubebuilder:validation:Optional
	WritableMember *string `json:"writableMember,omitempty"`
}

// NpmConfig defines npm specific configuration.
type NpmConfig struct {
	// RemoveNonCataloged removes non-cataloged versions.
	// +kubebuilder:validation:Optional
	RemoveNonCataloged *bool `json:"removeNonCataloged,omitempty"`

	// RemoveQuarantined removes quarantined components.
	// +kubebuilder:validation:Optional
	RemoveQuarantined *bool `json:"removeQuarantined,omitempty"`
}

// AptConfig defines APT specific configuration.
type AptConfig struct {
	// Distribution for APT repository.
	// +kubebuilder:validation:Optional
	Distribution *string `json:"distribution,omitempty"`

	// Flat determines if the repository uses flat format.
	// +kubebuilder:validation:Optional
	Flat *bool `json:"flat,omitempty"`
}

// YumConfig defines Yum specific configuration.
type YumConfig struct {
	// RepodataDepth is the repodata depth.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	// +kubebuilder:validation:Optional
	RepodataDepth *int32 `json:"repodataDepth,omitempty"`

	// DeployPolicy for Yum repositories.
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE
	// +kubebuilder:validation:Optional
	DeployPolicy *string `json:"deployPolicy,omitempty"`
}

// YumSigningConfig defines Yum signing configuration.
type YumSigningConfig struct {
	// Keypair is the PGP signing key pair (armored private key).
	// +kubebuilder:validation:Optional
	Keypair *string `json:"keypair,omitempty"`

	// Passphrase to access PGP signing key.
	// +kubebuilder:validation:Optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// AptSigningConfig defines APT signing configuration for hosted repositories.
type AptSigningConfig struct {
	// Keypair is the PGP signing key pair (armored private key).
	// +kubebuilder:validation:Required
	Keypair string `json:"keypair"`

	// Passphrase to access PGP signing key.
	// +kubebuilder:validation:Optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// DockerProxyConfig defines Docker proxy specific configuration.
type DockerProxyConfig struct {
	// IndexType is the type of Docker index (HUB, REGISTRY, CUSTOM).
	// +kubebuilder:validation:Enum=HUB;REGISTRY;CUSTOM
	// +kubebuilder:validation:Optional
	IndexType *string `json:"indexType,omitempty"`

	// IndexURL is the URL of the Docker index to use (for CUSTOM type).
	// +kubebuilder:validation:Optional
	IndexURL *string `json:"indexUrl,omitempty"`

	// CacheForeignLayers allows downloading and caching foreign layers.
	// +kubebuilder:validation:Optional
	CacheForeignLayers *bool `json:"cacheForeignLayers,omitempty"`

	// ForeignLayerUrlWhitelist is a list of regex patterns for allowed foreign layer URLs.
	// +kubebuilder:validation:Optional
	ForeignLayerUrlWhitelist []string `json:"foreignLayerUrlWhitelist,omitempty"`
}

// NugetProxyConfig defines NuGet proxy specific configuration.
type NugetProxyConfig struct {
	// QueryCacheItemMaxAge is how long to cache query results in seconds.
	// +kubebuilder:default=3600
	// +kubebuilder:validation:Optional
	QueryCacheItemMaxAge *int32 `json:"queryCacheItemMaxAge,omitempty"`

	// NugetVersion is the NuGet protocol version (V2 or V3).
	// +kubebuilder:validation:Enum=V2;V3
	// +kubebuilder:default=V3
	// +kubebuilder:validation:Optional
	NugetVersion *string `json:"nugetVersion,omitempty"`
}

// BowerConfig defines Bower specific configuration.
type BowerConfig struct {
	// RewritePackageUrls forces Bower to retrieve packages through this proxy.
	// +kubebuilder:validation:Optional
	RewritePackageUrls *bool `json:"rewritePackageUrls,omitempty"`
}

// CargoConfig defines Cargo specific configuration.
type CargoConfig struct {
	// RequireAuthentication indicates if this repository requires authentication.
	// +kubebuilder:validation:Optional
	RequireAuthentication *bool `json:"requireAuthentication,omitempty"`
}

// RepositoryObservation represents the observed state of a Repository.
type RepositoryObservation struct {
	// URL is the URL of the repository.
	URL *string `json:"url,omitempty"`
}

// RepositorySpec defines the desired state of Repository.
type RepositorySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`

	ForProvider RepositoryParameters `json:"forProvider"`
}

// RepositoryStatus defines the observed state of Repository.
type RepositoryStatus struct {
	xpv2.ManagedResourceStatus `json:",inline"`

	AtProvider RepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,nexus}

// Repository is the Schema for the repositories API.
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository.
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Repository `json:"items"`
}

// Repository type metadata.
var (
	RepositoryKind             = reflect.TypeFor[Repository]().Name()
	RepositoryGroupKind        = schema.GroupKind{Group: APIGroup, Kind: RepositoryKind}.String()
	RepositoryKindAPIVersion   = RepositoryKind + "." + SchemeGroupVersion.String()
	RepositoryGroupVersionKind = SchemeGroupVersion.WithKind(RepositoryKind)
)

// init registers this type with the SchemeBuilder.
func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
