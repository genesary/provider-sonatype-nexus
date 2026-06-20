// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/cleanuppolicies"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	nxtask "github.com/datadrivers/go-nexus-client/nexus3/schema/task"

	"github.com/genesary/provider-sonatype-nexus/internal/clients/nexus"
)

// errMockNotConfigured is returned when a mock function has not been
// configured.
var errMockNotConfigured = errors.New("mock function not configured")

var _ nexus.Client = &MockClient{}

// MockClient is a mock implementation of nexus.Client.
type MockClient struct {
	MockBlobStore     *MockBlobStoreService
	MockCleanupPolicy *MockCleanupPolicyService
	MockRepository    *MockRepositoryService
	MockSecurity      *MockSecurityService
	MockSSL           *MockSSLService
	MockTask          *MockTaskService
}

// NewMockClient creates a new MockClient with default mocks.
func NewMockClient() *MockClient {
	return &MockClient{
		MockBlobStore:     &MockBlobStoreService{},
		MockCleanupPolicy: &MockCleanupPolicyService{},
		MockRepository:    &MockRepositoryService{},
		MockSecurity:      &MockSecurityService{},
		MockSSL:           &MockSSLService{},
		MockTask:          &MockTaskService{},
	}
}

// BlobStore returns the mock blob store service.
func (m *MockClient) BlobStore() nexus.BlobStoreService {
	return m.MockBlobStore
}

// CleanupPolicy returns the mock cleanup policy service.
func (m *MockClient) CleanupPolicy() nexus.CleanupPolicyService {
	return m.MockCleanupPolicy
}

// Repository returns the mock repository service.
func (m *MockClient) Repository() nexus.RepositoryService {
	return m.MockRepository
}

// Security returns the mock security service.
func (m *MockClient) Security() nexus.SecurityService {
	return m.MockSecurity
}

// SSL returns the mock SSL service.
func (m *MockClient) SSL() nexus.SSLService {
	return m.MockSSL
}

// Task returns the mock task service.
func (m *MockClient) Task() nexus.TaskService {
	return m.MockTask
}

// MockTaskService is a mock implementation of nexus.TaskService.
type MockTaskService struct {
	GetTaskByNameFn func(ctx context.Context, name string) (*nxtask.Task, error)
	CreateTaskFn    func(ctx context.Context, t *nxtask.TaskCreateStruct) (*nxtask.Task, error)
	UpdateTaskFn    func(ctx context.Context, id string, t *nxtask.TaskCreateStruct) error
	DeleteTaskFn    func(ctx context.Context, id string) error
}

// GetTaskByName mock implementation.
func (m *MockTaskService) GetTaskByName(ctx context.Context, name string) (*nxtask.Task, error) {
	if m.GetTaskByNameFn != nil {
		return m.GetTaskByNameFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateTask mock implementation.
func (m *MockTaskService) CreateTask(ctx context.Context, t *nxtask.TaskCreateStruct) (*nxtask.Task, error) {
	if m.CreateTaskFn != nil {
		return m.CreateTaskFn(ctx, t)
	}

	return nil, errMockNotConfigured
}

// UpdateTask mock implementation.
func (m *MockTaskService) UpdateTask(ctx context.Context, id string, t *nxtask.TaskCreateStruct) error {
	if m.UpdateTaskFn != nil {
		return m.UpdateTaskFn(ctx, id, t)
	}

	return errMockNotConfigured
}

// DeleteTask mock implementation.
func (m *MockTaskService) DeleteTask(ctx context.Context, id string) error {
	if m.DeleteTaskFn != nil {
		return m.DeleteTaskFn(ctx, id)
	}

	return errMockNotConfigured
}

// MockCleanupPolicyService is a mock implementation of
// nexus.CleanupPolicyService.
type MockCleanupPolicyService struct {
	GetCleanupPolicyFn    func(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error)
	ListCleanupPoliciesFn func(ctx context.Context) ([]*cleanuppolicies.CleanupPolicy, error)
	CreateCleanupPolicyFn func(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	UpdateCleanupPolicyFn func(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error
	DeleteCleanupPolicyFn func(ctx context.Context, name string) error

	GetCleanupPolicyCalls    []string
	CreateCleanupPolicyCalls []*cleanuppolicies.CleanupPolicy
	UpdateCleanupPolicyCalls []*cleanuppolicies.CleanupPolicy
	DeleteCleanupPolicyCalls []string
}

// GetCleanupPolicy mock implementation.
func (m *MockCleanupPolicyService) GetCleanupPolicy(ctx context.Context, name string) (*cleanuppolicies.CleanupPolicy, error) {
	m.GetCleanupPolicyCalls = append(m.GetCleanupPolicyCalls, name)
	if m.GetCleanupPolicyFn != nil {
		return m.GetCleanupPolicyFn(ctx, name)
	}
	//nolint:nilnil // intentionally testing nil policy with nil error case
	return nil, nil
}

// ListCleanupPolicies mock implementation.
func (m *MockCleanupPolicyService) ListCleanupPolicies(ctx context.Context) ([]*cleanuppolicies.CleanupPolicy, error) {
	if m.ListCleanupPoliciesFn != nil {
		return m.ListCleanupPoliciesFn(ctx)
	}

	return nil, nil
}

// CreateCleanupPolicy mock implementation.
func (m *MockCleanupPolicyService) CreateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	m.CreateCleanupPolicyCalls = append(m.CreateCleanupPolicyCalls, policy)
	if m.CreateCleanupPolicyFn != nil {
		return m.CreateCleanupPolicyFn(ctx, policy)
	}

	return nil
}

// UpdateCleanupPolicy mock implementation.
func (m *MockCleanupPolicyService) UpdateCleanupPolicy(ctx context.Context, policy *cleanuppolicies.CleanupPolicy) error {
	m.UpdateCleanupPolicyCalls = append(m.UpdateCleanupPolicyCalls, policy)
	if m.UpdateCleanupPolicyFn != nil {
		return m.UpdateCleanupPolicyFn(ctx, policy)
	}

	return nil
}

// DeleteCleanupPolicy mock implementation.
func (m *MockCleanupPolicyService) DeleteCleanupPolicy(ctx context.Context, name string) error {
	m.DeleteCleanupPolicyCalls = append(m.DeleteCleanupPolicyCalls, name)
	if m.DeleteCleanupPolicyFn != nil {
		return m.DeleteCleanupPolicyFn(ctx, name)
	}

	return nil
}

// MockBlobStoreService is a mock implementation of nexus.BlobStoreService.
type MockBlobStoreService struct {
	GetFileFn    func(ctx context.Context, name string) (*blobstore.File, error)
	GetS3Fn      func(ctx context.Context, name string) (*blobstore.S3, error)
	CreateFileFn func(ctx context.Context, bs *blobstore.File) error
	CreateS3Fn   func(ctx context.Context, bs *blobstore.S3) error
	UpdateFileFn func(ctx context.Context, name string, bs *blobstore.File) error
	UpdateS3Fn   func(ctx context.Context, name string, bs *blobstore.S3) error
	DeleteFn     func(ctx context.Context, name string) error
	ListFn       func(ctx context.Context) ([]blobstore.Generic, error)

	// Call tracking
	GetFileCalls    []string
	GetS3Calls      []string
	CreateFileCalls []*blobstore.File
	CreateS3Calls   []*blobstore.S3
	UpdateFileCalls []UpdateFileCall
	UpdateS3Calls   []UpdateS3Call
	DeleteCalls     []string
	ListCalls       int
}

// UpdateFileCall tracks calls to UpdateFile.
type UpdateFileCall struct {
	Name      string
	BlobStore *blobstore.File
}

// UpdateS3Call tracks calls to UpdateS3.
type UpdateS3Call struct {
	Name      string
	BlobStore *blobstore.S3
}

// GetFile mock implementation.
func (m *MockBlobStoreService) GetFile(ctx context.Context, name string) (*blobstore.File, error) {
	m.GetFileCalls = append(m.GetFileCalls, name)
	if m.GetFileFn != nil {
		return m.GetFileFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// GetS3 mock implementation.
func (m *MockBlobStoreService) GetS3(ctx context.Context, name string) (*blobstore.S3, error) {
	m.GetS3Calls = append(m.GetS3Calls, name)
	if m.GetS3Fn != nil {
		return m.GetS3Fn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateFile mock implementation.
func (m *MockBlobStoreService) CreateFile(ctx context.Context, bs *blobstore.File) error {
	m.CreateFileCalls = append(m.CreateFileCalls, bs)
	if m.CreateFileFn != nil {
		return m.CreateFileFn(ctx, bs)
	}

	return nil
}

// CreateS3 mock implementation.
func (m *MockBlobStoreService) CreateS3(ctx context.Context, bs *blobstore.S3) error {
	m.CreateS3Calls = append(m.CreateS3Calls, bs)
	if m.CreateS3Fn != nil {
		return m.CreateS3Fn(ctx, bs)
	}

	return nil
}

// UpdateFile mock implementation.
func (m *MockBlobStoreService) UpdateFile(ctx context.Context, name string, bs *blobstore.File) error {
	m.UpdateFileCalls = append(m.UpdateFileCalls, UpdateFileCall{Name: name, BlobStore: bs})
	if m.UpdateFileFn != nil {
		return m.UpdateFileFn(ctx, name, bs)
	}

	return nil
}

// UpdateS3 mock implementation.
func (m *MockBlobStoreService) UpdateS3(ctx context.Context, name string, bs *blobstore.S3) error {
	m.UpdateS3Calls = append(m.UpdateS3Calls, UpdateS3Call{Name: name, BlobStore: bs})
	if m.UpdateS3Fn != nil {
		return m.UpdateS3Fn(ctx, name, bs)
	}

	return nil
}

// Delete mock implementation.
func (m *MockBlobStoreService) Delete(ctx context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, name)
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, name)
	}

	return nil
}

// List mock implementation.
func (m *MockBlobStoreService) List(ctx context.Context) ([]blobstore.Generic, error) {
	m.ListCalls++
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}

	return nil, errMockNotConfigured
}

// MockRepositoryService is a mock implementation of nexus.RepositoryService.
type MockRepositoryService struct {
	// Maven
	GetMavenHostedFn    func(ctx context.Context, name string) (*repository.MavenHostedRepository, error)
	CreateMavenHostedFn func(ctx context.Context, repo repository.MavenHostedRepository) error
	UpdateMavenHostedFn func(ctx context.Context, name string, repo repository.MavenHostedRepository) error
	DeleteMavenHostedFn func(ctx context.Context, name string) error

	GetMavenProxyFn    func(ctx context.Context, name string) (*repository.MavenProxyRepository, error)
	CreateMavenProxyFn func(ctx context.Context, repo repository.MavenProxyRepository) error
	UpdateMavenProxyFn func(ctx context.Context, name string, repo repository.MavenProxyRepository) error
	DeleteMavenProxyFn func(ctx context.Context, name string) error

	GetMavenGroupFn    func(ctx context.Context, name string) (*repository.MavenGroupRepository, error)
	CreateMavenGroupFn func(ctx context.Context, repo repository.MavenGroupRepository) error
	UpdateMavenGroupFn func(ctx context.Context, name string, repo repository.MavenGroupRepository) error
	DeleteMavenGroupFn func(ctx context.Context, name string) error

	// Docker
	GetDockerHostedFn    func(ctx context.Context, name string) (*repository.DockerHostedRepository, error)
	CreateDockerHostedFn func(ctx context.Context, repo repository.DockerHostedRepository) error
	UpdateDockerHostedFn func(ctx context.Context, name string, repo repository.DockerHostedRepository) error
	DeleteDockerHostedFn func(ctx context.Context, name string) error

	GetDockerProxyFn    func(ctx context.Context, name string) (*repository.DockerProxyRepository, error)
	CreateDockerProxyFn func(ctx context.Context, repo repository.DockerProxyRepository) error
	UpdateDockerProxyFn func(ctx context.Context, name string, repo repository.DockerProxyRepository) error
	DeleteDockerProxyFn func(ctx context.Context, name string) error

	GetDockerGroupFn    func(ctx context.Context, name string) (*repository.DockerGroupRepository, error)
	CreateDockerGroupFn func(ctx context.Context, repo repository.DockerGroupRepository) error
	UpdateDockerGroupFn func(ctx context.Context, name string, repo repository.DockerGroupRepository) error
	DeleteDockerGroupFn func(ctx context.Context, name string) error

	// npm
	GetNpmHostedFn    func(ctx context.Context, name string) (*repository.NpmHostedRepository, error)
	CreateNpmHostedFn func(ctx context.Context, repo repository.NpmHostedRepository) error
	UpdateNpmHostedFn func(ctx context.Context, name string, repo repository.NpmHostedRepository) error
	DeleteNpmHostedFn func(ctx context.Context, name string) error

	GetNpmProxyFn    func(ctx context.Context, name string) (*repository.NpmProxyRepository, error)
	CreateNpmProxyFn func(ctx context.Context, repo repository.NpmProxyRepository) error
	UpdateNpmProxyFn func(ctx context.Context, name string, repo repository.NpmProxyRepository) error
	DeleteNpmProxyFn func(ctx context.Context, name string) error

	GetNpmGroupFn    func(ctx context.Context, name string) (*repository.NpmGroupRepository, error)
	CreateNpmGroupFn func(ctx context.Context, repo repository.NpmGroupRepository) error
	UpdateNpmGroupFn func(ctx context.Context, name string, repo repository.NpmGroupRepository) error
	DeleteNpmGroupFn func(ctx context.Context, name string) error

	// Raw
	GetRawHostedFn    func(ctx context.Context, name string) (*repository.RawHostedRepository, error)
	CreateRawHostedFn func(ctx context.Context, repo repository.RawHostedRepository) error
	UpdateRawHostedFn func(ctx context.Context, name string, repo repository.RawHostedRepository) error
	DeleteRawHostedFn func(ctx context.Context, name string) error

	GetRawProxyFn    func(ctx context.Context, name string) (*repository.RawProxyRepository, error)
	CreateRawProxyFn func(ctx context.Context, repo repository.RawProxyRepository) error
	UpdateRawProxyFn func(ctx context.Context, name string, repo repository.RawProxyRepository) error
	DeleteRawProxyFn func(ctx context.Context, name string) error

	GetRawGroupFn    func(ctx context.Context, name string) (*repository.RawGroupRepository, error)
	CreateRawGroupFn func(ctx context.Context, repo repository.RawGroupRepository) error
	UpdateRawGroupFn func(ctx context.Context, name string, repo repository.RawGroupRepository) error
	DeleteRawGroupFn func(ctx context.Context, name string) error

	// APT
	GetAptHostedFn    func(ctx context.Context, name string) (*repository.AptHostedRepository, error)
	CreateAptHostedFn func(ctx context.Context, repo repository.AptHostedRepository) error
	UpdateAptHostedFn func(ctx context.Context, name string, repo repository.AptHostedRepository) error
	DeleteAptHostedFn func(ctx context.Context, name string) error

	GetAptProxyFn    func(ctx context.Context, name string) (*repository.AptProxyRepository, error)
	CreateAptProxyFn func(ctx context.Context, repo repository.AptProxyRepository) error
	UpdateAptProxyFn func(ctx context.Context, name string, repo repository.AptProxyRepository) error
	DeleteAptProxyFn func(ctx context.Context, name string) error

	// Bower
	GetBowerHostedFn    func(ctx context.Context, name string) (*repository.BowerHostedRepository, error)
	CreateBowerHostedFn func(ctx context.Context, repo repository.BowerHostedRepository) error
	UpdateBowerHostedFn func(ctx context.Context, name string, repo repository.BowerHostedRepository) error
	DeleteBowerHostedFn func(ctx context.Context, name string) error

	GetBowerProxyFn    func(ctx context.Context, name string) (*repository.BowerProxyRepository, error)
	CreateBowerProxyFn func(ctx context.Context, repo repository.BowerProxyRepository) error
	UpdateBowerProxyFn func(ctx context.Context, name string, repo repository.BowerProxyRepository) error
	DeleteBowerProxyFn func(ctx context.Context, name string) error

	GetBowerGroupFn    func(ctx context.Context, name string) (*repository.BowerGroupRepository, error)
	CreateBowerGroupFn func(ctx context.Context, repo repository.BowerGroupRepository) error
	UpdateBowerGroupFn func(ctx context.Context, name string, repo repository.BowerGroupRepository) error
	DeleteBowerGroupFn func(ctx context.Context, name string) error

	// Cargo
	GetCargoHostedFn    func(ctx context.Context, name string) (*repository.CargoHostedRepository, error)
	CreateCargoHostedFn func(ctx context.Context, repo repository.CargoHostedRepository) error
	UpdateCargoHostedFn func(ctx context.Context, name string, repo repository.CargoHostedRepository) error
	DeleteCargoHostedFn func(ctx context.Context, name string) error

	GetCargoProxyFn    func(ctx context.Context, name string) (*repository.CargoProxyRepository, error)
	CreateCargoProxyFn func(ctx context.Context, repo repository.CargoProxyRepository) error
	UpdateCargoProxyFn func(ctx context.Context, name string, repo repository.CargoProxyRepository) error
	DeleteCargoProxyFn func(ctx context.Context, name string) error

	GetCargoGroupFn    func(ctx context.Context, name string) (*repository.CargoGroupRepository, error)
	CreateCargoGroupFn func(ctx context.Context, repo repository.CargoGroupRepository) error
	UpdateCargoGroupFn func(ctx context.Context, name string, repo repository.CargoGroupRepository) error
	DeleteCargoGroupFn func(ctx context.Context, name string) error

	// Cocoapods
	GetCocoapodsProxyFn    func(ctx context.Context, name string) (*repository.CocoapodsProxyRepository, error)
	CreateCocoapodsProxyFn func(ctx context.Context, repo repository.CocoapodsProxyRepository) error
	UpdateCocoapodsProxyFn func(ctx context.Context, name string, repo repository.CocoapodsProxyRepository) error
	DeleteCocoapodsProxyFn func(ctx context.Context, name string) error

	// Conan
	GetConanProxyFn    func(ctx context.Context, name string) (*repository.ConanProxyRepository, error)
	CreateConanProxyFn func(ctx context.Context, repo repository.ConanProxyRepository) error
	UpdateConanProxyFn func(ctx context.Context, name string, repo repository.ConanProxyRepository) error
	DeleteConanProxyFn func(ctx context.Context, name string) error

	// Conda
	GetCondaProxyFn    func(ctx context.Context, name string) (*repository.CondaProxyRepository, error)
	CreateCondaProxyFn func(ctx context.Context, repo repository.CondaProxyRepository) error
	UpdateCondaProxyFn func(ctx context.Context, name string, repo repository.CondaProxyRepository) error
	DeleteCondaProxyFn func(ctx context.Context, name string) error

	// GitLFS
	GetGitLfsHostedFn    func(ctx context.Context, name string) (*repository.GitLfsHostedRepository, error)
	CreateGitLfsHostedFn func(ctx context.Context, repo repository.GitLfsHostedRepository) error
	UpdateGitLfsHostedFn func(ctx context.Context, name string, repo repository.GitLfsHostedRepository) error
	DeleteGitLfsHostedFn func(ctx context.Context, name string) error

	// Go
	GetGoProxyFn    func(ctx context.Context, name string) (*repository.GoProxyRepository, error)
	CreateGoProxyFn func(ctx context.Context, repo repository.GoProxyRepository) error
	UpdateGoProxyFn func(ctx context.Context, name string, repo repository.GoProxyRepository) error
	DeleteGoProxyFn func(ctx context.Context, name string) error

	GetGoGroupFn    func(ctx context.Context, name string) (*repository.GoGroupRepository, error)
	CreateGoGroupFn func(ctx context.Context, repo repository.GoGroupRepository) error
	UpdateGoGroupFn func(ctx context.Context, name string, repo repository.GoGroupRepository) error
	DeleteGoGroupFn func(ctx context.Context, name string) error

	// Helm
	GetHelmHostedFn    func(ctx context.Context, name string) (*repository.HelmHostedRepository, error)
	CreateHelmHostedFn func(ctx context.Context, repo repository.HelmHostedRepository) error
	UpdateHelmHostedFn func(ctx context.Context, name string, repo repository.HelmHostedRepository) error
	DeleteHelmHostedFn func(ctx context.Context, name string) error

	GetHelmProxyFn    func(ctx context.Context, name string) (*repository.HelmProxyRepository, error)
	CreateHelmProxyFn func(ctx context.Context, repo repository.HelmProxyRepository) error
	UpdateHelmProxyFn func(ctx context.Context, name string, repo repository.HelmProxyRepository) error
	DeleteHelmProxyFn func(ctx context.Context, name string) error

	// NuGet
	GetNugetHostedFn    func(ctx context.Context, name string) (*repository.NugetHostedRepository, error)
	CreateNugetHostedFn func(ctx context.Context, repo repository.NugetHostedRepository) error
	UpdateNugetHostedFn func(ctx context.Context, name string, repo repository.NugetHostedRepository) error
	DeleteNugetHostedFn func(ctx context.Context, name string) error

	GetNugetProxyFn    func(ctx context.Context, name string) (*repository.NugetProxyRepository, error)
	CreateNugetProxyFn func(ctx context.Context, repo repository.NugetProxyRepository) error
	UpdateNugetProxyFn func(ctx context.Context, name string, repo repository.NugetProxyRepository) error
	DeleteNugetProxyFn func(ctx context.Context, name string) error

	GetNugetGroupFn    func(ctx context.Context, name string) (*repository.NugetGroupRepository, error)
	CreateNugetGroupFn func(ctx context.Context, repo repository.NugetGroupRepository) error
	UpdateNugetGroupFn func(ctx context.Context, name string, repo repository.NugetGroupRepository) error
	DeleteNugetGroupFn func(ctx context.Context, name string) error

	// PyPI
	GetPypiHostedFn    func(ctx context.Context, name string) (*repository.PypiHostedRepository, error)
	CreatePypiHostedFn func(ctx context.Context, repo repository.PypiHostedRepository) error
	UpdatePypiHostedFn func(ctx context.Context, name string, repo repository.PypiHostedRepository) error
	DeletePypiHostedFn func(ctx context.Context, name string) error

	GetPypiProxyFn    func(ctx context.Context, name string) (*repository.PypiProxyRepository, error)
	CreatePypiProxyFn func(ctx context.Context, repo repository.PypiProxyRepository) error
	UpdatePypiProxyFn func(ctx context.Context, name string, repo repository.PypiProxyRepository) error
	DeletePypiProxyFn func(ctx context.Context, name string) error

	GetPypiGroupFn    func(ctx context.Context, name string) (*repository.PypiGroupRepository, error)
	CreatePypiGroupFn func(ctx context.Context, repo repository.PypiGroupRepository) error
	UpdatePypiGroupFn func(ctx context.Context, name string, repo repository.PypiGroupRepository) error
	DeletePypiGroupFn func(ctx context.Context, name string) error

	// R
	GetRHostedFn    func(ctx context.Context, name string) (*repository.RHostedRepository, error)
	CreateRHostedFn func(ctx context.Context, repo repository.RHostedRepository) error
	UpdateRHostedFn func(ctx context.Context, name string, repo repository.RHostedRepository) error
	DeleteRHostedFn func(ctx context.Context, name string) error

	GetRProxyFn    func(ctx context.Context, name string) (*repository.RProxyRepository, error)
	CreateRProxyFn func(ctx context.Context, repo repository.RProxyRepository) error
	UpdateRProxyFn func(ctx context.Context, name string, repo repository.RProxyRepository) error
	DeleteRProxyFn func(ctx context.Context, name string) error

	GetRGroupFn    func(ctx context.Context, name string) (*repository.RGroupRepository, error)
	CreateRGroupFn func(ctx context.Context, repo repository.RGroupRepository) error
	UpdateRGroupFn func(ctx context.Context, name string, repo repository.RGroupRepository) error
	DeleteRGroupFn func(ctx context.Context, name string) error

	// RubyGems
	GetRubygemsHostedFn    func(ctx context.Context, name string) (*repository.RubyGemsHostedRepository, error)
	CreateRubygemsHostedFn func(ctx context.Context, repo repository.RubyGemsHostedRepository) error
	UpdateRubygemsHostedFn func(ctx context.Context, name string, repo repository.RubyGemsHostedRepository) error
	DeleteRubygemsHostedFn func(ctx context.Context, name string) error

	GetRubygemsProxyFn    func(ctx context.Context, name string) (*repository.RubyGemsProxyRepository, error)
	CreateRubygemsProxyFn func(ctx context.Context, repo repository.RubyGemsProxyRepository) error
	UpdateRubygemsProxyFn func(ctx context.Context, name string, repo repository.RubyGemsProxyRepository) error
	DeleteRubygemsProxyFn func(ctx context.Context, name string) error

	GetRubygemsGroupFn    func(ctx context.Context, name string) (*repository.RubyGemsGroupRepository, error)
	CreateRubygemsGroupFn func(ctx context.Context, repo repository.RubyGemsGroupRepository) error
	UpdateRubygemsGroupFn func(ctx context.Context, name string, repo repository.RubyGemsGroupRepository) error
	DeleteRubygemsGroupFn func(ctx context.Context, name string) error

	// Yum
	GetYumHostedFn    func(ctx context.Context, name string) (*repository.YumHostedRepository, error)
	CreateYumHostedFn func(ctx context.Context, repo repository.YumHostedRepository) error
	UpdateYumHostedFn func(ctx context.Context, name string, repo repository.YumHostedRepository) error
	DeleteYumHostedFn func(ctx context.Context, name string) error

	GetYumProxyFn    func(ctx context.Context, name string) (*repository.YumProxyRepository, error)
	CreateYumProxyFn func(ctx context.Context, repo repository.YumProxyRepository) error
	UpdateYumProxyFn func(ctx context.Context, name string, repo repository.YumProxyRepository) error
	DeleteYumProxyFn func(ctx context.Context, name string) error

	GetYumGroupFn    func(ctx context.Context, name string) (*repository.YumGroupRepository, error)
	CreateYumGroupFn func(ctx context.Context, repo repository.YumGroupRepository) error
	UpdateYumGroupFn func(ctx context.Context, name string, repo repository.YumGroupRepository) error
	DeleteYumGroupFn func(ctx context.Context, name string) error
}

// GetMavenHosted retrieves a MavenHosted repository by name.
func (m *MockRepositoryService) GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
	if m.GetMavenHostedFn != nil {
		return m.GetMavenHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateMavenHosted creates a new MavenHosted repository.
func (m *MockRepositoryService) CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error {
	if m.CreateMavenHostedFn != nil {
		return m.CreateMavenHostedFn(ctx, repo)
	}

	return nil
}

// UpdateMavenHosted updates an existing MavenHosted repository.
func (m *MockRepositoryService) UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
	if m.UpdateMavenHostedFn != nil {
		return m.UpdateMavenHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteMavenHosted deletes a MavenHosted repository by name.
func (m *MockRepositoryService) DeleteMavenHosted(ctx context.Context, name string) error {
	if m.DeleteMavenHostedFn != nil {
		return m.DeleteMavenHostedFn(ctx, name)
	}

	return nil
}

// GetMavenProxy retrieves a MavenProxy repository by name.
func (m *MockRepositoryService) GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error) {
	if m.GetMavenProxyFn != nil {
		return m.GetMavenProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateMavenProxy creates a new MavenProxy repository.
func (m *MockRepositoryService) CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error {
	if m.CreateMavenProxyFn != nil {
		return m.CreateMavenProxyFn(ctx, repo)
	}

	return nil
}

// UpdateMavenProxy updates an existing MavenProxy repository.
func (m *MockRepositoryService) UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error {
	if m.UpdateMavenProxyFn != nil {
		return m.UpdateMavenProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteMavenProxy deletes a MavenProxy repository by name.
func (m *MockRepositoryService) DeleteMavenProxy(ctx context.Context, name string) error {
	if m.DeleteMavenProxyFn != nil {
		return m.DeleteMavenProxyFn(ctx, name)
	}

	return nil
}

// GetMavenGroup retrieves a MavenGroup repository by name.
func (m *MockRepositoryService) GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error) {
	if m.GetMavenGroupFn != nil {
		return m.GetMavenGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateMavenGroup creates a new MavenGroup repository.
func (m *MockRepositoryService) CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error {
	if m.CreateMavenGroupFn != nil {
		return m.CreateMavenGroupFn(ctx, repo)
	}

	return nil
}

// UpdateMavenGroup updates an existing MavenGroup repository.
func (m *MockRepositoryService) UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error {
	if m.UpdateMavenGroupFn != nil {
		return m.UpdateMavenGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteMavenGroup deletes a MavenGroup repository by name.
func (m *MockRepositoryService) DeleteMavenGroup(ctx context.Context, name string) error {
	if m.DeleteMavenGroupFn != nil {
		return m.DeleteMavenGroupFn(ctx, name)
	}

	return nil
}

// GetDockerHosted retrieves a DockerHosted repository by name.
func (m *MockRepositoryService) GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error) {
	if m.GetDockerHostedFn != nil {
		return m.GetDockerHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateDockerHosted creates a new DockerHosted repository.
func (m *MockRepositoryService) CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error {
	if m.CreateDockerHostedFn != nil {
		return m.CreateDockerHostedFn(ctx, repo)
	}

	return nil
}

// UpdateDockerHosted updates an existing DockerHosted repository.
func (m *MockRepositoryService) UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error {
	if m.UpdateDockerHostedFn != nil {
		return m.UpdateDockerHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteDockerHosted deletes a DockerHosted repository by name.
func (m *MockRepositoryService) DeleteDockerHosted(ctx context.Context, name string) error {
	if m.DeleteDockerHostedFn != nil {
		return m.DeleteDockerHostedFn(ctx, name)
	}

	return nil
}

// GetDockerProxy retrieves a DockerProxy repository by name.
func (m *MockRepositoryService) GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error) {
	if m.GetDockerProxyFn != nil {
		return m.GetDockerProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateDockerProxy creates a new DockerProxy repository.
func (m *MockRepositoryService) CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error {
	if m.CreateDockerProxyFn != nil {
		return m.CreateDockerProxyFn(ctx, repo)
	}

	return nil
}

// UpdateDockerProxy updates an existing DockerProxy repository.
func (m *MockRepositoryService) UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error {
	if m.UpdateDockerProxyFn != nil {
		return m.UpdateDockerProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteDockerProxy deletes a DockerProxy repository by name.
func (m *MockRepositoryService) DeleteDockerProxy(ctx context.Context, name string) error {
	if m.DeleteDockerProxyFn != nil {
		return m.DeleteDockerProxyFn(ctx, name)
	}

	return nil
}

// GetDockerGroup retrieves a DockerGroup repository by name.
func (m *MockRepositoryService) GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error) {
	if m.GetDockerGroupFn != nil {
		return m.GetDockerGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateDockerGroup creates a new DockerGroup repository.
func (m *MockRepositoryService) CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error {
	if m.CreateDockerGroupFn != nil {
		return m.CreateDockerGroupFn(ctx, repo)
	}

	return nil
}

// UpdateDockerGroup updates an existing DockerGroup repository.
func (m *MockRepositoryService) UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error {
	if m.UpdateDockerGroupFn != nil {
		return m.UpdateDockerGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteDockerGroup deletes a DockerGroup repository by name.
func (m *MockRepositoryService) DeleteDockerGroup(ctx context.Context, name string) error {
	if m.DeleteDockerGroupFn != nil {
		return m.DeleteDockerGroupFn(ctx, name)
	}

	return nil
}

// GetNpmHosted retrieves a NpmHosted repository by name.
func (m *MockRepositoryService) GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error) {
	if m.GetNpmHostedFn != nil {
		return m.GetNpmHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNpmHosted creates a new NpmHosted repository.
func (m *MockRepositoryService) CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error {
	if m.CreateNpmHostedFn != nil {
		return m.CreateNpmHostedFn(ctx, repo)
	}

	return nil
}

// UpdateNpmHosted updates an existing NpmHosted repository.
func (m *MockRepositoryService) UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error {
	if m.UpdateNpmHostedFn != nil {
		return m.UpdateNpmHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteNpmHosted deletes a NpmHosted repository by name.
func (m *MockRepositoryService) DeleteNpmHosted(ctx context.Context, name string) error {
	if m.DeleteNpmHostedFn != nil {
		return m.DeleteNpmHostedFn(ctx, name)
	}

	return nil
}

// GetNpmProxy retrieves a NpmProxy repository by name.
func (m *MockRepositoryService) GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error) {
	if m.GetNpmProxyFn != nil {
		return m.GetNpmProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNpmProxy creates a new NpmProxy repository.
func (m *MockRepositoryService) CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error {
	if m.CreateNpmProxyFn != nil {
		return m.CreateNpmProxyFn(ctx, repo)
	}

	return nil
}

// UpdateNpmProxy updates an existing NpmProxy repository.
func (m *MockRepositoryService) UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error {
	if m.UpdateNpmProxyFn != nil {
		return m.UpdateNpmProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteNpmProxy deletes a NpmProxy repository by name.
func (m *MockRepositoryService) DeleteNpmProxy(ctx context.Context, name string) error {
	if m.DeleteNpmProxyFn != nil {
		return m.DeleteNpmProxyFn(ctx, name)
	}

	return nil
}

// GetNpmGroup retrieves a NpmGroup repository by name.
func (m *MockRepositoryService) GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error) {
	if m.GetNpmGroupFn != nil {
		return m.GetNpmGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNpmGroup creates a new NpmGroup repository.
func (m *MockRepositoryService) CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error {
	if m.CreateNpmGroupFn != nil {
		return m.CreateNpmGroupFn(ctx, repo)
	}

	return nil
}

// UpdateNpmGroup updates an existing NpmGroup repository.
func (m *MockRepositoryService) UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error {
	if m.UpdateNpmGroupFn != nil {
		return m.UpdateNpmGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteNpmGroup deletes a NpmGroup repository by name.
func (m *MockRepositoryService) DeleteNpmGroup(ctx context.Context, name string) error {
	if m.DeleteNpmGroupFn != nil {
		return m.DeleteNpmGroupFn(ctx, name)
	}

	return nil
}

// GetRawHosted retrieves a RawHosted repository by name.
func (m *MockRepositoryService) GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error) {
	if m.GetRawHostedFn != nil {
		return m.GetRawHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRawHosted creates a new RawHosted repository.
func (m *MockRepositoryService) CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error {
	if m.CreateRawHostedFn != nil {
		return m.CreateRawHostedFn(ctx, repo)
	}

	return nil
}

// UpdateRawHosted updates an existing RawHosted repository.
func (m *MockRepositoryService) UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error {
	if m.UpdateRawHostedFn != nil {
		return m.UpdateRawHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteRawHosted deletes a RawHosted repository by name.
func (m *MockRepositoryService) DeleteRawHosted(ctx context.Context, name string) error {
	if m.DeleteRawHostedFn != nil {
		return m.DeleteRawHostedFn(ctx, name)
	}

	return nil
}

// GetRawProxy retrieves a RawProxy repository by name.
func (m *MockRepositoryService) GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error) {
	if m.GetRawProxyFn != nil {
		return m.GetRawProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRawProxy creates a new RawProxy repository.
func (m *MockRepositoryService) CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error {
	if m.CreateRawProxyFn != nil {
		return m.CreateRawProxyFn(ctx, repo)
	}

	return nil
}

// UpdateRawProxy updates an existing RawProxy repository.
func (m *MockRepositoryService) UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error {
	if m.UpdateRawProxyFn != nil {
		return m.UpdateRawProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteRawProxy deletes a RawProxy repository by name.
func (m *MockRepositoryService) DeleteRawProxy(ctx context.Context, name string) error {
	if m.DeleteRawProxyFn != nil {
		return m.DeleteRawProxyFn(ctx, name)
	}

	return nil
}

// GetRawGroup retrieves a RawGroup repository by name.
func (m *MockRepositoryService) GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error) {
	if m.GetRawGroupFn != nil {
		return m.GetRawGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRawGroup creates a new RawGroup repository.
func (m *MockRepositoryService) CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error {
	if m.CreateRawGroupFn != nil {
		return m.CreateRawGroupFn(ctx, repo)
	}

	return nil
}

// UpdateRawGroup updates an existing RawGroup repository.
func (m *MockRepositoryService) UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error {
	if m.UpdateRawGroupFn != nil {
		return m.UpdateRawGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteRawGroup deletes a RawGroup repository by name.
func (m *MockRepositoryService) DeleteRawGroup(ctx context.Context, name string) error {
	if m.DeleteRawGroupFn != nil {
		return m.DeleteRawGroupFn(ctx, name)
	}

	return nil
}

// GetAptHosted retrieves a AptHosted repository by name.
func (m *MockRepositoryService) GetAptHosted(ctx context.Context, name string) (*repository.AptHostedRepository, error) {
	if m.GetAptHostedFn != nil {
		return m.GetAptHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateAptHosted creates a new AptHosted repository.
func (m *MockRepositoryService) CreateAptHosted(ctx context.Context, repo repository.AptHostedRepository) error {
	if m.CreateAptHostedFn != nil {
		return m.CreateAptHostedFn(ctx, repo)
	}

	return nil
}

// UpdateAptHosted updates an existing AptHosted repository.
func (m *MockRepositoryService) UpdateAptHosted(ctx context.Context, name string, repo repository.AptHostedRepository) error {
	if m.UpdateAptHostedFn != nil {
		return m.UpdateAptHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteAptHosted deletes a AptHosted repository by name.
func (m *MockRepositoryService) DeleteAptHosted(ctx context.Context, name string) error {
	if m.DeleteAptHostedFn != nil {
		return m.DeleteAptHostedFn(ctx, name)
	}

	return nil
}

// GetAptProxy retrieves a AptProxy repository by name.
func (m *MockRepositoryService) GetAptProxy(ctx context.Context, name string) (*repository.AptProxyRepository, error) {
	if m.GetAptProxyFn != nil {
		return m.GetAptProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateAptProxy creates a new AptProxy repository.
func (m *MockRepositoryService) CreateAptProxy(ctx context.Context, repo repository.AptProxyRepository) error {
	if m.CreateAptProxyFn != nil {
		return m.CreateAptProxyFn(ctx, repo)
	}

	return nil
}

// UpdateAptProxy updates an existing AptProxy repository.
func (m *MockRepositoryService) UpdateAptProxy(ctx context.Context, name string, repo repository.AptProxyRepository) error {
	if m.UpdateAptProxyFn != nil {
		return m.UpdateAptProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteAptProxy deletes a AptProxy repository by name.
func (m *MockRepositoryService) DeleteAptProxy(ctx context.Context, name string) error {
	if m.DeleteAptProxyFn != nil {
		return m.DeleteAptProxyFn(ctx, name)
	}

	return nil
}

// GetBowerHosted retrieves a BowerHosted repository by name.
func (m *MockRepositoryService) GetBowerHosted(ctx context.Context, name string) (*repository.BowerHostedRepository, error) {
	if m.GetBowerHostedFn != nil {
		return m.GetBowerHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateBowerHosted creates a new BowerHosted repository.
func (m *MockRepositoryService) CreateBowerHosted(ctx context.Context, repo repository.BowerHostedRepository) error {
	if m.CreateBowerHostedFn != nil {
		return m.CreateBowerHostedFn(ctx, repo)
	}

	return nil
}

// UpdateBowerHosted updates an existing BowerHosted repository.
func (m *MockRepositoryService) UpdateBowerHosted(ctx context.Context, name string, repo repository.BowerHostedRepository) error {
	if m.UpdateBowerHostedFn != nil {
		return m.UpdateBowerHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteBowerHosted deletes a BowerHosted repository by name.
func (m *MockRepositoryService) DeleteBowerHosted(ctx context.Context, name string) error {
	if m.DeleteBowerHostedFn != nil {
		return m.DeleteBowerHostedFn(ctx, name)
	}

	return nil
}

// GetBowerProxy retrieves a BowerProxy repository by name.
func (m *MockRepositoryService) GetBowerProxy(ctx context.Context, name string) (*repository.BowerProxyRepository, error) {
	if m.GetBowerProxyFn != nil {
		return m.GetBowerProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateBowerProxy creates a new BowerProxy repository.
func (m *MockRepositoryService) CreateBowerProxy(ctx context.Context, repo repository.BowerProxyRepository) error {
	if m.CreateBowerProxyFn != nil {
		return m.CreateBowerProxyFn(ctx, repo)
	}

	return nil
}

// UpdateBowerProxy updates an existing BowerProxy repository.
func (m *MockRepositoryService) UpdateBowerProxy(ctx context.Context, name string, repo repository.BowerProxyRepository) error {
	if m.UpdateBowerProxyFn != nil {
		return m.UpdateBowerProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteBowerProxy deletes a BowerProxy repository by name.
func (m *MockRepositoryService) DeleteBowerProxy(ctx context.Context, name string) error {
	if m.DeleteBowerProxyFn != nil {
		return m.DeleteBowerProxyFn(ctx, name)
	}

	return nil
}

// GetBowerGroup retrieves a BowerGroup repository by name.
func (m *MockRepositoryService) GetBowerGroup(ctx context.Context, name string) (*repository.BowerGroupRepository, error) {
	if m.GetBowerGroupFn != nil {
		return m.GetBowerGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateBowerGroup creates a new BowerGroup repository.
func (m *MockRepositoryService) CreateBowerGroup(ctx context.Context, repo repository.BowerGroupRepository) error {
	if m.CreateBowerGroupFn != nil {
		return m.CreateBowerGroupFn(ctx, repo)
	}

	return nil
}

// UpdateBowerGroup updates an existing BowerGroup repository.
func (m *MockRepositoryService) UpdateBowerGroup(ctx context.Context, name string, repo repository.BowerGroupRepository) error {
	if m.UpdateBowerGroupFn != nil {
		return m.UpdateBowerGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteBowerGroup deletes a BowerGroup repository by name.
func (m *MockRepositoryService) DeleteBowerGroup(ctx context.Context, name string) error {
	if m.DeleteBowerGroupFn != nil {
		return m.DeleteBowerGroupFn(ctx, name)
	}

	return nil
}

// GetCargoHosted retrieves a CargoHosted repository by name.
func (m *MockRepositoryService) GetCargoHosted(ctx context.Context, name string) (*repository.CargoHostedRepository, error) {
	if m.GetCargoHostedFn != nil {
		return m.GetCargoHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCargoHosted creates a new CargoHosted repository.
func (m *MockRepositoryService) CreateCargoHosted(ctx context.Context, repo repository.CargoHostedRepository) error {
	if m.CreateCargoHostedFn != nil {
		return m.CreateCargoHostedFn(ctx, repo)
	}

	return nil
}

// UpdateCargoHosted updates an existing CargoHosted repository.
func (m *MockRepositoryService) UpdateCargoHosted(ctx context.Context, name string, repo repository.CargoHostedRepository) error {
	if m.UpdateCargoHostedFn != nil {
		return m.UpdateCargoHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteCargoHosted deletes a CargoHosted repository by name.
func (m *MockRepositoryService) DeleteCargoHosted(ctx context.Context, name string) error {
	if m.DeleteCargoHostedFn != nil {
		return m.DeleteCargoHostedFn(ctx, name)
	}

	return nil
}

// GetCargoProxy retrieves a CargoProxy repository by name.
func (m *MockRepositoryService) GetCargoProxy(ctx context.Context, name string) (*repository.CargoProxyRepository, error) {
	if m.GetCargoProxyFn != nil {
		return m.GetCargoProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCargoProxy creates a new CargoProxy repository.
func (m *MockRepositoryService) CreateCargoProxy(ctx context.Context, repo repository.CargoProxyRepository) error {
	if m.CreateCargoProxyFn != nil {
		return m.CreateCargoProxyFn(ctx, repo)
	}

	return nil
}

// UpdateCargoProxy updates an existing CargoProxy repository.
func (m *MockRepositoryService) UpdateCargoProxy(ctx context.Context, name string, repo repository.CargoProxyRepository) error {
	if m.UpdateCargoProxyFn != nil {
		return m.UpdateCargoProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteCargoProxy deletes a CargoProxy repository by name.
func (m *MockRepositoryService) DeleteCargoProxy(ctx context.Context, name string) error {
	if m.DeleteCargoProxyFn != nil {
		return m.DeleteCargoProxyFn(ctx, name)
	}

	return nil
}

// GetCargoGroup retrieves a CargoGroup repository by name.
func (m *MockRepositoryService) GetCargoGroup(ctx context.Context, name string) (*repository.CargoGroupRepository, error) {
	if m.GetCargoGroupFn != nil {
		return m.GetCargoGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCargoGroup creates a new CargoGroup repository.
func (m *MockRepositoryService) CreateCargoGroup(ctx context.Context, repo repository.CargoGroupRepository) error {
	if m.CreateCargoGroupFn != nil {
		return m.CreateCargoGroupFn(ctx, repo)
	}

	return nil
}

// UpdateCargoGroup updates an existing CargoGroup repository.
func (m *MockRepositoryService) UpdateCargoGroup(ctx context.Context, name string, repo repository.CargoGroupRepository) error {
	if m.UpdateCargoGroupFn != nil {
		return m.UpdateCargoGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteCargoGroup deletes a CargoGroup repository by name.
func (m *MockRepositoryService) DeleteCargoGroup(ctx context.Context, name string) error {
	if m.DeleteCargoGroupFn != nil {
		return m.DeleteCargoGroupFn(ctx, name)
	}

	return nil
}

// GetCocoapodsProxy retrieves a CocoapodsProxy repository by name.
func (m *MockRepositoryService) GetCocoapodsProxy(ctx context.Context, name string) (*repository.CocoapodsProxyRepository, error) {
	if m.GetCocoapodsProxyFn != nil {
		return m.GetCocoapodsProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCocoapodsProxy creates a new CocoapodsProxy repository.
func (m *MockRepositoryService) CreateCocoapodsProxy(ctx context.Context, repo repository.CocoapodsProxyRepository) error {
	if m.CreateCocoapodsProxyFn != nil {
		return m.CreateCocoapodsProxyFn(ctx, repo)
	}

	return nil
}

// UpdateCocoapodsProxy updates an existing CocoapodsProxy repository.
func (m *MockRepositoryService) UpdateCocoapodsProxy(ctx context.Context, name string, repo repository.CocoapodsProxyRepository) error {
	if m.UpdateCocoapodsProxyFn != nil {
		return m.UpdateCocoapodsProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteCocoapodsProxy deletes a CocoapodsProxy repository by name.
func (m *MockRepositoryService) DeleteCocoapodsProxy(ctx context.Context, name string) error {
	if m.DeleteCocoapodsProxyFn != nil {
		return m.DeleteCocoapodsProxyFn(ctx, name)
	}

	return nil
}

// GetConanProxy retrieves a ConanProxy repository by name.
func (m *MockRepositoryService) GetConanProxy(ctx context.Context, name string) (*repository.ConanProxyRepository, error) {
	if m.GetConanProxyFn != nil {
		return m.GetConanProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateConanProxy creates a new ConanProxy repository.
func (m *MockRepositoryService) CreateConanProxy(ctx context.Context, repo repository.ConanProxyRepository) error {
	if m.CreateConanProxyFn != nil {
		return m.CreateConanProxyFn(ctx, repo)
	}

	return nil
}

// UpdateConanProxy updates an existing ConanProxy repository.
func (m *MockRepositoryService) UpdateConanProxy(ctx context.Context, name string, repo repository.ConanProxyRepository) error {
	if m.UpdateConanProxyFn != nil {
		return m.UpdateConanProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteConanProxy deletes a ConanProxy repository by name.
func (m *MockRepositoryService) DeleteConanProxy(ctx context.Context, name string) error {
	if m.DeleteConanProxyFn != nil {
		return m.DeleteConanProxyFn(ctx, name)
	}

	return nil
}

// GetCondaProxy retrieves a CondaProxy repository by name.
func (m *MockRepositoryService) GetCondaProxy(ctx context.Context, name string) (*repository.CondaProxyRepository, error) {
	if m.GetCondaProxyFn != nil {
		return m.GetCondaProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateCondaProxy creates a new CondaProxy repository.
func (m *MockRepositoryService) CreateCondaProxy(ctx context.Context, repo repository.CondaProxyRepository) error {
	if m.CreateCondaProxyFn != nil {
		return m.CreateCondaProxyFn(ctx, repo)
	}

	return nil
}

// UpdateCondaProxy updates an existing CondaProxy repository.
func (m *MockRepositoryService) UpdateCondaProxy(ctx context.Context, name string, repo repository.CondaProxyRepository) error {
	if m.UpdateCondaProxyFn != nil {
		return m.UpdateCondaProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteCondaProxy deletes a CondaProxy repository by name.
func (m *MockRepositoryService) DeleteCondaProxy(ctx context.Context, name string) error {
	if m.DeleteCondaProxyFn != nil {
		return m.DeleteCondaProxyFn(ctx, name)
	}

	return nil
}

// GetGitLfsHosted retrieves a GitLfsHosted repository by name.
func (m *MockRepositoryService) GetGitLfsHosted(ctx context.Context, name string) (*repository.GitLfsHostedRepository, error) {
	if m.GetGitLfsHostedFn != nil {
		return m.GetGitLfsHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateGitLfsHosted creates a new GitLfsHosted repository.
func (m *MockRepositoryService) CreateGitLfsHosted(ctx context.Context, repo repository.GitLfsHostedRepository) error {
	if m.CreateGitLfsHostedFn != nil {
		return m.CreateGitLfsHostedFn(ctx, repo)
	}

	return nil
}

// UpdateGitLfsHosted updates an existing GitLfsHosted repository.
func (m *MockRepositoryService) UpdateGitLfsHosted(ctx context.Context, name string, repo repository.GitLfsHostedRepository) error {
	if m.UpdateGitLfsHostedFn != nil {
		return m.UpdateGitLfsHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteGitLfsHosted deletes a GitLfsHosted repository by name.
func (m *MockRepositoryService) DeleteGitLfsHosted(ctx context.Context, name string) error {
	if m.DeleteGitLfsHostedFn != nil {
		return m.DeleteGitLfsHostedFn(ctx, name)
	}

	return nil
}

// GetGoProxy retrieves a GoProxy repository by name.
func (m *MockRepositoryService) GetGoProxy(ctx context.Context, name string) (*repository.GoProxyRepository, error) {
	if m.GetGoProxyFn != nil {
		return m.GetGoProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateGoProxy creates a new GoProxy repository.
func (m *MockRepositoryService) CreateGoProxy(ctx context.Context, repo repository.GoProxyRepository) error {
	if m.CreateGoProxyFn != nil {
		return m.CreateGoProxyFn(ctx, repo)
	}

	return nil
}

// UpdateGoProxy updates an existing GoProxy repository.
func (m *MockRepositoryService) UpdateGoProxy(ctx context.Context, name string, repo repository.GoProxyRepository) error {
	if m.UpdateGoProxyFn != nil {
		return m.UpdateGoProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteGoProxy deletes a GoProxy repository by name.
func (m *MockRepositoryService) DeleteGoProxy(ctx context.Context, name string) error {
	if m.DeleteGoProxyFn != nil {
		return m.DeleteGoProxyFn(ctx, name)
	}

	return nil
}

// GetGoGroup retrieves a GoGroup repository by name.
func (m *MockRepositoryService) GetGoGroup(ctx context.Context, name string) (*repository.GoGroupRepository, error) {
	if m.GetGoGroupFn != nil {
		return m.GetGoGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateGoGroup creates a new GoGroup repository.
func (m *MockRepositoryService) CreateGoGroup(ctx context.Context, repo repository.GoGroupRepository) error {
	if m.CreateGoGroupFn != nil {
		return m.CreateGoGroupFn(ctx, repo)
	}

	return nil
}

// UpdateGoGroup updates an existing GoGroup repository.
func (m *MockRepositoryService) UpdateGoGroup(ctx context.Context, name string, repo repository.GoGroupRepository) error {
	if m.UpdateGoGroupFn != nil {
		return m.UpdateGoGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteGoGroup deletes a GoGroup repository by name.
func (m *MockRepositoryService) DeleteGoGroup(ctx context.Context, name string) error {
	if m.DeleteGoGroupFn != nil {
		return m.DeleteGoGroupFn(ctx, name)
	}

	return nil
}

// GetHelmHosted retrieves a HelmHosted repository by name.
func (m *MockRepositoryService) GetHelmHosted(ctx context.Context, name string) (*repository.HelmHostedRepository, error) {
	if m.GetHelmHostedFn != nil {
		return m.GetHelmHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateHelmHosted creates a new HelmHosted repository.
func (m *MockRepositoryService) CreateHelmHosted(ctx context.Context, repo repository.HelmHostedRepository) error {
	if m.CreateHelmHostedFn != nil {
		return m.CreateHelmHostedFn(ctx, repo)
	}

	return nil
}

// UpdateHelmHosted updates an existing HelmHosted repository.
func (m *MockRepositoryService) UpdateHelmHosted(ctx context.Context, name string, repo repository.HelmHostedRepository) error {
	if m.UpdateHelmHostedFn != nil {
		return m.UpdateHelmHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteHelmHosted deletes a HelmHosted repository by name.
func (m *MockRepositoryService) DeleteHelmHosted(ctx context.Context, name string) error {
	if m.DeleteHelmHostedFn != nil {
		return m.DeleteHelmHostedFn(ctx, name)
	}

	return nil
}

// GetHelmProxy retrieves a HelmProxy repository by name.
func (m *MockRepositoryService) GetHelmProxy(ctx context.Context, name string) (*repository.HelmProxyRepository, error) {
	if m.GetHelmProxyFn != nil {
		return m.GetHelmProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateHelmProxy creates a new HelmProxy repository.
func (m *MockRepositoryService) CreateHelmProxy(ctx context.Context, repo repository.HelmProxyRepository) error {
	if m.CreateHelmProxyFn != nil {
		return m.CreateHelmProxyFn(ctx, repo)
	}

	return nil
}

// UpdateHelmProxy updates an existing HelmProxy repository.
func (m *MockRepositoryService) UpdateHelmProxy(ctx context.Context, name string, repo repository.HelmProxyRepository) error {
	if m.UpdateHelmProxyFn != nil {
		return m.UpdateHelmProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteHelmProxy deletes a HelmProxy repository by name.
func (m *MockRepositoryService) DeleteHelmProxy(ctx context.Context, name string) error {
	if m.DeleteHelmProxyFn != nil {
		return m.DeleteHelmProxyFn(ctx, name)
	}

	return nil
}

// GetNugetHosted retrieves a NugetHosted repository by name.
func (m *MockRepositoryService) GetNugetHosted(ctx context.Context, name string) (*repository.NugetHostedRepository, error) {
	if m.GetNugetHostedFn != nil {
		return m.GetNugetHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNugetHosted creates a new NugetHosted repository.
func (m *MockRepositoryService) CreateNugetHosted(ctx context.Context, repo repository.NugetHostedRepository) error {
	if m.CreateNugetHostedFn != nil {
		return m.CreateNugetHostedFn(ctx, repo)
	}

	return nil
}

// UpdateNugetHosted updates an existing NugetHosted repository.
func (m *MockRepositoryService) UpdateNugetHosted(ctx context.Context, name string, repo repository.NugetHostedRepository) error {
	if m.UpdateNugetHostedFn != nil {
		return m.UpdateNugetHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteNugetHosted deletes a NugetHosted repository by name.
func (m *MockRepositoryService) DeleteNugetHosted(ctx context.Context, name string) error {
	if m.DeleteNugetHostedFn != nil {
		return m.DeleteNugetHostedFn(ctx, name)
	}

	return nil
}

// GetNugetProxy retrieves a NugetProxy repository by name.
func (m *MockRepositoryService) GetNugetProxy(ctx context.Context, name string) (*repository.NugetProxyRepository, error) {
	if m.GetNugetProxyFn != nil {
		return m.GetNugetProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNugetProxy creates a new NugetProxy repository.
func (m *MockRepositoryService) CreateNugetProxy(ctx context.Context, repo repository.NugetProxyRepository) error {
	if m.CreateNugetProxyFn != nil {
		return m.CreateNugetProxyFn(ctx, repo)
	}

	return nil
}

// UpdateNugetProxy updates an existing NugetProxy repository.
func (m *MockRepositoryService) UpdateNugetProxy(ctx context.Context, name string, repo repository.NugetProxyRepository) error {
	if m.UpdateNugetProxyFn != nil {
		return m.UpdateNugetProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteNugetProxy deletes a NugetProxy repository by name.
func (m *MockRepositoryService) DeleteNugetProxy(ctx context.Context, name string) error {
	if m.DeleteNugetProxyFn != nil {
		return m.DeleteNugetProxyFn(ctx, name)
	}

	return nil
}

// GetNugetGroup retrieves a NugetGroup repository by name.
func (m *MockRepositoryService) GetNugetGroup(ctx context.Context, name string) (*repository.NugetGroupRepository, error) {
	if m.GetNugetGroupFn != nil {
		return m.GetNugetGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateNugetGroup creates a new NugetGroup repository.
func (m *MockRepositoryService) CreateNugetGroup(ctx context.Context, repo repository.NugetGroupRepository) error {
	if m.CreateNugetGroupFn != nil {
		return m.CreateNugetGroupFn(ctx, repo)
	}

	return nil
}

// UpdateNugetGroup updates an existing NugetGroup repository.
func (m *MockRepositoryService) UpdateNugetGroup(ctx context.Context, name string, repo repository.NugetGroupRepository) error {
	if m.UpdateNugetGroupFn != nil {
		return m.UpdateNugetGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteNugetGroup deletes a NugetGroup repository by name.
func (m *MockRepositoryService) DeleteNugetGroup(ctx context.Context, name string) error {
	if m.DeleteNugetGroupFn != nil {
		return m.DeleteNugetGroupFn(ctx, name)
	}

	return nil
}

// GetPypiHosted retrieves a PypiHosted repository by name.
func (m *MockRepositoryService) GetPypiHosted(ctx context.Context, name string) (*repository.PypiHostedRepository, error) {
	if m.GetPypiHostedFn != nil {
		return m.GetPypiHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreatePypiHosted creates a new PypiHosted repository.
func (m *MockRepositoryService) CreatePypiHosted(ctx context.Context, repo repository.PypiHostedRepository) error {
	if m.CreatePypiHostedFn != nil {
		return m.CreatePypiHostedFn(ctx, repo)
	}

	return nil
}

// UpdatePypiHosted updates an existing PypiHosted repository.
func (m *MockRepositoryService) UpdatePypiHosted(ctx context.Context, name string, repo repository.PypiHostedRepository) error {
	if m.UpdatePypiHostedFn != nil {
		return m.UpdatePypiHostedFn(ctx, name, repo)
	}

	return nil
}

// DeletePypiHosted deletes a PypiHosted repository by name.
func (m *MockRepositoryService) DeletePypiHosted(ctx context.Context, name string) error {
	if m.DeletePypiHostedFn != nil {
		return m.DeletePypiHostedFn(ctx, name)
	}

	return nil
}

// GetPypiProxy retrieves a PypiProxy repository by name.
func (m *MockRepositoryService) GetPypiProxy(ctx context.Context, name string) (*repository.PypiProxyRepository, error) {
	if m.GetPypiProxyFn != nil {
		return m.GetPypiProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreatePypiProxy creates a new PypiProxy repository.
func (m *MockRepositoryService) CreatePypiProxy(ctx context.Context, repo repository.PypiProxyRepository) error {
	if m.CreatePypiProxyFn != nil {
		return m.CreatePypiProxyFn(ctx, repo)
	}

	return nil
}

// UpdatePypiProxy updates an existing PypiProxy repository.
func (m *MockRepositoryService) UpdatePypiProxy(ctx context.Context, name string, repo repository.PypiProxyRepository) error {
	if m.UpdatePypiProxyFn != nil {
		return m.UpdatePypiProxyFn(ctx, name, repo)
	}

	return nil
}

// DeletePypiProxy deletes a PypiProxy repository by name.
func (m *MockRepositoryService) DeletePypiProxy(ctx context.Context, name string) error {
	if m.DeletePypiProxyFn != nil {
		return m.DeletePypiProxyFn(ctx, name)
	}

	return nil
}

// GetPypiGroup retrieves a PypiGroup repository by name.
func (m *MockRepositoryService) GetPypiGroup(ctx context.Context, name string) (*repository.PypiGroupRepository, error) {
	if m.GetPypiGroupFn != nil {
		return m.GetPypiGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreatePypiGroup creates a new PypiGroup repository.
func (m *MockRepositoryService) CreatePypiGroup(ctx context.Context, repo repository.PypiGroupRepository) error {
	if m.CreatePypiGroupFn != nil {
		return m.CreatePypiGroupFn(ctx, repo)
	}

	return nil
}

// UpdatePypiGroup updates an existing PypiGroup repository.
func (m *MockRepositoryService) UpdatePypiGroup(ctx context.Context, name string, repo repository.PypiGroupRepository) error {
	if m.UpdatePypiGroupFn != nil {
		return m.UpdatePypiGroupFn(ctx, name, repo)
	}

	return nil
}

// DeletePypiGroup deletes a PypiGroup repository by name.
func (m *MockRepositoryService) DeletePypiGroup(ctx context.Context, name string) error {
	if m.DeletePypiGroupFn != nil {
		return m.DeletePypiGroupFn(ctx, name)
	}

	return nil
}

// GetRHosted retrieves a RHosted repository by name.
func (m *MockRepositoryService) GetRHosted(ctx context.Context, name string) (*repository.RHostedRepository, error) {
	if m.GetRHostedFn != nil {
		return m.GetRHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRHosted creates a new RHosted repository.
func (m *MockRepositoryService) CreateRHosted(ctx context.Context, repo repository.RHostedRepository) error {
	if m.CreateRHostedFn != nil {
		return m.CreateRHostedFn(ctx, repo)
	}

	return nil
}

// UpdateRHosted updates an existing RHosted repository.
func (m *MockRepositoryService) UpdateRHosted(ctx context.Context, name string, repo repository.RHostedRepository) error {
	if m.UpdateRHostedFn != nil {
		return m.UpdateRHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteRHosted deletes a RHosted repository by name.
func (m *MockRepositoryService) DeleteRHosted(ctx context.Context, name string) error {
	if m.DeleteRHostedFn != nil {
		return m.DeleteRHostedFn(ctx, name)
	}

	return nil
}

// GetRProxy retrieves a RProxy repository by name.
func (m *MockRepositoryService) GetRProxy(ctx context.Context, name string) (*repository.RProxyRepository, error) {
	if m.GetRProxyFn != nil {
		return m.GetRProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRProxy creates a new RProxy repository.
func (m *MockRepositoryService) CreateRProxy(ctx context.Context, repo repository.RProxyRepository) error {
	if m.CreateRProxyFn != nil {
		return m.CreateRProxyFn(ctx, repo)
	}

	return nil
}

// UpdateRProxy updates an existing RProxy repository.
func (m *MockRepositoryService) UpdateRProxy(ctx context.Context, name string, repo repository.RProxyRepository) error {
	if m.UpdateRProxyFn != nil {
		return m.UpdateRProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteRProxy deletes a RProxy repository by name.
func (m *MockRepositoryService) DeleteRProxy(ctx context.Context, name string) error {
	if m.DeleteRProxyFn != nil {
		return m.DeleteRProxyFn(ctx, name)
	}

	return nil
}

// GetRGroup retrieves a RGroup repository by name.
func (m *MockRepositoryService) GetRGroup(ctx context.Context, name string) (*repository.RGroupRepository, error) {
	if m.GetRGroupFn != nil {
		return m.GetRGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRGroup creates a new RGroup repository.
func (m *MockRepositoryService) CreateRGroup(ctx context.Context, repo repository.RGroupRepository) error {
	if m.CreateRGroupFn != nil {
		return m.CreateRGroupFn(ctx, repo)
	}

	return nil
}

// UpdateRGroup updates an existing RGroup repository.
func (m *MockRepositoryService) UpdateRGroup(ctx context.Context, name string, repo repository.RGroupRepository) error {
	if m.UpdateRGroupFn != nil {
		return m.UpdateRGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteRGroup deletes a RGroup repository by name.
func (m *MockRepositoryService) DeleteRGroup(ctx context.Context, name string) error {
	if m.DeleteRGroupFn != nil {
		return m.DeleteRGroupFn(ctx, name)
	}

	return nil
}

// GetRubygemsHosted retrieves a RubygemsHosted repository by name.
func (m *MockRepositoryService) GetRubygemsHosted(ctx context.Context, name string) (*repository.RubyGemsHostedRepository, error) {
	if m.GetRubygemsHostedFn != nil {
		return m.GetRubygemsHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRubygemsHosted creates a new RubygemsHosted repository.
func (m *MockRepositoryService) CreateRubygemsHosted(ctx context.Context, repo repository.RubyGemsHostedRepository) error {
	if m.CreateRubygemsHostedFn != nil {
		return m.CreateRubygemsHostedFn(ctx, repo)
	}

	return nil
}

// UpdateRubygemsHosted updates an existing RubygemsHosted repository.
func (m *MockRepositoryService) UpdateRubygemsHosted(ctx context.Context, name string, repo repository.RubyGemsHostedRepository) error {
	if m.UpdateRubygemsHostedFn != nil {
		return m.UpdateRubygemsHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteRubygemsHosted deletes a RubygemsHosted repository by name.
func (m *MockRepositoryService) DeleteRubygemsHosted(ctx context.Context, name string) error {
	if m.DeleteRubygemsHostedFn != nil {
		return m.DeleteRubygemsHostedFn(ctx, name)
	}

	return nil
}

// GetRubygemsProxy retrieves a RubygemsProxy repository by name.
func (m *MockRepositoryService) GetRubygemsProxy(ctx context.Context, name string) (*repository.RubyGemsProxyRepository, error) {
	if m.GetRubygemsProxyFn != nil {
		return m.GetRubygemsProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRubygemsProxy creates a new RubygemsProxy repository.
func (m *MockRepositoryService) CreateRubygemsProxy(ctx context.Context, repo repository.RubyGemsProxyRepository) error {
	if m.CreateRubygemsProxyFn != nil {
		return m.CreateRubygemsProxyFn(ctx, repo)
	}

	return nil
}

// UpdateRubygemsProxy updates an existing RubygemsProxy repository.
func (m *MockRepositoryService) UpdateRubygemsProxy(ctx context.Context, name string, repo repository.RubyGemsProxyRepository) error {
	if m.UpdateRubygemsProxyFn != nil {
		return m.UpdateRubygemsProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteRubygemsProxy deletes a RubygemsProxy repository by name.
func (m *MockRepositoryService) DeleteRubygemsProxy(ctx context.Context, name string) error {
	if m.DeleteRubygemsProxyFn != nil {
		return m.DeleteRubygemsProxyFn(ctx, name)
	}

	return nil
}

// GetRubygemsGroup retrieves a RubygemsGroup repository by name.
func (m *MockRepositoryService) GetRubygemsGroup(ctx context.Context, name string) (*repository.RubyGemsGroupRepository, error) {
	if m.GetRubygemsGroupFn != nil {
		return m.GetRubygemsGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateRubygemsGroup creates a new RubygemsGroup repository.
func (m *MockRepositoryService) CreateRubygemsGroup(ctx context.Context, repo repository.RubyGemsGroupRepository) error {
	if m.CreateRubygemsGroupFn != nil {
		return m.CreateRubygemsGroupFn(ctx, repo)
	}

	return nil
}

// UpdateRubygemsGroup updates an existing RubygemsGroup repository.
func (m *MockRepositoryService) UpdateRubygemsGroup(ctx context.Context, name string, repo repository.RubyGemsGroupRepository) error {
	if m.UpdateRubygemsGroupFn != nil {
		return m.UpdateRubygemsGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteRubygemsGroup deletes a RubygemsGroup repository by name.
func (m *MockRepositoryService) DeleteRubygemsGroup(ctx context.Context, name string) error {
	if m.DeleteRubygemsGroupFn != nil {
		return m.DeleteRubygemsGroupFn(ctx, name)
	}

	return nil
}

// GetYumHosted retrieves a YumHosted repository by name.
func (m *MockRepositoryService) GetYumHosted(ctx context.Context, name string) (*repository.YumHostedRepository, error) {
	if m.GetYumHostedFn != nil {
		return m.GetYumHostedFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateYumHosted creates a new YumHosted repository.
func (m *MockRepositoryService) CreateYumHosted(ctx context.Context, repo repository.YumHostedRepository) error {
	if m.CreateYumHostedFn != nil {
		return m.CreateYumHostedFn(ctx, repo)
	}

	return nil
}

// UpdateYumHosted updates an existing YumHosted repository.
func (m *MockRepositoryService) UpdateYumHosted(ctx context.Context, name string, repo repository.YumHostedRepository) error {
	if m.UpdateYumHostedFn != nil {
		return m.UpdateYumHostedFn(ctx, name, repo)
	}

	return nil
}

// DeleteYumHosted deletes a YumHosted repository by name.
func (m *MockRepositoryService) DeleteYumHosted(ctx context.Context, name string) error {
	if m.DeleteYumHostedFn != nil {
		return m.DeleteYumHostedFn(ctx, name)
	}

	return nil
}

// GetYumProxy retrieves a YumProxy repository by name.
func (m *MockRepositoryService) GetYumProxy(ctx context.Context, name string) (*repository.YumProxyRepository, error) {
	if m.GetYumProxyFn != nil {
		return m.GetYumProxyFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateYumProxy creates a new YumProxy repository.
func (m *MockRepositoryService) CreateYumProxy(ctx context.Context, repo repository.YumProxyRepository) error {
	if m.CreateYumProxyFn != nil {
		return m.CreateYumProxyFn(ctx, repo)
	}

	return nil
}

// UpdateYumProxy updates an existing YumProxy repository.
func (m *MockRepositoryService) UpdateYumProxy(ctx context.Context, name string, repo repository.YumProxyRepository) error {
	if m.UpdateYumProxyFn != nil {
		return m.UpdateYumProxyFn(ctx, name, repo)
	}

	return nil
}

// DeleteYumProxy deletes a YumProxy repository by name.
func (m *MockRepositoryService) DeleteYumProxy(ctx context.Context, name string) error {
	if m.DeleteYumProxyFn != nil {
		return m.DeleteYumProxyFn(ctx, name)
	}

	return nil
}

// GetYumGroup retrieves a YumGroup repository by name.
func (m *MockRepositoryService) GetYumGroup(ctx context.Context, name string) (*repository.YumGroupRepository, error) {
	if m.GetYumGroupFn != nil {
		return m.GetYumGroupFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// CreateYumGroup creates a new YumGroup repository.
func (m *MockRepositoryService) CreateYumGroup(ctx context.Context, repo repository.YumGroupRepository) error {
	if m.CreateYumGroupFn != nil {
		return m.CreateYumGroupFn(ctx, repo)
	}

	return nil
}

// UpdateYumGroup updates an existing YumGroup repository.
func (m *MockRepositoryService) UpdateYumGroup(ctx context.Context, name string, repo repository.YumGroupRepository) error {
	if m.UpdateYumGroupFn != nil {
		return m.UpdateYumGroupFn(ctx, name, repo)
	}

	return nil
}

// DeleteYumGroup deletes a YumGroup repository by name.
func (m *MockRepositoryService) DeleteYumGroup(ctx context.Context, name string) error {
	if m.DeleteYumGroupFn != nil {
		return m.DeleteYumGroupFn(ctx, name)
	}

	return nil
}

// MockSecurityService is a mock implementation of nexus.SecurityService.
type MockSecurityService struct {
	// User methods
	GetUserFn        func(ctx context.Context, id string) (*security.User, error)
	CreateUserFn     func(ctx context.Context, user security.User) error
	UpdateUserFn     func(ctx context.Context, id string, user security.User) error
	DeleteUserFn     func(ctx context.Context, id string) error
	ChangePasswordFn func(ctx context.Context, id, password string) error

	// Role methods
	GetRoleFn    func(ctx context.Context, id string) (*security.Role, error)
	CreateRoleFn func(ctx context.Context, role security.Role) error
	UpdateRoleFn func(ctx context.Context, id string, role security.Role) error
	DeleteRoleFn func(ctx context.Context, id string) error

	// Content Selector methods
	GetContentSelectorFn    func(ctx context.Context, name string) (*security.ContentSelector, error)
	CreateContentSelectorFn func(ctx context.Context, cs security.ContentSelector) error
	UpdateContentSelectorFn func(ctx context.Context, name string, cs security.ContentSelector) error
	DeleteContentSelectorFn func(ctx context.Context, name string) error

	// Privilege methods
	GetPrivilegeFn                  func(ctx context.Context, name string) (*security.Privilege, error)
	DeletePrivilegeFn               func(ctx context.Context, name string) error
	CreatePrivilegeApplicationFn    func(ctx context.Context, p security.PrivilegeApplication) error
	UpdatePrivilegeApplicationFn    func(ctx context.Context, name string, p security.PrivilegeApplication) error
	CreatePrivilegeRepositoryViewFn func(ctx context.Context, p security.PrivilegeRepositoryView) error
	UpdatePrivilegeRepositoryViewFn func(ctx context.Context, name string, p security.PrivilegeRepositoryView) error
	CreatePrivilegeWildcardFn       func(ctx context.Context, p security.PrivilegeWildcard) error
	UpdatePrivilegeWildcardFn       func(ctx context.Context, name string, p security.PrivilegeWildcard) error

	// Realm methods
	ListActiveRealmsFn func(ctx context.Context) ([]string, error)
	ActivateRealmsFn   func(ctx context.Context, ids []string) error

	// Anonymous Access methods
	GetAnonymousAccessFn    func(ctx context.Context) (*security.AnonymousAccessSettings, error)
	UpdateAnonymousAccessFn func(ctx context.Context, settings security.AnonymousAccessSettings) error

	// User Token methods
	GetUserTokenConfigFn    func(ctx context.Context) (*security.UserTokenConfiguration, error)
	UpdateUserTokenConfigFn func(ctx context.Context, config security.UserTokenConfiguration) error

	// LDAP methods
	GetLDAPFn    func(ctx context.Context, name string) (*security.LDAP, error)
	CreateLDAPFn func(ctx context.Context, ldap security.LDAP) error
	UpdateLDAPFn func(ctx context.Context, name string, ldap security.LDAP) error
	DeleteLDAPFn func(ctx context.Context, name string) error

	// SAML methods
	GetSAMLFn    func(ctx context.Context) (*security.SAML, error)
	UpdateSAMLFn func(ctx context.Context, saml security.SAML) error
	DeleteSAMLFn func(ctx context.Context) error

	// Call tracking
	GetUserCalls        []string
	CreateUserCalls     []security.User
	UpdateUserCalls     []UpdateUserCall
	DeleteUserCalls     []string
	ChangePasswordCalls []ChangePasswordCall
	GetRoleCalls        []string
	CreateRoleCalls     []security.Role
	UpdateRoleCalls     []UpdateRoleCall
	DeleteRoleCalls     []string
}

// UpdateUserCall tracks calls to UpdateUser.
type UpdateUserCall struct {
	ID   string
	User security.User
}

// ChangePasswordCall tracks calls to ChangePassword.
type ChangePasswordCall struct {
	ID       string
	Password string
}

// UpdateRoleCall tracks calls to UpdateRole.
type UpdateRoleCall struct {
	ID   string
	Role security.Role
}

// GetUser mock implementation.
func (m *MockSecurityService) GetUser(ctx context.Context, id string) (*security.User, error) {
	m.GetUserCalls = append(m.GetUserCalls, id)
	if m.GetUserFn != nil {
		return m.GetUserFn(ctx, id)
	}

	return nil, errMockNotConfigured
}

// CreateUser mock implementation.
func (m *MockSecurityService) CreateUser(ctx context.Context, user security.User) error {
	m.CreateUserCalls = append(m.CreateUserCalls, user)
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, user)
	}

	return nil
}

// UpdateUser mock implementation.
func (m *MockSecurityService) UpdateUser(ctx context.Context, id string, user security.User) error {
	m.UpdateUserCalls = append(m.UpdateUserCalls, UpdateUserCall{ID: id, User: user})
	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, id, user)
	}

	return nil
}

// DeleteUser mock implementation.
func (m *MockSecurityService) DeleteUser(ctx context.Context, id string) error {
	m.DeleteUserCalls = append(m.DeleteUserCalls, id)
	if m.DeleteUserFn != nil {
		return m.DeleteUserFn(ctx, id)
	}

	return nil
}

// ChangePassword mock implementation.
func (m *MockSecurityService) ChangePassword(ctx context.Context, id, password string) error {
	m.ChangePasswordCalls = append(m.ChangePasswordCalls, ChangePasswordCall{ID: id, Password: password})
	if m.ChangePasswordFn != nil {
		return m.ChangePasswordFn(ctx, id, password)
	}

	return nil
}

// GetRole mock implementation.
func (m *MockSecurityService) GetRole(ctx context.Context, id string) (*security.Role, error) {
	m.GetRoleCalls = append(m.GetRoleCalls, id)
	if m.GetRoleFn != nil {
		return m.GetRoleFn(ctx, id)
	}

	return nil, errMockNotConfigured
}

// CreateRole mock implementation.
func (m *MockSecurityService) CreateRole(ctx context.Context, role security.Role) error {
	m.CreateRoleCalls = append(m.CreateRoleCalls, role)
	if m.CreateRoleFn != nil {
		return m.CreateRoleFn(ctx, role)
	}

	return nil
}

// UpdateRole mock implementation.
func (m *MockSecurityService) UpdateRole(ctx context.Context, id string, role security.Role) error {
	m.UpdateRoleCalls = append(m.UpdateRoleCalls, UpdateRoleCall{ID: id, Role: role})
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, id, role)
	}

	return nil
}

// DeleteRole mock implementation.
func (m *MockSecurityService) DeleteRole(ctx context.Context, id string) error {
	m.DeleteRoleCalls = append(m.DeleteRoleCalls, id)
	if m.DeleteRoleFn != nil {
		return m.DeleteRoleFn(ctx, id)
	}

	return nil
}

// ListAvailableRealms returns all available realm identifiers.
func (m *MockSecurityService) ListAvailableRealms(ctx context.Context) ([]security.Realm, error) {
	return nil, errMockNotConfigured
}

// ListActiveRealms returns the list of currently active realm identifiers.
func (m *MockSecurityService) ListActiveRealms(ctx context.Context) ([]string, error) {
	if m.ListActiveRealmsFn != nil {
		return m.ListActiveRealmsFn(ctx)
	}

	return nil, errMockNotConfigured
}

// ActivateRealms sets the active realms to the provided list of identifiers.
func (m *MockSecurityService) ActivateRealms(ctx context.Context, ids []string) error {
	if m.ActivateRealmsFn != nil {
		return m.ActivateRealmsFn(ctx, ids)
	}

	return nil
}

// GetContentSelector retrieves a content selector by name.
func (m *MockSecurityService) GetContentSelector(ctx context.Context, name string) (*security.ContentSelector, error) {
	if m.GetContentSelectorFn != nil {
		return m.GetContentSelectorFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// ListContentSelectors returns all content selectors.
func (m *MockSecurityService) ListContentSelectors(ctx context.Context) ([]security.ContentSelector, error) {
	return nil, errMockNotConfigured
}

// CreateContentSelector creates a new content selector.
func (m *MockSecurityService) CreateContentSelector(ctx context.Context, cs security.ContentSelector) error {
	if m.CreateContentSelectorFn != nil {
		return m.CreateContentSelectorFn(ctx, cs)
	}

	return nil
}

// UpdateContentSelector updates an existing content selector.
func (m *MockSecurityService) UpdateContentSelector(ctx context.Context, name string, cs security.ContentSelector) error {
	if m.UpdateContentSelectorFn != nil {
		return m.UpdateContentSelectorFn(ctx, name, cs)
	}

	return nil
}

// DeleteContentSelector deletes a content selector by name.
func (m *MockSecurityService) DeleteContentSelector(ctx context.Context, name string) error {
	if m.DeleteContentSelectorFn != nil {
		return m.DeleteContentSelectorFn(ctx, name)
	}

	return nil
}

// GetPrivilege retrieves a privilege by name.
func (m *MockSecurityService) GetPrivilege(ctx context.Context, name string) (*security.Privilege, error) {
	if m.GetPrivilegeFn != nil {
		return m.GetPrivilegeFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// ListPrivileges returns all privileges.
func (m *MockSecurityService) ListPrivileges(ctx context.Context) ([]security.Privilege, error) {
	return nil, errMockNotConfigured
}

// DeletePrivilege deletes a privilege by name.
func (m *MockSecurityService) DeletePrivilege(ctx context.Context, name string) error {
	if m.DeletePrivilegeFn != nil {
		return m.DeletePrivilegeFn(ctx, name)
	}

	return nil
}

// CreatePrivilegeApplication creates a new application privilege.
func (m *MockSecurityService) CreatePrivilegeApplication(ctx context.Context, p security.PrivilegeApplication) error {
	if m.CreatePrivilegeApplicationFn != nil {
		return m.CreatePrivilegeApplicationFn(ctx, p)
	}

	return nil
}

// UpdatePrivilegeApplication updates an existing application privilege.
func (m *MockSecurityService) UpdatePrivilegeApplication(ctx context.Context, name string, p security.PrivilegeApplication) error {
	if m.UpdatePrivilegeApplicationFn != nil {
		return m.UpdatePrivilegeApplicationFn(ctx, name, p)
	}

	return nil
}

// CreatePrivilegeRepositoryView creates a new repository view privilege.
func (m *MockSecurityService) CreatePrivilegeRepositoryView(ctx context.Context, p security.PrivilegeRepositoryView) error {
	if m.CreatePrivilegeRepositoryViewFn != nil {
		return m.CreatePrivilegeRepositoryViewFn(ctx, p)
	}

	return nil
}

// UpdatePrivilegeRepositoryView updates an existing repository view privilege.
func (m *MockSecurityService) UpdatePrivilegeRepositoryView(ctx context.Context, name string, p security.PrivilegeRepositoryView) error {
	if m.UpdatePrivilegeRepositoryViewFn != nil {
		return m.UpdatePrivilegeRepositoryViewFn(ctx, name, p)
	}

	return nil
}

// CreatePrivilegeRepositoryAdmin creates a new repository admin privilege.
func (m *MockSecurityService) CreatePrivilegeRepositoryAdmin(ctx context.Context, p security.PrivilegeRepositoryAdmin) error {
	return nil
}

// UpdatePrivilegeRepositoryAdmin updates an existing repository admin
// privilege.
func (m *MockSecurityService) UpdatePrivilegeRepositoryAdmin(ctx context.Context, name string, p security.PrivilegeRepositoryAdmin) error {
	return nil
}

// CreatePrivilegeRepositoryContentSelector creates a new repository content
// selector privilege.
func (m *MockSecurityService) CreatePrivilegeRepositoryContentSelector(ctx context.Context, p security.PrivilegeRepositoryContentSelector) error {
	return nil
}

// UpdatePrivilegeRepositoryContentSelector updates an existing repository
// content selector privilege.
func (m *MockSecurityService) UpdatePrivilegeRepositoryContentSelector(ctx context.Context, name string, p security.PrivilegeRepositoryContentSelector) error {
	return nil
}

// CreatePrivilegeScript creates a new script privilege.
func (m *MockSecurityService) CreatePrivilegeScript(ctx context.Context, p security.PrivilegeScript) error {
	return nil
}

// UpdatePrivilegeScript updates an existing script privilege.
func (m *MockSecurityService) UpdatePrivilegeScript(ctx context.Context, name string, p security.PrivilegeScript) error {
	return nil
}

// CreatePrivilegeWildcard creates a new wildcard privilege.
func (m *MockSecurityService) CreatePrivilegeWildcard(ctx context.Context, p security.PrivilegeWildcard) error {
	if m.CreatePrivilegeWildcardFn != nil {
		return m.CreatePrivilegeWildcardFn(ctx, p)
	}

	return nil
}

// UpdatePrivilegeWildcard updates an existing wildcard privilege.
func (m *MockSecurityService) UpdatePrivilegeWildcard(ctx context.Context, name string, p security.PrivilegeWildcard) error {
	if m.UpdatePrivilegeWildcardFn != nil {
		return m.UpdatePrivilegeWildcardFn(ctx, name, p)
	}

	return nil
}

// GetAnonymousAccess retrieves the anonymous access configuration.
func (m *MockSecurityService) GetAnonymousAccess(ctx context.Context) (*security.AnonymousAccessSettings, error) {
	if m.GetAnonymousAccessFn != nil {
		return m.GetAnonymousAccessFn(ctx)
	}

	return nil, errMockNotConfigured
}

// UpdateAnonymousAccess updates the anonymous access configuration.
func (m *MockSecurityService) UpdateAnonymousAccess(ctx context.Context, settings security.AnonymousAccessSettings) error {
	if m.UpdateAnonymousAccessFn != nil {
		return m.UpdateAnonymousAccessFn(ctx, settings)
	}

	return nil
}

// GetSAML retrieves the SAML configuration.
func (m *MockSecurityService) GetSAML(ctx context.Context) (*security.SAML, error) {
	if m.GetSAMLFn != nil {
		return m.GetSAMLFn(ctx)
	}

	return nil, errMockNotConfigured
}

// ApplySAML sets the SAML configuration.
func (m *MockSecurityService) ApplySAML(ctx context.Context, saml security.SAML) error {
	if m.UpdateSAMLFn != nil {
		return m.UpdateSAMLFn(ctx, saml)
	}

	return nil
}

// DeleteSAML removes the SAML configuration.
func (m *MockSecurityService) DeleteSAML(ctx context.Context) error {
	if m.DeleteSAMLFn != nil {
		return m.DeleteSAMLFn(ctx)
	}

	return nil
}

// GetLDAP retrieves an LDAP connection by name.
func (m *MockSecurityService) GetLDAP(ctx context.Context, name string) (*security.LDAP, error) {
	if m.GetLDAPFn != nil {
		return m.GetLDAPFn(ctx, name)
	}

	return nil, errMockNotConfigured
}

// ListLDAP returns all configured LDAP connections.
func (m *MockSecurityService) ListLDAP(ctx context.Context) ([]security.LDAP, error) {
	return nil, errMockNotConfigured
}

// CreateLDAP creates a new LDAP connection.
func (m *MockSecurityService) CreateLDAP(ctx context.Context, ldap security.LDAP) error {
	if m.CreateLDAPFn != nil {
		return m.CreateLDAPFn(ctx, ldap)
	}

	return nil
}

// UpdateLDAP updates an existing LDAP connection.
func (m *MockSecurityService) UpdateLDAP(ctx context.Context, name string, ldap security.LDAP) error {
	if m.UpdateLDAPFn != nil {
		return m.UpdateLDAPFn(ctx, name, ldap)
	}

	return nil
}

// DeleteLDAP deletes an LDAP connection by name.
func (m *MockSecurityService) DeleteLDAP(ctx context.Context, name string) error {
	if m.DeleteLDAPFn != nil {
		return m.DeleteLDAPFn(ctx, name)
	}

	return nil
}

// GetUserTokenConfiguration retrieves the user token configuration.
func (m *MockSecurityService) GetUserTokenConfiguration(ctx context.Context) (*security.UserTokenConfiguration, error) {
	if m.GetUserTokenConfigFn != nil {
		return m.GetUserTokenConfigFn(ctx)
	}

	return nil, errMockNotConfigured
}

// UpdateUserTokenConfiguration updates the user token configuration.
func (m *MockSecurityService) UpdateUserTokenConfiguration(ctx context.Context, config security.UserTokenConfiguration) error {
	if m.UpdateUserTokenConfigFn != nil {
		return m.UpdateUserTokenConfigFn(ctx, config)
	}

	return nil
}

// MockSSLService is a mock implementation of nexus.SSLService.
type MockSSLService struct {
	AddCertificateFn    func(ctx context.Context, cert *security.SSLCertificate) error
	RemoveCertificateFn func(ctx context.Context, id string) error
	ListCertificatesFn  func(ctx context.Context) ([]security.SSLCertificate, error)
}

// AddCertificate mock implementation.
func (m *MockSSLService) AddCertificate(ctx context.Context, cert *security.SSLCertificate) error {
	if m.AddCertificateFn != nil {
		return m.AddCertificateFn(ctx, cert)
	}

	return nil
}

// RemoveCertificate mock implementation.
func (m *MockSSLService) RemoveCertificate(ctx context.Context, id string) error {
	if m.RemoveCertificateFn != nil {
		return m.RemoveCertificateFn(ctx, id)
	}

	return nil
}

// ListCertificates mock implementation.
func (m *MockSSLService) ListCertificates(ctx context.Context) ([]security.SSLCertificate, error) {
	if m.ListCertificatesFn != nil {
		return m.ListCertificatesFn(ctx)
	}

	return nil, errMockNotConfigured
}
