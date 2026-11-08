# Referencing Other Managed Resources

Back to the [README](../README.md).

Fields that name another Nexus object can instead reference the managed
resource that manages it, so that Crossplane fills them in once that resource
is `Ready`, which means the provider has observed it in Nexus. A reference
names the managed resource (`metadata.name`), and resolves to its Nexus name.
A selector picks managed resources by label. Both only match managed resources
in the referencing resource's own namespace.

Until every resource it references is `Ready`, the referencing resource is not
created or updated, and its `Synced` condition explains which reference is
waiting. A selector waits on every resource it matches, except the referencing
resource itself. A reference or selector whose `policy.resolution` is
`Optional` skips the resources that are not `Ready` instead; since resolved
values are kept, add `policy.resolve: Always` for it to pick them up once they
are `Ready`. Note that a
`Repository` whose optional `storage.blobStoreNameRef` or
`storage.blobStoreNameSelector` resolves to nothing fails to reconcile, because
Nexus needs a blob store.

| Kind | Field | Reference | Selector | Referenced kind |
| ------ | ------- | ----------- | ---------- | ----------------- |
| `Repository` | `storage.blobStoreName` | `storage.blobStoreNameRef` | `storage.blobStoreNameSelector` | `BlobStore` |
| `Repository` | `cleanup.policyNames` | `cleanup.policyNameRefs` | `cleanup.policyNameSelector` | `CleanupPolicy` |
| `Repository` | `group.memberNames` | `group.memberNameRefs` | `group.memberNameSelector` | `Repository` |
| `Repository` | `routingRule` | `routingRuleRef` | `routingRuleSelector` | `RoutingRule` |
| `Privilege` | `repository` | `repositoryRef` | `repositorySelector` | `Repository` |
| `Privilege` | `contentSelector` | `contentSelectorRef` | `contentSelectorSelector` | `ContentSelector` |
| `Role` | `privileges` | `privilegeRefs` | `privilegeSelector` | `Privilege` |
| `Role` | `roles` | `roleRefs` | `roleSelector` | `Role` |
| `User` | `roles` | `roleRefs` | `roleSelector` | `Role` |

```yaml
apiVersion: iam.nexus.crossplane.io/v1alpha1
kind: Role
metadata:
  name: maven-readers
  namespace: team-a
spec:
  forProvider:
    id: team-a-maven-readers
    name: Team A Maven readers
    privilegeRefs:
      - name: maven-releases-view
  providerConfigRef:
    kind: ProviderConfig
    name: default
```

As with every Crossplane provider, resolved values replace the field, and a
field that already has values is not resolved again unless the reference's
`policy.resolve` is `Always`. A list therefore holds either literal names or
references, not both. To reference a built-in Nexus object, such as an
`nx-*` privilege, observe it with a managed resource whose
`managementPolicies` is `["Observe"]`.
