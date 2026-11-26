package v1alpha1

import (
	"reflect"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RepositoryParameters defines the desired state of a Repository.
// +kubebuilder:validation:XValidation:rule="self.format != 'terraform' || self.type == 'group' || !has(self.terraform)",message="terraform is only supported on terraform group repositories: Nexus ignores requireAuthentication on the other types."
// +kubebuilder:validation:XValidation:rule="!has(self.httpClient) || !has(self.httpClient.authentication) || !has(self.httpClient.authentication.preemptive) || !self.httpClient.authentication.preemptive || (self.type == 'proxy' && self.format in ['maven2', 'pypi', 'terraform'])",message="httpClient.authentication.preemptive is only supported on maven2, pypi and terraform proxy repositories: Nexus ignores it on the other formats."
// +kubebuilder:validation:XValidation:rule="!has(self.ociProxy) || !has(self.ociProxy.ecrAuth)",message="ociProxy.ecrAuth is not supported: Nexus only offers AWS ECR authentication on docker proxy repositories."
type RepositoryParameters struct {
	// Name of the repository.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable."
	Name string `json:"name"`

	// Format of the repository (maven2, npm, docker, raw, etc.).
	// +kubebuilder:validation:Enum=maven2;npm;docker;raw;nuget;pypi;rubygems;yum;apt;helm;go;r;conan;conda;cocoapods;gitlfs;p2;cargo;huggingface;alpine;ansiblegalaxy;composer;oci;pub;swift;terraform
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Format is immutable."
	Format string `json:"format"`

	// Type of the repository (hosted, proxy, group).
	// +kubebuilder:validation:Enum=hosted;proxy;group
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable."
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

	// Component configuration for hosted repositories.
	// +kubebuilder:validation:Optional
	Component *ComponentConfig `json:"component,omitempty"`

	// Firewall configuration for proxy repositories. It requires Sonatype
	// Repository Firewall: without an IQ Server connection, Nexus only
	// accepts the DISABLED mode. Nexus keeps the mode an update leaves out,
	// so removing this block does not reset it: set DISABLED instead.
	// +kubebuilder:validation:Optional
	Firewall *FirewallConfig `json:"firewall,omitempty"`

	// Replication configures the pre-emptive pull of a proxy repository. It
	// requires Nexus Repository Pro: Community rejects enabling it.
	// +kubebuilder:validation:Optional
	Replication *ReplicationConfig `json:"replication,omitempty"`

	// Maven specific configuration.
	// +kubebuilder:validation:Optional
	Maven *MavenConfig `json:"maven,omitempty"`

	// Docker specific configuration.
	// +kubebuilder:validation:Optional
	Docker *ContainerConfig `json:"docker,omitempty"`

	// DockerProxy specific configuration for Docker proxy repositories.
	// +kubebuilder:validation:Optional
	DockerProxy *ContainerProxyConfig `json:"dockerProxy,omitempty"`

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
	//
	// Deprecated: Nexus moved these settings to the Sonatype Repository
	// Firewall configuration and no longer accepts them on a repository, so
	// this field is ignored.
	//
	// +kubebuilder:validation:Optional
	Npm *NpmConfig `json:"npm,omitempty"`

	// Apt specific configuration.
	// +kubebuilder:validation:Optional
	Apt *AptConfig `json:"apt,omitempty"`

	// AptSigning configuration for APT repositories. It is required for
	// hosted repositories and verifies the upstream index of proxy
	// repositories.
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

	// Raw specific configuration for raw repositories.
	// +kubebuilder:validation:Optional
	Raw *RawConfig `json:"raw,omitempty"`

	// ConanProxy specific configuration for Conan proxy repositories.
	// +kubebuilder:validation:Optional
	ConanProxy *ConanProxyConfig `json:"conanProxy,omitempty"`

	// Pypi specific configuration for PyPI proxy repositories.
	// +kubebuilder:validation:Optional
	Pypi *PypiProxyConfig `json:"pypi,omitempty"`

	// Xet configures the Xet storage protocol of a Hugging Face proxy
	// repository.
	// +kubebuilder:validation:Optional
	Xet *XetConfig `json:"xet,omitempty"`

	// Bower specific configuration.
	//
	// Deprecated: Nexus no longer supports the Bower format, so this field is
	// ignored.
	//
	// +kubebuilder:validation:Optional
	Bower *BowerConfig `json:"bower,omitempty"`

	// Cargo specific configuration.
	// +kubebuilder:validation:Optional
	Cargo *RequireAuthenticationConfig `json:"cargo,omitempty"`

	// Oci specific configuration for OCI repositories. It takes the same
	// settings as docker.
	// +kubebuilder:validation:Optional
	Oci *ContainerConfig `json:"oci,omitempty"`

	// OciProxy specific configuration for OCI proxy repositories. It takes
	// the same settings as dockerProxy.
	// +kubebuilder:validation:Optional
	OciProxy *ContainerProxyConfig `json:"ociProxy,omitempty"`

	// Cosign signature enforcement for OCI repositories. Enforcement is NONE
	// when it is left out.
	// +kubebuilder:validation:Optional
	Cosign *CosignConfig `json:"cosign,omitempty"`

	// AlpineSigning is the RSA key Nexus signs the APK index of an alpine
	// repository with. It is required for proxy and group repositories,
	// unless the managementPolicies only observe the repository.
	// +kubebuilder:validation:Optional
	AlpineSigning *AlpineSigningConfig `json:"alpineSigning,omitempty"`

	// Swift specific configuration for Swift proxy repositories.
	// +kubebuilder:validation:Optional
	Swift *RequireAuthenticationConfig `json:"swift,omitempty"`

	// Terraform specific configuration for Terraform group repositories.
	// +kubebuilder:validation:Optional
	Terraform *RequireAuthenticationConfig `json:"terraform,omitempty"`

	// TerraformSigning is the PGP key Nexus signs the providers of a
	// Terraform hosted repository with. It is required for hosted
	// repositories, unless the managementPolicies only observe the
	// repository.
	// +kubebuilder:validation:Optional
	TerraformSigning *TerraformSigningConfig `json:"terraformSigning,omitempty"`

	// RoutingRule is the name of the routing rule for proxy repositories.
	// +kubebuilder:validation:Optional
	RoutingRule *string `json:"routingRule,omitempty"`

	// RoutingRuleRef resolves routingRule from a RoutingRule managed resource
	// in the Repository's namespace.
	// +kubebuilder:validation:Optional
	RoutingRuleRef *xpv2.Reference `json:"routingRuleRef,omitempty"`

	// RoutingRuleSelector selects a RoutingRule managed resource in the
	// Repository's namespace to resolve routingRule.
	// +kubebuilder:validation:Optional
	RoutingRuleSelector *xpv2.Selector `json:"routingRuleSelector,omitempty"`
}

// RepositoryStorage defines storage configuration for a repository.
// +kubebuilder:validation:XValidation:rule="has(self.blobStoreName) || has(self.blobStoreNameRef) || has(self.blobStoreNameSelector)",message="one of blobStoreName, blobStoreNameRef or blobStoreNameSelector is required."
// +kubebuilder:validation:XValidation:rule="!has(self.latestPolicy) || !self.latestPolicy || (has(self.writePolicy) && self.writePolicy == 'ALLOW_ONCE')",message="latestPolicy requires the ALLOW_ONCE writePolicy: Nexus drops it otherwise."
type RepositoryStorage struct {
	// BlobStoreName is the name of the blob store to use. It is required,
	// unless it is resolved through blobStoreNameRef or blobStoreNameSelector.
	// +kubebuilder:validation:Optional
	BlobStoreName string `json:"blobStoreName,omitempty"`

	// BlobStoreNameRef resolves blobStoreName from a BlobStore managed
	// resource in the Repository's namespace.
	// +kubebuilder:validation:Optional
	BlobStoreNameRef *xpv2.Reference `json:"blobStoreNameRef,omitempty"`

	// BlobStoreNameSelector selects a BlobStore managed resource in the
	// Repository's namespace to resolve blobStoreName.
	// +kubebuilder:validation:Optional
	BlobStoreNameSelector *xpv2.Selector `json:"blobStoreNameSelector,omitempty"`

	// StrictContentTypeValidation enables strict content type validation.
	// +kubebuilder:default=true
	// +kubebuilder:validation:Optional
	StrictContentTypeValidation *bool `json:"strictContentTypeValidation,omitempty"`

	// WritePolicy for hosted repositories.
	// +kubebuilder:validation:Enum=ALLOW;ALLOW_ONCE;DENY
	// +kubebuilder:validation:Optional
	WritePolicy *string `json:"writePolicy,omitempty"`

	// LatestPolicy allows redeploying the 'latest' tag of a Docker or OCI
	// hosted repository, deferring to the write policy for every other tag.
	// It requires the ALLOW_ONCE writePolicy, and is turned off when left
	// unset.
	// +kubebuilder:validation:Optional
	LatestPolicy *bool `json:"latestPolicy,omitempty"`
}

// RepositoryCleanup defines cleanup configuration for a repository.
type RepositoryCleanup struct {
	// PolicyNames is a list of cleanup policy names.
	// +kubebuilder:validation:Optional
	PolicyNames []string `json:"policyNames,omitempty"`

	// PolicyNameRefs resolve policyNames from CleanupPolicy managed resources
	// in the Repository's namespace. Resolved names replace policyNames.
	// +kubebuilder:validation:Optional
	PolicyNameRefs []xpv2.Reference `json:"policyNameRefs,omitempty"`

	// PolicyNameSelector selects CleanupPolicy managed resources in the
	// Repository's namespace to resolve policyNames.
	// +kubebuilder:validation:Optional
	PolicyNameSelector *xpv2.Selector `json:"policyNameSelector,omitempty"`
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

// ContainerConfig defines the registry settings of a Docker or OCI
// repository.
type ContainerConfig struct {
	// V1Enabled allows clients to use the V1 registry API.
	// +kubebuilder:validation:Optional
	V1Enabled *bool `json:"v1Enabled,omitempty"`

	// ForceBasicAuth forces basic authentication. It defaults to true; the
	// Docker or OCI Bearer Token realm is required when it is disabled.
	// +kubebuilder:validation:Optional
	ForceBasicAuth *bool `json:"forceBasicAuth,omitempty"`

	// HTTPPort creates an HTTP connector at this port.
	// +kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// HTTPSPort creates an HTTPS connector at this port.
	// +kubebuilder:validation:Optional
	HTTPSPort *int32 `json:"httpsPort,omitempty"`

	// Subdomain serves the repository at this subdomain.
	// +kubebuilder:validation:Optional
	Subdomain *string `json:"subdomain,omitempty"`

	// PathEnabled allows the repository name in image paths.
	// +kubebuilder:validation:Optional
	PathEnabled *bool `json:"pathEnabled,omitempty"`
}

// DockerConfig is the former name of ContainerConfig.
//
// Deprecated: use ContainerConfig, which the oci block shares.
type DockerConfig = ContainerConfig

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

	// PreserveEncodedCharacters keeps encoded characters such as %2B, %23
	// and %20 encoded when proxying to the remote, which AWS S3, Cloudflare
	// and Azure Blob Storage remotes require. Only enable it for trusted
	// remotes. It is turned off when left unset.
	// +kubebuilder:validation:Optional
	PreserveEncodedCharacters *bool `json:"preserveEncodedCharacters,omitempty"`
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
	// Type of authentication. username and ntlm authenticate with username
	// and passwordSecretRef, bearerToken with bearerTokenSecretRef.
	// +kubebuilder:validation:Enum=username;ntlm;bearerToken
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`

	// Username for authentication.
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`

	// PasswordSecretRef is a reference to a secret containing the password.
	// The secret must be in the same namespace as the Repository.
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv2.LocalSecretKeySelector `json:"passwordSecretRef,omitempty"`

	// NTLMHost is the NTLM host.
	// +kubebuilder:validation:Optional
	NTLMHost *string `json:"ntlmHost,omitempty"`

	// NTLMDomain is the NTLM domain.
	// +kubebuilder:validation:Optional
	NTLMDomain *string `json:"ntlmDomain,omitempty"`

	// BearerTokenSecretRef is a reference to a secret containing the bearer
	// token of the bearerToken authentication type. The secret must be in
	// the same namespace as the Repository.
	// +kubebuilder:validation:Optional
	BearerTokenSecretRef *xpv2.LocalSecretKeySelector `json:"bearerTokenSecretRef,omitempty"`

	// Preemptive sends the credentials with every request instead of
	// waiting for the remote to ask for them. Use with caution. Nexus only
	// supports it on maven2, pypi and terraform proxy repositories. It is
	// turned off when left unset.
	// +kubebuilder:validation:Optional
	Preemptive *bool `json:"preemptive,omitempty"`
}

// GroupConfig defines group configuration for group repositories.
type GroupConfig struct {
	// MemberNames is a list of member repository names.
	// +kubebuilder:validation:Optional
	MemberNames []string `json:"memberNames,omitempty"`

	// MemberNameRefs resolve memberNames from Repository managed resources in
	// the group Repository's namespace. Resolved names replace memberNames, in
	// the order of the references.
	// +kubebuilder:validation:Optional
	MemberNameRefs []xpv2.Reference `json:"memberNameRefs,omitempty"`

	// MemberNameSelector selects Repository managed resources in the group
	// Repository's namespace, other than the group itself, to resolve
	// memberNames, sorted by their Nexus names.
	// +kubebuilder:validation:Optional
	MemberNameSelector *xpv2.Selector `json:"memberNameSelector,omitempty"`

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

	// EnforceDistribution restricts the requests of a proxy repository to
	// the distribution, which is then required. It is turned off when left
	// unset.
	// +kubebuilder:validation:Optional
	EnforceDistribution *bool `json:"enforceDistribution,omitempty"`
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

// ContainerProxyConfig defines the index settings of a Docker or OCI proxy
// repository.
type ContainerProxyConfig struct {
	// IndexType is the type of index (HUB, REGISTRY, CUSTOM). It defaults to
	// HUB for docker and to REGISTRY for oci, as Nexus does.
	// +kubebuilder:validation:Enum=HUB;REGISTRY;CUSTOM
	// +kubebuilder:validation:Optional
	IndexType *string `json:"indexType,omitempty"`

	// IndexURL is the URL of the index to use (for CUSTOM type).
	// +kubebuilder:validation:Optional
	IndexURL *string `json:"indexUrl,omitempty"`

	// CacheForeignLayers allows downloading and caching foreign layers.
	// +kubebuilder:validation:Optional
	CacheForeignLayers *bool `json:"cacheForeignLayers,omitempty"`

	// ForeignLayerUrlWhitelist is a list of regex patterns for allowed foreign layer URLs.
	// +kubebuilder:validation:Optional
	ForeignLayerUrlWhitelist []string `json:"foreignLayerUrlWhitelist,omitempty"`

	// EcrAuth authenticates a Docker proxy repository to AWS Elastic
	// Container Registry. Nexus does not offer it on OCI proxy repositories.
	// +kubebuilder:validation:Optional
	EcrAuth *EcrAuthConfig `json:"ecrAuth,omitempty"`
}

// EcrAuthConfig defines the AWS ECR authentication of a Docker proxy
// repository.
type EcrAuthConfig struct {
	// Enabled turns AWS ECR authentication on.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// AwsRegion is the AWS region of the ECR registry, such as us-east-1.
	// +kubebuilder:validation:Optional
	AwsRegion *string `json:"awsRegion,omitempty"`

	// RegistryID is the 12-digit AWS account ID hosting the ECR registry.
	// +kubebuilder:validation:Pattern=`^[0-9]{12}$`
	// +kubebuilder:validation:Optional
	RegistryID *string `json:"registryId,omitempty"`

	// AccessKeyID is the AWS access key ID used to authenticate to ECR.
	// Nexus masks it on read.
	// +kubebuilder:validation:Optional
	AccessKeyID *string `json:"accessKeyId,omitempty"`

	// SecretAccessKeySecretRef is a reference to a secret containing the AWS
	// secret access key. The secret must be in the same namespace as the
	// Repository.
	// +kubebuilder:validation:Optional
	SecretAccessKeySecretRef *xpv2.LocalSecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`

	// SessionTokenSecretRef is a reference to a secret containing the AWS
	// session token of short-lived STS credentials. The secret must be in
	// the same namespace as the Repository.
	// +kubebuilder:validation:Optional
	SessionTokenSecretRef *xpv2.LocalSecretKeySelector `json:"sessionTokenSecretRef,omitempty"`
}

// DockerProxyConfig is the former name of ContainerProxyConfig.
//
// Deprecated: use ContainerProxyConfig, which the ociProxy block shares.
type DockerProxyConfig = ContainerProxyConfig

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

	// AllowAnonymousSymbolAccess allows unauthenticated access to the symbol
	// server, which debuggers require. It defaults to true, as in Nexus.
	// +kubebuilder:validation:Optional
	AllowAnonymousSymbolAccess *bool `json:"allowAnonymousSymbolAccess,omitempty"`

	// SymbolServerURL is the upstream symbol server to proxy.
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Optional
	SymbolServerURL *string `json:"symbolServerUrl,omitempty"`
}

// RawConfig defines raw specific configuration.
type RawConfig struct {
	// ContentDisposition is the Content-Disposition header Nexus serves
	// content with. Nexus defaults it to ATTACHMENT, and keeps it when an
	// update leaves it out.
	// +kubebuilder:validation:Enum=INLINE;ATTACHMENT
	// +kubebuilder:validation:Optional
	ContentDisposition *string `json:"contentDisposition,omitempty"`
}

// ConanProxyConfig defines Conan proxy specific configuration.
type ConanProxyConfig struct {
	// ConanVersion is the Conan protocol version of the remote. Nexus
	// defaults it to V2 and cannot change it once the repository exists.
	// +kubebuilder:validation:Enum=V1;V2
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="conanVersion is immutable: Nexus cannot change it."
	// +kubebuilder:validation:Optional
	ConanVersion *string `json:"conanVersion,omitempty"`
}

// PypiProxyConfig defines PyPI proxy specific configuration.
type PypiProxyConfig struct {
	// IndexPath is the path of the remote index: /simple for standard
	// remotes such as PyPI.org, or the empty string for remotes serving it
	// at their root. It defaults to /simple, as in Nexus.
	// +kubebuilder:validation:Enum=/simple;""
	// +kubebuilder:validation:Optional
	IndexPath *string `json:"indexPath,omitempty"`
}

// XetConfig defines the Xet storage protocol of a Hugging Face proxy
// repository.
type XetConfig struct {
	// XetCasURL is the Xet content addressed storage service of the remote.
	// Nexus uses the remote's default when it is left unset.
	// +kubebuilder:validation:Optional
	XetCasURL *string `json:"xetCasUrl,omitempty"`

	// XetTransferHostAllowList lists the hosts Nexus may download Xet
	// content from. Nexus uses the remote's defaults when it is left unset.
	// +kubebuilder:validation:Optional
	XetTransferHostAllowList []string `json:"xetTransferHostAllowList,omitempty"`
}

// ComponentConfig defines the component settings of a hosted repository.
type ComponentConfig struct {
	// ProprietaryComponents marks the components of the repository as
	// proprietary, which protects them from namespace confusion attacks. It
	// requires Sonatype Repository Firewall to take effect. It is turned
	// off when left unset.
	// +kubebuilder:validation:Optional
	ProprietaryComponents *bool `json:"proprietaryComponents,omitempty"`
}

// FirewallConfig defines the Sonatype Repository Firewall mode of a proxy
// repository.
type FirewallConfig struct {
	// Mode is the firewall mode. Every mode but DISABLED requires an IQ
	// Server connection.
	// +kubebuilder:validation:Enum=DISABLED;AUDIT;QUARANTINE;PCCS
	// +kubebuilder:validation:Required
	Mode string `json:"mode"`
}

// ReplicationConfig defines the pre-emptive pull of a proxy repository.
type ReplicationConfig struct {
	// PreemptivePullEnabled pulls new content from the remote before
	// clients ask for it.
	// +kubebuilder:validation:Required
	PreemptivePullEnabled bool `json:"preemptivePullEnabled"`

	// AssetPathRegex restricts the pre-emptive pull to the asset paths it
	// matches.
	// +kubebuilder:validation:Optional
	AssetPathRegex *string `json:"assetPathRegex,omitempty"`
}

// BowerConfig defines Bower specific configuration.
type BowerConfig struct {
	// RewritePackageUrls forces Bower to retrieve packages through this proxy.
	// +kubebuilder:validation:Optional
	RewritePackageUrls *bool `json:"rewritePackageUrls,omitempty"`
}

// RequireAuthenticationConfig defines whether a Cargo, Swift or Terraform
// repository overrides anonymous access.
type RequireAuthenticationConfig struct {
	// RequireAuthentication indicates if this repository requires
	// authentication. It is turned off when left unset.
	// +kubebuilder:validation:Optional
	RequireAuthentication *bool `json:"requireAuthentication,omitempty"`
}

// CargoConfig is the former name of RequireAuthenticationConfig.
//
// Deprecated: use RequireAuthenticationConfig, which the swift and terraform
// blocks share.
type CargoConfig = RequireAuthenticationConfig

// CosignConfig defines the cosign signature enforcement of an OCI repository.
// +kubebuilder:validation:XValidation:rule="self.enforcement == 'KEYLESS' || (!has(self.identityRegex) && !has(self.issuerRegex))",message="identityRegex and issuerRegex require KEYLESS enforcement: Nexus drops them otherwise."
type CosignConfig struct {
	// Enforcement is the cosign enforcement mode. NONE disables enforcement;
	// KEYLESS requires a valid cosign signature attached as an OCI referrer
	// for every pull. Nexus 3.96 does not implement KEYLESS yet and rejects
	// it.
	// +kubebuilder:validation:Enum=NONE;KEYLESS
	// +kubebuilder:validation:Required
	Enforcement string `json:"enforcement"`

	// IdentityRegex is matched against the Fulcio certificate Subject
	// Alternative Name. It requires KEYLESS enforcement.
	// +kubebuilder:validation:Optional
	IdentityRegex *string `json:"identityRegex,omitempty"`

	// IssuerRegex is matched against the OIDC issuer extension of the Fulcio
	// certificate. It requires KEYLESS enforcement.
	// +kubebuilder:validation:Optional
	IssuerRegex *string `json:"issuerRegex,omitempty"`
}

// AlpineSigningConfig defines the signing key of an alpine repository.
type AlpineSigningConfig struct {
	// Keypair is the RSA signing key pair (PEM armored private key).
	// +kubebuilder:validation:Required
	Keypair string `json:"keypair"`

	// Passphrase to access the RSA signing key.
	// +kubebuilder:validation:Optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// TerraformSigningConfig defines the signing key of a Terraform hosted
// repository.
type TerraformSigningConfig struct {
	// Keypair is the PGP signing key pair (armored private key).
	// +kubebuilder:validation:Required
	Keypair string `json:"keypair"`

	// Passphrase to access the PGP signing key.
	// +kubebuilder:validation:Optional
	Passphrase *string `json:"passphrase,omitempty"`
}

// RepositoryObservation represents the observed state of a Repository.
//
// Every field is reported for every format; a field a format has no concept of
// simply stays empty. A hosted repository reports no remoteUrl, a proxy
// repository reports no memberNames.
type RepositoryObservation struct {
	// URL is the URL Nexus serves the repository at.
	URL *string `json:"url,omitempty"`

	// Name is the repository name observed in Nexus.
	Name string `json:"name,omitempty"`

	// Online reports whether Nexus is currently serving the repository.
	Online *bool `json:"online,omitempty"`

	// BlobStoreName is the blob store backing the repository.
	BlobStoreName string `json:"blobStoreName,omitempty"`

	// StrictContentTypeValidation reports whether Nexus validates that
	// uploaded content matches the format of the repository.
	StrictContentTypeValidation *bool `json:"strictContentTypeValidation,omitempty"`

	// WritePolicy is the deployment policy of a hosted repository.
	WritePolicy string `json:"writePolicy,omitempty"`

	// CleanupPolicyNames are the cleanup policies applied to the repository.
	CleanupPolicyNames []string `json:"cleanupPolicyNames,omitempty"`

	// RemoteURL is the remote repository a proxy repository proxies.
	RemoteURL string `json:"remoteUrl,omitempty"`

	// MemberNames are the members of a group repository.
	MemberNames []string `json:"memberNames,omitempty"`

	// RoutingRuleName is the routing rule assigned to the repository.
	RoutingRuleName string `json:"routingRuleName,omitempty"`

	// CredentialsFingerprint identifies the credentials last applied to
	// Nexus, which Nexus never reports back. It changes when a referenced
	// Secret changes or is repointed, which triggers an update.
	CredentialsFingerprint string `json:"credentialsFingerprint,omitempty"`
}

// RepositorySpec defines the desired state of Repository.
//
// The signing keys Nexus requires are only required when the provider may
// create or update the repository: one imported with Observe alone is never
// written to.
// +kubebuilder:validation:XValidation:rule="(has(self.managementPolicies) && self.managementPolicies.all(p, p != '*' && p != 'Create' && p != 'Update')) || self.forProvider.format != 'alpine' || self.forProvider.type == 'hosted' || has(self.forProvider.alpineSigning)",message="alpineSigning is required for alpine proxy and group repositories the provider may create or update."
// +kubebuilder:validation:XValidation:rule="(has(self.managementPolicies) && self.managementPolicies.all(p, p != '*' && p != 'Create' && p != 'Update')) || self.forProvider.format != 'terraform' || self.forProvider.type != 'hosted' || has(self.forProvider.terraformSigning)",message="terraformSigning is required for terraform hosted repositories the provider may create or update."
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
