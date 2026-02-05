package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// +optional
	Online *bool `json:"online,omitempty"`

	// Storage configuration.
	// +optional
	Storage *RepositoryStorage `json:"storage,omitempty"`

	// Cleanup configuration.
	// +optional
	Cleanup *RepositoryCleanup `json:"cleanup,omitempty"`

	// Maven specific configuration.
	// +optional
	Maven *MavenConfig `json:"maven,omitempty"`

	// Docker specific configuration.
	// +optional
	Docker *DockerConfig `json:"docker,omitempty"`

	// DockerProxy specific configuration for Docker proxy repositories.
	// +optional
	DockerProxy *DockerProxyConfig `json:"dockerProxy,omitempty"`

	// Proxy configuration for proxy repositories.
	// +optional
	Proxy *ProxyConfig `json:"proxy,omitempty"`

	// NegativeCache configuration for proxy repositories.
	// +optional
	NegativeCache *NegativeCacheConfig `json:"negativeCache,omitempty"`

	// HTTPClient configuration for proxy repositories.
	// +optional
	HTTPClient *HTTPClientConfig `json:"httpClient,omitempty"`

	// Group configuration for group repositories.
	// +optional
	Group *GroupConfig `json:"group,omitempty"`

	// Npm specific configuration.
	// +optional
	Npm *NpmConfig `json:"npm,omitempty"`

	// Apt specific configuration.
	// +optional
	Apt *AptConfig `json:"apt,omitempty"`

	// AptSigning configuration for APT hosted repositories.
	// +optional
	AptSigning *AptSigningConfig `json:"aptSigning,omitempty"`

	// Yum specific configuration.
	// +optional
	Yum *YumConfig `json:"yum,omitempty"`

	// YumSigning configuration for Yum repositories.
	// +optional
	YumSigning *YumSigningConfig `json:"yumSigning,omitempty"`

	// NugetProxy specific configuration for NuGet proxy repositories.
	// +optional
	NugetProxy *NugetProxyConfig `json:"nugetProxy,omitempty"`

	// Bower specific configuration.
	// +optional
	Bower *BowerConfig `json:"bower,omitempty"`

	// Cargo specific configuration.
	// +optional
	Cargo *CargoConfig `json:"cargo,omitempty"`

	// RoutingRule is the name of the routing rule for proxy repositories.
	// +optional
	RoutingRule *string `json:"routingRule,omitempty"`
}

// RepositoryStorage defines storage configuration for a repository.
type RepositoryStorage struct {
	// BlobStoreName is the name of the blob store to use.
	// +kubebuilder:validation:Required
	BlobStoreName string `json:"blobStoreName"`

	// StrictContentTypeValidation enables strict content type validation.
	// +kubebuilder:default=true
	// +optional
	StrictContentTypeValidation *bool `json:"strictContentTypeValidation,omitempty"`

	// WritePolicy for hosted repositories.
	// +kubebuilder:validation:Enum=ALLOW;ALLOW_ONCE;DENY
	// +optional
	WritePolicy *string `json:"writePolicy,omitempty"`
}

// RepositoryCleanup defines cleanup configuration for a repository.
type RepositoryCleanup struct {
	// PolicyNames is a list of cleanup policy names.
	// +optional
	PolicyNames []string `json:"policyNames,omitempty"`
}

// MavenConfig defines Maven specific configuration.
type MavenConfig struct {
	// VersionPolicy for Maven repositories.
	// +kubebuilder:validation:Enum=RELEASE;SNAPSHOT;MIXED
	// +optional
	VersionPolicy *string `json:"versionPolicy,omitempty"`

	// LayoutPolicy for Maven repositories.
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE
	// +optional
	LayoutPolicy *string `json:"layoutPolicy,omitempty"`

	// ContentDisposition for Maven repositories.
	// +kubebuilder:validation:Enum=INLINE;ATTACHMENT
	// +optional
	ContentDisposition *string `json:"contentDisposition,omitempty"`
}

// DockerConfig defines Docker specific configuration.
type DockerConfig struct {
	// V1Enabled allows clients to use V1 Docker registry API.
	// +optional
	V1Enabled *bool `json:"v1Enabled,omitempty"`

	// ForceBasicAuth forces basic authentication.
	// +optional
	ForceBasicAuth *bool `json:"forceBasicAuth,omitempty"`

	// HTTPPort for Docker registry.
	// +optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// HTTPSPort for Docker registry.
	// +optional
	HTTPSPort *int32 `json:"httpsPort,omitempty"`

	// Subdomain for Docker registry.
	// +optional
	Subdomain *string `json:"subdomain,omitempty"`
}

// ProxyConfig defines proxy configuration for proxy repositories.
type ProxyConfig struct {
	// RemoteURL is the URL of the remote repository.
	// +kubebuilder:validation:Required
	RemoteURL string `json:"remoteUrl"`

	// ContentMaxAge is the maximum age of cached content in minutes.
	// +kubebuilder:default=1440
	// +optional
	ContentMaxAge *int32 `json:"contentMaxAge,omitempty"`

	// MetadataMaxAge is the maximum age of cached metadata in minutes.
	// +kubebuilder:default=1440
	// +optional
	MetadataMaxAge *int32 `json:"metadataMaxAge,omitempty"`
}

// NegativeCacheConfig defines negative cache configuration.
type NegativeCacheConfig struct {
	// Enabled determines if negative caching is enabled.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// TimeToLive is the time to live for negative cache entries in minutes.
	// +kubebuilder:default=1440
	// +optional
	TimeToLive *int32 `json:"timeToLive,omitempty"`
}

// HTTPClientConfig defines HTTP client configuration for proxy repositories.
type HTTPClientConfig struct {
	// Blocked determines if the repository is blocked.
	// +optional
	Blocked *bool `json:"blocked,omitempty"`

	// AutoBlock determines if the repository is auto-blocked.
	// +kubebuilder:default=true
	// +optional
	AutoBlock *bool `json:"autoBlock,omitempty"`

	// Connection configuration.
	// +optional
	Connection *HTTPClientConnection `json:"connection,omitempty"`

	// Authentication configuration.
	// +optional
	Authentication *HTTPClientAuthentication `json:"authentication,omitempty"`
}

// HTTPClientConnection defines HTTP client connection configuration.
type HTTPClientConnection struct {
	// Retries is the number of retries.
	// +optional
	Retries *int32 `json:"retries,omitempty"`

	// UserAgentSuffix is a custom user agent suffix.
	// +optional
	UserAgentSuffix *string `json:"userAgentSuffix,omitempty"`

	// Timeout is the connection timeout in seconds.
	// +optional
	Timeout *int32 `json:"timeout,omitempty"`

	// EnableCircularRedirects enables circular redirects.
	// +optional
	EnableCircularRedirects *bool `json:"enableCircularRedirects,omitempty"`

	// EnableCookies enables cookies.
	// +optional
	EnableCookies *bool `json:"enableCookies,omitempty"`

	// UseTrustStore uses the trust store.
	// +optional
	UseTrustStore *bool `json:"useTrustStore,omitempty"`
}

// HTTPClientAuthentication defines HTTP client authentication configuration.
type HTTPClientAuthentication struct {
	// Type of authentication.
	// +kubebuilder:validation:Enum=username;ntlm
	// +optional
	Type *string `json:"type,omitempty"`

	// Username for authentication.
	// +optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef is a reference to a secret containing the password.
	// +optional
	PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// NTLMHost is the NTLM host.
	// +optional
	NTLMHost *string `json:"ntlmHost,omitempty"`

	// NTLMDomain is the NTLM domain.
	// +optional
	NTLMDomain *string `json:"ntlmDomain,omitempty"`
}

// GroupConfig defines group configuration for group repositories.
type GroupConfig struct {
	// MemberNames is a list of member repository names.
	// +optional
	MemberNames []string `json:"memberNames,omitempty"`

	// WritableMember is the writable member of the group (for Docker and Maven).
	// +optional
	WritableMember *string `json:"writableMember,omitempty"`
}

// NpmConfig defines npm specific configuration.
type NpmConfig struct {
	// RemoveNonCataloged removes non-cataloged versions.
	// +optional
	RemoveNonCataloged *bool `json:"removeNonCataloged,omitempty"`

	// RemoveQuarantined removes quarantined components.
	// +optional
	RemoveQuarantined *bool `json:"removeQuarantined,omitempty"`
}

// AptConfig defines APT specific configuration.
type AptConfig struct {
	// Distribution for APT repository.
	// +optional
	Distribution *string `json:"distribution,omitempty"`

	// Flat determines if the repository uses flat format.
	// +optional
	Flat *bool `json:"flat,omitempty"`
}

// YumConfig defines Yum specific configuration.
type YumConfig struct {
	// RepodataDepth is the repodata depth.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	// +optional
	RepodataDepth *int32 `json:"repodataDepth,omitempty"`

	// DeployPolicy for Yum repositories.
	// +kubebuilder:validation:Enum=STRICT;PERMISSIVE
	// +optional
	DeployPolicy *string `json:"deployPolicy,omitempty"`
}

// YumSigningConfig defines Yum signing configuration.
type YumSigningConfig struct {
	// Keypair is the PGP signing key pair (armored private key).
	// +optional
	Keypair *string `json:"keypair,omitempty"`

	// Passphrase to access PGP signing key.
	// +optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// AptSigningConfig defines APT signing configuration for hosted repositories.
type AptSigningConfig struct {
	// Keypair is the PGP signing key pair (armored private key).
	// +kubebuilder:validation:Required
	Keypair string `json:"keypair"`

	// Passphrase to access PGP signing key.
	// +optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// DockerProxyConfig defines Docker proxy specific configuration.
type DockerProxyConfig struct {
	// IndexType is the type of Docker index (HUB, REGISTRY, CUSTOM).
	// +kubebuilder:validation:Enum=HUB;REGISTRY;CUSTOM
	// +optional
	IndexType *string `json:"indexType,omitempty"`

	// IndexURL is the URL of the Docker index to use (for CUSTOM type).
	// +optional
	IndexURL *string `json:"indexUrl,omitempty"`

	// CacheForeignLayers allows downloading and caching foreign layers.
	// +optional
	CacheForeignLayers *bool `json:"cacheForeignLayers,omitempty"`

	// ForeignLayerUrlWhitelist is a list of regex patterns for allowed foreign layer URLs.
	// +optional
	ForeignLayerUrlWhitelist []string `json:"foreignLayerUrlWhitelist,omitempty"`
}

// NugetProxyConfig defines NuGet proxy specific configuration.
type NugetProxyConfig struct {
	// QueryCacheItemMaxAge is how long to cache query results in seconds.
	// +kubebuilder:default=3600
	// +optional
	QueryCacheItemMaxAge *int32 `json:"queryCacheItemMaxAge,omitempty"`

	// NugetVersion is the NuGet protocol version (V2 or V3).
	// +kubebuilder:validation:Enum=V2;V3
	// +kubebuilder:default=V3
	// +optional
	NugetVersion *string `json:"nugetVersion,omitempty"`
}

// BowerConfig defines Bower specific configuration.
type BowerConfig struct {
	// RewritePackageUrls forces Bower to retrieve packages through this proxy.
	// +optional
	RewritePackageUrls *bool `json:"rewritePackageUrls,omitempty"`
}

// CargoConfig defines Cargo specific configuration.
type CargoConfig struct {
	// RequireAuthentication indicates if this repository requires authentication.
	// +optional
	RequireAuthentication *bool `json:"requireAuthentication,omitempty"`
}

// RepositoryObservation represents the observed state of a Repository.
type RepositoryObservation struct {
	// URL is the URL of the repository.
	URL *string `json:"url,omitempty"`
}

// RepositorySpec defines the desired state of Repository.
type RepositorySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepositoryParameters `json:"forProvider"`
}

// RepositoryStatus defines the observed state of Repository.
type RepositoryStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepositoryObservation `json:"atProvider,omitempty"`
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
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
