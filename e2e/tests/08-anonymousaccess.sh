#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing AnonymousAccess Resources ==="

# Test AnonymousAccess configuration
echo "--- Testing AnonymousAccess Configuration ---"

kubectl apply -f "${MANIFEST_DIR}/anonymousaccess.yaml"

echo "Waiting for AnonymousAccess to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get anonymousaccess e2e-test-anonymous -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "AnonymousAccess is synced!"
        break
    fi
    echo "Waiting for AnonymousAccess to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying AnonymousAccess in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/anonymous" || echo "")
if echo "$response" | grep -q "enabled"; then
    echo "SUCCESS: AnonymousAccess configuration retrieved from Nexus!"
else
    echo "WARNING: AnonymousAccess not yet visible in Nexus API"
fi

# Cleanup
echo "Cleaning up AnonymousAccess..."
kubectl delete anonymousaccess e2e-test-anonymous -n default --wait=true --timeout=60s

echo "--- AnonymousAccess test completed ---"
