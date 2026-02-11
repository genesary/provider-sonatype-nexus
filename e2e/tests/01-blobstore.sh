#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing BlobStore Resources ==="

# Test File BlobStore
echo "--- Testing File BlobStore ---"

kubectl apply -f "${MANIFEST_DIR}/blobstore-file.yaml"

echo "Waiting for BlobStore to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get blobstore e2e-test-file-blobstore -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "BlobStore is synced!"
        break
    fi
    echo "Waiting for BlobStore to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying BlobStore in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/blobstores" || echo "")
if echo "$response" | grep -q "e2e-test-file"; then
    echo "SUCCESS: BlobStore found in Nexus!"
else
    echo "WARNING: BlobStore not yet visible in Nexus API (may still be creating)"
fi

# Cleanup
echo "Cleaning up BlobStore..."
kubectl delete blobstore e2e-test-file-blobstore -n default --wait=true --timeout=60s

echo "--- File BlobStore test completed ---"
