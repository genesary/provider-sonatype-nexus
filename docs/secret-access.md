# Kubernetes Secret Access

Back to the [README](../README.md).

By default the provider reads Kubernetes Secrets through the controller-runtime
cache. The first read starts an informer that lists and watches every Secret
the provider can access and keeps them in memory, which requires cluster-wide
`list` and `watch` on Secrets.

`--no-enable-secret-cache` (or `ENABLE_SECRET_CACHE=false`) sends each Secret
read directly to the Kubernetes API instead, so the provider only needs access
to the Secrets it actually uses. Other kinds stay cached.

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-sonatype-nexus
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --no-enable-secret-cache
```

Things to know before disabling it:

- **Permissions:** the provider needs `get` on every Secret it reads:
  `ProviderConfig` credentials and the Secret references of managed resources
  (user passwords, repository HTTP client and LDAP bind passwords, S3 keys, IQ
  Server credentials, license files). It needs `get`, `create` and `patch` on
  connection Secrets (`writeConnectionSecretToRef`) in their namespace. `get`
  and `patch` can be restricted with `resourceNames`, `create` cannot.
- **Existing grants:** the flag does not change RBAC. Crossplane still grants
  the provider every verb on every Secret of the cluster, and a narrower Role
  does not take that away. See
  [Removing Crossplane's Secret Grant](#removing-crossplanes-secret-grant).
- **API traffic:** every Secret read becomes an API request. Credentials are
  read on every reconcile, and these requests share the provider's Kubernetes
  client rate limit, derived from `--max-reconcile-rate`. Each operation reads
  a given Secret once, even when it uses several of its keys.

## Removing Crossplane's Secret Grant

Crossplane's RBAC manager creates a `crossplane:provider:<revision>:system`
ClusterRole for every provider revision and binds it to the ServiceAccount the
provider runs as. Besides the provider's own resources, that role allows every
verb on `secrets`, `configmaps`, `events` and `leases` in all namespaces. The
grant is fixed (checked against Crossplane 2.4.3):

- The RBAC manager re-applies the role whenever its rules differ, so editing it
  does not last.
- The binding follows the ServiceAccount of the provider Deployment, so a
  custom ServiceAccount is bound too.

The only way to drop it is to install Crossplane without the RBAC manager
(`--set rbacManager.deploy=false` on the `crossplane` Helm chart) and write the
RBAC yourself. That setting applies to the whole cluster: no provider gets
generated roles anymore, Crossplane itself loses the aggregated access to
composite resources it gets for each XRD, and the `crossplane-admin`, `-edit`,
`-view` and `-browse` aggregated ClusterRoles are not installed. Only choose it
if you already manage RBAC for every package on the cluster.

With the RBAC manager disabled, give the provider a stable ServiceAccount name,
because the generated one changes with each revision:

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-sonatype-nexus
spec:
  serviceAccountTemplate:
    metadata:
      name: provider-sonatype-nexus
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --no-enable-secret-cache
```

Then grant it what the RBAC manager would have, without Secrets, and bind it to
`provider-sonatype-nexus` in the namespace Crossplane runs providers in
(`crossplane-system` by default):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: provider-sonatype-nexus
rules:
  # The provider's own resources.
  - apiGroups:
      - nexus.crossplane.io
      - content.nexus.crossplane.io
      - iam.nexus.crossplane.io
      - instance.nexus.crossplane.io
    resources: ["*"]
    verbs: [get, list, watch, create, update, patch]
  - apiGroups:
      - nexus.crossplane.io
      - content.nexus.crossplane.io
      - iam.nexus.crossplane.io
      - instance.nexus.crossplane.io
    resources: ["*/finalizers"]
    verbs: [update]
  # Events, and leader election with --leader-election.
  - apiGroups: [""]
    resources: [events]
    verbs: [create, patch]
  - apiGroups: [coordination.k8s.io]
    resources: [leases]
    verbs: [get, create, update]
  # Safe-start: waiting for CRDs to be installed.
  - apiGroups: [apiextensions.k8s.io]
    resources: [customresourcedefinitions]
    verbs: [get, list, watch]
```

Finally, grant Secret access per namespace with a Role and RoleBinding in each
namespace that holds provider Secrets, as described under **Permissions** above.
