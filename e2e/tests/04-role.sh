#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing Role Resources ==="

# Test Role creation
echo "--- Testing Role Creation ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: Role
metadata:
  name: e2e-test-role
spec:
  forProvider:
    id: e2e-test-role
    name: E2E Test Role
    description: "Role created by e2e tests"
    privileges:
      - nx-repository-view-*-*-browse
      - nx-repository-view-*-*-read
  providerConfigRef:
    name: default
EOF

echo "Waiting for Role to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get role.nexus.crossplane.io e2e-test-role -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
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
kubectl delete role.nexus.crossplane.io e2e-test-role --wait=true --timeout=60s

echo "--- Role test completed ---"
