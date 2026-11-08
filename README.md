# Crossplane Provider for Sonatype Nexus Repository Manager

A [Crossplane](https://crossplane.io/) provider for managing [Sonatype Nexus Repository Manager](https://www.sonatype.com/products/nexus-repository) resources.

## Overview

This provider enables you to manage Nexus Repository Manager resources using Kubernetes custom resources. It follows the standard Crossplane provider pattern and uses the [nexus-repo-api-client-go](https://github.com/sonatype-nexus-community/nexus-repo-api-client-go) library, generated from the Nexus OpenAPI specification, for communicating with the Nexus API.

## Supported Resources

| Resource | API group | Description |
| ---------- | ----------- | ------------- |
| `ProviderConfig` | `nexus.crossplane.io` | Connection to a Nexus instance, used by the managed resources of its namespace |
| `ClusterProviderConfig` | `nexus.crossplane.io` | Connection to a Nexus instance, usable from any namespace |
| `Repository` | `content.nexus.crossplane.io` | Repositories for various formats (hosted, proxy, group) |
| `CleanupPolicy` | `content.nexus.crossplane.io` | Component cleanup policies |
| `ContentSelector` | `content.nexus.crossplane.io` | Content selectors for fine-grained access control |
| `RoutingRule` | `content.nexus.crossplane.io` | Routing rules for proxy repositories |
| `User` | `iam.nexus.crossplane.io` | Nexus user accounts |
| `Role` | `iam.nexus.crossplane.io` | Security roles with privileges |
| `Privilege` | `iam.nexus.crossplane.io` | Security privileges (application, repository-view, wildcard, ...) |
| `AnonymousAccess` | `iam.nexus.crossplane.io` | Anonymous access settings |
| `LDAP` | `iam.nexus.crossplane.io` | LDAP server configuration |
| `SAML` | `iam.nexus.crossplane.io` | SAML identity provider configuration |
| `UserTokenConfiguration` | `iam.nexus.crossplane.io` | User token settings |
| `BlobStore` | `instance.nexus.crossplane.io` | File or S3-based blob storage |
| `Capability` | `instance.nexus.crossplane.io` | Nexus capabilities |
| `IQServerConfiguration` | `instance.nexus.crossplane.io` | Sonatype IQ Server integration |
| `License` | `instance.nexus.crossplane.io` | Nexus Pro license |
| `SecurityRealm` | `instance.nexus.crossplane.io` | Active security realms and their order |
| `SecuritySSLTruststore` | `instance.nexus.crossplane.io` | Certificates of the Nexus SSL truststore |
| `SSRFProtection` | `instance.nexus.crossplane.io` | SSRF protection and its allowed IP and domain exceptions |

Repositories are supported for Maven2, npm, Docker and 16 other formats. See [Supported Repository Formats](docs/repository-formats.md) for the full list and the types of each.

### Instance-wide settings

`AnonymousAccess`, `IQServerConfiguration`, `License`, `SAML`, `SecurityRealm`, `SSRFProtection` and `UserTokenConfiguration` each manage a setting Nexus holds once per instance. Declare at most one of each kind per Nexus instance, across all namespaces and ProviderConfigs that point at it. The provider does not enforce this: two resources of the same kind keep overwriting each other's settings. Deleting an `IQServerConfiguration`, `License`, `SAML` or `SSRFProtection` also disables, removes or resets the setting in Nexus, even while another resource of that kind still declares it.

## Quick Start

**1. Install the provider** (requires a cluster with Crossplane and a Nexus instance):

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-sonatype-nexus
spec:
  package: ghcr.io/genesary/provider-sonatype-nexus:v0.3.1
```

Or with the CLI: `crossplane xpkg install provider ghcr.io/genesary/provider-sonatype-nexus:v0.3.1`

**2. Store the credentials.** Each sensitive field is referenced from its own Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nexus-username
  namespace: default
type: Opaque
stringData:
  username: admin
---
apiVersion: v1
kind: Secret
metadata:
  name: nexus-password
  namespace: default
type: Opaque
stringData:
  password: admin123
```

**3. Create a ProviderConfig** in the namespace of the managed resources it serves, `default` in every example:

```yaml
apiVersion: nexus.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
  namespace: default
spec:
  url: http://nexus.example.com:8081
  insecureSkipVerify: false
  username:
    source: Secret
    secretRef:
      name: nexus-username
      key: username
  password:
    source: Secret
    secretRef:
      name: nexus-password
      key: password
```

Secret references on namespaced objects (ProviderConfig and every managed resource) have no `namespace` field: they always refer to Secrets in the object's own namespace. Only a `ClusterProviderConfig` names the namespace of its Secrets.

**4. Create managed resources.** Every managed resource is namespaced and names its provider config with `providerConfigRef`. See [Usage Examples](docs/examples.md) and the [`examples`](examples) directory.

## Documentation

| Guide | Contents |
| ------- | ---------- |
| [Configuration](docs/configuration.md) | Provider flags, activating managed resources, metrics |
| [Kubernetes Secret Access](docs/secret-access.md) | Secret cache, required permissions, removing Crossplane's Secret grant |
| [Referencing Other Managed Resources](docs/references.md) | Reference and selector fields between kinds |
| [Usage Examples](docs/examples.md) | A blob store, repository, privilege and role chain, and the `examples` directory |
| [Supported Repository Formats](docs/repository-formats.md) | Formats and repository types |
| [Development](docs/development.md) | Building, testing, e2e tests, project structure |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. See [Development](docs/development.md) to get started.

## License

Apache 2.0 - See [LICENSE](LICENSE) for more information.
