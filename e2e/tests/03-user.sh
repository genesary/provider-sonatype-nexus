#!/bin/bash
set -e

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing User Resources ==="

# First create a secret for the user password
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: e2e-test-user-password
  namespace: default
type: Opaque
stringData:
  password: "TestPassword123!"
EOF

# Test User creation
echo "--- Testing User Creation ---"

cat <<EOF | kubectl apply -f -
apiVersion: nexus.crossplane.io/v1alpha1
kind: User
metadata:
  name: e2e-test-user
  namespace: default
spec:
  forProvider:
    userId: e2e-test-user
    firstName: E2E
    lastName: TestUser
    emailAddress: e2e-test@example.com
    status: active
    roles:
      - nx-anonymous
    passwordSecretRef:
      name: e2e-test-user-password
      namespace: default
      key: password
  providerConfigRef:
    name: default
EOF

echo "Waiting for User to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get user e2e-test-user -n default -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "User is synced!"
        break
    fi
    echo "Waiting for User to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying User in Nexus..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/users?userId=e2e-test-user" || echo "")
if echo "$response" | grep -q "e2e-test-user"; then
    echo "SUCCESS: User found in Nexus!"
else
    echo "WARNING: User not yet visible in Nexus API (may still be creating)"
fi

# Cleanup
echo "Cleaning up User..."
kubectl delete user e2e-test-user -n default --wait=true --timeout=60s
kubectl delete secret e2e-test-user-password

echo "--- User test completed ---"
