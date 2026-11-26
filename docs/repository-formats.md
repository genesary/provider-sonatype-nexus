# Supported Repository Formats

Back to the [README](../README.md).

The provider supports the following repository formats:

- **Maven2** (hosted, proxy, group)
- **npm** (hosted, proxy, group)
- **Docker** (hosted, proxy, group)
- **OCI** (hosted, proxy, group)
- **Raw** (hosted, proxy, group)
- **NuGet** (hosted, proxy, group)
- **PyPI** (hosted, proxy, group)
- **RubyGems** (hosted, proxy, group)
- **Yum** (hosted, proxy, group)
- **APT** (hosted, proxy)
- **Alpine (APK)** (hosted, proxy, group)
- **Ansible Galaxy** (hosted, proxy, group)
- **Helm** (hosted, proxy, group)
- **Go** (hosted, proxy, group)
- **R** (hosted, proxy, group)
- **Conan** (hosted, proxy, group)
- **Conda** (hosted, proxy, group)
- **Composer** (hosted, proxy, group)
- **Pub** (hosted, proxy, group)
- **Swift** (hosted, proxy, group)
- **Terraform** (hosted, proxy, group)
- **Cargo** (hosted, proxy, group)
- **Cocoapods** (proxy)
- **Git LFS** (hosted)
- **Hugging Face** (proxy)
- **P2** (proxy)

Set `format` to the Nexus format name: `maven2`, `npm`, `docker`, `oci`,
`raw`, `nuget`, `pypi`, `rubygems`, `yum`, `apt`, `alpine`, `ansiblegalaxy`,
`helm`, `go`, `r`, `conan`, `conda`, `composer`, `pub`, `swift`, `terraform`,
`cargo`, `cocoapods`, `gitlfs`, `huggingface` or `p2`.

## Format specific settings

Most formats only use the shared `storage`, `cleanup`, `component`, `proxy`,
`negativeCache`, `httpClient`, `firewall`, `replication`, `routingRule` and
`group` blocks. The following formats add their own:

| Format | Field | Types | Notes |
|---|---|---|---|
| `maven2` | `maven` | hosted, proxy | Version, layout and content disposition policies. |
| `docker` | `docker` | all | Connector ports, subdomain, V1 API, basic auth, repository name in image paths. |
| `docker` | `dockerProxy` | proxy | Index type (default `HUB`) and URL, foreign layers, AWS ECR authentication (`ecrAuth`). |
| `docker`, `oci` | `storage.latestPolicy` | hosted | Allow redeploying the `latest` tag. Requires the `ALLOW_ONCE` write policy, as Nexus drops it otherwise. |
| `oci` | `oci` | all | Same settings as `docker`. |
| `oci` | `ociProxy` | proxy | Same settings as `dockerProxy`, but the index type defaults to `REGISTRY`, as in Nexus. Nexus offers no `ecrAuth` on OCI. |
| `oci` | `cosign` | all | Signature enforcement, `NONE` when left out. Nexus 3.96 only accepts `NONE`. The regexes require `KEYLESS`. |
| `apt` | `apt`, `aptSigning` | hosted, proxy | Distribution, flat layout, distribution enforcement (proxy), PGP signing key (required on hosted). |
| `yum` | `yum`, `yumSigning` | all | Repodata depth, deploy policy, PGP signing key. |
| `nuget` | `nugetProxy` | proxy | Protocol version, query cache max age, symbol server URL and anonymous access (default `true`). |
| `raw` | `raw` | all | Content disposition. Nexus keeps it when the block is removed. |
| `conan` | `conanProxy` | proxy | Protocol version (Nexus defaults to `V2`). Immutable, as Nexus cannot change it. |
| `pypi` | `pypi` | proxy | Remote index path: `/simple` (default) or `""` for remotes serving their index at the root. |
| `huggingface` | `xet` | proxy | Xet CAS URL and transfer host allow list. |
| `cargo` | `cargo` | proxy, group | Require authentication. |
| `swift` | `swift` | proxy | Require authentication. |
| `terraform` | `terraform` | group | Require authentication. Rejected on hosted and proxy repositories, where Nexus ignores it. |
| `terraform` | `terraformSigning` | hosted | **Required.** Armored PGP private key. |
| `alpine` | `alpineSigning` | all | PEM RSA private key. **Required** for proxy and group repositories. |

## Shared settings

| Field | Types | Notes |
|---|---|---|
| `component.proprietaryComponents` | hosted | Marks components as proprietary against namespace confusion attacks. Takes effect with Sonatype Repository Firewall. |
| `proxy.preserveEncodedCharacters` | proxy | Keeps `%2B`, `%23`, `%20`... encoded towards the remote, as AWS S3, Cloudflare and Azure Blob Storage remotes require. |
| `httpClient.authentication.type` | proxy | `username`, `ntlm` or `bearerToken`. A bearer token is read from `bearerTokenSecretRef`. |
| `httpClient.authentication.preemptive` | maven2, pypi and terraform proxy | Sends the credentials up front. Rejected on the other formats, which Nexus ignores it on. |
| `firewall.mode` | proxy | `DISABLED`, `AUDIT`, `QUARANTINE` or `PCCS`. Every mode but `DISABLED` requires an IQ Server connection. Nexus keeps the mode when the block is removed, so set `DISABLED` to turn it off. |
| `replication` | proxy | Pre-emptive pull of the remote's new content, optionally restricted by `assetPathRegex`. Nexus Repository Pro only: Community rejects enabling it. |

The settings Nexus resets when an update leaves them out
(`storage.latestPolicy`, `component`, `proxy.preserveEncodedCharacters`,
`apt.enforceDistribution`, `nugetProxy.allowAnonymousSymbolAccess` and
`pypi.indexPath`) are always sent with their defaults, for the same reason as
the `dockerProxy` block below.

Nexus 3.96 stores the `replication` and `xet` blocks, and the `component` block
of Ansible Galaxy hosted repositories, without reporting them back. The
provider only compares them when Nexus reports them, so it cannot notice them
changed directly in Nexus there.

The raw `forwardQueryParameters` and `excludedQueryParameters` settings of the
API are not offered: Nexus 3.96 accepts them but does not store them.

## Credentials and drift

The signing keys marked **Required** are not required when the
`managementPolicies` only observe the repository (no `*`, `Create` or
`Update`), since the provider never writes it then. This lets you import an
existing repository without handing the provider its private key.

The `dockerProxy`, `ociProxy` and `cosign` blocks are always sent with their
defaults when left out. Nexus resets a block that an update leaves out, and
sending the defaults also lets the provider notice when someone changes them
directly in Nexus.

Signing keys, proxy credentials and the AWS ECR access key are not compared
with what Nexus reports: Nexus redacts or masks them on read, except the
Terraform signing key, which it echoes back but which is left out all the same.
The provider records a fingerprint of the ones it applied instead, and updates
the repository when the spec, or a referenced password, bearer token or AWS
Secret, changes.
