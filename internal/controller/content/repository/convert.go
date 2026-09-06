package repository

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
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
func (d desiredRepo) cleanup() *repository.Cleanup {
	spec := d.params().Cleanup
	if spec == nil {
		return nil
	}

	policies := spec.PolicyNames
	if policies == nil {
		policies = []string{}
	}

	return &repository.Cleanup{PolicyNames: policies}
}

// routingRule converts the routing rule assigned to a proxy repository.
func (d desiredRepo) routingRule() *string {
	return d.params().RoutingRule
}

// hostedStorage converts the storage configuration of a hosted repository.
func (d desiredRepo) hostedStorage() repository.HostedStorage {
	storage := repository.HostedStorage{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
		WritePolicy:                 new(repository.StorageWritePolicyAllow),
	}

	spec := d.params().Storage
	if spec == nil {
		return storage
	}

	storage.BlobStoreName = spec.BlobStoreName
	storage.StrictContentTypeValidation = ptr.Deref(spec.StrictContentTypeValidation, true)

	if spec.WritePolicy != nil {
		storage.WritePolicy = new(repository.StorageWritePolicy(*spec.WritePolicy))
	}

	return storage
}

// dockerHostedStorage converts the storage configuration of a hosted Docker
// repository, which carries the write policy as a plain value.
func (d desiredRepo) dockerHostedStorage() repository.DockerHostedStorage {
	hosted := d.hostedStorage()

	return repository.DockerHostedStorage{
		BlobStoreName:               hosted.BlobStoreName,
		StrictContentTypeValidation: hosted.StrictContentTypeValidation,
		WritePolicy:                 ptr.Deref(hosted.WritePolicy, repository.StorageWritePolicyAllow),
	}
}

// storage converts the storage configuration of a proxy or group repository,
// neither of which has a write policy.
func (d desiredRepo) storage() repository.Storage {
	storage := repository.Storage{
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
func (d desiredRepo) proxy() repository.Proxy {
	proxy := repository.Proxy{
		ContentMaxAge:  defaultMaxAge,
		MetadataMaxAge: defaultMaxAge,
	}

	spec := d.params().Proxy
	if spec == nil {
		return proxy
	}

	proxy.RemoteURL = spec.RemoteURL
	proxy.ContentMaxAge = int(ptr.Deref(spec.ContentMaxAge, defaultMaxAge))
	proxy.MetadataMaxAge = int(ptr.Deref(spec.MetadataMaxAge, defaultMaxAge))

	return proxy
}

// negativeCache converts the negative cache configuration of a proxy
// repository.
func (d desiredRepo) negativeCache() repository.NegativeCache {
	negativeCache := repository.NegativeCache{
		Enabled: true,
		TTL:     defaultNegativeCacheTTL,
	}

	spec := d.params().NegativeCache
	if spec == nil {
		return negativeCache
	}

	negativeCache.Enabled = ptr.Deref(spec.Enabled, true)
	negativeCache.TTL = int(ptr.Deref(spec.TimeToLive, defaultNegativeCacheTTL))

	return negativeCache
}

// httpClient converts the outbound HTTP client configuration of a proxy
// repository.
func (d desiredRepo) httpClient() repository.HTTPClient {
	client := repository.HTTPClient{Blocked: false, AutoBlock: true}

	spec := d.params().HTTPClient
	if spec == nil {
		return client
	}

	client.Blocked = ptr.Deref(spec.Blocked, false)
	client.AutoBlock = ptr.Deref(spec.AutoBlock, true)
	client.Connection = d.httpConnection()

	auth := d.httpAuth()
	if auth != nil {
		client.Authentication = &repository.HTTPClientAuthentication{
			Type:       auth.Type,
			Username:   auth.Username,
			NTLMHost:   auth.NTLMHost,
			NTLMDomain: auth.NTLMDomain,
			Password:   auth.Password,
		}
	}

	return client
}

// httpClientWithPreemptiveAuth converts the outbound HTTP client configuration
// of a proxy repository whose format also accepts pre-emptive authentication.
func (d desiredRepo) httpClientWithPreemptiveAuth() repository.HTTPClientWithPreemptiveAuth {
	client := d.httpClient()
	preemptive := repository.HTTPClientWithPreemptiveAuth{
		Blocked:    client.Blocked,
		AutoBlock:  client.AutoBlock,
		Connection: client.Connection,
	}

	if client.Authentication != nil {
		preemptive.Authentication = d.httpAuth()
	}

	return preemptive
}

// httpConnection converts the connection tuning of a proxy repository's HTTP
// client.
func (d desiredRepo) httpConnection() *repository.HTTPClientConnection {
	if d.params().HTTPClient == nil {
		return nil
	}

	spec := d.params().HTTPClient.Connection
	if spec == nil {
		return nil
	}

	connection := &repository.HTTPClientConnection{
		UserAgentSuffix:         ptr.Deref(spec.UserAgentSuffix, ""),
		EnableCircularRedirects: spec.EnableCircularRedirects,
		EnableCookies:           spec.EnableCookies,
		UseTrustStore:           spec.UseTrustStore,
	}

	if spec.Retries != nil {
		connection.Retries = new(int(*spec.Retries))
	}

	if spec.Timeout != nil {
		connection.Timeout = new(int(*spec.Timeout))
	}

	return connection
}

// httpAuth converts the credentials of a proxy repository's HTTP client. The
// password is the one the controller resolved from the referenced secret.
func (d desiredRepo) httpAuth() *repository.HTTPClientAuthenticationWithPreemptive {
	if d.params().HTTPClient == nil || d.params().HTTPClient.Authentication == nil {
		return nil
	}

	spec := d.params().HTTPClient.Authentication

	return &repository.HTTPClientAuthenticationWithPreemptive{
		Type:       repository.HTTPClientAuthenticationType(ptr.Deref(spec.Type, "")),
		Username:   ptr.Deref(spec.Username, ""),
		NTLMHost:   ptr.Deref(spec.NTLMHost, ""),
		NTLMDomain: ptr.Deref(spec.NTLMDomain, ""),
		Password:   d.httpPassword,
	}
}

// group converts the membership of a group repository.
func (d desiredRepo) group() repository.Group {
	spec := d.params().Group
	if spec == nil {
		return repository.Group{}
	}

	return repository.Group{MemberNames: spec.MemberNames}
}

// groupDeploy converts the membership of a group repository whose format also
// supports a writable member.
func (d desiredRepo) groupDeploy() repository.GroupDeploy {
	spec := d.params().Group
	if spec == nil {
		return repository.GroupDeploy{}
	}

	return repository.GroupDeploy{MemberNames: spec.MemberNames, WritableMember: spec.WritableMember}
}

// maven converts the Maven specific configuration.
func (d desiredRepo) maven() repository.Maven {
	maven := repository.Maven{
		VersionPolicy: repository.MavenVersionPolicyRelease,
		LayoutPolicy:  repository.MavenLayoutPolicyStrict,
	}

	spec := d.params().Maven
	if spec == nil {
		return maven
	}

	if spec.VersionPolicy != nil {
		maven.VersionPolicy = repository.MavenVersionPolicy(*spec.VersionPolicy)
	}

	if spec.LayoutPolicy != nil {
		maven.LayoutPolicy = repository.MavenLayoutPolicy(*spec.LayoutPolicy)
	}

	if spec.ContentDisposition != nil {
		maven.ContentDisposition = new(repository.MavenContentDisposition(*spec.ContentDisposition))
	}

	return maven
}

// docker converts the Docker specific configuration.
func (d desiredRepo) docker() repository.Docker {
	docker := repository.Docker{V1Enabled: false, ForceBasicAuth: true}

	spec := d.params().Docker
	if spec == nil {
		return docker
	}

	docker.V1Enabled = ptr.Deref(spec.V1Enabled, false)
	docker.ForceBasicAuth = ptr.Deref(spec.ForceBasicAuth, true)
	docker.Subdomain = spec.Subdomain

	if spec.HTTPPort != nil {
		docker.HTTPPort = new(int(*spec.HTTPPort))
	}

	if spec.HTTPSPort != nil {
		docker.HTTPSPort = new(int(*spec.HTTPSPort))
	}

	return docker
}

// dockerProxy converts the Docker index configuration of a Docker proxy
// repository.
func (d desiredRepo) dockerProxy() repository.DockerProxy {
	dockerProxy := repository.DockerProxy{IndexType: repository.DockerProxyIndexTypeHub}

	spec := d.params().DockerProxy
	if spec == nil {
		return dockerProxy
	}

	if spec.IndexType != nil {
		dockerProxy.IndexType = repository.DockerProxyIndexType(*spec.IndexType)
	}

	dockerProxy.IndexURL = spec.IndexURL
	dockerProxy.CacheForeignLayers = spec.CacheForeignLayers
	dockerProxy.ForeignLayerUrlWhitelist = spec.ForeignLayerUrlWhitelist

	return dockerProxy
}

// npm converts the npm specific configuration of a proxy repository.
func (d desiredRepo) npm() *repository.Npm {
	spec := d.params().Npm
	if spec == nil {
		return nil
	}

	return &repository.Npm{
		RemoveNonCataloged: ptr.Deref(spec.RemoveNonCataloged, false),
		RemoveQuarantined:  ptr.Deref(spec.RemoveQuarantined, false),
	}
}

// apt converts the APT specific configuration of a hosted repository.
func (d desiredRepo) apt() repository.AptHosted {
	spec := d.params().Apt
	if spec == nil {
		return repository.AptHosted{}
	}

	return repository.AptHosted{Distribution: ptr.Deref(spec.Distribution, "")}
}

// aptProxy converts the APT specific configuration of a proxy repository.
func (d desiredRepo) aptProxy() repository.AptProxy {
	spec := d.params().Apt
	if spec == nil {
		return repository.AptProxy{}
	}

	return repository.AptProxy{
		Distribution: ptr.Deref(spec.Distribution, ""),
		Flat:         ptr.Deref(spec.Flat, false),
	}
}

// aptSigning converts the APT signing key of a hosted repository.
func (d desiredRepo) aptSigning() repository.AptSigning {
	spec := d.params().AptSigning
	if spec == nil {
		return repository.AptSigning{}
	}

	return repository.AptSigning{Keypair: spec.Keypair, Passphrase: spec.Passphrase}
}

// yum converts the Yum specific configuration of a hosted repository.
func (d desiredRepo) yum() repository.Yum {
	yum := repository.Yum{}

	spec := d.params().Yum
	if spec == nil {
		return yum
	}

	yum.RepodataDepth = int(ptr.Deref(spec.RepodataDepth, 0))

	if spec.DeployPolicy != nil {
		yum.DeployPolicy = new(repository.YumDeployPolicy(*spec.DeployPolicy))
	}

	return yum
}

// yumSigning converts the Yum signing key of a proxy or group repository.
func (d desiredRepo) yumSigning() *repository.YumSigning {
	spec := d.params().YumSigning
	if spec == nil {
		return nil
	}

	return &repository.YumSigning{Keypair: spec.Keypair, Passphrase: spec.Passphrase}
}

// nugetProxy converts the NuGet specific configuration of a proxy repository.
func (d desiredRepo) nugetProxy() repository.NugetProxy {
	nugetProxy := repository.NugetProxy{
		QueryCacheItemMaxAge: defaultNugetQueryCacheItemMaxAge,
		NugetVersion:         repository.NugetVersion3,
	}

	spec := d.params().NugetProxy
	if spec == nil {
		return nugetProxy
	}

	nugetProxy.QueryCacheItemMaxAge = int(ptr.Deref(spec.QueryCacheItemMaxAge, defaultNugetQueryCacheItemMaxAge))

	if spec.NugetVersion != nil {
		nugetProxy.NugetVersion = repository.NugetVersion(*spec.NugetVersion)
	}

	return nugetProxy
}

// bower converts the Bower specific configuration of a proxy repository.
func (d desiredRepo) bower() repository.Bower {
	spec := d.params().Bower
	if spec == nil {
		return repository.Bower{}
	}

	return repository.Bower{RewritePackageUrls: ptr.Deref(spec.RewritePackageUrls, false)}
}

// cargo converts the Cargo specific configuration.
func (d desiredRepo) cargo() repository.Cargo {
	spec := d.params().Cargo
	if spec == nil {
		return repository.Cargo{}
	}

	return repository.Cargo{RequireAuthentication: ptr.Deref(spec.RequireAuthentication, false)}
}
