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

Most formats only use the shared `storage`, `cleanup`, `proxy`,
`negativeCache`, `httpClient`, `routingRule` and `group` blocks. The following
formats add their own:

| Format | Field | Types | Notes |
|---|---|---|---|
| `maven2` | `maven` | hosted, proxy | Version, layout and content disposition policies. |
| `docker` | `docker` | all | Connector ports, subdomain, V1 API, basic auth. |
| `docker` | `dockerProxy` | proxy | Index type and URL, foreign layers. |
| `oci` | `oci` | all | Same settings as `docker`, plus `pathEnabled`. |
| `oci` | `ociProxy` | proxy | Same settings as `dockerProxy`. |
| `oci` | `cosign` | all | Signature enforcement. Nexus 3.96 only accepts `NONE`, and drops the regexes in that mode. |
| `apt` | `apt`, `aptSigning` | hosted, proxy | Distribution, flat layout, PGP signing key. |
| `yum` | `yum`, `yumSigning` | all | Repodata depth, deploy policy, PGP signing key. |
| `nuget` | `nugetProxy` | proxy | Protocol version, query cache max age. |
| `cargo` | `cargo` | proxy, group | Require authentication. |
| `swift` | `swift` | proxy | Require authentication. |
| `terraform` | `terraform` | group | Require authentication. Nexus ignores it on proxies. |
| `terraform` | `terraformSigning` | hosted | **Required.** Armored PGP private key. |
| `alpine` | `alpineSigning` | all | PEM RSA private key. **Required** for proxy and group repositories. |

Nexus redacts signing keys and proxy credentials on read, so they are not
compared with what Nexus reports. The provider records a fingerprint of the
ones it applied instead, and updates the repository when the spec, or the
referenced password Secret, changes.
