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
	"context"
	"net/http"

	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"

	nexusclient "github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// absentAsNil turns the outcome of a generated client read into the value it
// read, (nil, nil) when Nexus answered 404, or the error otherwise.
func absentAsNil[T any](value *T, resp *http.Response, err error) (*T, error) {
	return ignoreAbsent(value, nexusclient.APIError(resp, err))
}

// ignoreAbsent returns (nil, nil) when err stems from Nexus not having the
// resource, (nil, err) on any other error, and (value, nil) otherwise.
func ignoreAbsent[T any](value *T, err error) (*T, error) {
	if nexusclient.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return value, nil
}

// available turns the outcome of a generated client read probing an edition
// specific endpoint into whether the endpoint exists: true on success, false
// when Nexus answered 404 (the endpoint is missing from this edition), and an
// error otherwise. Any other failure, such as a 401, a network error or a
// response that does not decode, is returned so that the caller fails rather
// than skipping a test that would have caught it.
func available[T any](_ T, resp *http.Response, err error) (bool, error) {
	err = nexusclient.APIError(resp, err)
	if nexusclient.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}

// FetchUser returns the Nexus user with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchUser(userID string) (*v395.ApiUser, error) {
	return ignoreAbsent(f.users.Get(context.Background(), userID))
}

// FetchRole returns the Nexus role with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchRole(roleID string) (*v395.RoleXOResponse, error) {
	return absentAsNil(f.Nexus.SecurityManagementRolesAPI.GetSecurityRoles(context.Background(), roleID).Execute())
}

// FetchPrivilege returns the Nexus privilege with the given name, or (nil, nil) if absent.
func (f *Framework) FetchPrivilege(name string) (*v395.ApiPrivilegeRequest, error) {
	return absentAsNil(f.Nexus.SecurityManagementPrivilegesAPI.GetSecurityPrivileges(context.Background(), name).Execute())
}

// FetchAnonymousAccess returns the current anonymous access settings from Nexus.
func (f *Framework) FetchAnonymousAccess() (*v395.AnonymousAccessSettingsXO, error) {
	settings, resp, err := f.Nexus.SecurityManagementAnonymousAccessAPI.ListSecurityAnonymous(context.Background()).Execute()

	return settings, nexusclient.APIError(resp, err)
}

// ListActiveRealms returns the list of active realm IDs from Nexus.
func (f *Framework) ListActiveRealms() ([]string, error) {
	realms, resp, err := f.Nexus.SecurityManagementRealmsAPI.ListSecurityRealmsActive(context.Background()).Execute()

	return realms, nexusclient.APIError(resp, err)
}

// ListSSLCertificates returns all certificates in the Nexus truststore.
func (f *Framework) ListSSLCertificates() ([]v395.ApiCertificate, error) {
	certs, resp, err := f.Nexus.SecurityCertificatesAPI.ListSecuritySslTruststore(context.Background()).Execute()

	return certs, nexusclient.APIError(resp, err)
}

// FetchSSRFProtection returns the Nexus SSRF protection settings.
func (f *Framework) FetchSSRFProtection() (*v395.SsrfProtectionConfigurationXO, error) {
	config, resp, err := f.Nexus.SecurityManagementSSRFProtectionAPI.ListSecuritySsrfProtection(context.Background()).Execute()

	return config, nexusclient.APIError(resp, err)
}

// SetSSRFProtection replaces the Nexus SSRF protection settings directly,
// bypassing the provider.
func (f *Framework) SetSSRFProtection(config v395.SsrfProtectionConfigurationXO) error {
	_, resp, err := f.Nexus.SecurityManagementSSRFProtectionAPI.UpdateSecuritySsrfProtection(context.Background()).SsrfProtectionConfigurationXO(config).Execute()

	return nexusclient.APIError(resp, err)
}

// HTTPSettingsAvailable reports whether the Nexus instance exposes the HTTP
// system settings REST API (Nexus Repository Pro feature). See available.
func (f *Framework) HTTPSettingsAvailable() (bool, error) {
	return available(f.Nexus.ManageSonatypeHTTPSystemSettingsAPI.ListHttp(context.Background()).Execute())
}

// FetchHTTPSettings returns the Nexus HTTP system settings.
func (f *Framework) FetchHTTPSettings() (*v395.HttpSettingsXo, error) {
	settings, resp, err := f.Nexus.ManageSonatypeHTTPSystemSettingsAPI.ListHttp(context.Background()).Execute()

	return settings, nexusclient.APIError(resp, err)
}

// CleanupPoliciesAvailable reports whether the Nexus instance exposes the
// cleanup-policies REST API (Nexus Repository Pro feature). See available.
func (f *Framework) CleanupPoliciesAvailable() (bool, error) {
	return available(f.Nexus.CleanupPoliciesAPI.ListCleanupPolicies(context.Background()).Execute())
}

// FetchCleanupPolicy returns the cleanup policy with the given name, or (nil, nil) if absent.
func (f *Framework) FetchCleanupPolicy(name string) (*v395.CleanupPolicyResourceXO, error) {
	return absentAsNil(f.Nexus.CleanupPoliciesAPI.GetCleanupPolicies(context.Background(), name).Execute())
}

// FetchContentSelector returns the content selector with the given name, or (nil, nil) if absent.
func (f *Framework) FetchContentSelector(name string) (*v395.ContentSelectorApiResponse, error) {
	return absentAsNil(f.Nexus.ContentSelectorsAPI.GetSecurityContentSelectors(context.Background(), name).Execute())
}

// FetchBlobStoreFile returns the file-type blobstore with the given name, or (nil, nil) if absent.
func (f *Framework) FetchBlobStoreFile(name string) (*v395.FileBlobStoreApiModel, error) {
	return absentAsNil(f.Nexus.BlobStoreAPI.GetBlobstoresFile(context.Background(), name).Execute())
}

// FetchMavenHostedRepo returns the Maven hosted repository with the given name.
func (f *Framework) FetchMavenHostedRepo(name string) (*v395.MavenHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetMavenHostedRepository(context.Background(), name).Execute())
}

// FetchMavenProxyRepo returns the Maven proxy repository with the given name.
func (f *Framework) FetchMavenProxyRepo(name string) (*v395.MavenProxyApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetMavenProxyRepository(context.Background(), name).Execute())
}

// FetchMavenGroupRepo returns the Maven group repository with the given name.
func (f *Framework) FetchMavenGroupRepo(name string) (*v395.MavenGroupApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetMavenGroupRepository(context.Background(), name).Execute())
}

// FetchDockerHostedRepo returns the Docker hosted repository with the given name.
func (f *Framework) FetchDockerHostedRepo(name string) (*v395.DockerHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetDockerHostedRepository(context.Background(), name).Execute())
}

// FetchNpmHostedRepo returns the npm hosted repository with the given name.
func (f *Framework) FetchNpmHostedRepo(name string) (*v395.NpmHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetNpmHostedRepository(context.Background(), name).Execute())
}

// FetchHelmHostedRepo returns the Helm hosted repository with the given name.
func (f *Framework) FetchHelmHostedRepo(name string) (*v395.HelmHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetHelmHostedRepository(context.Background(), name).Execute())
}

// FetchHelmProxyRepo returns the Helm proxy repository with the given name.
func (f *Framework) FetchHelmProxyRepo(name string) (*v395.HelmProxyApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetHelmProxyRepository(context.Background(), name).Execute())
}

// FetchPypiProxyRepo returns the PyPI proxy repository with the given name.
func (f *Framework) FetchPypiProxyRepo(name string) (*v395.PyPiProxyApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetPypiProxyRepository(context.Background(), name).Execute())
}

// FetchAptHostedRepo returns the APT hosted repository with the given name.
func (f *Framework) FetchAptHostedRepo(name string) (*v395.AptHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetAptHostedRepository(context.Background(), name).Execute())
}

// FetchCargoHostedRepo returns the Cargo hosted repository with the given name.
func (f *Framework) FetchCargoHostedRepo(name string) (*v395.CargoHostedApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetCargoHostedRepository(context.Background(), name).Execute())
}

// FetchCargoProxyRepo returns the Cargo proxy repository with the given name.
func (f *Framework) FetchCargoProxyRepo(name string) (*v395.CargoProxyApiRepository, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetCargoProxyRepository(context.Background(), name).Execute())
}

// FetchRepository returns the format agnostic summary of the repository with
// the given name, or (nil, nil) if absent.
func (f *Framework) FetchRepository(name string) (*v395.RepositoryXO, error) {
	return absentAsNil(f.Nexus.RepositoryManagementAPI.GetRepositories(context.Background(), name).Execute())
}

// FetchCapability returns the capability with the given ID, or (nil, nil) if absent.
func (f *Framework) FetchCapability(id string) (*v395.CapabilityDTO, error) {
	return ignoreAbsent(f.capabilities.Get(context.Background(), id))
}

// FetchRoutingRule returns the routing rule with the given name, or (nil, nil) if absent.
func (f *Framework) FetchRoutingRule(name string) (*v395.RoutingRuleXO, error) {
	return absentAsNil(f.Nexus.RoutingRulesAPI.GetRoutingRules(context.Background(), name).Execute())
}

// FetchTask returns the scheduled task with the given name, or (nil, nil) if
// absent.
func (f *Framework) FetchTask(name string) (*v395.TaskXO, error) {
	return ignoreAbsent(f.tasks.GetByName(context.Background(), name, ""))
}

// UpdateTask replaces the scheduled task with the given ID directly,
// bypassing the provider.
func (f *Framework) UpdateTask(id string, task v395.UpdateTasksRequest) error {
	return f.tasks.Update(context.Background(), id, task)
}

// FetchScript returns the script with the given name, or (nil, nil) if absent.
func (f *Framework) FetchScript(name string) (*v395.ScriptXO, error) {
	return absentAsNil(f.Nexus.ScriptAPI.GetScript(context.Background(), name).Execute())
}

// SetScript replaces the script with the given name directly, bypassing the
// provider.
func (f *Framework) SetScript(script v395.ScriptXO) error {
	return nexusclient.APIError(f.Nexus.ScriptAPI.UpdateScript(context.Background(), script.Name).ScriptXO(script).Execute())
}

// FetchEmailConfiguration returns the Nexus email server configuration.
func (f *Framework) FetchEmailConfiguration() (*v395.ApiEmailConfiguration, error) {
	config, resp, err := f.Nexus.EmailAPI.ListEmail(context.Background()).Execute()

	return config, nexusclient.APIError(resp, err)
}

// SetEmailConfiguration replaces the Nexus email server configuration
// directly, bypassing the provider.
func (f *Framework) SetEmailConfiguration(config v395.ApiEmailConfiguration) error {
	return nexusclient.APIError(f.Nexus.EmailAPI.UpdateEmail(context.Background()).ApiEmailConfiguration(config).Execute())
}

// ResetEmailConfiguration resets the Nexus email server configuration
// directly, bypassing the provider.
func (f *Framework) ResetEmailConfiguration() error {
	return nexusclient.APIError(f.Nexus.EmailAPI.DeleteEmail(context.Background()).Execute())
}
