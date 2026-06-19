#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing License Resources ==="

# Test License CR creation
echo "--- Testing License CR Creation ---"

kubectl apply -f "${MANIFEST_DIR}/license.yaml"

echo "Waiting for License CR to be accepted by the API server..."
sleep 5

# Verify the CR was created successfully in Kubernetes
license_cr=$(kubectl get license.iam.nexus.crossplane.io e2e-test-license -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
if [ "$license_cr" = "e2e-test-license" ]; then
    echo "SUCCESS: License CR exists in Kubernetes!"
else
    echo "ERROR: License CR was not created"
    exit 1
fi

# Wait for the controller to attempt reconciliation (Synced condition set)
echo "Waiting for License to be reconciled by the controller..."
for i in {1..20}; do
    synced=$(kubectl get license.iam.nexus.crossplane.io e2e-test-license \
        -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "")
    if [ -n "$synced" ]; then
        echo "License reconciled (Synced=${synced}) after $((i * 3))s"
        break
    fi
    echo "Waiting for controller reconciliation... ($i/20)"
    sleep 3
done

# Check if Nexus has a real license installed (only meaningful with a real license file)
echo "--- Checking Nexus license status ---"
license_status=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" \
    "${NEXUS_URL}/service/rest/v1/system/license" \
    -o /dev/null -w "%{http_code}" 2>/dev/null || echo "000")

if [ "$license_status" = "200" ]; then
    echo "SUCCESS: Nexus has a license installed!"
    fingerprint=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" \
        "${NEXUS_URL}/service/rest/v1/system/license" | \
        python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('fingerprint','N/A'))" 2>/dev/null || echo "N/A")
    echo "  Fingerprint: ${fingerprint}"
elif [ "$license_status" = "404" ]; then
    echo "INFO: No license installed on Nexus (placeholder secret used — expected in CI)"
else
    echo "WARNING: License endpoint returned HTTP ${license_status}"
fi

# Cleanup
echo "--- Cleaning up License resources ---"
kubectl delete license.iam.nexus.crossplane.io e2e-test-license --wait=true --timeout=60s || true
kubectl delete secret e2e-license-file -n default || true

echo "--- License test completed ---"
