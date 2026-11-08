# Development

Back to the [README](../README.md).

## Prerequisites

- Go 1.26+
- Docker
- The `build` submodule: `make submodules`

## Building

```bash
# Generate code (DeepCopy methods, managed resource methods, CRDs)
make generate

# Build the provider binary, image and Crossplane package
make build

# Run unit tests
make test

# Run the linters
make lint

# Generate, lint and test: run before opening a pull request
make reviewable
```

## Running E2E Tests

`make e2e.run` builds nothing: run `make build` first. It then creates a kind
cluster, installs Crossplane and the locally built provider package, starts
Nexus in the cluster, and runs the Go e2e suite in `internal/test/e2e`.

```bash
make build
make e2e.run
```

The following environment variables tune it:

| Variable | Default | Description |
| ---------- | --------- | ------------- |
| `skipcleanup` | | `true` keeps the kind cluster after the run. |
| `KIND_NODE_IMAGE_TAG` | see `Makefile` | kindest/node tag, and so Kubernetes version, of the cluster. |
| `CROSSPLANE_VERSION` | see `Makefile` | Crossplane chart version. |
| `E2E_TEST_PARALLEL` | `16` | Number of e2e tests run at once. |
| `E2E_TEST_TIMEOUT` | `30m` | Timeout of the Go e2e suite. |
| `E2E_CREATION_GRACE_PERIOD` | `5s` | `--creation-grace-period` of the provider under test. |
| `ENABLE_SECRET_CACHE` | `true` | `--enable-secret-cache` of the provider under test. |
| `E2E_PROVIDER_ARGS` | | Extra provider flags, space separated, e.g. `--poll=10s`. |

`make e2e.test` only runs the Go suite, against an existing cluster. It
expects `NEXUS_URL`, `NEXUS_USER` and `NEXUS_PASS`.

## Project Structure

```plaintext
.
├── apis/                       # API types and generated CRD methods
│   ├── v1alpha1/               # ProviderConfig, ClusterProviderConfig, ProviderConfigUsage
│   ├── content/v1alpha1/       # Repository, CleanupPolicy, ContentSelector, RoutingRule
│   ├── iam/v1alpha1/           # User, Role, Privilege, LDAP, SAML, ...
│   └── instance/v1alpha1/      # BlobStore, Capability, License, SecurityRealm, ...
├── cmd/provider/               # Provider entry point and flags
├── internal/
│   ├── clients/                # Nexus API clients, by API group
│   │   └── nexus/              # Shared HTTP client, credentials, errors
│   ├── controller/             # Controllers, by API group
│   ├── helpers/                # Shared reconciler options and helpers
│   └── test/e2e/               # E2E test suite
├── cluster/                    # Provider image and e2e scripts
├── e2e/                        # E2E kind cluster and Nexus manifests
├── examples/                   # Example manifests
└── package/                    # Crossplane package metadata and generated CRDs
```

## Running Locally

```bash
# Apply CRDs to your cluster
kubectl apply -f package/crds/

# Run the provider locally (requires kubeconfig)
go run ./cmd/provider --debug
```
