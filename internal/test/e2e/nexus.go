//go:build e2e

/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	nexussdk "github.com/datadrivers/go-nexus-client/nexus3/schema/capability"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

// FetchUser returns the Nexus user with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchUser(userID string) (*security.User, error) {
	return f.Nexus.Security.User.Get(userID, nil)
}

// FetchRole returns the Nexus role with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchRole(roleID string) (*security.Role, error) {
	return f.Nexus.Security.Role.Get(roleID)
}

// FetchPrivilege returns the Nexus privilege with the given name, or (nil, nil) if absent.
func (f *Framework) FetchPrivilege(name string) (*security.Privilege, error) {
	return f.Nexus.Security.Privilege.Get(name)
}

// FetchAnonymousAccess returns the current anonymous access settings from Nexus.
func (f *Framework) FetchAnonymousAccess() (*security.AnonymousAccessSettings, error) {
	return f.Nexus.Security.Anonymous.Read()
}

// ListActiveRealms returns the list of active realm IDs from Nexus.
func (f *Framework) ListActiveRealms() ([]string, error) {
	return f.Nexus.Security.Realm.ListActive()
}

// ListSSLCertificates returns all certificates in the Nexus truststore.
func (f *Framework) ListSSLCertificates() ([]security.SSLCertificate, error) {
	result, err := f.Nexus.Security.SSL.ListCertificates()
	if err != nil || result == nil {
		return nil, err
	}

	return *result, nil
}

// CleanupPoliciesAvailable returns true if the Nexus instance exposes the
// cleanup-policies REST API (Nexus Repository Pro feature).
func (f *Framework) CleanupPoliciesAvailable() bool {
	_, err := f.Nexus.CleanupPolicy.List()

	return err == nil
}

// FetchCleanupPolicy returns the cleanup policy with the given name, or (nil, nil) if absent.
func (f *Framework) FetchCleanupPolicy(name string) (*cleanuppolicies.CleanupPolicy, error) {
	return f.Nexus.CleanupPolicy.Get(name)
}

// FetchContentSelector returns the content selector with the given name, or (nil, nil) if absent.
func (f *Framework) FetchContentSelector(name string) (*security.ContentSelector, error) {
	return f.Nexus.Security.ContentSelector.Get(name)
}

// FetchBlobStoreFile returns the file-type blobstore with the given name, or (nil, nil) if absent.
func (f *Framework) FetchBlobStoreFile(name string) (*blobstore.File, error) {
	return f.Nexus.BlobStore.File.Get(name)
}

// FetchMavenHostedRepo returns the Maven hosted repository with the given name.
func (f *Framework) FetchMavenHostedRepo(name string) (*repository.MavenHostedRepository, error) {
	return f.Nexus.Repository.Maven.Hosted.Get(name)
}

// FetchMavenProxyRepo returns the Maven proxy repository with the given name.
func (f *Framework) FetchMavenProxyRepo(name string) (*repository.MavenProxyRepository, error) {
	return f.Nexus.Repository.Maven.Proxy.Get(name)
}

// FetchMavenGroupRepo returns the Maven group repository with the given name.
func (f *Framework) FetchMavenGroupRepo(name string) (*repository.MavenGroupRepository, error) {
	return f.Nexus.Repository.Maven.Group.Get(name)
}

// FetchDockerHostedRepo returns the Docker hosted repository with the given name.
func (f *Framework) FetchDockerHostedRepo(name string) (*repository.DockerHostedRepository, error) {
	return f.Nexus.Repository.Docker.Hosted.Get(name)
}

// FetchNpmHostedRepo returns the npm hosted repository with the given name.
func (f *Framework) FetchNpmHostedRepo(name string) (*repository.NpmHostedRepository, error) {
	return f.Nexus.Repository.Npm.Hosted.Get(name)
}

// FetchHelmHostedRepo returns the Helm hosted repository with the given name.
func (f *Framework) FetchHelmHostedRepo(name string) (*repository.HelmHostedRepository, error) {
	return f.Nexus.Repository.Helm.Hosted.Get(name)
}

// FetchHelmProxyRepo returns the Helm proxy repository with the given name.
func (f *Framework) FetchHelmProxyRepo(name string) (*repository.HelmProxyRepository, error) {
	return f.Nexus.Repository.Helm.Proxy.Get(name)
}

// FetchPypiProxyRepo returns the PyPI proxy repository with the given name.
func (f *Framework) FetchPypiProxyRepo(name string) (*repository.PypiProxyRepository, error) {
	return f.Nexus.Repository.Pypi.Proxy.Get(name)
}

// FetchCapability returns the capability with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchCapability(id string) (*nexussdk.Capability, error) {
	return f.Nexus.Capability.Get(id)
}
