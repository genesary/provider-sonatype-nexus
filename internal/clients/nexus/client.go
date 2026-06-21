// Package nexus provides a client interface for Sonatype Nexus
// Repository Manager.
package nexus

import (
	"context"
	"encoding/json"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/datadrivers/go-nexus-client/nexus3"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/datadrivers/go-nexus-client/nexus3/schema"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// Credentials contains the credentials for connecting to Nexus.
type Credentials struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// SSLService provides methods for managing SSL truststore certificates.
type SSLService interface {
	AddCertificate(ctx context.Context, cert *security.SSLCertificate) error
	RemoveCertificate(ctx context.Context, id string) error
	ListCertificates(ctx context.Context) ([]security.SSLCertificate, error)
}

// CleanupPolicyService provides methods for managing cleanup policies.
type CleanupPolicyService interface {
	GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error)
	ListCleanupPolicies(ctx context.Context) ([]*cleanuppolicies.CleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	DeleteCleanupPolicy(ctx context.Context, name string) error
}

// ScriptService manages Nexus Groovy scripts.
type ScriptService interface {
	GetScript(ctx context.Context, name string) (*schema.Script, error)
	ListScripts(ctx context.Context) ([]schema.Script, error)
	CreateScript(ctx context.Context, script *schema.Script) error
	UpdateScript(ctx context.Context, script *schema.Script) error
	DeleteScript(ctx context.Context, name string) error
}

// Client is an interface for interacting with the Nexus API.
type Client interface {
	BlobStore() BlobStoreService
	CleanupPolicy() CleanupPolicyService
	Repository() RepositoryService
	Script() ScriptService
	Security() SecurityService
	SSL() SSLService
}

// BlobStoreService provides methods for managing blob stores.
type BlobStoreService interface {
	GetFile(ctx context.Context, name string) (*blobstore.File, error)
	GetS3(ctx context.Context, name string) (*blobstore.S3, error)
	CreateFile(ctx context.Context, bs *blobstore.File) error
	CreateS3(ctx context.Context, bs *blobstore.S3) error
	UpdateFile(ctx context.Context, name string, bs *blobstore.File) error
	UpdateS3(ctx context.Context, name string, bs *blobstore.S3) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]blobstore.Generic, error)
}

// RepositoryService provides methods for managing repositories.
type RepositoryService interface {
	// Maven
	GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error)
	CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error
	UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error
	DeleteMavenHosted(ctx context.Context, name string) error

	GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error)
	CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error
	UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error
	DeleteMavenProxy(ctx context.Context, name string) error

	GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error)
	CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error
	UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error
	DeleteMavenGroup(ctx context.Context, name string) error

	// Docker
	GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error)
	CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error
	UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error
	DeleteDockerHosted(ctx context.Context, name string) error

	GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error)
	CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error
	UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error
	DeleteDockerProxy(ctx context.Context, name string) error

	GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error)
	CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error
	UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error
	DeleteDockerGroup(ctx context.Context, name string) error

	// npm
	GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error)
	CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error
	UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error
	DeleteNpmHosted(ctx context.Context, name string) error

	GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error)
	CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error
	UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error
	DeleteNpmProxy(ctx context.Context, name string) error

	GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error)
	CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error
	UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error
	DeleteNpmGroup(ctx context.Context, name string) error

	// Raw
	GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error)
	CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error
	UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error
	DeleteRawHosted(ctx context.Context, name string) error

	GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error)
	CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error
	UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error
	DeleteRawProxy(ctx context.Context, name string) error

	GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error)
	CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error
	UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error
	DeleteRawGroup(ctx context.Context, name string) error

	// APT
	GetAptHosted(ctx context.Context, name string) (*repository.AptHostedRepository, error)
	CreateAptHosted(ctx context.Context, repo repository.AptHostedRepository) error
	UpdateAptHosted(ctx context.Context, name string, repo repository.AptHostedRepository) error
	DeleteAptHosted(ctx context.Context, name string) error

	GetAptProxy(ctx context.Context, name string) (*repository.AptProxyRepository, error)
	CreateAptProxy(ctx context.Context, repo repository.AptProxyRepository) error
	UpdateAptProxy(ctx context.Context, name string, repo repository.AptProxyRepository) error
	DeleteAptProxy(ctx context.Context, name string) error

	// Bower
	GetBowerHosted(ctx context.Context, name string) (*repository.BowerHostedRepository, error)
	CreateBowerHosted(ctx context.Context, repo repository.BowerHostedRepository) error
	UpdateBowerHosted(ctx context.Context, name string, repo repository.BowerHostedRepository) error
	DeleteBowerHosted(ctx context.Context, name string) error

	GetBowerProxy(ctx context.Context, name string) (*repository.BowerProxyRepository, error)
	CreateBowerProxy(ctx context.Context, repo repository.BowerProxyRepository) error
	UpdateBowerProxy(ctx context.Context, name string, repo repository.BowerProxyRepository) error
	DeleteBowerProxy(ctx context.Context, name string) error

	GetBowerGroup(ctx context.Context, name string) (*repository.BowerGroupRepository, error)
	CreateBowerGroup(ctx context.Context, repo repository.BowerGroupRepository) error
	UpdateBowerGroup(ctx context.Context, name string, repo repository.BowerGroupRepository) error
	DeleteBowerGroup(ctx context.Context, name string) error

	// Cargo
	GetCargoHosted(ctx context.Context, name string) (*repository.CargoHostedRepository, error)
	CreateCargoHosted(ctx context.Context, repo repository.CargoHostedRepository) error
	UpdateCargoHosted(ctx context.Context, name string, repo repository.CargoHostedRepository) error
	DeleteCargoHosted(ctx context.Context, name string) error

	GetCargoProxy(ctx context.Context, name string) (*repository.CargoProxyRepository, error)
	CreateCargoProxy(ctx context.Context, repo repository.CargoProxyRepository) error
	UpdateCargoProxy(ctx context.Context, name string, repo repository.CargoProxyRepository) error
	DeleteCargoProxy(ctx context.Context, name string) error

	GetCargoGroup(ctx context.Context, name string) (*repository.CargoGroupRepository, error)
	CreateCargoGroup(ctx context.Context, repo repository.CargoGroupRepository) error
	UpdateCargoGroup(ctx context.Context, name string, repo repository.CargoGroupRepository) error
	DeleteCargoGroup(ctx context.Context, name string) error

	// Cocoapods
	GetCocoapodsProxy(ctx context.Context, name string) (*repository.CocoapodsProxyRepository, error)
	CreateCocoapodsProxy(ctx context.Context, repo repository.CocoapodsProxyRepository) error
	UpdateCocoapodsProxy(ctx context.Context, name string, repo repository.CocoapodsProxyRepository) error
	DeleteCocoapodsProxy(ctx context.Context, name string) error

	// Conan
	GetConanProxy(ctx context.Context, name string) (*repository.ConanProxyRepository, error)
	CreateConanProxy(ctx context.Context, repo repository.ConanProxyRepository) error
	UpdateConanProxy(ctx context.Context, name string, repo repository.ConanProxyRepository) error
	DeleteConanProxy(ctx context.Context, name string) error

	// Conda
	GetCondaProxy(ctx context.Context, name string) (*repository.CondaProxyRepository, error)
	CreateCondaProxy(ctx context.Context, repo repository.CondaProxyRepository) error
	UpdateCondaProxy(ctx context.Context, name string, repo repository.CondaProxyRepository) error
	DeleteCondaProxy(ctx context.Context, name string) error

	// Git LFS
	GetGitLfsHosted(ctx context.Context, name string) (*repository.GitLfsHostedRepository, error)
	CreateGitLfsHosted(ctx context.Context, repo repository.GitLfsHostedRepository) error
	UpdateGitLfsHosted(ctx context.Context, name string, repo repository.GitLfsHostedRepository) error
	DeleteGitLfsHosted(ctx context.Context, name string) error

	// Go
	GetGoProxy(ctx context.Context, name string) (*repository.GoProxyRepository, error)
	CreateGoProxy(ctx context.Context, repo repository.GoProxyRepository) error
	UpdateGoProxy(ctx context.Context, name string, repo repository.GoProxyRepository) error
	DeleteGoProxy(ctx context.Context, name string) error

	GetGoGroup(ctx context.Context, name string) (*repository.GoGroupRepository, error)
	CreateGoGroup(ctx context.Context, repo repository.GoGroupRepository) error
	UpdateGoGroup(ctx context.Context, name string, repo repository.GoGroupRepository) error
	DeleteGoGroup(ctx context.Context, name string) error

	// Helm
	GetHelmHosted(ctx context.Context, name string) (*repository.HelmHostedRepository, error)
	CreateHelmHosted(ctx context.Context, repo repository.HelmHostedRepository) error
	UpdateHelmHosted(ctx context.Context, name string, repo repository.HelmHostedRepository) error
	DeleteHelmHosted(ctx context.Context, name string) error

	GetHelmProxy(ctx context.Context, name string) (*repository.HelmProxyRepository, error)
	CreateHelmProxy(ctx context.Context, repo repository.HelmProxyRepository) error
	UpdateHelmProxy(ctx context.Context, name string, repo repository.HelmProxyRepository) error
	DeleteHelmProxy(ctx context.Context, name string) error

	// NuGet
	GetNugetHosted(ctx context.Context, name string) (*repository.NugetHostedRepository, error)
	CreateNugetHosted(ctx context.Context, repo repository.NugetHostedRepository) error
	UpdateNugetHosted(ctx context.Context, name string, repo repository.NugetHostedRepository) error
	DeleteNugetHosted(ctx context.Context, name string) error

	GetNugetProxy(ctx context.Context, name string) (*repository.NugetProxyRepository, error)
	CreateNugetProxy(ctx context.Context, repo repository.NugetProxyRepository) error
	UpdateNugetProxy(ctx context.Context, name string, repo repository.NugetProxyRepository) error
	DeleteNugetProxy(ctx context.Context, name string) error

	GetNugetGroup(ctx context.Context, name string) (*repository.NugetGroupRepository, error)
	CreateNugetGroup(ctx context.Context, repo repository.NugetGroupRepository) error
	UpdateNugetGroup(ctx context.Context, name string, repo repository.NugetGroupRepository) error
	DeleteNugetGroup(ctx context.Context, name string) error

	// PyPI
	GetPypiHosted(ctx context.Context, name string) (*repository.PypiHostedRepository, error)
	CreatePypiHosted(ctx context.Context, repo repository.PypiHostedRepository) error
	UpdatePypiHosted(ctx context.Context, name string, repo repository.PypiHostedRepository) error
	DeletePypiHosted(ctx context.Context, name string) error

	GetPypiProxy(ctx context.Context, name string) (*repository.PypiProxyRepository, error)
	CreatePypiProxy(ctx context.Context, repo repository.PypiProxyRepository) error
	UpdatePypiProxy(ctx context.Context, name string, repo repository.PypiProxyRepository) error
	DeletePypiProxy(ctx context.Context, name string) error

	GetPypiGroup(ctx context.Context, name string) (*repository.PypiGroupRepository, error)
	CreatePypiGroup(ctx context.Context, repo repository.PypiGroupRepository) error
	UpdatePypiGroup(ctx context.Context, name string, repo repository.PypiGroupRepository) error
	DeletePypiGroup(ctx context.Context, name string) error

	// R
	GetRHosted(ctx context.Context, name string) (*repository.RHostedRepository, error)
	CreateRHosted(ctx context.Context, repo repository.RHostedRepository) error
	UpdateRHosted(ctx context.Context, name string, repo repository.RHostedRepository) error
	DeleteRHosted(ctx context.Context, name string) error

	GetRProxy(ctx context.Context, name string) (*repository.RProxyRepository, error)
	CreateRProxy(ctx context.Context, repo repository.RProxyRepository) error
	UpdateRProxy(ctx context.Context, name string, repo repository.RProxyRepository) error
	DeleteRProxy(ctx context.Context, name string) error

	GetRGroup(ctx context.Context, name string) (*repository.RGroupRepository, error)
	CreateRGroup(ctx context.Context, repo repository.RGroupRepository) error
	UpdateRGroup(ctx context.Context, name string, repo repository.RGroupRepository) error
	DeleteRGroup(ctx context.Context, name string) error

	// RubyGems
	GetRubygemsHosted(ctx context.Context, name string) (*repository.RubyGemsHostedRepository, error)
	CreateRubygemsHosted(ctx context.Context, repo repository.RubyGemsHostedRepository) error
	UpdateRubygemsHosted(ctx context.Context, name string, repo repository.RubyGemsHostedRepository) error
	DeleteRubygemsHosted(ctx context.Context, name string) error

	GetRubygemsProxy(ctx context.Context, name string) (*repository.RubyGemsProxyRepository, error)
	CreateRubygemsProxy(ctx context.Context, repo repository.RubyGemsProxyRepository) error
	UpdateRubygemsProxy(ctx context.Context, name string, repo repository.RubyGemsProxyRepository) error
	DeleteRubygemsProxy(ctx context.Context, name string) error

	GetRubygemsGroup(ctx context.Context, name string) (*repository.RubyGemsGroupRepository, error)
	CreateRubygemsGroup(ctx context.Context, repo repository.RubyGemsGroupRepository) error
	UpdateRubygemsGroup(ctx context.Context, name string, repo repository.RubyGemsGroupRepository) error
	DeleteRubygemsGroup(ctx context.Context, name string) error

	// Yum
	GetYumHosted(ctx context.Context, name string) (*repository.YumHostedRepository, error)
	CreateYumHosted(ctx context.Context, repo repository.YumHostedRepository) error
	UpdateYumHosted(ctx context.Context, name string, repo repository.YumHostedRepository) error
	DeleteYumHosted(ctx context.Context, name string) error

	GetYumProxy(ctx context.Context, name string) (*repository.YumProxyRepository, error)
	CreateYumProxy(ctx context.Context, repo repository.YumProxyRepository) error
	UpdateYumProxy(ctx context.Context, name string, repo repository.YumProxyRepository) error
	DeleteYumProxy(ctx context.Context, name string) error

	GetYumGroup(ctx context.Context, name string) (*repository.YumGroupRepository, error)
	CreateYumGroup(ctx context.Context, repo repository.YumGroupRepository) error
	UpdateYumGroup(ctx context.Context, name string, repo repository.YumGroupRepository) error
	DeleteYumGroup(ctx context.Context, name string) error
}

// SecurityService provides methods for managing security resources.
type SecurityService interface {
	// User management
	GetUser(ctx context.Context, id string) (*security.User, error)
	CreateUser(ctx context.Context, user security.User) error
	UpdateUser(ctx context.Context, id string, user security.User) error
	DeleteUser(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, id, password string) error

	// Role management
	GetRole(ctx context.Context, id string) (*security.Role, error)
	CreateRole(ctx context.Context, role security.Role) error
	UpdateRole(ctx context.Context, id string, role security.Role) error
	DeleteRole(ctx context.Context, id string) error

	// Realm management
	ListAvailableRealms(ctx context.Context) ([]security.Realm, error)
	ListActiveRealms(ctx context.Context) ([]string, error)
	ActivateRealms(ctx context.Context, ids []string) error

	// Content Selector management
	GetContentSelector(ctx context.Context, name string) (*security.ContentSelector, error)
	ListContentSelectors(ctx context.Context) ([]security.ContentSelector, error)
	CreateContentSelector(ctx context.Context, cs security.ContentSelector) error
	UpdateContentSelector(ctx context.Context, name string, cs security.ContentSelector) error
	DeleteContentSelector(ctx context.Context, name string) error

	// Privilege management
	GetPrivilege(ctx context.Context, name string) (*security.Privilege, error)
	ListPrivileges(ctx context.Context) ([]security.Privilege, error)
	DeletePrivilege(ctx context.Context, name string) error
	CreatePrivilegeApplication(ctx context.Context, p security.PrivilegeApplication) error
	UpdatePrivilegeApplication(ctx context.Context, name string, p security.PrivilegeApplication) error
	CreatePrivilegeRepositoryView(ctx context.Context, p security.PrivilegeRepositoryView) error
	UpdatePrivilegeRepositoryView(ctx context.Context, name string, p security.PrivilegeRepositoryView) error
	CreatePrivilegeRepositoryAdmin(ctx context.Context, p security.PrivilegeRepositoryAdmin) error
	UpdatePrivilegeRepositoryAdmin(ctx context.Context, name string, p security.PrivilegeRepositoryAdmin) error
	CreatePrivilegeRepositoryContentSelector(ctx context.Context, p security.PrivilegeRepositoryContentSelector) error
	UpdatePrivilegeRepositoryContentSelector(ctx context.Context, name string, p security.PrivilegeRepositoryContentSelector) error
	CreatePrivilegeScript(ctx context.Context, p security.PrivilegeScript) error
	UpdatePrivilegeScript(ctx context.Context, name string, p security.PrivilegeScript) error
	CreatePrivilegeWildcard(ctx context.Context, p security.PrivilegeWildcard) error
	UpdatePrivilegeWildcard(ctx context.Context, name string, p security.PrivilegeWildcard) error

	// Anonymous access management
	GetAnonymousAccess(ctx context.Context) (*security.AnonymousAccessSettings, error)
	UpdateAnonymousAccess(ctx context.Context, settings security.AnonymousAccessSettings) error

	// SAML management
	GetSAML(ctx context.Context) (*security.SAML, error)
	ApplySAML(ctx context.Context, saml security.SAML) error
	DeleteSAML(ctx context.Context) error

	// LDAP management
	GetLDAP(ctx context.Context, name string) (*security.LDAP, error)
	ListLDAP(ctx context.Context) ([]security.LDAP, error)
	CreateLDAP(ctx context.Context, ldap security.LDAP) error
	UpdateLDAP(ctx context.Context, name string, ldap security.LDAP) error
	DeleteLDAP(ctx context.Context, name string) error

	// User Token management
	GetUserTokenConfiguration(ctx context.Context) (*security.UserTokenConfiguration, error)
	UpdateUserTokenConfiguration(ctx context.Context, config security.UserTokenConfiguration) error
}

// nexusClientWrapper implements the Client interface.
type nexusClientWrapper struct {
	client *nexus3.NexusClient
}

// blobStoreService implements BlobStoreService.
type blobStoreService struct {
	client *nexus3.NexusClient
}

// repositoryService implements RepositoryService.
type repositoryService struct {
	client *nexus3.NexusClient
}

// securityService implements SecurityService.
type securityService struct {
	client *nexus3.NexusClient
}

// sslService implements SSLService.
type sslService struct {
	client *nexus3.NexusClient
}

// scriptService implements ScriptService.
type scriptService struct {
	client *nexus3.NexusClient
}

// cleanupPolicyService implements CleanupPolicyService.
type cleanupPolicyService struct {
	client *nexus3.NexusClient
}

// NewClient creates a new Nexus client from the provided credentials.
func NewClient(creds Credentials) (Client, error) {
	cfg := client.Config{
		URL:      creds.URL,
		Username: creds.Username,
		Password: creds.Password,
		Insecure: creds.Insecure,
	}

	nc := nexus3.NewClient(cfg)
	if nc == nil {
		return nil, errors.New("failed to create Nexus client")
	}

	return &nexusClientWrapper{client: nc}, nil
}

// clusterProviderConfigKind is the Kind name for ClusterProviderConfig.
const clusterProviderConfigKind = "ClusterProviderConfig"

// GetCredentials resolves the ProviderConfig or ClusterProviderConfig
// referenced by mg and extracts Nexus credentials from the referenced secret.
// It dispatches on providerConfigRef.Kind: ClusterProviderConfig is looked up
// cluster-wide; ProviderConfig (or empty) is looked up in mg's namespace.
func GetCredentials(ctx context.Context, kube kubeclient.Client, managed interface {
	GetNamespace() string
	GetProviderConfigReference() *xpv2.ProviderConfigReference
}) (Credentials, error) {
	ref := managed.GetProviderConfigReference()
	if ref == nil {
		return Credentials{}, errors.New("providerConfigRef is not set")
	}

	if ref.Kind == clusterProviderConfigKind {
		cpc := &v1alpha1.ClusterProviderConfig{}

		err := kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc)
		if err != nil {
			return Credentials{}, errors.Wrap(err, "cannot get ClusterProviderConfig")
		}

		return GetCredentialsFromSpec(ctx, kube, cpc.Spec)
	}

	providerConfig := &v1alpha1.ProviderConfig{}

	err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: managed.GetNamespace()}, providerConfig)
	if err != nil {
		return Credentials{}, errors.Wrap(err, "cannot get ProviderConfig")
	}

	return GetCredentialsFromSpec(ctx, kube, providerConfig.Spec)
}

// GetCredentialsFromSpec extracts Nexus credentials from a ProviderConfigSpec.
func GetCredentialsFromSpec(ctx context.Context, kube kubeclient.Client, spec v1alpha1.ProviderConfigSpec) (Credentials, error) {
	var creds Credentials

	if spec.Credentials.Source != "Secret" {
		return creds, errors.New("only Secret source is supported")
	}

	if spec.Credentials.SecretRef == nil {
		return creds, errors.New("secretRef is required when source is Secret")
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{
		Name:      spec.Credentials.SecretRef.Name,
		Namespace: spec.Credentials.SecretRef.Namespace,
	}, secret)
	if err != nil {
		return creds, errors.Wrap(err, "failed to get credentials secret")
	}

	key := "credentials"
	if spec.Credentials.SecretRef.Key != "" {
		key = spec.Credentials.SecretRef.Key
	}

	data, ok := secret.Data[key]
	if !ok {
		return creds, errors.Errorf("secret does not contain key %q", key)
	}

	err = json.Unmarshal(data, &creds)
	if err != nil {
		return creds, errors.Wrap(err, "failed to unmarshal credentials")
	}

	return creds, nil
}

// GetCredentialsFromSecret extracts Nexus credentials from a
// Kubernetes secret.
//
// Deprecated: Use GetCredentials which correctly handles both
// ProviderConfig (namespace-scoped) and ClusterProviderConfig (cluster-scoped).
func GetCredentialsFromSecret(ctx context.Context, kube kubeclient.Client, providerConfig *v1alpha1.ProviderConfig) (Credentials, error) {
	return GetCredentialsFromSpec(ctx, kube, providerConfig.Spec)
}

// GetSecretValue retrieves a value from a Kubernetes secret using a
// SecretKeySelector.
func GetSecretValue(ctx context.Context, kube kubeclient.Client, selector *xpv2.SecretKeySelector) (string, error) {
	if selector == nil {
		return "", errors.New("secretKeySelector is nil")
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: selector.Namespace,
	}, secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	data, ok := secret.Data[selector.Key]
	if !ok {
		return "", errors.Errorf("secret does not contain key %q", selector.Key)
	}

	return string(data), nil
}

// BlobStore returns the BlobStoreService.
func (c *nexusClientWrapper) BlobStore() BlobStoreService {
	return &blobStoreService{client: c.client}
}

// CleanupPolicy returns the CleanupPolicyService.
func (c *nexusClientWrapper) CleanupPolicy() CleanupPolicyService {
	return &cleanupPolicyService{client: c.client}
}

// Repository returns the RepositoryService.
func (c *nexusClientWrapper) Repository() RepositoryService {
	return &repositoryService{client: c.client}
}

// Script returns the ScriptService.
func (c *nexusClientWrapper) Script() ScriptService {
	return &scriptService{client: c.client}
}

// Security returns the SecurityService.
func (c *nexusClientWrapper) Security() SecurityService {
	return &securityService{client: c.client}
}

// SSL returns the SSLService.
func (c *nexusClientWrapper) SSL() SSLService {
	return &sslService{client: c.client}
}

// CleanupPolicyService implementations

// GetCleanupPolicy gets a cleanup policy by name.
func (s *cleanupPolicyService) GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error) {
	return s.client.CleanupPolicy.Get(name)
}

// ListCleanupPolicies lists all cleanup policies.
func (s *cleanupPolicyService) ListCleanupPolicies(ctx context.Context) ([]*cleanuppolicies.CleanupPolicy, error) {
	return s.client.CleanupPolicy.List()
}

// CreateCleanupPolicy creates a new cleanup policy.
func (s *cleanupPolicyService) CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	return s.client.CleanupPolicy.Create(policy)
}

// UpdateCleanupPolicy updates an existing cleanup policy.
func (s *cleanupPolicyService) UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	return s.client.CleanupPolicy.Update(policy)
}

// DeleteCleanupPolicy deletes a cleanup policy by name.
func (s *cleanupPolicyService) DeleteCleanupPolicy(ctx context.Context, name string) error {
	return s.client.CleanupPolicy.Delete(name)
}

// ScriptService implementations

// GetScript gets a script by name.
func (s *scriptService) GetScript(ctx context.Context, name string) (*schema.Script, error) {
	return s.client.Script.Get(name)
}

// ListScripts lists all scripts.
func (s *scriptService) ListScripts(ctx context.Context) ([]schema.Script, error) {
	return s.client.Script.List()
}

// CreateScript creates a new script.
func (s *scriptService) CreateScript(ctx context.Context, script *schema.Script) error {
	return s.client.Script.Create(script)
}

// UpdateScript updates an existing script.
func (s *scriptService) UpdateScript(ctx context.Context, script *schema.Script) error {
	return s.client.Script.Update(script)
}

// DeleteScript deletes a script by name.
func (s *scriptService) DeleteScript(ctx context.Context, name string) error {
	return s.client.Script.Delete(name)
}

// BlobStoreService implementations

// GetFile gets a File blob store by name.
func (s *blobStoreService) GetFile(ctx context.Context, name string) (*blobstore.File, error) {
	return s.client.BlobStore.File.Get(name)
}

// GetS3 gets an S3 blob store by name.
func (s *blobStoreService) GetS3(ctx context.Context, name string) (*blobstore.S3, error) {
	return s.client.BlobStore.S3.Get(name)
}

// CreateFile creates a File blob store.
func (s *blobStoreService) CreateFile(ctx context.Context, bs *blobstore.File) error {
	return s.client.BlobStore.File.Create(bs)
}

// CreateS3 creates an S3 blob store.
func (s *blobStoreService) CreateS3(ctx context.Context, bs *blobstore.S3) error {
	return s.client.BlobStore.S3.Create(bs)
}

// UpdateFile updates a File blob store.
func (s *blobStoreService) UpdateFile(ctx context.Context, name string, bs *blobstore.File) error {
	return s.client.BlobStore.File.Update(name, bs)
}

// UpdateS3 updates an S3 blob store.
func (s *blobStoreService) UpdateS3(ctx context.Context, name string, bs *blobstore.S3) error {
	return s.client.BlobStore.S3.Update(name, bs)
}

// Delete deletes a blob store by name.
func (s *blobStoreService) Delete(ctx context.Context, name string) error {
	return s.client.BlobStore.Delete(name)
}

// List lists all blob stores.
func (s *blobStoreService) List(ctx context.Context) ([]blobstore.Generic, error) {
	return s.client.BlobStore.List()
}

// RepositoryService implementations - Maven

// GetMavenHosted gets a Maven hosted repository.
func (s *repositoryService) GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
	return s.client.Repository.Maven.Hosted.Get(name)
}

// CreateMavenHosted creates a Maven hosted repository.
func (s *repositoryService) CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error {
	return s.client.Repository.Maven.Hosted.Create(repo)
}

// UpdateMavenHosted updates a Maven hosted repository.
func (s *repositoryService) UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
	return s.client.Repository.Maven.Hosted.Update(name, repo)
}

// DeleteMavenHosted deletes a Maven hosted repository.
func (s *repositoryService) DeleteMavenHosted(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Hosted.Delete(name)
}

// GetMavenProxy gets a Maven proxy repository.
func (s *repositoryService) GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error) {
	return s.client.Repository.Maven.Proxy.Get(name)
}

// CreateMavenProxy creates a Maven proxy repository.
func (s *repositoryService) CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error {
	return s.client.Repository.Maven.Proxy.Create(repo)
}

// UpdateMavenProxy updates a Maven proxy repository.
func (s *repositoryService) UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error {
	return s.client.Repository.Maven.Proxy.Update(name, repo)
}

// DeleteMavenProxy deletes a Maven proxy repository.
func (s *repositoryService) DeleteMavenProxy(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Proxy.Delete(name)
}

// GetMavenGroup gets a Maven group repository.
func (s *repositoryService) GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error) {
	return s.client.Repository.Maven.Group.Get(name)
}

// CreateMavenGroup creates a Maven group repository.
func (s *repositoryService) CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error {
	return s.client.Repository.Maven.Group.Create(repo)
}

// UpdateMavenGroup updates a Maven group repository.
func (s *repositoryService) UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error {
	return s.client.Repository.Maven.Group.Update(name, repo)
}

// DeleteMavenGroup deletes a Maven group repository.
func (s *repositoryService) DeleteMavenGroup(ctx context.Context, name string) error {
	return s.client.Repository.Maven.Group.Delete(name)
}

// RepositoryService implementations - Docker

// GetDockerHosted gets a Docker hosted repository.
func (s *repositoryService) GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error) {
	return s.client.Repository.Docker.Hosted.Get(name)
}

// CreateDockerHosted creates a Docker hosted repository.
func (s *repositoryService) CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error {
	return s.client.Repository.Docker.Hosted.Create(repo)
}

// UpdateDockerHosted updates a Docker hosted repository.
func (s *repositoryService) UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error {
	return s.client.Repository.Docker.Hosted.Update(name, repo)
}

// DeleteDockerHosted deletes a Docker hosted repository.
func (s *repositoryService) DeleteDockerHosted(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Hosted.Delete(name)
}

// GetDockerProxy gets a Docker proxy repository.
func (s *repositoryService) GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error) {
	return s.client.Repository.Docker.Proxy.Get(name)
}

// CreateDockerProxy creates a Docker proxy repository.
func (s *repositoryService) CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error {
	return s.client.Repository.Docker.Proxy.Create(repo)
}

// UpdateDockerProxy updates a Docker proxy repository.
func (s *repositoryService) UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error {
	return s.client.Repository.Docker.Proxy.Update(name, repo)
}

// DeleteDockerProxy deletes a Docker proxy repository.
func (s *repositoryService) DeleteDockerProxy(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Proxy.Delete(name)
}

// GetDockerGroup gets a Docker group repository.
func (s *repositoryService) GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error) {
	return s.client.Repository.Docker.Group.Get(name)
}

// CreateDockerGroup creates a Docker group repository.
func (s *repositoryService) CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error {
	return s.client.Repository.Docker.Group.Create(repo)
}

// UpdateDockerGroup updates a Docker group repository.
func (s *repositoryService) UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error {
	return s.client.Repository.Docker.Group.Update(name, repo)
}

// DeleteDockerGroup deletes a Docker group repository.
func (s *repositoryService) DeleteDockerGroup(ctx context.Context, name string) error {
	return s.client.Repository.Docker.Group.Delete(name)
}

// RepositoryService implementations - npm

// GetNpmHosted gets an npm hosted repository.
func (s *repositoryService) GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error) {
	return s.client.Repository.Npm.Hosted.Get(name)
}

// CreateNpmHosted creates an npm hosted repository.
func (s *repositoryService) CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error {
	return s.client.Repository.Npm.Hosted.Create(repo)
}

// UpdateNpmHosted updates an npm hosted repository.
func (s *repositoryService) UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error {
	return s.client.Repository.Npm.Hosted.Update(name, repo)
}

// DeleteNpmHosted deletes an npm hosted repository.
func (s *repositoryService) DeleteNpmHosted(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Hosted.Delete(name)
}

// GetNpmProxy gets an npm proxy repository.
func (s *repositoryService) GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error) {
	return s.client.Repository.Npm.Proxy.Get(name)
}

// CreateNpmProxy creates an npm proxy repository.
func (s *repositoryService) CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error {
	return s.client.Repository.Npm.Proxy.Create(repo)
}

// UpdateNpmProxy updates an npm proxy repository.
func (s *repositoryService) UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error {
	return s.client.Repository.Npm.Proxy.Update(name, repo)
}

// DeleteNpmProxy deletes an npm proxy repository.
func (s *repositoryService) DeleteNpmProxy(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Proxy.Delete(name)
}

// GetNpmGroup gets an npm group repository.
func (s *repositoryService) GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error) {
	return s.client.Repository.Npm.Group.Get(name)
}

// CreateNpmGroup creates an npm group repository.
func (s *repositoryService) CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error {
	return s.client.Repository.Npm.Group.Create(repo)
}

// UpdateNpmGroup updates an npm group repository.
func (s *repositoryService) UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error {
	return s.client.Repository.Npm.Group.Update(name, repo)
}

// DeleteNpmGroup deletes an npm group repository.
func (s *repositoryService) DeleteNpmGroup(ctx context.Context, name string) error {
	return s.client.Repository.Npm.Group.Delete(name)
}

// RepositoryService implementations - Raw

// GetRawHosted gets a Raw hosted repository.
func (s *repositoryService) GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error) {
	return s.client.Repository.Raw.Hosted.Get(name)
}

// CreateRawHosted creates a Raw hosted repository.
func (s *repositoryService) CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error {
	return s.client.Repository.Raw.Hosted.Create(repo)
}

// UpdateRawHosted updates a Raw hosted repository.
func (s *repositoryService) UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error {
	return s.client.Repository.Raw.Hosted.Update(name, repo)
}

// DeleteRawHosted deletes a Raw hosted repository.
func (s *repositoryService) DeleteRawHosted(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Hosted.Delete(name)
}

// GetRawProxy gets a Raw proxy repository.
func (s *repositoryService) GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error) {
	return s.client.Repository.Raw.Proxy.Get(name)
}

// CreateRawProxy creates a Raw proxy repository.
func (s *repositoryService) CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error {
	return s.client.Repository.Raw.Proxy.Create(repo)
}

// UpdateRawProxy updates a Raw proxy repository.
func (s *repositoryService) UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error {
	return s.client.Repository.Raw.Proxy.Update(name, repo)
}

// DeleteRawProxy deletes a Raw proxy repository.
func (s *repositoryService) DeleteRawProxy(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Proxy.Delete(name)
}

// GetRawGroup gets a Raw group repository.
func (s *repositoryService) GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error) {
	return s.client.Repository.Raw.Group.Get(name)
}

// CreateRawGroup creates a Raw group repository.
func (s *repositoryService) CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error {
	return s.client.Repository.Raw.Group.Create(repo)
}

// UpdateRawGroup updates a Raw group repository.
func (s *repositoryService) UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error {
	return s.client.Repository.Raw.Group.Update(name, repo)
}

// DeleteRawGroup deletes a Raw group repository.
func (s *repositoryService) DeleteRawGroup(ctx context.Context, name string) error {
	return s.client.Repository.Raw.Group.Delete(name)
}

// RepositoryService implementations - APT

// GetAptHosted gets an APT hosted repository.
func (s *repositoryService) GetAptHosted(ctx context.Context, name string) (*repository.AptHostedRepository, error) {
	return s.client.Repository.Apt.Hosted.Get(name)
}

// CreateAptHosted creates an APT hosted repository.
func (s *repositoryService) CreateAptHosted(ctx context.Context, repo repository.AptHostedRepository) error {
	return s.client.Repository.Apt.Hosted.Create(repo)
}

// UpdateAptHosted updates an APT hosted repository.
func (s *repositoryService) UpdateAptHosted(ctx context.Context, name string, repo repository.AptHostedRepository) error {
	return s.client.Repository.Apt.Hosted.Update(name, repo)
}

// DeleteAptHosted deletes an APT hosted repository.
func (s *repositoryService) DeleteAptHosted(ctx context.Context, name string) error {
	return s.client.Repository.Apt.Hosted.Delete(name)
}

// GetAptProxy gets an APT proxy repository.
func (s *repositoryService) GetAptProxy(ctx context.Context, name string) (*repository.AptProxyRepository, error) {
	return s.client.Repository.Apt.Proxy.Get(name)
}

// CreateAptProxy creates an APT proxy repository.
func (s *repositoryService) CreateAptProxy(ctx context.Context, repo repository.AptProxyRepository) error {
	return s.client.Repository.Apt.Proxy.Create(repo)
}

// UpdateAptProxy updates an APT proxy repository.
func (s *repositoryService) UpdateAptProxy(ctx context.Context, name string, repo repository.AptProxyRepository) error {
	return s.client.Repository.Apt.Proxy.Update(name, repo)
}

// DeleteAptProxy deletes an APT proxy repository.
func (s *repositoryService) DeleteAptProxy(ctx context.Context, name string) error {
	return s.client.Repository.Apt.Proxy.Delete(name)
}

// RepositoryService implementations - Bower

// GetBowerHosted gets a Bower hosted repository.
func (s *repositoryService) GetBowerHosted(ctx context.Context, name string) (*repository.BowerHostedRepository, error) {
	return s.client.Repository.Bower.Hosted.Get(name)
}

// CreateBowerHosted creates a Bower hosted repository.
func (s *repositoryService) CreateBowerHosted(ctx context.Context, repo repository.BowerHostedRepository) error {
	return s.client.Repository.Bower.Hosted.Create(repo)
}

// UpdateBowerHosted updates a Bower hosted repository.
func (s *repositoryService) UpdateBowerHosted(ctx context.Context, name string, repo repository.BowerHostedRepository) error {
	return s.client.Repository.Bower.Hosted.Update(name, repo)
}

// DeleteBowerHosted deletes a Bower hosted repository.
func (s *repositoryService) DeleteBowerHosted(ctx context.Context, name string) error {
	return s.client.Repository.Bower.Hosted.Delete(name)
}

// GetBowerProxy gets a Bower proxy repository.
func (s *repositoryService) GetBowerProxy(ctx context.Context, name string) (*repository.BowerProxyRepository, error) {
	return s.client.Repository.Bower.Proxy.Get(name)
}

// CreateBowerProxy creates a Bower proxy repository.
func (s *repositoryService) CreateBowerProxy(ctx context.Context, repo repository.BowerProxyRepository) error {
	return s.client.Repository.Bower.Proxy.Create(repo)
}

// UpdateBowerProxy updates a Bower proxy repository.
func (s *repositoryService) UpdateBowerProxy(ctx context.Context, name string, repo repository.BowerProxyRepository) error {
	return s.client.Repository.Bower.Proxy.Update(name, repo)
}

// DeleteBowerProxy deletes a Bower proxy repository.
func (s *repositoryService) DeleteBowerProxy(ctx context.Context, name string) error {
	return s.client.Repository.Bower.Proxy.Delete(name)
}

// GetBowerGroup gets a Bower group repository.
func (s *repositoryService) GetBowerGroup(ctx context.Context, name string) (*repository.BowerGroupRepository, error) {
	return s.client.Repository.Bower.Group.Get(name)
}

// CreateBowerGroup creates a Bower group repository.
func (s *repositoryService) CreateBowerGroup(ctx context.Context, repo repository.BowerGroupRepository) error {
	return s.client.Repository.Bower.Group.Create(repo)
}

// UpdateBowerGroup updates a Bower group repository.
func (s *repositoryService) UpdateBowerGroup(ctx context.Context, name string, repo repository.BowerGroupRepository) error {
	return s.client.Repository.Bower.Group.Update(name, repo)
}

// DeleteBowerGroup deletes a Bower group repository.
func (s *repositoryService) DeleteBowerGroup(ctx context.Context, name string) error {
	return s.client.Repository.Bower.Group.Delete(name)
}

// RepositoryService implementations - Cargo

// GetCargoHosted gets a Cargo hosted repository.
func (s *repositoryService) GetCargoHosted(ctx context.Context, name string) (*repository.CargoHostedRepository, error) {
	return s.client.Repository.Cargo.Hosted.Get(name)
}

// CreateCargoHosted creates a Cargo hosted repository.
func (s *repositoryService) CreateCargoHosted(ctx context.Context, repo repository.CargoHostedRepository) error {
	return s.client.Repository.Cargo.Hosted.Create(repo)
}

// UpdateCargoHosted updates a Cargo hosted repository.
func (s *repositoryService) UpdateCargoHosted(ctx context.Context, name string, repo repository.CargoHostedRepository) error {
	return s.client.Repository.Cargo.Hosted.Update(name, repo)
}

// DeleteCargoHosted deletes a Cargo hosted repository.
func (s *repositoryService) DeleteCargoHosted(ctx context.Context, name string) error {
	return s.client.Repository.Cargo.Hosted.Delete(name)
}

// GetCargoProxy gets a Cargo proxy repository.
func (s *repositoryService) GetCargoProxy(ctx context.Context, name string) (*repository.CargoProxyRepository, error) {
	return s.client.Repository.Cargo.Proxy.Get(name)
}

// CreateCargoProxy creates a Cargo proxy repository.
func (s *repositoryService) CreateCargoProxy(ctx context.Context, repo repository.CargoProxyRepository) error {
	return s.client.Repository.Cargo.Proxy.Create(repo)
}

// UpdateCargoProxy updates a Cargo proxy repository.
func (s *repositoryService) UpdateCargoProxy(ctx context.Context, name string, repo repository.CargoProxyRepository) error {
	return s.client.Repository.Cargo.Proxy.Update(name, repo)
}

// DeleteCargoProxy deletes a Cargo proxy repository.
func (s *repositoryService) DeleteCargoProxy(ctx context.Context, name string) error {
	return s.client.Repository.Cargo.Proxy.Delete(name)
}

// GetCargoGroup gets a Cargo group repository.
func (s *repositoryService) GetCargoGroup(ctx context.Context, name string) (*repository.CargoGroupRepository, error) {
	return s.client.Repository.Cargo.Group.Get(name)
}

// CreateCargoGroup creates a Cargo group repository.
func (s *repositoryService) CreateCargoGroup(ctx context.Context, repo repository.CargoGroupRepository) error {
	return s.client.Repository.Cargo.Group.Create(repo)
}

// UpdateCargoGroup updates a Cargo group repository.
func (s *repositoryService) UpdateCargoGroup(ctx context.Context, name string, repo repository.CargoGroupRepository) error {
	return s.client.Repository.Cargo.Group.Update(name, repo)
}

// DeleteCargoGroup deletes a Cargo group repository.
func (s *repositoryService) DeleteCargoGroup(ctx context.Context, name string) error {
	return s.client.Repository.Cargo.Group.Delete(name)
}

// RepositoryService implementations - Cocoapods

// GetCocoapodsProxy gets a Cocoapods proxy repository.
func (s *repositoryService) GetCocoapodsProxy(ctx context.Context, name string) (*repository.CocoapodsProxyRepository, error) {
	return s.client.Repository.Cocoapods.Proxy.Get(name)
}

// CreateCocoapodsProxy creates a Cocoapods proxy repository.
func (s *repositoryService) CreateCocoapodsProxy(ctx context.Context, repo repository.CocoapodsProxyRepository) error {
	return s.client.Repository.Cocoapods.Proxy.Create(repo)
}

// UpdateCocoapodsProxy updates a Cocoapods proxy repository.
func (s *repositoryService) UpdateCocoapodsProxy(ctx context.Context, name string, repo repository.CocoapodsProxyRepository) error {
	return s.client.Repository.Cocoapods.Proxy.Update(name, repo)
}

// DeleteCocoapodsProxy deletes a Cocoapods proxy repository.
func (s *repositoryService) DeleteCocoapodsProxy(ctx context.Context, name string) error {
	return s.client.Repository.Cocoapods.Proxy.Delete(name)
}

// RepositoryService implementations - Conan

// GetConanProxy gets a Conan proxy repository.
func (s *repositoryService) GetConanProxy(ctx context.Context, name string) (*repository.ConanProxyRepository, error) {
	return s.client.Repository.Conan.Proxy.Get(name)
}

// CreateConanProxy creates a Conan proxy repository.
func (s *repositoryService) CreateConanProxy(ctx context.Context, repo repository.ConanProxyRepository) error {
	return s.client.Repository.Conan.Proxy.Create(repo)
}

// UpdateConanProxy updates a Conan proxy repository.
func (s *repositoryService) UpdateConanProxy(ctx context.Context, name string, repo repository.ConanProxyRepository) error {
	return s.client.Repository.Conan.Proxy.Update(name, repo)
}

// DeleteConanProxy deletes a Conan proxy repository.
func (s *repositoryService) DeleteConanProxy(ctx context.Context, name string) error {
	return s.client.Repository.Conan.Proxy.Delete(name)
}

// RepositoryService implementations - Conda

// GetCondaProxy gets a Conda proxy repository.
func (s *repositoryService) GetCondaProxy(ctx context.Context, name string) (*repository.CondaProxyRepository, error) {
	return s.client.Repository.Conda.Proxy.Get(name)
}

// CreateCondaProxy creates a Conda proxy repository.
func (s *repositoryService) CreateCondaProxy(ctx context.Context, repo repository.CondaProxyRepository) error {
	return s.client.Repository.Conda.Proxy.Create(repo)
}

// UpdateCondaProxy updates a Conda proxy repository.
func (s *repositoryService) UpdateCondaProxy(ctx context.Context, name string, repo repository.CondaProxyRepository) error {
	return s.client.Repository.Conda.Proxy.Update(name, repo)
}

// DeleteCondaProxy deletes a Conda proxy repository.
func (s *repositoryService) DeleteCondaProxy(ctx context.Context, name string) error {
	return s.client.Repository.Conda.Proxy.Delete(name)
}

// RepositoryService implementations - GitLFS

// GetGitLfsHosted gets a GitLFS hosted repository.
func (s *repositoryService) GetGitLfsHosted(ctx context.Context, name string) (*repository.GitLfsHostedRepository, error) {
	return s.client.Repository.GitLfs.Hosted.Get(name)
}

// CreateGitLfsHosted creates a GitLFS hosted repository.
func (s *repositoryService) CreateGitLfsHosted(ctx context.Context, repo repository.GitLfsHostedRepository) error {
	return s.client.Repository.GitLfs.Hosted.Create(repo)
}

// UpdateGitLfsHosted updates a GitLFS hosted repository.
func (s *repositoryService) UpdateGitLfsHosted(ctx context.Context, name string, repo repository.GitLfsHostedRepository) error {
	return s.client.Repository.GitLfs.Hosted.Update(name, repo)
}

// DeleteGitLfsHosted deletes a GitLFS hosted repository.
func (s *repositoryService) DeleteGitLfsHosted(ctx context.Context, name string) error {
	return s.client.Repository.GitLfs.Hosted.Delete(name)
}

// RepositoryService implementations - Go

// GetGoProxy gets a Go proxy repository.
func (s *repositoryService) GetGoProxy(ctx context.Context, name string) (*repository.GoProxyRepository, error) {
	return s.client.Repository.Go.Proxy.Get(name)
}

// CreateGoProxy creates a Go proxy repository.
func (s *repositoryService) CreateGoProxy(ctx context.Context, repo repository.GoProxyRepository) error {
	return s.client.Repository.Go.Proxy.Create(repo)
}

// UpdateGoProxy updates a Go proxy repository.
func (s *repositoryService) UpdateGoProxy(ctx context.Context, name string, repo repository.GoProxyRepository) error {
	return s.client.Repository.Go.Proxy.Update(name, repo)
}

// DeleteGoProxy deletes a Go proxy repository.
func (s *repositoryService) DeleteGoProxy(ctx context.Context, name string) error {
	return s.client.Repository.Go.Proxy.Delete(name)
}

// GetGoGroup gets a Go group repository.
func (s *repositoryService) GetGoGroup(ctx context.Context, name string) (*repository.GoGroupRepository, error) {
	return s.client.Repository.Go.Group.Get(name)
}

// CreateGoGroup creates a Go group repository.
func (s *repositoryService) CreateGoGroup(ctx context.Context, repo repository.GoGroupRepository) error {
	return s.client.Repository.Go.Group.Create(repo)
}

// UpdateGoGroup updates a Go group repository.
func (s *repositoryService) UpdateGoGroup(ctx context.Context, name string, repo repository.GoGroupRepository) error {
	return s.client.Repository.Go.Group.Update(name, repo)
}

// DeleteGoGroup deletes a Go group repository.
func (s *repositoryService) DeleteGoGroup(ctx context.Context, name string) error {
	return s.client.Repository.Go.Group.Delete(name)
}

// RepositoryService implementations - Helm

// GetHelmHosted gets a Helm hosted repository.
func (s *repositoryService) GetHelmHosted(ctx context.Context, name string) (*repository.HelmHostedRepository, error) {
	return s.client.Repository.Helm.Hosted.Get(name)
}

// CreateHelmHosted creates a Helm hosted repository.
func (s *repositoryService) CreateHelmHosted(ctx context.Context, repo repository.HelmHostedRepository) error {
	return s.client.Repository.Helm.Hosted.Create(repo)
}

// UpdateHelmHosted updates a Helm hosted repository.
func (s *repositoryService) UpdateHelmHosted(ctx context.Context, name string, repo repository.HelmHostedRepository) error {
	return s.client.Repository.Helm.Hosted.Update(name, repo)
}

// DeleteHelmHosted deletes a Helm hosted repository.
func (s *repositoryService) DeleteHelmHosted(ctx context.Context, name string) error {
	return s.client.Repository.Helm.Hosted.Delete(name)
}

// GetHelmProxy gets a Helm proxy repository.
func (s *repositoryService) GetHelmProxy(ctx context.Context, name string) (*repository.HelmProxyRepository, error) {
	return s.client.Repository.Helm.Proxy.Get(name)
}

// CreateHelmProxy creates a Helm proxy repository.
func (s *repositoryService) CreateHelmProxy(ctx context.Context, repo repository.HelmProxyRepository) error {
	return s.client.Repository.Helm.Proxy.Create(repo)
}

// UpdateHelmProxy updates a Helm proxy repository.
func (s *repositoryService) UpdateHelmProxy(ctx context.Context, name string, repo repository.HelmProxyRepository) error {
	return s.client.Repository.Helm.Proxy.Update(name, repo)
}

// DeleteHelmProxy deletes a Helm proxy repository.
func (s *repositoryService) DeleteHelmProxy(ctx context.Context, name string) error {
	return s.client.Repository.Helm.Proxy.Delete(name)
}

// RepositoryService implementations - NuGet

// GetNugetHosted gets a NuGet hosted repository.
func (s *repositoryService) GetNugetHosted(ctx context.Context, name string) (*repository.NugetHostedRepository, error) {
	return s.client.Repository.Nuget.Hosted.Get(name)
}

// CreateNugetHosted creates a NuGet hosted repository.
func (s *repositoryService) CreateNugetHosted(ctx context.Context, repo repository.NugetHostedRepository) error {
	return s.client.Repository.Nuget.Hosted.Create(repo)
}

// UpdateNugetHosted updates a NuGet hosted repository.
func (s *repositoryService) UpdateNugetHosted(ctx context.Context, name string, repo repository.NugetHostedRepository) error {
	return s.client.Repository.Nuget.Hosted.Update(name, repo)
}

// DeleteNugetHosted deletes a NuGet hosted repository.
func (s *repositoryService) DeleteNugetHosted(ctx context.Context, name string) error {
	return s.client.Repository.Nuget.Hosted.Delete(name)
}

// GetNugetProxy gets a NuGet proxy repository.
func (s *repositoryService) GetNugetProxy(ctx context.Context, name string) (*repository.NugetProxyRepository, error) {
	return s.client.Repository.Nuget.Proxy.Get(name)
}

// CreateNugetProxy creates a NuGet proxy repository.
func (s *repositoryService) CreateNugetProxy(ctx context.Context, repo repository.NugetProxyRepository) error {
	return s.client.Repository.Nuget.Proxy.Create(repo)
}

// UpdateNugetProxy updates a NuGet proxy repository.
func (s *repositoryService) UpdateNugetProxy(ctx context.Context, name string, repo repository.NugetProxyRepository) error {
	return s.client.Repository.Nuget.Proxy.Update(name, repo)
}

// DeleteNugetProxy deletes a NuGet proxy repository.
func (s *repositoryService) DeleteNugetProxy(ctx context.Context, name string) error {
	return s.client.Repository.Nuget.Proxy.Delete(name)
}

// GetNugetGroup gets a NuGet group repository.
func (s *repositoryService) GetNugetGroup(ctx context.Context, name string) (*repository.NugetGroupRepository, error) {
	return s.client.Repository.Nuget.Group.Get(name)
}

// CreateNugetGroup creates a NuGet group repository.
func (s *repositoryService) CreateNugetGroup(ctx context.Context, repo repository.NugetGroupRepository) error {
	return s.client.Repository.Nuget.Group.Create(repo)
}

// UpdateNugetGroup updates a NuGet group repository.
func (s *repositoryService) UpdateNugetGroup(ctx context.Context, name string, repo repository.NugetGroupRepository) error {
	return s.client.Repository.Nuget.Group.Update(name, repo)
}

// DeleteNugetGroup deletes a NuGet group repository.
func (s *repositoryService) DeleteNugetGroup(ctx context.Context, name string) error {
	return s.client.Repository.Nuget.Group.Delete(name)
}

// RepositoryService implementations - PyPI

// GetPypiHosted gets a PyPI hosted repository.
func (s *repositoryService) GetPypiHosted(ctx context.Context, name string) (*repository.PypiHostedRepository, error) {
	return s.client.Repository.Pypi.Hosted.Get(name)
}

// CreatePypiHosted creates a PyPI hosted repository.
func (s *repositoryService) CreatePypiHosted(ctx context.Context, repo repository.PypiHostedRepository) error {
	return s.client.Repository.Pypi.Hosted.Create(repo)
}

// UpdatePypiHosted updates a PyPI hosted repository.
func (s *repositoryService) UpdatePypiHosted(ctx context.Context, name string, repo repository.PypiHostedRepository) error {
	return s.client.Repository.Pypi.Hosted.Update(name, repo)
}

// DeletePypiHosted deletes a PyPI hosted repository.
func (s *repositoryService) DeletePypiHosted(ctx context.Context, name string) error {
	return s.client.Repository.Pypi.Hosted.Delete(name)
}

// GetPypiProxy gets a PyPI proxy repository.
func (s *repositoryService) GetPypiProxy(ctx context.Context, name string) (*repository.PypiProxyRepository, error) {
	return s.client.Repository.Pypi.Proxy.Get(name)
}

// CreatePypiProxy creates a PyPI proxy repository.
func (s *repositoryService) CreatePypiProxy(ctx context.Context, repo repository.PypiProxyRepository) error {
	return s.client.Repository.Pypi.Proxy.Create(repo)
}

// UpdatePypiProxy updates a PyPI proxy repository.
func (s *repositoryService) UpdatePypiProxy(ctx context.Context, name string, repo repository.PypiProxyRepository) error {
	return s.client.Repository.Pypi.Proxy.Update(name, repo)
}

// DeletePypiProxy deletes a PyPI proxy repository.
func (s *repositoryService) DeletePypiProxy(ctx context.Context, name string) error {
	return s.client.Repository.Pypi.Proxy.Delete(name)
}

// GetPypiGroup gets a PyPI group repository.
func (s *repositoryService) GetPypiGroup(ctx context.Context, name string) (*repository.PypiGroupRepository, error) {
	return s.client.Repository.Pypi.Group.Get(name)
}

// CreatePypiGroup creates a PyPI group repository.
func (s *repositoryService) CreatePypiGroup(ctx context.Context, repo repository.PypiGroupRepository) error {
	return s.client.Repository.Pypi.Group.Create(repo)
}

// UpdatePypiGroup updates a PyPI group repository.
func (s *repositoryService) UpdatePypiGroup(ctx context.Context, name string, repo repository.PypiGroupRepository) error {
	return s.client.Repository.Pypi.Group.Update(name, repo)
}

// DeletePypiGroup deletes a PyPI group repository.
func (s *repositoryService) DeletePypiGroup(ctx context.Context, name string) error {
	return s.client.Repository.Pypi.Group.Delete(name)
}

// RepositoryService implementations - R

// GetRHosted gets an R hosted repository.
func (s *repositoryService) GetRHosted(ctx context.Context, name string) (*repository.RHostedRepository, error) {
	return s.client.Repository.R.Hosted.Get(name)
}

// CreateRHosted creates an R hosted repository.
func (s *repositoryService) CreateRHosted(ctx context.Context, repo repository.RHostedRepository) error {
	return s.client.Repository.R.Hosted.Create(repo)
}

// UpdateRHosted updates an R hosted repository.
func (s *repositoryService) UpdateRHosted(ctx context.Context, name string, repo repository.RHostedRepository) error {
	return s.client.Repository.R.Hosted.Update(name, repo)
}

// DeleteRHosted deletes an R hosted repository.
func (s *repositoryService) DeleteRHosted(ctx context.Context, name string) error {
	return s.client.Repository.R.Hosted.Delete(name)
}

// GetRProxy gets an R proxy repository.
func (s *repositoryService) GetRProxy(ctx context.Context, name string) (*repository.RProxyRepository, error) {
	return s.client.Repository.R.Proxy.Get(name)
}

// CreateRProxy creates an R proxy repository.
func (s *repositoryService) CreateRProxy(ctx context.Context, repo repository.RProxyRepository) error {
	return s.client.Repository.R.Proxy.Create(repo)
}

// UpdateRProxy updates an R proxy repository.
func (s *repositoryService) UpdateRProxy(ctx context.Context, name string, repo repository.RProxyRepository) error {
	return s.client.Repository.R.Proxy.Update(name, repo)
}

// DeleteRProxy deletes an R proxy repository.
func (s *repositoryService) DeleteRProxy(ctx context.Context, name string) error {
	return s.client.Repository.R.Proxy.Delete(name)
}

// GetRGroup gets an R group repository.
func (s *repositoryService) GetRGroup(ctx context.Context, name string) (*repository.RGroupRepository, error) {
	return s.client.Repository.R.Group.Get(name)
}

// CreateRGroup creates an R group repository.
func (s *repositoryService) CreateRGroup(ctx context.Context, repo repository.RGroupRepository) error {
	return s.client.Repository.R.Group.Create(repo)
}

// UpdateRGroup updates an R group repository.
func (s *repositoryService) UpdateRGroup(ctx context.Context, name string, repo repository.RGroupRepository) error {
	return s.client.Repository.R.Group.Update(name, repo)
}

// DeleteRGroup deletes an R group repository.
func (s *repositoryService) DeleteRGroup(ctx context.Context, name string) error {
	return s.client.Repository.R.Group.Delete(name)
}

// RepositoryService implementations - RubyGems

// GetRubygemsHosted gets a RubyGems hosted repository.
func (s *repositoryService) GetRubygemsHosted(ctx context.Context, name string) (*repository.RubyGemsHostedRepository, error) {
	return s.client.Repository.RubyGems.Hosted.Get(name)
}

// CreateRubygemsHosted creates a RubyGems hosted repository.
func (s *repositoryService) CreateRubygemsHosted(ctx context.Context, repo repository.RubyGemsHostedRepository) error {
	return s.client.Repository.RubyGems.Hosted.Create(repo)
}

// UpdateRubygemsHosted updates a RubyGems hosted repository.
func (s *repositoryService) UpdateRubygemsHosted(ctx context.Context, name string, repo repository.RubyGemsHostedRepository) error {
	return s.client.Repository.RubyGems.Hosted.Update(name, repo)
}

// DeleteRubygemsHosted deletes a RubyGems hosted repository.
func (s *repositoryService) DeleteRubygemsHosted(ctx context.Context, name string) error {
	return s.client.Repository.RubyGems.Hosted.Delete(name)
}

// GetRubygemsProxy gets a RubyGems proxy repository.
func (s *repositoryService) GetRubygemsProxy(ctx context.Context, name string) (*repository.RubyGemsProxyRepository, error) {
	return s.client.Repository.RubyGems.Proxy.Get(name)
}

// CreateRubygemsProxy creates a RubyGems proxy repository.
func (s *repositoryService) CreateRubygemsProxy(ctx context.Context, repo repository.RubyGemsProxyRepository) error {
	return s.client.Repository.RubyGems.Proxy.Create(repo)
}

// UpdateRubygemsProxy updates a RubyGems proxy repository.
func (s *repositoryService) UpdateRubygemsProxy(ctx context.Context, name string, repo repository.RubyGemsProxyRepository) error {
	return s.client.Repository.RubyGems.Proxy.Update(name, repo)
}

// DeleteRubygemsProxy deletes a RubyGems proxy repository.
func (s *repositoryService) DeleteRubygemsProxy(ctx context.Context, name string) error {
	return s.client.Repository.RubyGems.Proxy.Delete(name)
}

// GetRubygemsGroup gets a RubyGems group repository.
func (s *repositoryService) GetRubygemsGroup(ctx context.Context, name string) (*repository.RubyGemsGroupRepository, error) {
	return s.client.Repository.RubyGems.Group.Get(name)
}

// CreateRubygemsGroup creates a RubyGems group repository.
func (s *repositoryService) CreateRubygemsGroup(ctx context.Context, repo repository.RubyGemsGroupRepository) error {
	return s.client.Repository.RubyGems.Group.Create(repo)
}

// UpdateRubygemsGroup updates a RubyGems group repository.
func (s *repositoryService) UpdateRubygemsGroup(ctx context.Context, name string, repo repository.RubyGemsGroupRepository) error {
	return s.client.Repository.RubyGems.Group.Update(name, repo)
}

// DeleteRubygemsGroup deletes a RubyGems group repository.
func (s *repositoryService) DeleteRubygemsGroup(ctx context.Context, name string) error {
	return s.client.Repository.RubyGems.Group.Delete(name)
}

// RepositoryService implementations - Yum

// GetYumHosted gets a Yum hosted repository.
func (s *repositoryService) GetYumHosted(ctx context.Context, name string) (*repository.YumHostedRepository, error) {
	return s.client.Repository.Yum.Hosted.Get(name)
}

// CreateYumHosted creates a Yum hosted repository.
func (s *repositoryService) CreateYumHosted(ctx context.Context, repo repository.YumHostedRepository) error {
	return s.client.Repository.Yum.Hosted.Create(repo)
}

// UpdateYumHosted updates a Yum hosted repository.
func (s *repositoryService) UpdateYumHosted(ctx context.Context, name string, repo repository.YumHostedRepository) error {
	return s.client.Repository.Yum.Hosted.Update(name, repo)
}

// DeleteYumHosted deletes a Yum hosted repository.
func (s *repositoryService) DeleteYumHosted(ctx context.Context, name string) error {
	return s.client.Repository.Yum.Hosted.Delete(name)
}

// GetYumProxy gets a Yum proxy repository.
func (s *repositoryService) GetYumProxy(ctx context.Context, name string) (*repository.YumProxyRepository, error) {
	return s.client.Repository.Yum.Proxy.Get(name)
}

// CreateYumProxy creates a Yum proxy repository.
func (s *repositoryService) CreateYumProxy(ctx context.Context, repo repository.YumProxyRepository) error {
	return s.client.Repository.Yum.Proxy.Create(repo)
}

// UpdateYumProxy updates a Yum proxy repository.
func (s *repositoryService) UpdateYumProxy(ctx context.Context, name string, repo repository.YumProxyRepository) error {
	return s.client.Repository.Yum.Proxy.Update(name, repo)
}

// DeleteYumProxy deletes a Yum proxy repository.
func (s *repositoryService) DeleteYumProxy(ctx context.Context, name string) error {
	return s.client.Repository.Yum.Proxy.Delete(name)
}

// GetYumGroup gets a Yum group repository.
func (s *repositoryService) GetYumGroup(ctx context.Context, name string) (*repository.YumGroupRepository, error) {
	return s.client.Repository.Yum.Group.Get(name)
}

// CreateYumGroup creates a Yum group repository.
func (s *repositoryService) CreateYumGroup(ctx context.Context, repo repository.YumGroupRepository) error {
	return s.client.Repository.Yum.Group.Create(repo)
}

// UpdateYumGroup updates a Yum group repository.
func (s *repositoryService) UpdateYumGroup(ctx context.Context, name string, repo repository.YumGroupRepository) error {
	return s.client.Repository.Yum.Group.Update(name, repo)
}

// DeleteYumGroup deletes a Yum group repository.
func (s *repositoryService) DeleteYumGroup(ctx context.Context, name string) error {
	return s.client.Repository.Yum.Group.Delete(name)
}

// SecurityService implementations

// GetUser gets a user by ID.
func (s *securityService) GetUser(ctx context.Context, id string) (*security.User, error) {
	return s.client.Security.User.Get(id, nil)
}

// CreateUser creates a new user.
func (s *securityService) CreateUser(ctx context.Context, user security.User) error {
	return s.client.Security.User.Create(user)
}

// UpdateUser updates an existing user.
func (s *securityService) UpdateUser(ctx context.Context, id string, user security.User) error {
	return s.client.Security.User.Update(id, user)
}

// DeleteUser deletes a user by ID.
func (s *securityService) DeleteUser(ctx context.Context, id string) error {
	return s.client.Security.User.Delete(id)
}

// ChangePassword changes a user's password.
func (s *securityService) ChangePassword(ctx context.Context, id, password string) error {
	return s.client.Security.User.ChangePassword(id, password)
}

// GetRole gets a role by ID.
func (s *securityService) GetRole(ctx context.Context, id string) (*security.Role, error) {
	return s.client.Security.Role.Get(id)
}

// CreateRole creates a new role.
func (s *securityService) CreateRole(ctx context.Context, role security.Role) error {
	return s.client.Security.Role.Create(role)
}

// UpdateRole updates an existing role.
func (s *securityService) UpdateRole(ctx context.Context, id string, role security.Role) error {
	return s.client.Security.Role.Update(id, role)
}

// DeleteRole deletes a role by ID.
func (s *securityService) DeleteRole(ctx context.Context, id string) error {
	return s.client.Security.Role.Delete(id)
}

// Realm management

// ListAvailableRealms lists all available security realms.
func (s *securityService) ListAvailableRealms(ctx context.Context) ([]security.Realm, error) {
	return s.client.Security.Realm.ListAvailable()
}

// ListActiveRealms lists all active security realm IDs.
func (s *securityService) ListActiveRealms(ctx context.Context) ([]string, error) {
	return s.client.Security.Realm.ListActive()
}

// ActivateRealms sets the active security realms.
func (s *securityService) ActivateRealms(ctx context.Context, ids []string) error {
	return s.client.Security.Realm.Activate(ids)
}

// Content Selector management

// GetContentSelector gets a content selector by name.
func (s *securityService) GetContentSelector(ctx context.Context, name string) (*security.ContentSelector, error) {
	return s.client.Security.ContentSelector.Get(name)
}

// ListContentSelectors lists all content selectors.
func (s *securityService) ListContentSelectors(ctx context.Context) ([]security.ContentSelector, error) {
	return s.client.Security.ContentSelector.List()
}

// CreateContentSelector creates a new content selector.
func (s *securityService) CreateContentSelector(ctx context.Context, cs security.ContentSelector) error {
	return s.client.Security.ContentSelector.Create(cs)
}

// UpdateContentSelector updates an existing content selector.
func (s *securityService) UpdateContentSelector(ctx context.Context, name string, cs security.ContentSelector) error {
	return s.client.Security.ContentSelector.Update(name, cs)
}

// DeleteContentSelector deletes a content selector by name.
func (s *securityService) DeleteContentSelector(ctx context.Context, name string) error {
	return s.client.Security.ContentSelector.Delete(name)
}

// Privilege management

// GetPrivilege gets a privilege by name.
func (s *securityService) GetPrivilege(ctx context.Context, name string) (*security.Privilege, error) {
	return s.client.Security.Privilege.Get(name)
}

// ListPrivileges lists all privileges.
func (s *securityService) ListPrivileges(ctx context.Context) ([]security.Privilege, error) {
	return s.client.Security.Privilege.List()
}

// DeletePrivilege deletes a privilege by name.
func (s *securityService) DeletePrivilege(ctx context.Context, name string) error {
	return s.client.Security.Privilege.Delete(name)
}

// CreatePrivilegeApplication creates an application privilege.
func (s *securityService) CreatePrivilegeApplication(ctx context.Context, p security.PrivilegeApplication) error {
	return s.client.Security.Privilege.Application.Create(p)
}

// UpdatePrivilegeApplication updates an application privilege.
func (s *securityService) UpdatePrivilegeApplication(ctx context.Context, name string, p security.PrivilegeApplication) error {
	return s.client.Security.Privilege.Application.Update(name, p)
}

// CreatePrivilegeRepositoryView creates a repository view privilege.
func (s *securityService) CreatePrivilegeRepositoryView(ctx context.Context, p security.PrivilegeRepositoryView) error {
	return s.client.Security.Privilege.RepositoryView.Create(p)
}

// UpdatePrivilegeRepositoryView updates a repository view privilege.
func (s *securityService) UpdatePrivilegeRepositoryView(ctx context.Context, name string, p security.PrivilegeRepositoryView) error {
	return s.client.Security.Privilege.RepositoryView.Update(name, p)
}

// CreatePrivilegeRepositoryAdmin creates a repository admin privilege.
func (s *securityService) CreatePrivilegeRepositoryAdmin(ctx context.Context, p security.PrivilegeRepositoryAdmin) error {
	return s.client.Security.Privilege.RepositoryAdmin.Create(p)
}

// UpdatePrivilegeRepositoryAdmin updates a repository admin privilege.
func (s *securityService) UpdatePrivilegeRepositoryAdmin(ctx context.Context, name string, p security.PrivilegeRepositoryAdmin) error {
	return s.client.Security.Privilege.RepositoryAdmin.Update(name, p)
}

// CreatePrivilegeRepositoryContentSelector creates a repository
// content selector privilege.
func (s *securityService) CreatePrivilegeRepositoryContentSelector(ctx context.Context, p security.PrivilegeRepositoryContentSelector) error {
	return s.client.Security.Privilege.RepositoryContentSelector.Create(p)
}

// UpdatePrivilegeRepositoryContentSelector updates a repository
// content selector privilege.
func (s *securityService) UpdatePrivilegeRepositoryContentSelector(ctx context.Context, name string, p security.PrivilegeRepositoryContentSelector) error {
	return s.client.Security.Privilege.RepositoryContentSelector.Update(name, p)
}

// CreatePrivilegeScript creates a script privilege.
func (s *securityService) CreatePrivilegeScript(ctx context.Context, p security.PrivilegeScript) error {
	return s.client.Security.Privilege.Script.Create(p)
}

// UpdatePrivilegeScript updates a script privilege.
func (s *securityService) UpdatePrivilegeScript(ctx context.Context, name string, p security.PrivilegeScript) error {
	return s.client.Security.Privilege.Script.Update(name, p)
}

// CreatePrivilegeWildcard creates a wildcard privilege.
func (s *securityService) CreatePrivilegeWildcard(ctx context.Context, p security.PrivilegeWildcard) error {
	return s.client.Security.Privilege.Wildcard.Create(p)
}

// UpdatePrivilegeWildcard updates a wildcard privilege.
func (s *securityService) UpdatePrivilegeWildcard(ctx context.Context, name string, p security.PrivilegeWildcard) error {
	return s.client.Security.Privilege.Wildcard.Update(name, p)
}

// Anonymous access management

// GetAnonymousAccess gets the anonymous access settings.
func (s *securityService) GetAnonymousAccess(ctx context.Context) (*security.AnonymousAccessSettings, error) {
	return s.client.Security.Anonymous.Read()
}

// UpdateAnonymousAccess updates the anonymous access settings.
func (s *securityService) UpdateAnonymousAccess(ctx context.Context, settings security.AnonymousAccessSettings) error {
	return s.client.Security.Anonymous.Update(settings)
}

// SAML management

// GetSAML gets the SAML configuration.
func (s *securityService) GetSAML(ctx context.Context) (*security.SAML, error) {
	return s.client.Security.SAML.Read()
}

// ApplySAML creates or updates the SAML configuration.
func (s *securityService) ApplySAML(ctx context.Context, saml security.SAML) error {
	return s.client.Security.SAML.Apply(saml)
}

// DeleteSAML deletes the SAML configuration.
func (s *securityService) DeleteSAML(ctx context.Context) error {
	return s.client.Security.SAML.Delete()
}

// LDAP management

// GetLDAP gets an LDAP server configuration by name.
func (s *securityService) GetLDAP(ctx context.Context, name string) (*security.LDAP, error) {
	return s.client.Security.LDAP.Get(name)
}

// ListLDAP lists all LDAP server configurations.
func (s *securityService) ListLDAP(ctx context.Context) ([]security.LDAP, error) {
	return s.client.Security.LDAP.List()
}

// CreateLDAP creates a new LDAP server configuration.
func (s *securityService) CreateLDAP(ctx context.Context, ldap security.LDAP) error {
	return s.client.Security.LDAP.Create(ldap)
}

// UpdateLDAP updates an existing LDAP server configuration.
func (s *securityService) UpdateLDAP(ctx context.Context, name string, ldap security.LDAP) error {
	return s.client.Security.LDAP.Update(name, ldap)
}

// DeleteLDAP deletes an LDAP server configuration by name.
func (s *securityService) DeleteLDAP(ctx context.Context, name string) error {
	return s.client.Security.LDAP.Delete(name)
}

// User Token management

// GetUserTokenConfiguration gets the user token configuration.
func (s *securityService) GetUserTokenConfiguration(ctx context.Context) (*security.UserTokenConfiguration, error) {
	return s.client.Security.UserTokens.Get()
}

// UpdateUserTokenConfiguration updates the user token configuration.
func (s *securityService) UpdateUserTokenConfiguration(ctx context.Context, config security.UserTokenConfiguration) error {
	return s.client.Security.UserTokens.Configure(config)
}

// SSL Truststore management

// AddCertificate adds a certificate to the Nexus truststore.
func (s *sslService) AddCertificate(ctx context.Context, cert *security.SSLCertificate) error {
	return s.client.Security.SSL.AddCertificate(cert)
}

// RemoveCertificate removes a certificate from the Nexus truststore by ID.
func (s *sslService) RemoveCertificate(ctx context.Context, id string) error {
	return s.client.Security.SSL.RemoveCertificate(id)
}

// ListCertificates retrieves all certificates in the Nexus truststore.
func (s *sslService) ListCertificates(ctx context.Context) ([]security.SSLCertificate, error) {
	certs, err := s.client.Security.SSL.ListCertificates()
	if err != nil {
		return nil, err
	}

	if certs == nil {
		return nil, nil
	}

	return *certs, nil
}
