package repository

import (
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	"k8s.io/utils/ptr"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

const (
	// repoTypeHosted is the Nexus repository type for hosted repositories.
	repoTypeHosted = "hosted"
	// repoTypeProxy is the Nexus repository type for proxy repositories.
	repoTypeProxy = "proxy"
	// repoTypeGroup is the Nexus repository type for group repositories.
	repoTypeGroup = "group"

	// defaultBlobStoreName is the blob store used when the spec omits storage.
	defaultBlobStoreName = "default"
	// defaultMaxAge is the default proxy content and metadata max age, in
	// minutes.
	defaultMaxAge = 1440
	// defaultNegativeCacheTTL is the default negative cache time to live, in
	// minutes.
	defaultNegativeCacheTTL = 1440
	// defaultNugetQueryCacheItemMaxAge is the default NuGet query cache max
	// age, in seconds. It mirrors the default of the CRD field.
	defaultNugetQueryCacheItemMaxAge = 3600
	// defaultNugetVersion is the NuGet protocol version of a proxy repository
	// whose spec sets none.
	defaultNugetVersion = "V3"
	// defaultWritePolicy is the write policy of a hosted repository whose
	// spec sets none.
	defaultWritePolicy = "ALLOW"
	// defaultMavenVersionPolicy is the Maven version policy used when the
	// spec sets none.
	defaultMavenVersionPolicy = "RELEASE"
	// defaultStrictPolicy is both the Maven layout policy and the Yum deploy
	// policy used when the spec sets none.
	defaultStrictPolicy = "STRICT"
	// defaultDockerIndexType is the Docker index of a Docker proxy repository
	// whose spec sets none.
	defaultDockerIndexType = "HUB"
)

// desiredRepo is the input every payload builder works from: the desired state
// of one Repository, plus the values the controller had to resolve from
// Kubernetes before the payload could be rendered.
//
// Its methods convert the parts of the spec that are shared between formats,
// so that a format only has to describe what is genuinely specific to it.
type desiredRepo struct {
	// repo is the Repository custom resource being reconciled.
	repo *repositoryv1alpha1.Repository
	// httpPassword is the proxy HTTP client password resolved from the secret
	// referenced by the spec. It is empty when the spec references none, and
	// when the caller does not need it - drift detection never compares
	// credentials, so Observe leaves it unset.
	httpPassword string
}

// params returns the desired repository parameters.
func (d desiredRepo) params() *repositoryv1alpha1.RepositoryParameters {
	return &d.repo.Spec.ForProvider
}

// name returns the Nexus repository name.
func (d desiredRepo) name() string {
	return d.params().Name
}

// online reports whether the repository should serve requests. Nexus
// repositories are online unless the spec says otherwise.
func (d desiredRepo) online() bool {
	return ptr.Deref(d.params().Online, true)
}

// cleanup converts the cleanup policy configuration.
//
// It returns nil when the spec declares no cleanup block at all, so that the
// field is left out of the payload rather than clearing policies set
// elsewhere. When the block is declared the policy list is always sent, empty
// included, so that removing every policy from the spec removes it from Nexus
// too. The list is never sent as null: Nexus rejects a null policyNames with
// an internal error rather than treating it as empty.
func (d desiredRepo) cleanup() *v395.CleanupPolicyAttributes {
	spec := d.params().Cleanup
	if spec == nil {
		return nil
	}

	policies := spec.PolicyNames
	if policies == nil {
		policies = []string{}
	}

	return &v395.CleanupPolicyAttributes{PolicyNames: policies}
}

// routingRule converts the routing rule assigned to a proxy repository.
func (d desiredRepo) routingRule() *string {
	return d.params().RoutingRule
}

// hostedStorage converts the storage configuration of a hosted repository.
func (d desiredRepo) hostedStorage() v395.HostedStorageAttributes {
	storage := v395.HostedStorageAttributes{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
		WritePolicy:                 defaultWritePolicy,
	}

	spec := d.params().Storage
	if spec == nil {
		return storage
	}

	storage.BlobStoreName = spec.BlobStoreName
	storage.StrictContentTypeValidation = ptr.Deref(spec.StrictContentTypeValidation, true)
	storage.WritePolicy = ptr.Deref(spec.WritePolicy, defaultWritePolicy)

	return storage
}

// dockerHostedStorage converts the storage configuration of a hosted Docker
// repository.
func (d desiredRepo) dockerHostedStorage() v395.DockerHostedStorageAttributes {
	hosted := d.hostedStorage()

	return v395.DockerHostedStorageAttributes{
		BlobStoreName:               hosted.BlobStoreName,
		StrictContentTypeValidation: hosted.StrictContentTypeValidation,
		WritePolicy:                 hosted.WritePolicy,
	}
}

// storage converts the storage configuration of a proxy or group repository,
// neither of which has a write policy.
func (d desiredRepo) storage() v395.StorageAttributes {
	storage := v395.StorageAttributes{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
	}

	spec := d.params().Storage
	if spec == nil {
		return storage
	}

	storage.BlobStoreName = spec.BlobStoreName
	storage.StrictContentTypeValidation = ptr.Deref(spec.StrictContentTypeValidation, true)

	return storage
}

// proxy converts the remote target configuration of a proxy repository.
func (d desiredRepo) proxy() v395.ProxyAttributes {
	proxy := v395.ProxyAttributes{
		ContentMaxAge:  defaultMaxAge,
		MetadataMaxAge: defaultMaxAge,
	}

	spec := d.params().Proxy
	if spec == nil {
		return proxy
	}

	proxy.RemoteUrl = spec.RemoteURL
	proxy.ContentMaxAge = ptr.Deref(spec.ContentMaxAge, defaultMaxAge)
	proxy.MetadataMaxAge = ptr.Deref(spec.MetadataMaxAge, defaultMaxAge)

	return proxy
}

// negativeCache converts the negative cache configuration of a proxy
// repository.
func (d desiredRepo) negativeCache() v395.NegativeCacheAttributes {
	negativeCache := v395.NegativeCacheAttributes{
		Enabled:    true,
		TimeToLive: defaultNegativeCacheTTL,
	}

	spec := d.params().NegativeCache
	if spec == nil {
		return negativeCache
	}

	negativeCache.Enabled = ptr.Deref(spec.Enabled, true)
	negativeCache.TimeToLive = ptr.Deref(spec.TimeToLive, defaultNegativeCacheTTL)

	return negativeCache
}

// httpClient converts the outbound HTTP client configuration of a proxy
// repository.
func (d desiredRepo) httpClient() v395.HttpClientAttributes {
	client := v395.HttpClientAttributes{Blocked: new(false), AutoBlock: new(true)}

	spec := d.params().HTTPClient
	if spec == nil {
		return client
	}

	client.Blocked = new(ptr.Deref(spec.Blocked, false))
	client.AutoBlock = new(ptr.Deref(spec.AutoBlock, true))
	client.Connection = d.httpConnection()

	auth := d.httpAuth()
	if auth != nil {
		client.Authentication = &v395.HttpClientConnectionAuthenticationAttributes{
			Type:       auth.Type,
			Username:   auth.Username,
			NtlmHost:   auth.NtlmHost,
			NtlmDomain: auth.NtlmDomain,
			Password:   auth.Password,
		}
	}

	return client
}

// httpClientWithPreemptiveAuth converts the outbound HTTP client configuration
// of a proxy repository whose format also accepts pre-emptive authentication.
func (d desiredRepo) httpClientWithPreemptiveAuth() v395.HttpClientAttributesWithPreemptiveAuth {
	client := d.httpClient()

	return v395.HttpClientAttributesWithPreemptiveAuth{
		Blocked:        client.Blocked,
		AutoBlock:      client.AutoBlock,
		Connection:     client.Connection,
		Authentication: d.httpAuth(),
	}
}

// httpConnection converts the connection tuning of a proxy repository's HTTP
// client.
func (d desiredRepo) httpConnection() *v395.HttpClientConnectionAttributes {
	if d.params().HTTPClient == nil {
		return nil
	}

	spec := d.params().HTTPClient.Connection
	if spec == nil {
		return nil
	}

	return &v395.HttpClientConnectionAttributes{
		UserAgentSuffix:         new(ptr.Deref(spec.UserAgentSuffix, "")),
		EnableCircularRedirects: spec.EnableCircularRedirects,
		EnableCookies:           spec.EnableCookies,
		UseTrustStore:           spec.UseTrustStore,
		Retries:                 spec.Retries,
		Timeout:                 spec.Timeout,
	}
}

// httpAuth converts the credentials of a proxy repository's HTTP client. The
// password is the one the controller resolved from the referenced secret.
func (d desiredRepo) httpAuth() *v395.HttpClientConnectionAuthenticationAttributesWithPreemptive {
	if d.params().HTTPClient == nil || d.params().HTTPClient.Authentication == nil {
		return nil
	}

	spec := d.params().HTTPClient.Authentication

	return &v395.HttpClientConnectionAuthenticationAttributesWithPreemptive{
		Type:       ptr.Deref(spec.Type, ""),
		Username:   new(ptr.Deref(spec.Username, "")),
		NtlmHost:   new(ptr.Deref(spec.NTLMHost, "")),
		NtlmDomain: new(ptr.Deref(spec.NTLMDomain, "")),
		Password:   new(d.httpPassword),
	}
}

// group converts the membership of a group repository.
func (d desiredRepo) group() v395.GroupAttributes {
	spec := d.params().Group
	if spec == nil {
		return v395.GroupAttributes{}
	}

	return v395.GroupAttributes{MemberNames: spec.MemberNames}
}

// groupDeploy converts the membership of a group repository whose format also
// supports a writable member.
func (d desiredRepo) groupDeploy() v395.GroupDeployAttributes {
	spec := d.params().Group
	if spec == nil {
		return v395.GroupDeployAttributes{}
	}

	return v395.GroupDeployAttributes{MemberNames: spec.MemberNames, WritableMember: spec.WritableMember}
}

// maven converts the Maven specific configuration.
func (d desiredRepo) maven() v395.MavenAttributes {
	maven := v395.MavenAttributes{
		VersionPolicy: defaultMavenVersionPolicy,
		LayoutPolicy:  defaultStrictPolicy,
	}

	spec := d.params().Maven
	if spec == nil {
		return maven
	}

	maven.VersionPolicy = ptr.Deref(spec.VersionPolicy, defaultMavenVersionPolicy)
	maven.LayoutPolicy = ptr.Deref(spec.LayoutPolicy, defaultStrictPolicy)
	maven.ContentDisposition = spec.ContentDisposition

	return maven
}

// docker converts the Docker specific configuration.
func (d desiredRepo) docker() v395.DockerAttributes {
	docker := v395.DockerAttributes{V1Enabled: false, ForceBasicAuth: true}

	spec := d.params().Docker
	if spec == nil {
		return docker
	}

	docker.V1Enabled = ptr.Deref(spec.V1Enabled, false)
	docker.ForceBasicAuth = ptr.Deref(spec.ForceBasicAuth, true)
	docker.Subdomain = spec.Subdomain
	docker.HttpPort = spec.HTTPPort
	docker.HttpsPort = spec.HTTPSPort

	return docker
}

// dockerProxy converts the Docker index configuration of a Docker proxy
// repository.
func (d desiredRepo) dockerProxy() v395.DockerProxyAttributes {
	dockerProxy := v395.DockerProxyAttributes{IndexType: defaultDockerIndexType}

	spec := d.params().DockerProxy
	if spec == nil {
		return dockerProxy
	}

	dockerProxy.IndexType = ptr.Deref(spec.IndexType, defaultDockerIndexType)
	dockerProxy.IndexUrl = spec.IndexURL
	dockerProxy.CacheForeignLayers = spec.CacheForeignLayers
	dockerProxy.ForeignLayerUrlWhitelist = spec.ForeignLayerUrlWhitelist

	return dockerProxy
}

// apt converts the APT specific configuration of a hosted repository.
func (d desiredRepo) apt() v395.AptHostedRepositoriesAttributes {
	spec := d.params().Apt
	if spec == nil {
		return v395.AptHostedRepositoriesAttributes{}
	}

	return v395.AptHostedRepositoriesAttributes{Distribution: ptr.Deref(spec.Distribution, "")}
}

// aptProxy converts the APT specific configuration of a proxy repository.
func (d desiredRepo) aptProxy() v395.AptProxyRepositoriesAttributes {
	spec := d.params().Apt
	if spec == nil {
		return v395.AptProxyRepositoriesAttributes{}
	}

	return v395.AptProxyRepositoriesAttributes{
		Distribution: new(ptr.Deref(spec.Distribution, "")),
		Flat:         ptr.Deref(spec.Flat, false),
	}
}

// aptSigning converts the APT signing key of a hosted repository.
func (d desiredRepo) aptSigning() v395.AptSigningRepositoriesAttributes {
	spec := d.params().AptSigning
	if spec == nil {
		return v395.AptSigningRepositoriesAttributes{}
	}

	return v395.AptSigningRepositoriesAttributes{Keypair: new(spec.Keypair), Passphrase: spec.Passphrase}
}

// yum converts the Yum specific configuration of a hosted repository.
func (d desiredRepo) yum() v395.YumAttributes {
	yum := v395.YumAttributes{DeployPolicy: defaultStrictPolicy}

	spec := d.params().Yum
	if spec == nil {
		return yum
	}

	yum.RepodataDepth = ptr.Deref(spec.RepodataDepth, 0)
	yum.DeployPolicy = ptr.Deref(spec.DeployPolicy, defaultStrictPolicy)

	return yum
}

// yumSigning converts the Yum signing key of a proxy or group repository.
func (d desiredRepo) yumSigning() *v395.YumSigningRepositoriesAttributes {
	spec := d.params().YumSigning
	if spec == nil {
		return nil
	}

	return &v395.YumSigningRepositoriesAttributes{Keypair: ptr.Deref(spec.Keypair, ""), Passphrase: spec.Passphrase}
}

// nugetProxy converts the NuGet specific configuration of a proxy repository.
func (d desiredRepo) nugetProxy() v395.NugetAttributes {
	nugetProxy := v395.NugetAttributes{
		QueryCacheItemMaxAge: new(int32(defaultNugetQueryCacheItemMaxAge)),
		NugetVersion:         new(defaultNugetVersion),
	}

	spec := d.params().NugetProxy
	if spec == nil {
		return nugetProxy
	}

	nugetProxy.QueryCacheItemMaxAge = new(ptr.Deref(spec.QueryCacheItemMaxAge, defaultNugetQueryCacheItemMaxAge))
	nugetProxy.NugetVersion = new(ptr.Deref(spec.NugetVersion, defaultNugetVersion))

	return nugetProxy
}

// cargo converts the Cargo specific configuration.
func (d desiredRepo) cargo() *v395.CargoAttributes {
	spec := d.params().Cargo
	if spec == nil {
		return &v395.CargoAttributes{RequireAuthentication: new(false)}
	}

	return &v395.CargoAttributes{RequireAuthentication: new(ptr.Deref(spec.RequireAuthentication, false))}
}
