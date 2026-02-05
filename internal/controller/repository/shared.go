package repository

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"

	"github.com/AYDEV-FR/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/AYDEV-FR/provider-sonatype-nexus/internal/utils"
)

// Shared configuration generators used by all format handlers.

// getOnline returns the online status, defaulting to true if not specified.
func getOnline(cr *v1alpha1.Repository) bool {
	return utils.BoolValueDefault(cr.Spec.ForProvider.Online, true)
}

// generateCleanup converts cleanup policy configuration.
func generateCleanup(cr *v1alpha1.Repository) *repository.Cleanup {
	if cr.Spec.ForProvider.Cleanup != nil && len(cr.Spec.ForProvider.Cleanup.PolicyNames) > 0 {
		return &repository.Cleanup{
			PolicyNames: cr.Spec.ForProvider.Cleanup.PolicyNames,
		}
	}
	return nil
}

// generateHostedStorage converts storage configuration for hosted repositories.
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

// generateDockerHostedStorage converts storage configuration for Docker hosted repositories.
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

// generateProxyStorage converts storage configuration for proxy/group repositories.
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

// generateProxyConfig converts proxy configuration.
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

// generateNegativeCache converts negative cache configuration.
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

// generateHTTPClient converts HTTP client configuration.
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

// generateHTTPClientWithPreemptiveAuth converts HTTP client configuration with preemptive auth.
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

// generateGroupConfig converts group configuration.
func generateGroupConfig(cr *v1alpha1.Repository) repository.Group {
	group := repository.Group{}

	if cr.Spec.ForProvider.Group != nil {
		group.MemberNames = cr.Spec.ForProvider.Group.MemberNames
	}

	return group
}

// generateGroupDeployConfig converts group configuration with writable member support.
func generateGroupDeployConfig(cr *v1alpha1.Repository) repository.GroupDeploy {
	group := repository.GroupDeploy{}

	if cr.Spec.ForProvider.Group != nil {
		group.MemberNames = cr.Spec.ForProvider.Group.MemberNames
		group.WritableMember = cr.Spec.ForProvider.Group.WritableMember
	}

	return group
}

// generateMavenConfig converts Maven-specific configuration.
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

// generateDockerConfig converts Docker-specific configuration.
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
		docker.Subdomain = cr.Spec.ForProvider.Docker.Subdomain
	}

	return docker
}
