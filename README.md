# Crossplane Provider for Sonatype Nexus Repository Manager

A [Crossplane](https://crossplane.io/) provider for managing [Sonatype Nexus Repository Manager](https://www.sonatype.com/products/nexus-repository) resources.

## Overview

This provider enables you to manage Nexus Repository Manager resources using Kubernetes custom resources. It follows the standard Crossplane provider pattern and uses the [go-nexus-client](https://github.com/datadrivers/go-nexus-client) library for communicating with the Nexus API.

## Supported Resources

| Resource | Description |
|----------|-------------|
| `ProviderConfig` | Configuration for connecting to Nexus |
| `BlobStore` | File or S3-based blob storage |
| `Repository` | Repositories for various formats (hosted, proxy, group) |
| `User` | Nexus user accounts |
| `Role` | Security roles with privileges |
| `Privilege` | Security privileges (application, repository-view, wildcard) |
| `ContentSelector` | Content selectors for fine-grained access control |
| `SecurityRealm` | Security realm configuration |
| `AnonymousAccess` | Anonymous access settings |
| `LDAP` | LDAP server configuration |
| `SAML` | SAML identity provider configuration |
| `UserTokenConfiguration` | User token settings |

### Supported Repository Formats

The provider supports the following repository formats:

- **Maven2** (hosted, proxy, group)
- **npm** (hosted, proxy, group)
- **Docker** (hosted, proxy, group)
- **Raw** (hosted, proxy, group)
- **NuGet** (hosted, proxy, group)
- **PyPI** (hosted, proxy, group)
- **RubyGems** (hosted, proxy, group)
- **Yum** (hosted, proxy, group)
- **APT** (hosted, proxy)
- **Helm** (hosted, proxy)
- **Go** (proxy, group)
- **R** (hosted, proxy, group)
- **Conan** (proxy)
- **Conda** (proxy)
- **Cocoapods** (proxy)
- **Bower** (hosted, proxy, group)
- **Git LFS** (hosted)
- **Cargo** (hosted, proxy, group)

## Installation

### Prerequisites

- Kubernetes cluster with Crossplane installed
- Sonatype Nexus Repository Manager instance

### Install the Provider

```bash
kubectl crossplane install provider ghcr.io/crossplane-contrib/provider-sonatype-nexus:latest
```

Or using a Provider manifest:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-sonatype-nexus
spec:
  package: ghcr.io/crossplane-contrib/provider-sonatype-nexus:latest
```

## Configuration

### Create a Secret with Nexus Credentials

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nexus-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  url: "http://nexus.example.com:8081"
  username: "admin"
  password: "admin123"
```

### Create a ProviderConfig

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: nexus-provider-config
spec:
  credentials:
    source: Secret
    secretRef:
      name: nexus-credentials
      namespace: crossplane-system
      key: ""
```

## Usage Examples

### BlobStore

#### File-based BlobStore

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: BlobStore
metadata:
  name: my-file-blobstore
spec:
  forProvider:
    name: my-file-blobstore
    type: File
    path: /nexus-data/blobs/my-file-blobstore
    softQuota:
      type: spaceRemainingQuota
      limit: 104857600  # 100MB
  providerConfigRef:
    name: nexus-provider-config
```

### Repository

#### Maven Hosted Repository

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: maven-releases
spec:
  forProvider:
    name: maven-releases
    format: maven2
    type: hosted
    online: true
    storage:
      blobStoreName: default
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
  providerConfigRef:
    name: nexus-provider-config
```

#### Docker Hosted Repository

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: docker-hosted
spec:
  forProvider:
    name: docker-hosted
    format: docker
    type: hosted
    online: true
    docker:
      httpPort: 8082
      forceBasicAuth: true
  providerConfigRef:
    name: nexus-provider-config
```

### User

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: User
metadata:
  name: developer-user
spec:
  forProvider:
    userId: developer
    firstName: John
    lastName: Developer
    email: john.developer@example.com
    status: active
    roles:
      - nx-repository-view-maven2-*-read
    passwordSecretRef:
      name: user-password-secret
      namespace: crossplane-system
      key: password
  providerConfigRef:
    name: nexus-provider-config
```

### Role

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: Role
metadata:
  name: developer-role
spec:
  forProvider:
    roleId: developer-role
    name: Developer Role
    description: Role for developers
    privileges:
      - nx-repository-view-maven2-*-read
      - nx-repository-view-npm-*-read
    roles: []
  providerConfigRef:
    name: nexus-provider-config
```

### Privilege

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: Privilege
metadata:
  name: custom-repo-view
spec:
  forProvider:
    name: custom-repo-view
    type: repository-view
    description: Custom repository view privilege
    repositoryView:
      format: maven2
      repository: maven-releases
      actions:
        - browse
        - read
  providerConfigRef:
    name: nexus-provider-config
```

### Content Selector

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: ContentSelector
metadata:
  name: release-selector
spec:
  forProvider:
    name: release-selector
    description: Select only release artifacts
    expression: format == "maven2" and path =^ "/com/example/"
  providerConfigRef:
    name: nexus-provider-config
```

### Security Realm

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: SecurityRealm
metadata:
  name: nexus-realms
spec:
  forProvider:
    realms:
      - NexusAuthenticatingRealm
      - NexusAuthorizingRealm
      - DockerToken
  providerConfigRef:
    name: nexus-provider-config
```

## Development

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- Kind (for e2e tests)

### Building

```bash
# Generate code (DeepCopy methods, CRDs)
make generate

# Build the provider binary
make build

# Run unit tests
make test

# Build Docker image
make docker-build
```

### Running E2E Tests

```bash
# Run full e2e test cycle (setup + tests)
make e2e

# Or with cleanup
make e2e-full
```

### Project Structure

```
.
├── apis/v1alpha1/              # CRD type definitions
│   ├── blobstore_types.go
│   ├── repository_types.go
│   ├── user_types.go
│   ├── role_types.go
│   ├── privilege_types.go
│   ├── contentselector_types.go
│   ├── securityrealm_types.go
│   ├── anonymousaccess_types.go
│   ├── ldap_types.go
│   ├── saml_types.go
│   ├── usertokenconfiguration_types.go
│   └── providerconfig_types.go
├── cmd/provider/               # Provider entry point
├── internal/
│   ├── clients/nexus/          # Nexus client wrapper
│   └── controller/             # Resource controllers
│       ├── blobstore/
│       ├── repository/
│       ├── user/
│       ├── role/
│       ├── privilege/
│       ├── contentselector/
│       ├── securityrealm/
│       ├── anonymousaccess/
│       ├── ldap/
│       ├── saml/
│       └── usertokenconfiguration/
├── e2e/                        # E2E test infrastructure
├── examples/                   # Example YAML manifests
├── package/crds/               # Generated CRDs
├── test/mocks/                 # Mock implementations
├── Dockerfile
├── Makefile
└── go.mod
```

### Running Locally

```bash
# Apply CRDs to your cluster
kubectl apply -f package/crds/

# Run the provider locally (requires kubeconfig)
go run ./cmd/provider --debug
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Apache 2.0 - See [LICENSE](LICENSE) for more information.
