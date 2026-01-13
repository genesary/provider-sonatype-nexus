package repository

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"

	"github.com/crossplane-contrib/provider-sonatype-nexus/apis/v1alpha1"
)

// Maven comparison functions

func isMavenHostedUpToDate(cr *v1alpha1.Repository, repo *repository.MavenHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil && repo.Storage.WritePolicy != nil &&
			string(*repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	if cr.Spec.ForProvider.Maven != nil {
		if cr.Spec.ForProvider.Maven.VersionPolicy != nil &&
			string(repo.Maven.VersionPolicy) != *cr.Spec.ForProvider.Maven.VersionPolicy {
			return false
		}
		if cr.Spec.ForProvider.Maven.LayoutPolicy != nil &&
			string(repo.Maven.LayoutPolicy) != *cr.Spec.ForProvider.Maven.LayoutPolicy {
			return false
		}
	}

	return true
}

func isMavenProxyUpToDate(cr *v1alpha1.Repository, repo *repository.MavenProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.Proxy.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	if cr.Spec.ForProvider.Maven != nil {
		if cr.Spec.ForProvider.Maven.VersionPolicy != nil &&
			string(repo.Maven.VersionPolicy) != *cr.Spec.ForProvider.Maven.VersionPolicy {
			return false
		}
		if cr.Spec.ForProvider.Maven.LayoutPolicy != nil &&
			string(repo.Maven.LayoutPolicy) != *cr.Spec.ForProvider.Maven.LayoutPolicy {
			return false
		}
	}

	return true
}

func isMavenGroupUpToDate(cr *v1alpha1.Repository, repo *repository.MavenGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !stringSlicesEqual(repo.Group.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}

// Docker comparison functions

func isDockerHostedUpToDate(cr *v1alpha1.Repository, repo *repository.DockerHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil &&
			string(repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	if cr.Spec.ForProvider.Docker != nil {
		if cr.Spec.ForProvider.Docker.ForceBasicAuth != nil &&
			repo.Docker.ForceBasicAuth != *cr.Spec.ForProvider.Docker.ForceBasicAuth {
			return false
		}
		if cr.Spec.ForProvider.Docker.V1Enabled != nil &&
			repo.Docker.V1Enabled != *cr.Spec.ForProvider.Docker.V1Enabled {
			return false
		}
	}

	return true
}

func isDockerProxyUpToDate(cr *v1alpha1.Repository, repo *repository.DockerProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.Proxy.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

func isDockerGroupUpToDate(cr *v1alpha1.Repository, repo *repository.DockerGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !stringSlicesEqual(repo.Group.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}

// npm comparison functions

func isNpmHostedUpToDate(cr *v1alpha1.Repository, repo *repository.NpmHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil && repo.Storage.WritePolicy != nil &&
			string(*repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	return true
}

func isNpmProxyUpToDate(cr *v1alpha1.Repository, repo *repository.NpmProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.Proxy.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

func isNpmGroupUpToDate(cr *v1alpha1.Repository, repo *repository.NpmGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !stringSlicesEqual(repo.Group.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}

// Raw comparison functions

func isRawHostedUpToDate(cr *v1alpha1.Repository, repo *repository.RawHostedRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Storage != nil {
		if repo.Storage.BlobStoreName != cr.Spec.ForProvider.Storage.BlobStoreName {
			return false
		}
		if cr.Spec.ForProvider.Storage.WritePolicy != nil && repo.Storage.WritePolicy != nil &&
			string(*repo.Storage.WritePolicy) != *cr.Spec.ForProvider.Storage.WritePolicy {
			return false
		}
	}

	return true
}

func isRawProxyUpToDate(cr *v1alpha1.Repository, repo *repository.RawProxyRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Proxy != nil {
		if repo.Proxy.RemoteURL != cr.Spec.ForProvider.Proxy.RemoteURL {
			return false
		}
	}

	return true
}

func isRawGroupUpToDate(cr *v1alpha1.Repository, repo *repository.RawGroupRepository) bool {
	if cr.Spec.ForProvider.Online != nil && repo.Online != *cr.Spec.ForProvider.Online {
		return false
	}

	if cr.Spec.ForProvider.Group != nil {
		if !stringSlicesEqual(repo.Group.MemberNames, cr.Spec.ForProvider.Group.MemberNames) {
			return false
		}
	}

	return true
}

// stringSlicesEqual compares two string slices for equality.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
