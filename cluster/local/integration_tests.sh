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

# local.xpkg.deploy.provider.<name> extracts every xpkg of the output
# directory under the same cache key, so the last one in glob order is
# deployed, whatever its version. Packages left by builds of earlier commits
# would then be installed instead of this one, with their older CRDs, while
# the runtime image is this build's. Remove them.
for xpkg in "${OUTPUT_DIR}"/xpkg/linux_*/"${PACKAGE_NAME}"-*.xpkg; do
    [ -e "${xpkg}" ] || continue
    case "${xpkg}" in
        */"${PACKAGE_NAME}-${VERSION}.xpkg") ;;
        *)
            echo "removing ${xpkg}, left by an earlier build"
            rm -f "${xpkg}"
            ;;
    esac
done

echo_step "creating kind cluster '${KIND_CLUSTER_NAME}' on kindest/node:${KIND_NODE_IMAGE_TAG:?run through make test-integration}"
"${KIND}" get kubeconfig --name "${KIND_CLUSTER_NAME}" >/dev/null 2>&1 || \
    "${KIND}" create cluster --name="${KIND_CLUSTER_NAME}" --config="${projectdir}/e2e/kind-config.yaml" \
        --image="kindest/node:${KIND_NODE_IMAGE_TAG}"
"${KUBECTL}" config use-context "kind-${KIND_CLUSTER_NAME}"

echo_step "installing Crossplane ${CROSSPLANE_VERSION}"
"${HELM}" repo add crossplane-build-module https://charts.crossplane.io/stable --force-update
"${HELM}" repo update
"${HELM}" get notes -n crossplane-system crossplane >/dev/null 2>&1 || \
    "${HELM}" install crossplane --create-namespace --namespace=crossplane-system \
        crossplane-build-module/crossplane --version "${CROSSPLANE_VERSION}"

# The tests delete their resources seconds after creating them. Within the
# creation grace period (30s by default), the provider does not trust Nexus
# reporting a resource gone, so each deletion would wait out the rest of
# those 30s. Nexus is strongly consistent, so a short period is enough.
E2E_CREATION_GRACE_PERIOD="${E2E_CREATION_GRACE_PERIOD:-5s}"

# Match the provider default; set to false to exercise direct Secret reads.
ENABLE_SECRET_CACHE="${ENABLE_SECRET_CACHE:-true}"

# Extra provider flags, space separated, e.g. E2E_PROVIDER_ARGS="--poll=10s".
# The script's own settings go through the provider's environment variables
# instead of flags: kingpin rejects a repeated flag, while a flag passed here
# simply overrides its variable. read -a splits on spaces without
# glob-expanding the words.
read -r -a extra_provider_args <<<"${E2E_PROVIDER_ARGS:-}"
provider_args=""
for arg in "${extra_provider_args[@]}"; do
    # Escape for a double-quoted YAML scalar.
    arg="${arg//\\/\\\\}"
    arg="${arg//\"/\\\"}"
    provider_args="${provider_args:+${provider_args}, }\"${arg}\""
done

# local.xpkg.deploy.provider.<name> uses DRC_FILE as the provider's
# DeploymentRuntimeConfig when it exists; it overrides the name and image.
DRC_FILE="$(mktemp)"
export DRC_FILE
# Keep the cluster teardown trap set above, unless skipcleanup is set.
trap 'rm -f "${DRC_FILE}"; [ "${skipcleanup}" = "true" ] || cleanup' EXIT
cat >"${DRC_FILE}" <<EOF_DRC
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: runtimeconfig-${PACKAGE_NAME}
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args: [${provider_args}]
              env:
                - name: DEBUG
                  value: "true"
                - name: CREATION_GRACE_PERIOD
                  value: "${E2E_CREATION_GRACE_PERIOD}"
                - name: ENABLE_SECRET_CACHE
                  value: "${ENABLE_SECRET_CACHE}"
EOF_DRC

echo_step "deploying ${PACKAGE_NAME} provider package (CREATION_GRACE_PERIOD=${E2E_CREATION_GRACE_PERIOD}, ENABLE_SECRET_CACHE=${ENABLE_SECRET_CACHE}, args=[${provider_args}])"
make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"

# Start Nexus now, so that its image pull and slow first boot (a few minutes)
# overlap with the provider becoming healthy and its CRDs being established,
# instead of starting after them. nexus_setup.sh re-applies the same manifest
# (idempotently) and waits for Nexus later.
echo_step "starting Nexus in the background"
"${KUBECTL}" apply -f "${projectdir}/e2e/manifests/nexus.yaml"

echo_step "waiting for provider to become healthy"
"${KUBECTL}" wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" \
    --for=condition=healthy --timeout=300s

echo_step "waiting for all provider CRDs to be Established"
# Poll until every CRD of the package appears (the label crossplane sets is on
# the revision, not the CRDs themselves, so select by group name instead). With
# safe-start, the managed resource CRDs are only created once their
# ManagedResourceDefinitions are activated, after the ProviderConfig ones, so
# the first CRDs to appear are not all of them.
crd_expected=$(find "${projectdir}/package/crds" -name '*.yaml' | wc -l)
crd_deadline=$(( $(date +%s) + 120 ))
while true; do
    crd_count=$("${KUBECTL}" get crd --no-headers 2>/dev/null | grep -c '\.nexus\.crossplane\.io' || true)
    [ "${crd_count}" -ge "${crd_expected}" ] && break
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
