// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/blobstore"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/repository"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"

	"github.com/crossplane-contrib/provider-sonatype-nexus/internal/clients/nexus"
)

var _ nexus.Client = &MockClient{}

// MockClient is a mock implementation of nexus.Client.
type MockClient struct {
	MockBlobStore  *MockBlobStoreService
	MockRepository *MockRepositoryService
	MockSecurity   *MockSecurityService
}

// NewMockClient creates a new MockClient with default mocks.
func NewMockClient() *MockClient {
	return &MockClient{
		MockBlobStore:  &MockBlobStoreService{},
		MockRepository: &MockRepositoryService{},
		MockSecurity:   &MockSecurityService{},
	}
}

// BlobStore returns the mock blob store service.
func (m *MockClient) BlobStore() nexus.BlobStoreService {
	return m.MockBlobStore
}

// Repository returns the mock repository service.
func (m *MockClient) Repository() nexus.RepositoryService {
	return m.MockRepository
}

// Security returns the mock security service.
func (m *MockClient) Security() nexus.SecurityService {
	return m.MockSecurity
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
	return nil, nil
}

// GetS3 mock implementation.
func (m *MockBlobStoreService) GetS3(ctx context.Context, name string) (*blobstore.S3, error) {
	m.GetS3Calls = append(m.GetS3Calls, name)
	if m.GetS3Fn != nil {
		return m.GetS3Fn(ctx, name)
	}
	return nil, nil
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
	return nil, nil
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
}

// Maven implementations
func (m *MockRepositoryService) GetMavenHosted(ctx context.Context, name string) (*repository.MavenHostedRepository, error) {
	if m.GetMavenHostedFn != nil {
		return m.GetMavenHostedFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateMavenHosted(ctx context.Context, repo repository.MavenHostedRepository) error {
	if m.CreateMavenHostedFn != nil {
		return m.CreateMavenHostedFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateMavenHosted(ctx context.Context, name string, repo repository.MavenHostedRepository) error {
	if m.UpdateMavenHostedFn != nil {
		return m.UpdateMavenHostedFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteMavenHosted(ctx context.Context, name string) error {
	if m.DeleteMavenHostedFn != nil {
		return m.DeleteMavenHostedFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetMavenProxy(ctx context.Context, name string) (*repository.MavenProxyRepository, error) {
	if m.GetMavenProxyFn != nil {
		return m.GetMavenProxyFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateMavenProxy(ctx context.Context, repo repository.MavenProxyRepository) error {
	if m.CreateMavenProxyFn != nil {
		return m.CreateMavenProxyFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateMavenProxy(ctx context.Context, name string, repo repository.MavenProxyRepository) error {
	if m.UpdateMavenProxyFn != nil {
		return m.UpdateMavenProxyFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteMavenProxy(ctx context.Context, name string) error {
	if m.DeleteMavenProxyFn != nil {
		return m.DeleteMavenProxyFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetMavenGroup(ctx context.Context, name string) (*repository.MavenGroupRepository, error) {
	if m.GetMavenGroupFn != nil {
		return m.GetMavenGroupFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateMavenGroup(ctx context.Context, repo repository.MavenGroupRepository) error {
	if m.CreateMavenGroupFn != nil {
		return m.CreateMavenGroupFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateMavenGroup(ctx context.Context, name string, repo repository.MavenGroupRepository) error {
	if m.UpdateMavenGroupFn != nil {
		return m.UpdateMavenGroupFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteMavenGroup(ctx context.Context, name string) error {
	if m.DeleteMavenGroupFn != nil {
		return m.DeleteMavenGroupFn(ctx, name)
	}
	return nil
}

// Docker implementations
func (m *MockRepositoryService) GetDockerHosted(ctx context.Context, name string) (*repository.DockerHostedRepository, error) {
	if m.GetDockerHostedFn != nil {
		return m.GetDockerHostedFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateDockerHosted(ctx context.Context, repo repository.DockerHostedRepository) error {
	if m.CreateDockerHostedFn != nil {
		return m.CreateDockerHostedFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateDockerHosted(ctx context.Context, name string, repo repository.DockerHostedRepository) error {
	if m.UpdateDockerHostedFn != nil {
		return m.UpdateDockerHostedFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteDockerHosted(ctx context.Context, name string) error {
	if m.DeleteDockerHostedFn != nil {
		return m.DeleteDockerHostedFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetDockerProxy(ctx context.Context, name string) (*repository.DockerProxyRepository, error) {
	if m.GetDockerProxyFn != nil {
		return m.GetDockerProxyFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateDockerProxy(ctx context.Context, repo repository.DockerProxyRepository) error {
	if m.CreateDockerProxyFn != nil {
		return m.CreateDockerProxyFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateDockerProxy(ctx context.Context, name string, repo repository.DockerProxyRepository) error {
	if m.UpdateDockerProxyFn != nil {
		return m.UpdateDockerProxyFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteDockerProxy(ctx context.Context, name string) error {
	if m.DeleteDockerProxyFn != nil {
		return m.DeleteDockerProxyFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetDockerGroup(ctx context.Context, name string) (*repository.DockerGroupRepository, error) {
	if m.GetDockerGroupFn != nil {
		return m.GetDockerGroupFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateDockerGroup(ctx context.Context, repo repository.DockerGroupRepository) error {
	if m.CreateDockerGroupFn != nil {
		return m.CreateDockerGroupFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateDockerGroup(ctx context.Context, name string, repo repository.DockerGroupRepository) error {
	if m.UpdateDockerGroupFn != nil {
		return m.UpdateDockerGroupFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteDockerGroup(ctx context.Context, name string) error {
	if m.DeleteDockerGroupFn != nil {
		return m.DeleteDockerGroupFn(ctx, name)
	}
	return nil
}

// npm implementations
func (m *MockRepositoryService) GetNpmHosted(ctx context.Context, name string) (*repository.NpmHostedRepository, error) {
	if m.GetNpmHostedFn != nil {
		return m.GetNpmHostedFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateNpmHosted(ctx context.Context, repo repository.NpmHostedRepository) error {
	if m.CreateNpmHostedFn != nil {
		return m.CreateNpmHostedFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateNpmHosted(ctx context.Context, name string, repo repository.NpmHostedRepository) error {
	if m.UpdateNpmHostedFn != nil {
		return m.UpdateNpmHostedFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteNpmHosted(ctx context.Context, name string) error {
	if m.DeleteNpmHostedFn != nil {
		return m.DeleteNpmHostedFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetNpmProxy(ctx context.Context, name string) (*repository.NpmProxyRepository, error) {
	if m.GetNpmProxyFn != nil {
		return m.GetNpmProxyFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateNpmProxy(ctx context.Context, repo repository.NpmProxyRepository) error {
	if m.CreateNpmProxyFn != nil {
		return m.CreateNpmProxyFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateNpmProxy(ctx context.Context, name string, repo repository.NpmProxyRepository) error {
	if m.UpdateNpmProxyFn != nil {
		return m.UpdateNpmProxyFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteNpmProxy(ctx context.Context, name string) error {
	if m.DeleteNpmProxyFn != nil {
		return m.DeleteNpmProxyFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetNpmGroup(ctx context.Context, name string) (*repository.NpmGroupRepository, error) {
	if m.GetNpmGroupFn != nil {
		return m.GetNpmGroupFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateNpmGroup(ctx context.Context, repo repository.NpmGroupRepository) error {
	if m.CreateNpmGroupFn != nil {
		return m.CreateNpmGroupFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateNpmGroup(ctx context.Context, name string, repo repository.NpmGroupRepository) error {
	if m.UpdateNpmGroupFn != nil {
		return m.UpdateNpmGroupFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteNpmGroup(ctx context.Context, name string) error {
	if m.DeleteNpmGroupFn != nil {
		return m.DeleteNpmGroupFn(ctx, name)
	}
	return nil
}

// Raw implementations
func (m *MockRepositoryService) GetRawHosted(ctx context.Context, name string) (*repository.RawHostedRepository, error) {
	if m.GetRawHostedFn != nil {
		return m.GetRawHostedFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateRawHosted(ctx context.Context, repo repository.RawHostedRepository) error {
	if m.CreateRawHostedFn != nil {
		return m.CreateRawHostedFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateRawHosted(ctx context.Context, name string, repo repository.RawHostedRepository) error {
	if m.UpdateRawHostedFn != nil {
		return m.UpdateRawHostedFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteRawHosted(ctx context.Context, name string) error {
	if m.DeleteRawHostedFn != nil {
		return m.DeleteRawHostedFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetRawProxy(ctx context.Context, name string) (*repository.RawProxyRepository, error) {
	if m.GetRawProxyFn != nil {
		return m.GetRawProxyFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateRawProxy(ctx context.Context, repo repository.RawProxyRepository) error {
	if m.CreateRawProxyFn != nil {
		return m.CreateRawProxyFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateRawProxy(ctx context.Context, name string, repo repository.RawProxyRepository) error {
	if m.UpdateRawProxyFn != nil {
		return m.UpdateRawProxyFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteRawProxy(ctx context.Context, name string) error {
	if m.DeleteRawProxyFn != nil {
		return m.DeleteRawProxyFn(ctx, name)
	}
	return nil
}

func (m *MockRepositoryService) GetRawGroup(ctx context.Context, name string) (*repository.RawGroupRepository, error) {
	if m.GetRawGroupFn != nil {
		return m.GetRawGroupFn(ctx, name)
	}
	return nil, nil
}

func (m *MockRepositoryService) CreateRawGroup(ctx context.Context, repo repository.RawGroupRepository) error {
	if m.CreateRawGroupFn != nil {
		return m.CreateRawGroupFn(ctx, repo)
	}
	return nil
}

func (m *MockRepositoryService) UpdateRawGroup(ctx context.Context, name string, repo repository.RawGroupRepository) error {
	if m.UpdateRawGroupFn != nil {
		return m.UpdateRawGroupFn(ctx, name, repo)
	}
	return nil
}

func (m *MockRepositoryService) DeleteRawGroup(ctx context.Context, name string) error {
	if m.DeleteRawGroupFn != nil {
		return m.DeleteRawGroupFn(ctx, name)
	}
	return nil
}

// MockSecurityService is a mock implementation of nexus.SecurityService.
type MockSecurityService struct {
	GetUserFn        func(ctx context.Context, id string) (*security.User, error)
	CreateUserFn     func(ctx context.Context, user security.User) error
	UpdateUserFn     func(ctx context.Context, id string, user security.User) error
	DeleteUserFn     func(ctx context.Context, id string) error
	ChangePasswordFn func(ctx context.Context, id string, password string) error

	GetRoleFn    func(ctx context.Context, id string) (*security.Role, error)
	CreateRoleFn func(ctx context.Context, role security.Role) error
	UpdateRoleFn func(ctx context.Context, id string, role security.Role) error
	DeleteRoleFn func(ctx context.Context, id string) error

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
	return nil, nil
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
func (m *MockSecurityService) ChangePassword(ctx context.Context, id string, password string) error {
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
	return nil, nil
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
