package repository

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"

	"github.com/crossplane-contrib/provider-sonatype-nexus/apis/v1alpha1"
)

// Maven repository generators

func generateMavenHosted(cr *v1alpha1.Repository) repository.MavenHostedRepository {
	repo := repository.MavenHostedRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateHostedStorage(cr)
	repo.Maven = generateMavenConfig(cr)

	if cr.Spec.ForProvider.Cleanup != nil && len(cr.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		repo.Cleanup = &repository.Cleanup{
			PolicyNames: cr.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}

	return repo
}

func generateMavenProxy(cr *v1alpha1.Repository) repository.MavenProxyRepository {
	repo := repository.MavenProxyRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Maven = generateMavenConfig(cr)
	repo.Proxy = generateProxyConfig(cr)
	repo.NegativeCache = generateNegativeCache(cr)
	repo.HTTPClient = generateHTTPClientWithPreemptiveAuth(cr)

	return repo
}

func generateMavenGroup(cr *v1alpha1.Repository) repository.MavenGroupRepository {
	repo := repository.MavenGroupRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Group = generateGroupConfig(cr)

	return repo
}

// Docker repository generators

func generateDockerHosted(cr *v1alpha1.Repository) repository.DockerHostedRepository {
	repo := repository.DockerHostedRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateDockerHostedStorage(cr)
	repo.Docker = generateDockerConfig(cr)

	if cr.Spec.ForProvider.Cleanup != nil && len(cr.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		repo.Cleanup = &repository.Cleanup{
			PolicyNames: cr.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}

	return repo
}

func generateDockerProxy(cr *v1alpha1.Repository) repository.DockerProxyRepository {
	repo := repository.DockerProxyRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Docker = generateDockerConfig(cr)
	repo.Proxy = generateProxyConfig(cr)
	repo.NegativeCache = generateNegativeCache(cr)
	repo.HTTPClient = generateHTTPClient(cr)
	repo.DockerProxy = repository.DockerProxy{
		IndexType: repository.DockerProxyIndexTypeHub,
	}

	return repo
}

func generateDockerGroup(cr *v1alpha1.Repository) repository.DockerGroupRepository {
	repo := repository.DockerGroupRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Docker = generateDockerConfig(cr)
	repo.Group = generateGroupDeployConfig(cr)

	return repo
}

// npm repository generators

func generateNpmHosted(cr *v1alpha1.Repository) repository.NpmHostedRepository {
	repo := repository.NpmHostedRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateHostedStorage(cr)

	if cr.Spec.ForProvider.Cleanup != nil && len(cr.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		repo.Cleanup = &repository.Cleanup{
			PolicyNames: cr.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}

	return repo
}

func generateNpmProxy(cr *v1alpha1.Repository) repository.NpmProxyRepository {
	repo := repository.NpmProxyRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Proxy = generateProxyConfig(cr)
	repo.NegativeCache = generateNegativeCache(cr)
	repo.HTTPClient = generateHTTPClient(cr)

	return repo
}

func generateNpmGroup(cr *v1alpha1.Repository) repository.NpmGroupRepository {
	repo := repository.NpmGroupRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Group = generateGroupDeployConfig(cr)

	return repo
}

// Raw repository generators

func generateRawHosted(cr *v1alpha1.Repository) repository.RawHostedRepository {
	repo := repository.RawHostedRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateHostedStorage(cr)

	if cr.Spec.ForProvider.Cleanup != nil && len(cr.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		repo.Cleanup = &repository.Cleanup{
			PolicyNames: cr.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}

	return repo
}

func generateRawProxy(cr *v1alpha1.Repository) repository.RawProxyRepository {
	repo := repository.RawProxyRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Proxy = generateProxyConfig(cr)
	repo.NegativeCache = generateNegativeCache(cr)
	repo.HTTPClient = generateHTTPClient(cr)

	return repo
}

func generateRawGroup(cr *v1alpha1.Repository) repository.RawGroupRepository {
	repo := repository.RawGroupRepository{
		Name:   cr.Spec.ForProvider.Name,
		Online: true,
	}

	if cr.Spec.ForProvider.Online != nil {
		repo.Online = *cr.Spec.ForProvider.Online
	}

	repo.Storage = generateProxyStorage(cr)
	repo.Group = generateGroupConfig(cr)

	return repo
}

// Shared configuration generators

func generateHostedStorage(cr *v1alpha1.Repository) repository.HostedStorage {
	defaultWritePolicy := repository.StorageWritePolicyAllow
	storage := repository.HostedStorage{
		BlobStoreName:               "default",
		StrictContentTypeValidation: true,
		WritePolicy:                 &defaultWritePolicy,
	}

	if cr.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = cr.Spec.ForProvider.Storage.BlobStoreName
		if cr.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *cr.Spec.ForProvider.Storage.StrictContentTypeValidation
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil {
			wp := repository.StorageWritePolicy(*cr.Spec.ForProvider.Storage.WritePolicy)
			storage.WritePolicy = &wp
		}
	}

	return storage
}

func generateDockerHostedStorage(cr *v1alpha1.Repository) repository.DockerHostedStorage {
	storage := repository.DockerHostedStorage{
		BlobStoreName:               "default",
		StrictContentTypeValidation: true,
		WritePolicy:                 repository.StorageWritePolicyAllow,
	}

	if cr.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = cr.Spec.ForProvider.Storage.BlobStoreName
		if cr.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *cr.Spec.ForProvider.Storage.StrictContentTypeValidation
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil {
			storage.WritePolicy = repository.StorageWritePolicy(*cr.Spec.ForProvider.Storage.WritePolicy)
		}
	}

	return storage
}

func generateProxyStorage(cr *v1alpha1.Repository) repository.Storage {
	storage := repository.Storage{
		BlobStoreName:               "default",
		StrictContentTypeValidation: true,
	}

	if cr.Spec.ForProvider.Storage != nil {
		storage.BlobStoreName = cr.Spec.ForProvider.Storage.BlobStoreName
		if cr.Spec.ForProvider.Storage.StrictContentTypeValidation != nil {
			storage.StrictContentTypeValidation = *cr.Spec.ForProvider.Storage.StrictContentTypeValidation
		}
	}

	return storage
}

func generateMavenConfig(cr *v1alpha1.Repository) repository.Maven {
	maven := repository.Maven{
		VersionPolicy: repository.MavenVersionPolicyRelease,
		LayoutPolicy:  repository.MavenLayoutPolicyStrict,
	}

	if cr.Spec.ForProvider.Maven != nil {
		if cr.Spec.ForProvider.Maven.VersionPolicy != nil {
			maven.VersionPolicy = repository.MavenVersionPolicy(*cr.Spec.ForProvider.Maven.VersionPolicy)
		}
		if cr.Spec.ForProvider.Maven.LayoutPolicy != nil {
			maven.LayoutPolicy = repository.MavenLayoutPolicy(*cr.Spec.ForProvider.Maven.LayoutPolicy)
		}
		if cr.Spec.ForProvider.Maven.ContentDisposition != nil {
			cd := repository.MavenContentDisposition(*cr.Spec.ForProvider.Maven.ContentDisposition)
			maven.ContentDisposition = &cd
		}
	}

	return maven
}

func generateDockerConfig(cr *v1alpha1.Repository) repository.Docker {
	docker := repository.Docker{
		V1Enabled:      false,
		ForceBasicAuth: true,
	}

	if cr.Spec.ForProvider.Docker != nil {
		if cr.Spec.ForProvider.Docker.V1Enabled != nil {
			docker.V1Enabled = *cr.Spec.ForProvider.Docker.V1Enabled
		}
		if cr.Spec.ForProvider.Docker.ForceBasicAuth != nil {
			docker.ForceBasicAuth = *cr.Spec.ForProvider.Docker.ForceBasicAuth
		}
		if cr.Spec.ForProvider.Docker.HTTPPort != nil {
			port := int(*cr.Spec.ForProvider.Docker.HTTPPort)
			docker.HTTPPort = &port
		}
		if cr.Spec.ForProvider.Docker.HTTPSPort != nil {
			port := int(*cr.Spec.ForProvider.Docker.HTTPSPort)
			docker.HTTPSPort = &port
		}
		if cr.Spec.ForProvider.Docker.Subdomain != nil {
			docker.Subdomain = cr.Spec.ForProvider.Docker.Subdomain
		}
	}

	return docker
}

func generateProxyConfig(cr *v1alpha1.Repository) repository.Proxy {
	proxy := repository.Proxy{
		ContentMaxAge:  1440,
		MetadataMaxAge: 1440,
	}

	if cr.Spec.ForProvider.Proxy != nil {
		proxy.RemoteURL = cr.Spec.ForProvider.Proxy.RemoteURL
		if cr.Spec.ForProvider.Proxy.ContentMaxAge != nil {
			proxy.ContentMaxAge = int(*cr.Spec.ForProvider.Proxy.ContentMaxAge)
		}
		if cr.Spec.ForProvider.Proxy.MetadataMaxAge != nil {
			proxy.MetadataMaxAge = int(*cr.Spec.ForProvider.Proxy.MetadataMaxAge)
		}
	}

	return proxy
}

func generateNegativeCache(cr *v1alpha1.Repository) repository.NegativeCache {
	nc := repository.NegativeCache{
		Enabled: true,
		TTL:     1440,
	}

	if cr.Spec.ForProvider.NegativeCache != nil {
		if cr.Spec.ForProvider.NegativeCache.Enabled != nil {
			nc.Enabled = *cr.Spec.ForProvider.NegativeCache.Enabled
		}
		if cr.Spec.ForProvider.NegativeCache.TimeToLive != nil {
			nc.TTL = int(*cr.Spec.ForProvider.NegativeCache.TimeToLive)
		}
	}

	return nc
}

func generateHTTPClient(cr *v1alpha1.Repository) repository.HTTPClient {
	hc := repository.HTTPClient{
		Blocked:   false,
		AutoBlock: true,
	}

	if cr.Spec.ForProvider.HTTPClient != nil {
		if cr.Spec.ForProvider.HTTPClient.Blocked != nil {
			hc.Blocked = *cr.Spec.ForProvider.HTTPClient.Blocked
		}
		if cr.Spec.ForProvider.HTTPClient.AutoBlock != nil {
			hc.AutoBlock = *cr.Spec.ForProvider.HTTPClient.AutoBlock
		}
	}

	return hc
}

func generateHTTPClientWithPreemptiveAuth(cr *v1alpha1.Repository) repository.HTTPClientWithPreemptiveAuth {
	hc := repository.HTTPClientWithPreemptiveAuth{
		Blocked:   false,
		AutoBlock: true,
	}

	if cr.Spec.ForProvider.HTTPClient != nil {
		if cr.Spec.ForProvider.HTTPClient.Blocked != nil {
			hc.Blocked = *cr.Spec.ForProvider.HTTPClient.Blocked
		}
		if cr.Spec.ForProvider.HTTPClient.AutoBlock != nil {
			hc.AutoBlock = *cr.Spec.ForProvider.HTTPClient.AutoBlock
		}
	}

	return hc
}

func generateGroupConfig(cr *v1alpha1.Repository) repository.Group {
	group := repository.Group{}

	if cr.Spec.ForProvider.Group != nil {
		group.MemberNames = cr.Spec.ForProvider.Group.MemberNames
	}

	return group
}

func generateGroupDeployConfig(cr *v1alpha1.Repository) repository.GroupDeploy {
	group := repository.GroupDeploy{}

	if cr.Spec.ForProvider.Group != nil {
		group.MemberNames = cr.Spec.ForProvider.Group.MemberNames
		if cr.Spec.ForProvider.Group.WritableMember != nil {
			group.WritableMember = cr.Spec.ForProvider.Group.WritableMember
		}
	}

	return group
}
