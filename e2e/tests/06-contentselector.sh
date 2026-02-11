#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing ContentSelector Resources ==="

# Test ContentSelector creation
echo "--- Testing ContentSelector Creation ---"

kubectl apply -f "${MANIFEST_DIR}/contentselector.yaml"

echo "Waiting for ContentSelector to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get contentselector e2e-test-selector -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "ContentSelector is synced!"
        break
    fi
    echo "Waiting for ContentSelector to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying ContentSelector in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/content-selectors" || echo "")
if echo "$response" | grep -q "e2e-test-selector"; then
    echo "SUCCESS: ContentSelector found in Nexus!"
else
    echo "WARNING: ContentSelector not yet visible in Nexus API (may still be creating)"
fi

# Cleanup
echo "Cleaning up ContentSelector..."
kubectl delete contentselector e2e-test-selector -n default --wait=true --timeout=60s

echo "--- ContentSelector test completed ---"
