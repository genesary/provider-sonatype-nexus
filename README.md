# Crossplane Provider for Sonatype Nexus Repository Manager

A [Crossplane](https://crossplane.io/) provider for managing [Sonatype Nexus Repository Manager](https://www.sonatype.com/products/nexus-repository) resources.

## Overview

This provider enables you to manage Nexus Repository Manager resources using Kubernetes custom resources. It follows the standard Crossplane provider pattern and uses the [go-nexus-client](https://github.com/datadrivers/go-nexus-client) library for communicating with the Nexus API.

## Supported Resources

| Resource | Description |
|----------|-------------|
| `ProviderConfig` | Configuration for connecting to Nexus |
| `BlobStore` | File or S3-based blob storage |
| `Repository` | Maven, NPM, Docker, and Raw repositories (hosted, proxy, group) |
| `User` | Nexus user accounts |
| `Role` | Security roles with privileges |

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

#### S3-based BlobStore

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: BlobStore
metadata:
  name: my-s3-blobstore
spec:
  forProvider:
    name: my-s3-blobstore
    type: S3
    s3Config:
      bucket: my-nexus-bucket
      region: us-east-1
      prefix: nexus-blobs
      expirationDays: 30
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
    blobStoreName: default
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
  providerConfigRef:
    name: nexus-provider-config
```

#### Maven Proxy Repository

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: maven-central-proxy
spec:
  forProvider:
    name: maven-central-proxy
    format: maven2
    type: proxy
    online: true
    blobStoreName: default
    proxy:
      remoteUrl: https://repo1.maven.org/maven2/
      contentMaxAge: 1440
      metadataMaxAge: 1440
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
      httpsPort: 8083
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

## Development

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- A Kubernetes cluster (minikube, kind, etc.)

### Building

```bash
# Generate code (DeepCopy methods, CRDs)
make generate

# Build the provider binary
make build

# Run tests
make test

# Build Docker image
make docker-build IMG=provider-sonatype-nexus:dev

# Push Docker image
make docker-push IMG=provider-sonatype-nexus:dev
```

### Project Structure

```
.
├── apis/
│   └── v1alpha1/           # CRD type definitions
│       ├── blobstore_types.go
│       ├── repository_types.go
│       ├── user_types.go
│       ├── role_types.go
│       ├── providerconfig_types.go
│       ├── managed.go      # Managed interface implementations
│       └── conditions.go   # Condition helpers
├── cmd/
│   └── provider/
│       └── main.go         # Provider entry point
├── internal/
│   ├── clients/
│   │   └── nexus/          # Nexus client wrapper
│   └── controller/
│       └── blobstore/      # BlobStore controller
├── test/
│   └── mocks/              # Mock implementations for testing
├── examples/               # Example YAML manifests
├── package/
│   └── crds/               # Generated CRDs
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
