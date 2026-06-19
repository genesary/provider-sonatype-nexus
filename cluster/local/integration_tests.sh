#!/usr/bin/env bash
set -e

# setting up colors
BLU='\033[0;34m'
GRN='\033[0;32m'
RED='\033[0;31m'
NOC='\033[0m' # No Color
echo_step() { printf "\n${BLU}>>>>>>> %s${NOC}\n" "$1"; }
echo_success() { printf "\n${GRN}%s${NOC}\n" "$1"; }
echo_error() { printf "\n${RED}%s${NOC}\n" "$1"; exit 1; }

PACKAGE_NAME="provider-sonatype-nexus"
projectdir="$( cd "$( dirname "${BASH_SOURCE[0]}")"/../.. && pwd )"

# Pull tool paths and project metadata from the build submodule.
eval "$(make --no-print-directory -C "${projectdir}" build.vars)"

# Use a build-id-tagged cluster name so concurrent CI runs do not collide.
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-${BUILD_REGISTRY:-build}-inttests}"
export KIND_CLUSTER_NAME

# Cleanup on exit unless skipcleanup is set.
if [ "${skipcleanup}" != "true" ]; then
    cleanup() {
        echo_step "cleaning up controlplane"
        "${KIND}" delete cluster --name="${KIND_CLUSTER_NAME}" || true
    }
    trap cleanup EXIT
fi

if [ ! -f "${OUTPUT_DIR}/xpkg/${PLATFORM}/${PACKAGE_NAME}-${VERSION}.xpkg" ]; then
    echo_error "xpkg not built — run 'make build' first"
fi

echo_step "creating kind cluster '${KIND_CLUSTER_NAME}' with Nexus NodePort mapping"
"${KIND}" get kubeconfig --name "${KIND_CLUSTER_NAME}" >/dev/null 2>&1 || \
    "${KIND}" create cluster --name="${KIND_CLUSTER_NAME}" --config="${projectdir}/e2e/kind-config.yaml"
"${KUBECTL}" config use-context "kind-${KIND_CLUSTER_NAME}"

echo_step "installing Crossplane ${CROSSPLANE_VERSION}"
"${HELM}" repo add crossplane-build-module https://charts.crossplane.io/stable --force-update
"${HELM}" repo update
"${HELM}" get notes -n crossplane-system crossplane >/dev/null 2>&1 || \
    "${HELM}" install crossplane --create-namespace --namespace=crossplane-system \
        crossplane-build-module/crossplane --version "${CROSSPLANE_VERSION}"

echo_step "deploying ${PACKAGE_NAME} provider package"
make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"

echo_step "granting provider service account permission to watch CRDs"
echo "waiting for provider service account to be created..."
provider_sa=""
for _ in $(seq 1 60); do
    provider_sa="$("${KUBECTL}" get sa -n crossplane-system -o name 2>/dev/null \
        | grep "${PACKAGE_NAME}" | head -1 | sed 's|^serviceaccount/||')"
    if [ -n "${provider_sa}" ]; then break; fi
    sleep 2
done
if [ -z "${provider_sa}" ]; then
    echo_error "provider service account did not appear within 120s"
fi
echo "binding ServiceAccount ${provider_sa} to CRD watch role"
"${KUBECTL}" apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ${PACKAGE_NAME}-crd-watcher
rules:
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${PACKAGE_NAME}-crd-watcher
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ${PACKAGE_NAME}-crd-watcher
subjects:
  - kind: ServiceAccount
    name: ${provider_sa}
    namespace: crossplane-system
EOF

echo_step "waiting for provider to become healthy"
"${KUBECTL}" wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" \
    --for=condition=healthy --timeout=300s

echo_step "waiting for all provider CRDs to be Established"
# Poll until at least one CRD in our API groups appears (the label crossplane sets
# is on the revision, not the CRDs themselves, so select by group name instead).
crd_deadline=$(( $(date +%s) + 120 ))
while true; do
    crd_count=$("${KUBECTL}" get crd --no-headers 2>/dev/null | grep -c '\.nexus\.crossplane\.io' || true)
    [ "${crd_count}" -gt 0 ] && break
    if [ "$(date +%s)" -ge "${crd_deadline}" ]; then
        echo_error "timed out waiting for provider CRDs to appear"
    fi
    sleep 3
done
"${KUBECTL}" get crd --no-headers 2>/dev/null \
    | grep '\.nexus\.crossplane\.io' | awk '{print $1}' \
    | xargs "${KUBECTL}" wait crd --for=condition=Established --timeout=120s

echo_step "deploying Nexus and configuring ProviderConfig"
KUBECTL="${KUBECTL}" "${projectdir}/cluster/local/nexus_setup.sh"

echo_step "running e2e Go suite"
e2e_status=0
NEXUS_URL="http://localhost:8081" \
NEXUS_USER="admin" \
NEXUS_PASS="admin123" \
make -C "${projectdir}" e2e.test || e2e_status=$?

if [ "${e2e_status}" -ne 0 ]; then
    echo_error "e2e Go suite failed (exit ${e2e_status})"
fi

echo_success "Integration tests succeeded!"
