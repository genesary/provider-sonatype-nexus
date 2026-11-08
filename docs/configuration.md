# Configuration

Back to the [README](../README.md).

## Provider Flags

The provider binary takes the flags below. Each one can also be set through
the environment variable in parentheses. Set them through a
[`DeploymentRuntimeConfig`](https://docs.crossplane.io/latest/packages/providers/#runtime-configuration),
for example `args: ["--debug", "--poll=5m"]`.

| Flag | Default | Description |
| ------ | --------- | ------------- |
| `--debug` (`DEBUG`) | `false` | Run with debug logging, including controller-runtime's own logs. |
| `--leader-election` (`LEADER_ELECTION`) | `false` | Use leader election, to run several replicas of the provider. |
| `--poll` (`POLL_INTERVAL`) | `1m` | How often each managed resource is checked for drift. |
| `--min-poll-interval` (`MIN_POLL_INTERVAL`) | `30s` | Shortest poll interval a resource may request with the `crossplane.io/poll-interval` annotation. Shorter requests are raised to it, or to `--poll` when that is shorter. `0` removes the limit. |
| `--poll-jitter-percentage` (`POLL_JITTER_PERCENTAGE`) | `10` | Spread each poll by up to this percentage of its interval, in both directions but never below the shortest allowed poll interval (see `--min-poll-interval`), so that resources created together do not poll Nexus together. `0` disables it, the maximum is `50`. |
| `--creation-grace-period` (`CREATION_GRACE_PERIOD`) | `30s` | How long after a create a report that the resource does not exist is not trusted. A resource deleted within this period keeps its finalizer until the period ends. `0` disables it. |
| `--sync` (`SYNC_INTERVAL`) | `1h` | How often every resource is re-listed from the API server and checked for drift. |
| `--max-reconcile-rate` (`MAX_RECONCILE_RATE`) | `10` | Maximum reconciles per second, across all controllers. It also sets the number of concurrent reconciles per controller and bounds the Kubernetes API client rate. |
| `--poll-state-metric` (`POLL_STATE_METRIC_INTERVAL`) | `5s` | How often the `crossplane_managed_resource_*` state metrics are recorded. |
| `--enable-management-policies` (`ENABLE_MANAGEMENT_POLICIES`) | `true` | Honor `spec.managementPolicies`, for example to observe a Nexus resource without managing it. Disable it with `--no-enable-management-policies` or `ENABLE_MANAGEMENT_POLICIES=false`: `--enable-management-policies=false` is rejected. |
| `--enable-secret-cache` (`ENABLE_SECRET_CACHE`) | `true` | Read Secrets through the informer cache. Disable it with `--no-enable-secret-cache` or `ENABLE_SECRET_CACHE=false`: `--enable-secret-cache=false` is rejected. See [Kubernetes Secret Access](secret-access.md). |
| `--enable-changelogs` (`ENABLE_CHANGE_LOGS`) | `false` | Alpha. Send a [change log](https://docs.crossplane.io/latest/guides/change-logs/) entry for every create, update and delete of a Nexus resource. Enable it with `--enable-changelogs` or `ENABLE_CHANGE_LOGS=true`: `--enable-changelogs=true` is rejected. |
| `--changelogs-socket-path` (`CHANGELOGS_SOCKET_PATH`) | `/var/run/changelogs/changelogs.sock` | Absolute path of the change logs sidecar socket, when change logs are enabled. |

## Activating Managed Resources

The package declares Crossplane's `safe-start` capability. Crossplane installs
a `ManagedResourceDefinition` for each kind and only creates its CRD once a
[`ManagedResourceActivationPolicy`](https://docs.crossplane.io/latest/managed-resources/managed-resource-activation-policies/)
activates it. The default policy installed with Crossplane activates every
kind. The provider starts the controller of each kind once its CRD is
established, so it runs fine with only some of its kinds activated. For
example, to only manage repositories and blob stores:

```yaml
apiVersion: apiextensions.crossplane.io/v1alpha1
kind: ManagedResourceActivationPolicy
metadata:
  name: nexus-storage
spec:
  activate:
    - repositories.content.nexus.crossplane.io
    - blobstores.instance.nexus.crossplane.io
```

## Metrics

The provider serves Prometheus metrics on port `8080` at `/metrics`. Besides
the controller-runtime metrics, it records the standard Crossplane managed
resource metrics for every kind:

- `crossplane_managed_resource_exists`, `_ready` and `_synced`: how many
  managed resources of each kind exist, are ready and are synced;
- `crossplane_managed_resource_first_time_to_readiness_seconds`,
  `_first_time_to_reconcile_seconds`, `_deletion_seconds` and
  `_drift_seconds`: how long they take to become ready, to be reconciled the
  first time, to be deleted, and how long they stay drifted.
