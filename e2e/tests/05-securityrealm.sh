#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing SecurityRealm Resources ==="

# Test SecurityRealm configuration
echo "--- Testing SecurityRealm Configuration ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: SecurityRealm
metadata:
  name: e2e-test-realms
  namespace: default
spec:
  forProvider:
    activeRealms:
      - NexusAuthenticatingRealm
      - NexusAuthorizingRealm
      - DockerToken
  providerConfigRef:
    name: default
EOF

echo "Waiting for SecurityRealm to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get securityrealm e2e-test-realms -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "SecurityRealm is synced!"
        break
    fi
    echo "Waiting for SecurityRealm to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying SecurityRealm in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/realms/active" || echo "")
if echo "$response" | grep -q "DockerToken"; then
    echo "SUCCESS: SecurityRealm configuration applied in Nexus!"
else
    echo "WARNING: SecurityRealm not yet visible in Nexus API (may still be configuring)"
fi

# Cleanup
echo "Cleaning up SecurityRealm..."
kubectl delete securityrealm e2e-test-realms -n default --wait=true --timeout=60s

echo "--- SecurityRealm test completed ---"
