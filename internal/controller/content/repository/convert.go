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
	// defaultOciIndexType is the index of an OCI proxy repository whose spec
	// sets none. Nexus defaults OCI proxies to REGISTRY, not to HUB.
	defaultOciIndexType = "REGISTRY"
	// cosignEnforcementNone is the cosign enforcement mode that verifies no
	// signature.
	cosignEnforcementNone = "NONE"
	// defaultPypiIndexPath is the remote index path of a PyPI proxy
	// repository whose spec sets none, the one Nexus defaults to.
	defaultPypiIndexPath = "/simple"
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
	// credentials, so Observe leaves the resolved secrets unset.
	httpPassword string
	// httpBearerToken is the proxy HTTP client bearer token resolved from
	// the secret referenced by the spec, empty like httpPassword.
	httpBearerToken string
	// ecrSecretAccessKey is the AWS secret access key of a Docker proxy's
	// ECR authentication, resolved from the secret referenced by the spec.
	ecrSecretAccessKey string
	// ecrSessionToken is the AWS session token of a Docker proxy's ECR
	// authentication, resolved from the secret referenced by the spec.
	ecrSessionToken string
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

// containerHostedStorage converts the storage configuration of a hosted Docker
// or OCI repository. The latest policy is always sent, false when the spec
// leaves it unset, since Nexus turns it off when an update leaves it out.
func (d desiredRepo) containerHostedStorage() v395.DockerHostedStorageAttributes {
	hosted := d.hostedStorage()

	var latestPolicy *bool
	if spec := d.params().Storage; spec != nil {
		latestPolicy = spec.LatestPolicy
	}

	return v395.DockerHostedStorageAttributes{
		BlobStoreName:               hosted.BlobStoreName,
		StrictContentTypeValidation: hosted.StrictContentTypeValidation,
		WritePolicy:                 hosted.WritePolicy,
		LatestPolicy:                new(ptr.Deref(latestPolicy, false)),
	}
}

// component converts the component settings of a hosted repository. It is
// always sent, since Nexus turns proprietaryComponents off when an update
// leaves it out.
func (d desiredRepo) component() v395.ComponentAttributes {
	spec := d.params().Component
	if spec == nil {
		return v395.ComponentAttributes{ProprietaryComponents: new(false)}
	}

	return v395.ComponentAttributes{ProprietaryComponents: new(ptr.Deref(spec.ProprietaryComponents, false))}
}

// firewall converts the firewall mode of a proxy repository, nil when the spec
// declares none. Nexus keeps the mode an update leaves out.
func (d desiredRepo) firewall() *v395.FirewallAttributes {
	spec := d.params().Firewall
	if spec == nil {
		return nil
	}

	return &v395.FirewallAttributes{Mode: new(spec.Mode)}
}

// replication converts the pre-emptive pull of a proxy repository, nil when
// the spec declares none.
func (d desiredRepo) replication() *v395.ReplicationAttributes {
	spec := d.params().Replication
	if spec == nil {
		return nil
	}

	return &v395.ReplicationAttributes{
		PreemptivePullEnabled: spec.PreemptivePullEnabled,
		AssetPathRegex:        spec.AssetPathRegex,
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

// proxy converts the remote target configuration of a proxy repository. The
// encoded characters setting is always sent, false when the spec leaves it
// unset, since Nexus turns it off when an update leaves it out.
func (d desiredRepo) proxy() v395.ProxyAttributes {
	proxy := v395.ProxyAttributes{
		ContentMaxAge:             defaultMaxAge,
		MetadataMaxAge:            defaultMaxAge,
		PreserveEncodedCharacters: new(false),
	}

	spec := d.params().Proxy
	if spec == nil {
		return proxy
	}

	proxy.RemoteUrl = spec.RemoteURL
	proxy.ContentMaxAge = ptr.Deref(spec.ContentMaxAge, defaultMaxAge)
	proxy.MetadataMaxAge = ptr.Deref(spec.MetadataMaxAge, defaultMaxAge)
	proxy.PreserveEncodedCharacters = new(ptr.Deref(spec.PreserveEncodedCharacters, false))

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

	// Pre-emptive authentication is left out: Nexus only honors it on the
	// formats whose payload is built by httpClientWithPreemptiveAuth.
	auth := d.httpAuth()
	if auth != nil {
		client.Authentication = &v395.HttpClientConnectionAuthenticationAttributes{
			Type:        auth.Type,
			Username:    auth.Username,
			NtlmHost:    auth.NtlmHost,
			NtlmDomain:  auth.NtlmDomain,
			Password:    auth.Password,
			BearerToken: auth.BearerToken,
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
// password and bearer token are the ones the controller resolved from the
// referenced secrets; the bearer token is left out when the spec references
// none.
func (d desiredRepo) httpAuth() *v395.HttpClientConnectionAuthenticationAttributesWithPreemptive {
	if d.params().HTTPClient == nil || d.params().HTTPClient.Authentication == nil {
		return nil
	}

	spec := d.params().HTTPClient.Authentication

	return &v395.HttpClientConnectionAuthenticationAttributesWithPreemptive{
		Type:        ptr.Deref(spec.Type, ""),
		Username:    new(ptr.Deref(spec.Username, "")),
		NtlmHost:    new(ptr.Deref(spec.NTLMHost, "")),
		NtlmDomain:  new(ptr.Deref(spec.NTLMDomain, "")),
		Password:    new(d.httpPassword),
		BearerToken: nonEmpty(d.httpBearerToken),
		Preemptive:  new(ptr.Deref(spec.Preemptive, false)),
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
	return containerAttributes(d.params().Docker)
}

// dockerProxy converts the index configuration of a Docker proxy repository,
// and its AWS ECR authentication, which Nexus removes when an update leaves
// it out.
func (d desiredRepo) dockerProxy() v395.DockerProxyAttributes {
	attributes := containerProxyAttributes(d.params().DockerProxy, defaultDockerIndexType)
	attributes.EcrAuth = d.ecrAuth()

	return attributes
}

// ecrAuth converts the AWS ECR authentication of a Docker proxy repository,
// nil when the spec declares none. The secret access key and session token
// are the ones the controller resolved from the referenced secrets, left out
// when the spec references none.
func (d desiredRepo) ecrAuth() *v395.EcrAuthAttributes {
	if d.params().DockerProxy == nil || d.params().DockerProxy.EcrAuth == nil {
		return nil
	}

	spec := d.params().DockerProxy.EcrAuth

	return &v395.EcrAuthAttributes{
		Enabled:         new(spec.Enabled),
		AwsRegion:       spec.AwsRegion,
		RegistryId:      spec.RegistryID,
		AccessKeyId:     spec.AccessKeyID,
		SecretAccessKey: nonEmpty(d.ecrSecretAccessKey),
		SessionToken:    nonEmpty(d.ecrSessionToken),
	}
}

// nonEmpty returns a pointer to value, nil when it is empty.
func nonEmpty(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

// containerAttributes converts the registry settings shared by Docker and OCI
// repositories.
func containerAttributes(spec *repositoryv1alpha1.ContainerConfig) v395.DockerAttributes {
	attributes := v395.DockerAttributes{V1Enabled: false, ForceBasicAuth: true}
	if spec == nil {
		return attributes
	}

	attributes.V1Enabled = ptr.Deref(spec.V1Enabled, false)
	attributes.ForceBasicAuth = ptr.Deref(spec.ForceBasicAuth, true)
	attributes.Subdomain = spec.Subdomain
	attributes.HttpPort = spec.HTTPPort
	attributes.HttpsPort = spec.HTTPSPort
	attributes.PathEnabled = spec.PathEnabled

	return attributes
}

// containerProxyAttributes converts the index settings shared by Docker and
// OCI proxy repositories. The index type is always sent, the format's default
// when the spec sets none, so that drift detection notices an index changed
// outside of the provider.
func containerProxyAttributes(spec *repositoryv1alpha1.ContainerProxyConfig, defaultIndexType string) v395.DockerProxyAttributes {
	attributes := v395.DockerProxyAttributes{IndexType: defaultIndexType}
	if spec == nil {
		return attributes
	}

	attributes.IndexType = ptr.Deref(spec.IndexType, defaultIndexType)
	attributes.IndexUrl = spec.IndexURL
	attributes.CacheForeignLayers = spec.CacheForeignLayers
	attributes.ForeignLayerUrlWhitelist = spec.ForeignLayerUrlWhitelist

	return attributes
}

// apt converts the APT specific configuration of a hosted repository.
func (d desiredRepo) apt() v395.AptHostedRepositoriesAttributes {
	spec := d.params().Apt
	if spec == nil {
		return v395.AptHostedRepositoriesAttributes{}
	}

	return v395.AptHostedRepositoriesAttributes{Distribution: ptr.Deref(spec.Distribution, "")}
}

// aptProxy converts the APT specific configuration of a proxy repository. The
// distribution enforcement is always sent, false when the spec leaves it
// unset, since Nexus turns it off when an update leaves it out.
func (d desiredRepo) aptProxy() v395.AptProxyRepositoriesAttributes {
	spec := d.params().Apt
	if spec == nil {
		return v395.AptProxyRepositoriesAttributes{EnforceDistribution: new(false)}
	}

	return v395.AptProxyRepositoriesAttributes{
		Distribution:        new(ptr.Deref(spec.Distribution, "")),
		Flat:                ptr.Deref(spec.Flat, false),
		EnforceDistribution: new(ptr.Deref(spec.EnforceDistribution, false)),
	}
}

// aptSigning converts the APT signing key of a hosted repository.
func (d desiredRepo) aptSigning() v395.AptSigningRepositoriesAttributes {
	return ptr.Deref(d.aptProxySigning(), v395.AptSigningRepositoriesAttributes{})
}

// aptProxySigning converts the APT signing key of a proxy repository, nil when
// the spec declares none.
func (d desiredRepo) aptProxySigning() *v395.AptSigningRepositoriesAttributes {
	spec := d.params().AptSigning
	if spec == nil {
		return nil
	}

	return &v395.AptSigningRepositoriesAttributes{Keypair: new(spec.Keypair), Passphrase: spec.Passphrase}
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
// Anonymous symbol access is always sent, true when the spec leaves it unset,
// since Nexus turns it back on when an update leaves it out.
func (d desiredRepo) nugetProxy() v395.NugetAttributes {
	nugetProxy := v395.NugetAttributes{
		QueryCacheItemMaxAge:       new(int32(defaultNugetQueryCacheItemMaxAge)),
		NugetVersion:               new(defaultNugetVersion),
		AllowAnonymousSymbolAccess: new(true),
	}

	spec := d.params().NugetProxy
	if spec == nil {
		return nugetProxy
	}

	nugetProxy.QueryCacheItemMaxAge = new(ptr.Deref(spec.QueryCacheItemMaxAge, defaultNugetQueryCacheItemMaxAge))
	nugetProxy.NugetVersion = new(ptr.Deref(spec.NugetVersion, defaultNugetVersion))
	nugetProxy.AllowAnonymousSymbolAccess = new(ptr.Deref(spec.AllowAnonymousSymbolAccess, true))
	nugetProxy.SymbolServerUrl = spec.SymbolServerURL

	return nugetProxy
}

// raw converts the raw specific configuration, nil when the spec declares
// none. Nexus keeps the content disposition an update leaves out.
func (d desiredRepo) raw() *v395.RawAttributes {
	spec := d.params().Raw
	if spec == nil {
		return nil
	}

	return &v395.RawAttributes{ContentDisposition: spec.ContentDisposition}
}

// conanProxy converts the Conan specific configuration of a proxy repository,
// nil when the spec sets no protocol version. Nexus picks V2 then, and rejects
// any change of version afterwards, so no default is ever sent.
func (d desiredRepo) conanProxy() *v395.ConanProxyAttributes {
	spec := d.params().ConanProxy
	if spec == nil || spec.ConanVersion == nil {
		return nil
	}

	return &v395.ConanProxyAttributes{ConanVersion: spec.ConanVersion}
}

// pypiProxy converts the PyPI specific configuration of a proxy repository.
// The index path is always sent, /simple when the spec leaves it unset, since
// Nexus resets it to /simple when an update leaves it out.
func (d desiredRepo) pypiProxy() *v395.PyPiProxyAttributes {
	var indexPath *string
	if spec := d.params().Pypi; spec != nil {
		indexPath = spec.IndexPath
	}

	return &v395.PyPiProxyAttributes{IndexPath: new(ptr.Deref(indexPath, defaultPypiIndexPath))}
}

// xet converts the Xet configuration of a Hugging Face proxy repository, nil
// when the spec declares none.
func (d desiredRepo) xet() *v395.HuggingFaceXetAttributes {
	spec := d.params().Xet
	if spec == nil {
		return nil
	}

	return &v395.HuggingFaceXetAttributes{
		XetCasUrl:                spec.XetCasURL,
		XetTransferHostAllowList: spec.XetTransferHostAllowList,
	}
}

// requireAuthentication converts whether a repository overrides anonymous
// access. It is always sent, false when the spec leaves it unset, so that
// removing the setting from the spec turns it off in Nexus too.
func requireAuthentication(spec *repositoryv1alpha1.RequireAuthenticationConfig) *bool {
	if spec == nil {
		return new(false)
	}

	return new(ptr.Deref(spec.RequireAuthentication, false))
}

// cargo converts the Cargo specific configuration.
func (d desiredRepo) cargo() *v395.CargoAttributes {
	return &v395.CargoAttributes{RequireAuthentication: requireAuthentication(d.params().Cargo)}
}

// swift converts the Swift specific configuration of a proxy repository.
func (d desiredRepo) swift() *v395.SwiftAttributes {
	return &v395.SwiftAttributes{RequireAuthentication: requireAuthentication(d.params().Swift)}
}

// terraform converts the Terraform specific configuration of a group
// repository.
func (d desiredRepo) terraform() *v395.TerraformAttributes {
	return &v395.TerraformAttributes{RequireAuthentication: requireAuthentication(d.params().Terraform)}
}

// terraformSigning converts the PGP key of a Terraform hosted repository.
// Nexus rejects a hosted repository without one, which the CRD enforces too.
func (d desiredRepo) terraformSigning() v395.TerraformSigningAttributes {
	spec := d.params().TerraformSigning
	if spec == nil {
		return v395.TerraformSigningAttributes{}
	}

	return v395.TerraformSigningAttributes{Keypair: spec.Keypair, Passphrase: spec.Passphrase}
}

// alpineSigning converts the RSA key of an alpine repository, nil when the
// spec gives none. Nexus requires one for proxy and group repositories, which
// the CRD enforces too.
func (d desiredRepo) alpineSigning() *v395.AlpineSigningRepositoriesAttributes {
	spec := d.params().AlpineSigning
	if spec == nil {
		return nil
	}

	return &v395.AlpineSigningRepositoriesAttributes{Keypair: spec.Keypair, Passphrase: spec.Passphrase}
}

// oci converts the OCI specific configuration, which shares the settings and
// defaults of the Docker block.
func (d desiredRepo) oci() v395.OciAttributes {
	return v395.OciAttributes(containerAttributes(d.params().Oci))
}

// ociProxy converts the index configuration of an OCI proxy repository. It is
// always sent, with the REGISTRY index Nexus defaults OCI proxies to when the
// spec sets none, so that removing the block resets a custom index and drift
// detection notices one set outside of the provider.
func (d desiredRepo) ociProxy() *v395.OciProxyAttributes {
	attributes := containerProxyAttributes(d.params().OciProxy, defaultOciIndexType)

	return &v395.OciProxyAttributes{
		IndexType:                &attributes.IndexType,
		IndexUrl:                 attributes.IndexUrl,
		CacheForeignLayers:       attributes.CacheForeignLayers,
		ForeignLayerUrlWhitelist: attributes.ForeignLayerUrlWhitelist,
	}
}

// cosign converts the cosign signature enforcement of an OCI repository. It is
// always sent, NONE when the spec declares no block, so that removing the
// block turns enforcement off and drift detection notices enforcement turned
// on outside of the provider. The CRD only accepts the certificate regexes
// with KEYLESS enforcement, since Nexus drops them otherwise.
func (d desiredRepo) cosign() *v395.OciCosignConfiguration {
	spec := d.params().Cosign
	if spec == nil {
		return &v395.OciCosignConfiguration{Enforcement: cosignEnforcementNone}
	}

	return &v395.OciCosignConfiguration{
		Enforcement:   spec.Enforcement,
		IdentityRegex: spec.IdentityRegex,
		IssuerRegex:   spec.IssuerRegex,
	}
}
