package repository

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"k8s.io/utils/ptr"

	repositoryv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/content/v1alpha1"
)

// Shared configuration generators used by all format handlers.

const (
	// defaultBlobStoreName is the default blob store name.
	defaultBlobStoreName = "default"
	// defaultContentMaxAge is the default content max age in minutes.
	defaultContentMaxAge = 1440
	// defaultMetadataMaxAge is the default metadata max age in minutes.
	defaultMetadataMaxAge = 1440
	// defaultNegativeCacheTTL is the default negative cache TTL in minutes.
	defaultNegativeCacheTTL = 1440
	// repoTypeHosted is the repository type name for hosted repositories.
	repoTypeHosted = "hosted"
	// repoTypeProxy is the repository type name for proxy repositories.
	repoTypeProxy = "proxy"
	// repoTypeGroup is the repository type name for group repositories.
	repoTypeGroup = "group"
)

// observeRepo is a generic helper for observing a repository using a typed
// getter function and an up-to-date checker.
func observeRepo[T any](
	name string,
	getter func(name string) (*T, error),
	checker func(repoCR *repositoryv1alpha1.Repository, repo *T) bool,
	repoCR *repositoryv1alpha1.Repository,
) (exists, upToDate bool) {
	repo, err := getter(name)
	if err != nil || repo == nil {
		return false, false
	}

	return true, checker(repoCR, repo)
}

// getOnline returns the online status, defaulting to true if not specified.
func getOnline(repo *repositoryv1alpha1.Repository) bool {
	return ptr.Deref(repo.Spec.ForProvider.Online, true)
}

// generateCleanup converts cleanup policy configuration.
func generateCleanup(repo *repositoryv1alpha1.Repository) *repository.Cleanup {
	if repo.Spec.ForProvider.Cleanup != nil && len(repo.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		return &repository.Cleanup{
			PolicyNames: repo.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}

	return nil
}

// generateHostedStorage converts storage configuration for hosted
// repositories.
func generateHostedStorage(repo *repositoryv1alpha1.Repository) repository.HostedStorage {
	defaultWritePolicy := repository.StorageWritePolicyAllow
	storage := repository.HostedStorage{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
		WritePolicy:                 &defaultWritePolicy,
	}

	if repo.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = repo.Spec.ForProvider.Storage.BlobStoreName
		if repo.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *repo.Spec.ForProvider.Storage.StrictContentTypeValidation
		}

		if repo.Spec.ForProvider.Storage.WritePolicy != nil {
			wp := repository.StorageWritePolicy(*repo.Spec.ForProvider.Storage.WritePolicy)
			storage.WritePolicy = &wp
		}
	}

	return storage
}

// generateDockerHostedStorage converts storage configuration for Docker
// hosted repositories.
func generateDockerHostedStorage(repo *repositoryv1alpha1.Repository) repository.DockerHostedStorage {
	storage := repository.DockerHostedStorage{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
		WritePolicy:                 repository.StorageWritePolicyAllow,
	}

	if repo.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = repo.Spec.ForProvider.Storage.BlobStoreName
		if repo.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *repo.Spec.ForProvider.Storage.StrictContentTypeValidation
		}

		if repo.Spec.ForProvider.Storage.WritePolicy != nil {
			storage.WritePolicy = repository.StorageWritePolicy(*repo.Spec.ForProvider.Storage.WritePolicy)
		}
	}

	return storage
}

// generateProxyStorage converts storage configuration for proxy and group
// repositories.
func generateProxyStorage(repo *repositoryv1alpha1.Repository) repository.Storage {
	storage := repository.Storage{
		BlobStoreName:               defaultBlobStoreName,
		StrictContentTypeValidation: true,
	}

	if repo.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = repo.Spec.ForProvider.Storage.BlobStoreName
		if repo.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *repo.Spec.ForProvider.Storage.StrictContentTypeValidation
		}
	}

	return storage
}

// generateProxyConfig converts proxy configuration.
func generateProxyConfig(repo *repositoryv1alpha1.Repository) repository.Proxy {
	proxy := repository.Proxy{
		ContentMaxAge:  defaultContentMaxAge,
		MetadataMaxAge: defaultMetadataMaxAge,
	}

	if repo.Spec.ForProvider.Proxy != nil {
		proxy.RemoteURL = repo.Spec.ForProvider.Proxy.RemoteURL
		if repo.Spec.ForProvider.Proxy.ContentMaxAge != nil {
			proxy.ContentMaxAge = int(*repo.Spec.ForProvider.Proxy.ContentMaxAge)
		}

		if repo.Spec.ForProvider.Proxy.MetadataMaxAge != nil {
			proxy.MetadataMaxAge = int(*repo.Spec.ForProvider.Proxy.MetadataMaxAge)
		}
	}

	return proxy
}

// generateNegativeCache converts negative cache configuration.
func generateNegativeCache(repo *repositoryv1alpha1.Repository) repository.NegativeCache {
	negCache := repository.NegativeCache{
		Enabled: true,
		TTL:     defaultNegativeCacheTTL,
	}

	if repo.Spec.ForProvider.NegativeCache != nil {
		if repo.Spec.ForProvider.NegativeCache.Enabled != nil {
			negCache.Enabled = *repo.Spec.ForProvider.NegativeCache.Enabled
		}

		if repo.Spec.ForProvider.NegativeCache.TimeToLive != nil {
			negCache.TTL = int(*repo.Spec.ForProvider.NegativeCache.TimeToLive)
		}
	}

	return negCache
}

// httpClientBaseFields holds the common fields for any HTTPClient
// configuration.
type httpClientBaseFields struct {
	blocked   bool
	autoBlock bool
	conn      *repository.HTTPClientConnection
}

// extractHTTPClientBase extracts common HTTPClient fields from the CR spec.
func extractHTTPClientBase(repo *repositoryv1alpha1.Repository) httpClientBaseFields {
	base := httpClientBaseFields{blocked: false, autoBlock: true}

	if repo.Spec.ForProvider.HTTPClient != nil {
		if repo.Spec.ForProvider.HTTPClient.Blocked != nil {
			base.blocked = *repo.Spec.ForProvider.HTTPClient.Blocked
		}

		if repo.Spec.ForProvider.HTTPClient.AutoBlock != nil {
			base.autoBlock = *repo.Spec.ForProvider.HTTPClient.AutoBlock
		}

		if repo.Spec.ForProvider.HTTPClient.Connection != nil {
			base.conn = generateHTTPClientConnection(repo.Spec.ForProvider.HTTPClient.Connection)
		}
	}

	return base
}

// generateHTTPClient converts HTTP client configuration.
func generateHTTPClient(ctx context.Context, repo *repositoryv1alpha1.Repository) repository.HTTPClient {
	base := extractHTTPClientBase(repo)
	httpClient := repository.HTTPClient{
		Blocked:    base.blocked,
		AutoBlock:  base.autoBlock,
		Connection: base.conn,
	}

	if repo.Spec.ForProvider.HTTPClient != nil && repo.Spec.ForProvider.HTTPClient.Authentication != nil {
		httpClient.Authentication = generateHTTPClientAuth(ctx, repo.Spec.ForProvider.HTTPClient.Authentication)
	}

	return httpClient
}

// generateHTTPClientWithPreemptiveAuth converts HTTP client configuration
// with preemptive auth.
func generateHTTPClientWithPreemptiveAuth(ctx context.Context, repo *repositoryv1alpha1.Repository) repository.HTTPClientWithPreemptiveAuth {
	base := extractHTTPClientBase(repo)
	httpClient := repository.HTTPClientWithPreemptiveAuth{
		Blocked:    base.blocked,
		AutoBlock:  base.autoBlock,
		Connection: base.conn,
	}

	if repo.Spec.ForProvider.HTTPClient != nil && repo.Spec.ForProvider.HTTPClient.Authentication != nil {
		httpClient.Authentication = generateHTTPClientAuthWithPreemptive(ctx, repo.Spec.ForProvider.HTTPClient.Authentication)
	}

	return httpClient
}

// generateHTTPClientConnection converts connection configuration.
func generateHTTPClientConnection(conn *repositoryv1alpha1.HTTPClientConnection) *repository.HTTPClientConnection {
	repoConn := &repository.HTTPClientConnection{}

	if conn.Retries != nil {
		retries := int(*conn.Retries)
		repoConn.Retries = &retries
	}

	if conn.UserAgentSuffix != nil {
		repoConn.UserAgentSuffix = *conn.UserAgentSuffix
	}

	if conn.Timeout != nil {
		timeout := int(*conn.Timeout)
		repoConn.Timeout = &timeout
	}

	repoConn.EnableCircularRedirects = conn.EnableCircularRedirects
	repoConn.EnableCookies = conn.EnableCookies
	repoConn.UseTrustStore = conn.UseTrustStore

	return repoConn
}

// authBaseFields holds common HTTP authentication fields.
type authBaseFields struct {
	authType   repository.HTTPClientAuthenticationType
	username   string
	ntlmHost   string
	ntlmDomain string
	password   string
}

// extractAuthBase extracts common authentication fields from the spec.
func extractAuthBase(ctx context.Context, auth *repositoryv1alpha1.HTTPClientAuthentication) authBaseFields {
	fields := authBaseFields{password: getResolvedPassword(ctx)}
	if auth.Type != nil {
		fields.authType = repository.HTTPClientAuthenticationType(*auth.Type)
	}

	if auth.Username != nil {
		fields.username = *auth.Username
	}

	if auth.NTLMHost != nil {
		fields.ntlmHost = *auth.NTLMHost
	}

	if auth.NTLMDomain != nil {
		fields.ntlmDomain = *auth.NTLMDomain
	}

	return fields
}

// generateHTTPClientAuth converts authentication configuration.
func generateHTTPClientAuth(ctx context.Context, auth *repositoryv1alpha1.HTTPClientAuthentication) *repository.HTTPClientAuthentication {
	fields := extractAuthBase(ctx, auth)

	return &repository.HTTPClientAuthentication{
		Type:       fields.authType,
		Username:   fields.username,
		NTLMHost:   fields.ntlmHost,
		NTLMDomain: fields.ntlmDomain,
		Password:   fields.password,
	}
}

// generateHTTPClientAuthWithPreemptive converts authentication configuration
// with preemptive support.
func generateHTTPClientAuthWithPreemptive(ctx context.Context, auth *repositoryv1alpha1.HTTPClientAuthentication) *repository.HTTPClientAuthenticationWithPreemptive {
	fields := extractAuthBase(ctx, auth)

	return &repository.HTTPClientAuthenticationWithPreemptive{
		Type:       fields.authType,
		Username:   fields.username,
		NTLMHost:   fields.ntlmHost,
		NTLMDomain: fields.ntlmDomain,
		Password:   fields.password,
	}
}

// generateGroupConfig converts group configuration.
func generateGroupConfig(repo *repositoryv1alpha1.Repository) repository.Group {
	group := repository.Group{}

	if repo.Spec.ForProvider.Group != nil {
		group.MemberNames = repo.Spec.ForProvider.Group.MemberNames
	}

	return group
}

// generateGroupDeployConfig converts group configuration with writable
// member support.
func generateGroupDeployConfig(repo *repositoryv1alpha1.Repository) repository.GroupDeploy {
	group := repository.GroupDeploy{}

	if repo.Spec.ForProvider.Group != nil {
		group.MemberNames = repo.Spec.ForProvider.Group.MemberNames
		group.WritableMember = repo.Spec.ForProvider.Group.WritableMember
	}

	return group
}

// generateMavenConfig converts Maven-specific configuration.
func generateMavenConfig(repo *repositoryv1alpha1.Repository) repository.Maven {
	maven := repository.Maven{
		VersionPolicy: repository.MavenVersionPolicyRelease,
		LayoutPolicy:  repository.MavenLayoutPolicyStrict,
	}

	if repo.Spec.ForProvider.Maven != nil {
		if repo.Spec.ForProvider.Maven.VersionPolicy != nil {
			maven.VersionPolicy = repository.MavenVersionPolicy(*repo.Spec.ForProvider.Maven.VersionPolicy)
		}

		if repo.Spec.ForProvider.Maven.LayoutPolicy != nil {
			maven.LayoutPolicy = repository.MavenLayoutPolicy(*repo.Spec.ForProvider.Maven.LayoutPolicy)
		}

		if repo.Spec.ForProvider.Maven.ContentDisposition != nil {
			cd := repository.MavenContentDisposition(*repo.Spec.ForProvider.Maven.ContentDisposition)
			maven.ContentDisposition = &cd
		}
	}

	return maven
}

// isSimpleHostedUpToDate checks common hosted repository fields: online
// status, blob store name, and write policy. Used by formats whose hosted
// repositories share the HostedStorage layout (npm, raw, etc.).
func isSimpleHostedUpToDate(
	repoCR *repositoryv1alpha1.Repository,
	online bool,
	blobStoreName string,
	writePolicy *repository.StorageWritePolicy,
) bool {
	if repoCR.Spec.ForProvider.Online != nil && online != *repoCR.Spec.ForProvider.Online {
		return false
	}

	if repoCR.Spec.ForProvider.Storage != nil {
		if blobStoreName != repoCR.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}

		if repoCR.Spec.ForProvider.Storage.WritePolicy != nil && writePolicy != nil &&
			string(*writePolicy) != *repoCR.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	return true
}

// generateDockerConfig converts Docker-specific configuration.
func generateDockerConfig(repo *repositoryv1alpha1.Repository) repository.Docker {
	docker := repository.Docker{
		V1Enabled:      false,
		ForceBasicAuth: true,
	}

	if repo.Spec.ForProvider.Docker != nil {
		if repo.Spec.ForProvider.Docker.V1Enabled != nil {
			docker.V1Enabled = *repo.Spec.ForProvider.Docker.V1Enabled
		}

		if repo.Spec.ForProvider.Docker.ForceBasicAuth != nil {
			docker.ForceBasicAuth = *repo.Spec.ForProvider.Docker.ForceBasicAuth
		}

		if repo.Spec.ForProvider.Docker.HTTPPort != nil {
			port := int(*repo.Spec.ForProvider.Docker.HTTPPort)
			docker.HTTPPort = &port
		}

		if repo.Spec.ForProvider.Docker.HTTPSPort != nil {
			port := int(*repo.Spec.ForProvider.Docker.HTTPSPort)
			docker.HTTPSPort = &port
		}

		docker.Subdomain = repo.Spec.ForProvider.Docker.Subdomain
	}

	return docker
}
