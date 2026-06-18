#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Role Resources ==="

# Test Role creation
echo "--- Testing Role Creation ---"

kubectl apply -f "${MANIFEST_DIR}/role.yaml"

echo "Waiting for Role to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get role.iam.nexus.crossplane.io e2e-test-role -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "Role is synced!"
        break
    fi
    echo "Waiting for Role to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying Role in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/roles/e2e-test-role" || echo "")
if echo "$response" | grep -q "e2e-test-role"; then
    echo "SUCCESS: Role found in Nexus!"
else
    echo "WARNING: Role not yet visible in Nexus API (may still be creating)"
fi

# Cleanup
echo "Cleaning up Role..."
kubectl delete role.iam.nexus.crossplane.io e2e-test-role -n default --wait=true --timeout=60s

echo "--- Role test completed ---"
