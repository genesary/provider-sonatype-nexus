#!/usr/bin/env bash
# Idempotent setup for an in-cluster Nexus instance used by e2e tests.
#
# Steps:
#   1. apply Deployment + Service from e2e/manifests/nexus.yaml
#   2. wait for the Deployment to become Available
#   3. poll localhost:8081 (mapped via kind NodePort) until Nexus reports ready
#   4. write username and password each to their own Kubernetes Secret in crossplane-system
#   5. apply a ProviderConfig and ClusterProviderConfig wired to the in-cluster Service
#
# Required environment:
#   KUBECTL (path to kubectl, falls back to "kubectl")
#
# Optional overrides have sensible defaults below.
set -euo pipefail

KUBECTL="${KUBECTL:-kubectl}"

NEXUS_NAMESPACE="${NEXUS_NAMESPACE:-nexus}"
NEXUS_DEPLOYMENT="${NEXUS_DEPLOYMENT:-nexus}"
NEXUS_ADMIN_USER="${NEXUS_ADMIN_USER:-admin}"
NEXUS_ADMIN_PASS="${NEXUS_ADMIN_PASS:-admin123}"
NEXUS_LOCAL_PORT="${NEXUS_LOCAL_PORT:-8081}"
NEXUS_USERNAME_SECRET_NAME="${NEXUS_USERNAME_SECRET_NAME:-nexus-username}"
NEXUS_PASSWORD_SECRET_NAME="${NEXUS_PASSWORD_SECRET_NAME:-nexus-password}"
NEXUS_SECRET_NAMESPACE="${NEXUS_SECRET_NAMESPACE:-crossplane-system}"
NEXUS_PROVIDERCONFIG_NAME="${NEXUS_PROVIDERCONFIG_NAME:-default}"
NEXUS_PROVIDERCONFIG_NAMESPACE="${NEXUS_PROVIDERCONFIG_NAMESPACE:-default}"
NEXUS_READY_TIMEOUT_SECS="${NEXUS_READY_TIMEOUT_SECS:-300}"

NEXUS_LOCAL_URL="http://localhost:${NEXUS_LOCAL_PORT}"
NEXUS_IN_CLUSTER_URL="http://${NEXUS_DEPLOYMENT}.${NEXUS_NAMESPACE}.svc.cluster.local:8081"

projectdir="$( cd "$( dirname "${BASH_SOURCE[0]}")"/../.. && pwd )"

log() { printf '\n>>> %s\n' "$*"; }

log "Applying Nexus manifests"
"${KUBECTL}" apply -f "${projectdir}/e2e/manifests/nexus.yaml"

log "Waiting for deployment/${NEXUS_DEPLOYMENT} to become Available (timeout 10m)"
"${KUBECTL}" wait --for=condition=Available \
    "deployment/${NEXUS_DEPLOYMENT}" \
    --namespace="${NEXUS_NAMESPACE}" \
    --timeout=10m

log "Waiting for Nexus API to be ready at ${NEXUS_LOCAL_URL}"
deadline=$((SECONDS + NEXUS_READY_TIMEOUT_SECS))
while [ "${SECONDS}" -lt "${deadline}" ]; do
    if curl -sf -o /dev/null "${NEXUS_LOCAL_URL}/service/rest/v1/status"; then
        log "Nexus is ready"
        break
    fi
    echo "  waiting for Nexus... (${SECONDS}s / ${deadline}s)"
    sleep 5
done
if ! curl -sf -o /dev/null "${NEXUS_LOCAL_URL}/service/rest/v1/status"; then
    echo "ERROR: Nexus did not become ready within ${NEXUS_READY_TIMEOUT_SECS}s" >&2
    exit 1
fi

log "Writing username Secret ${NEXUS_SECRET_NAMESPACE}/${NEXUS_USERNAME_SECRET_NAME}"
"${KUBECTL}" create secret generic "${NEXUS_USERNAME_SECRET_NAME}" \
    --namespace="${NEXUS_SECRET_NAMESPACE}" \
    --from-literal=username="${NEXUS_ADMIN_USER}" \
    --dry-run=client -o yaml | "${KUBECTL}" apply -f -

log "Writing password Secret ${NEXUS_SECRET_NAMESPACE}/${NEXUS_PASSWORD_SECRET_NAME}"
"${KUBECTL}" create secret generic "${NEXUS_PASSWORD_SECRET_NAME}" \
    --namespace="${NEXUS_SECRET_NAMESPACE}" \
    --from-literal=password="${NEXUS_ADMIN_PASS}" \
    --dry-run=client -o yaml | "${KUBECTL}" apply -f -

log "Applying ProviderConfig '${NEXUS_PROVIDERCONFIG_NAME}' in namespace '${NEXUS_PROVIDERCONFIG_NAMESPACE}'"
"${KUBECTL}" apply -f - <<EOF
apiVersion: nexus.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: ${NEXUS_PROVIDERCONFIG_NAME}
  namespace: ${NEXUS_PROVIDERCONFIG_NAMESPACE}
spec:
  url: ${NEXUS_IN_CLUSTER_URL}
  insecureSkipVerify: true
  username:
    source: Secret
    secretRef:
      name: ${NEXUS_USERNAME_SECRET_NAME}
      namespace: ${NEXUS_SECRET_NAMESPACE}
      key: username
  password:
    source: Secret
    secretRef:
      name: ${NEXUS_PASSWORD_SECRET_NAME}
      namespace: ${NEXUS_SECRET_NAMESPACE}
      key: password
EOF

log "Applying ClusterProviderConfig '${NEXUS_PROVIDERCONFIG_NAME}' (for cluster-scoped resources)"
"${KUBECTL}" apply -f - <<EOF
apiVersion: nexus.crossplane.io/v1alpha1
kind: ClusterProviderConfig
metadata:
  name: ${NEXUS_PROVIDERCONFIG_NAME}
spec:
  url: ${NEXUS_IN_CLUSTER_URL}
  insecureSkipVerify: true
  username:
    source: Secret
    secretRef:
      name: ${NEXUS_USERNAME_SECRET_NAME}
      namespace: ${NEXUS_SECRET_NAMESPACE}
      key: username
  password:
    source: Secret
    secretRef:
      name: ${NEXUS_PASSWORD_SECRET_NAME}
      namespace: ${NEXUS_SECRET_NAMESPACE}
      key: password
EOF

log "Nexus ready at ${NEXUS_IN_CLUSTER_URL} (credentials in ${NEXUS_SECRET_NAMESPACE}/${NEXUS_USERNAME_SECRET_NAME} and ${NEXUS_SECRET_NAMESPACE}/${NEXUS_PASSWORD_SECRET_NAME})"
