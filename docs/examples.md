# Usage Examples

Back to the [README](../README.md).

Every managed resource is namespaced, and names its provider config with
`providerConfigRef`. This chain creates a blob store, a Maven repository
stored in it, a privilege to read that repository, and a role granting that
privilege, each referencing the previous one:

```yaml
apiVersion: instance.nexus.crossplane.io/v1alpha1
kind: BlobStore
metadata:
  name: maven
  namespace: default
spec:
  forProvider:
    name: maven
    type: File
    path: /nexus-data/blobs/maven
  providerConfigRef:
    kind: ProviderConfig
    name: default
---
apiVersion: content.nexus.crossplane.io/v1alpha1
kind: Repository
metadata:
  name: maven-releases
  namespace: default
spec:
  forProvider:
    name: maven-releases
    format: maven2
    type: hosted
    storage:
      blobStoreNameRef:
        name: maven
      writePolicy: ALLOW_ONCE
    maven:
      versionPolicy: RELEASE
      layoutPolicy: STRICT
  providerConfigRef:
    kind: ProviderConfig
    name: default
---
apiVersion: iam.nexus.crossplane.io/v1alpha1
kind: Privilege
metadata:
  name: maven-releases-read
  namespace: default
spec:
  forProvider:
    name: maven-releases-read
    type: repository-view
    format: maven2
    repositoryRef:
      name: maven-releases
    actions:
      - BROWSE
      - READ
  providerConfigRef:
    kind: ProviderConfig
    name: default
---
apiVersion: iam.nexus.crossplane.io/v1alpha1
kind: Role
metadata:
  name: maven-readers
  namespace: default
spec:
  forProvider:
    id: maven-readers
    name: Maven readers
    privilegeRefs:
      - name: maven-releases-read
  providerConfigRef:
    kind: ProviderConfig
    name: default
```

The [`examples`](../examples) directory has an example of every kind but `License`:

| Directory | Examples |
| ----------- | ---------- |
| [`examples/content`](../examples/content) | Repositories (Maven hosted, proxy and group, Docker, npm, Hugging Face, P2), `CleanupPolicy`, `ContentSelector`, `RoutingRule` |
| [`examples/iam`](../examples/iam) | `User`, `Role`, `Privilege` (application, repository-view, wildcard), `AnonymousAccess`, `LDAP`, `SAML`, `UserTokenConfiguration`, a role built with references |
| [`examples/instance`](../examples/instance) | `BlobStore` (File and S3), `Capability`, `HTTPClient`, `IQServerConfiguration`, `Script`, `SecurityRealm`, `SecuritySSLTruststore`, `SSRFProtection`, `Task` |
| [`examples/providerconfig.yaml`](../examples/providerconfig.yaml) | `ProviderConfig`, `ClusterProviderConfig` and their Secrets |
