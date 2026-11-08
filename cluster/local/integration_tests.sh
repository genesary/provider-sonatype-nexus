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

# controlplane.up cannot choose the node image nor the cluster config, but
# reuses an existing cluster of the same name, so create it here to run on the
# pinned Kubernetes version.
echo_step "creating kind cluster '${KIND_CLUSTER_NAME}' on kindest/node:${KIND_NODE_IMAGE_TAG:?run through make test-integration}"
if ! "${KIND}" get kubeconfig --name="${KIND_CLUSTER_NAME}" >/dev/null 2>&1; then
    "${KIND}" create cluster --name="${KIND_CLUSTER_NAME}" --image="kindest/node:${KIND_NODE_IMAGE_TAG}" \
        --config="${projectdir}/cluster/local/kind.yaml"
fi

nexus_manifest="${projectdir}/cluster/local/nexus.yaml"

# By default the kind node pulls the Nexus image itself (the manifest pins a
# tag, so the pull policy is IfNotPresent). Pre-loading it through the host
# Docker daemon only pays off when the host cache outlives the cluster, as on
# a dev machine: `kind load` re-exports and re-imports the image, which costs
# more than a direct pull on an ephemeral CI runner. Set
# E2E_PRELOAD_IMAGES=true to reuse the host cache.
if [ "${E2E_PRELOAD_IMAGES:-false}" = "true" ]; then
    # kind_load_image loads a pulled image into the kind cluster. Docker hosts
    # using the containerd image store keep the multi-arch index but only the
    # host platform's layers, which makes `kind load docker-image` fail with
    # "content digest ... not found"; fall back to a single-platform archive.
    kind_load_image() {
        local image=$1 archive
        if "${KIND}" load docker-image "${image}" --name="${KIND_CLUSTER_NAME}"; then
            return 0
        fi
        echo "kind load docker-image failed for ${image}, retrying with a single-platform archive"
        archive="$(mktemp)"
        docker save --platform "$(docker version --format '{{.Server.Os}}/{{.Server.Arch}}')" -o "${archive}" "${image}"
        "${KIND}" load image-archive "${archive}" --name="${KIND_CLUSTER_NAME}"
        rm -f "${archive}"
    }

    # The image is pinned in the manifest, which Renovate keeps up to date;
    # read it from there so the pin lives in a single place.
    nexus_image="$(sed -n 's/^[[:space:]]*image:[[:space:]]*\(docker\.io\/sonatype\/nexus3:[^[:space:]]*\)$/\1/p' "${nexus_manifest}" | head -n1)"
    [ -n "${nexus_image}" ] || echo_error "could not read the Nexus image from ${nexus_manifest}"

    echo_step "pre-loading the Nexus image from the host Docker cache"
    docker pull "${nexus_image}"
    kind_load_image "${nexus_image}"
fi

echo_step "installing crossplane"
make -C "${projectdir}" controlplane.up

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
# local.xpkg.deploy.provider.<name> patches Crossplane with a dev sidecar,
# extracts the xpkg into the cache under the cache key Crossplane >=2.2
# expects (xpkg.crossplane.internal/dev/<name>@<digest>), kubectl-cps the
# cache into the sidecar, loads the controller image into kind, and creates a
# Provider + DeploymentRuntimeConfig wired to the local image.
# See build/makelib/local.xpkg.mk.
make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"

# Start Nexus now, so that its image pull and slow first boot (a few minutes)
# overlap with the provider establishing instead of running after it. Not any
# earlier: the Crossplane pod and its dev sidecar pull their own images while
# local.xpkg.deploy.provider waits on them, and the kind node also pulling
# Nexus then slows those waits down. nexus_setup.sh re-applies the same
# manifest (idempotently) and waits for Nexus later, once the provider CRDs
# its ProviderConfigs need are established.
echo_step "starting Nexus in the background"
"${KUBECTL}" apply -f "${nexus_manifest}"

# Crossplane's ProviderRevision establish reconciler has a known flake with
# large packages: on a fresh install it sometimes establishes only a subset of
# the package's CRDs and never converges on the rest, with no error or warning
# logged. Deleting and recreating the Provider forces a fresh
# ProviderRevision, which reaches full establishment on a later attempt, so
# poll for every CRD and retry a bounded number of times rather than failing
# on the first partial establish. With safe-start, the managed resource CRDs
# are only created once their ManagedResourceDefinitions are activated, after
# the ProviderConfig ones, so the first CRDs to appear are not all of them.
# The label crossplane sets is on the revision, not the CRDs themselves, so
# select them by group name instead.
expected_crd_count="$(find "${projectdir}/package/crds" -name '*.yaml' | wc -l | tr -d ' ')"

provider_crds() {
    "${KUBECTL}" get crds -o name 2>/dev/null | grep '\.nexus\.crossplane\.io$' || true
}

wait_for_crds_created() {
    local timeout=$1 elapsed=0
    while [ "${elapsed}" -lt "${timeout}" ]; do
        if [ "$(provider_crds | grep -c . || true)" -ge "${expected_crd_count}" ]; then
            return 0
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    return 1
}

echo_step "waiting for all ${expected_crd_count} provider CRDs to establish"
max_attempts=5
attempt=1
until wait_for_crds_created 90; do
    count="$(provider_crds | grep -c . || true)"
    if [ "${attempt}" -ge "${max_attempts}" ]; then
        echo_error "provider only created ${count}/${expected_crd_count} CRDs after ${max_attempts} attempts"
    fi
    echo "only ${count}/${expected_crd_count} CRDs created after attempt ${attempt}/${max_attempts}; recreating Provider to force a fresh ProviderRevision"
    "${KUBECTL}" delete "provider.pkg.crossplane.io/${PACKAGE_NAME}" --wait=true --timeout=60s || true
    make -C "${projectdir}" "local.xpkg.deploy.provider.${PACKAGE_NAME}"
    attempt=$((attempt + 1))
done
provider_crds | xargs "${KUBECTL}" wait --for=condition=Established --timeout=120s

echo_step "waiting for provider to become healthy"
"${KUBECTL}" wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" \
    --for=condition=healthy --timeout=300s

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
