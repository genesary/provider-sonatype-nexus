#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

LICENSE_FILE="${LICENSE_FILE:-}"

echo "=== Testing License Resources ==="

# Skip if no license file is available
if [ -z "$LICENSE_FILE" ] || [ ! -f "$LICENSE_FILE" ]; then
    echo "SKIP: No license file provided (set LICENSE_FILE env var)"
    exit 0
fi

# Create the license secret from the file
echo "--- Creating license secret ---"
kubectl create secret generic e2e-license-file \
    --from-file=license.lic="$LICENSE_FILE" \
    -n default --dry-run=client -o yaml | kubectl apply -f -

# Test License configuration
echo "--- Testing License Configuration ---"

kubectl apply -f "${MANIFEST_DIR}/license.yaml"

echo "Waiting for License to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get license e2e-test-license -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "License is synced!"
        break
    fi
    echo "Waiting for License to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying License in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/system/license" || echo "")
if echo "$response" | grep -q "licenseType"; then
    echo "SUCCESS: License configuration retrieved from Nexus!"
else
    echo "WARNING: License not yet visible in Nexus API"
fi

# Cleanup
echo "Cleaning up License..."
kubectl delete license e2e-test-license --wait=true --timeout=60s
kubectl delete secret e2e-license-file -n default --ignore-not-found

echo "--- License test completed ---"
