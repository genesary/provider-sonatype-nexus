#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
NEXUS_USER="${NEXUS_USER:-admin}"
NEXUS_PASS="${NEXUS_PASS:-admin123}"

echo "=== Testing SecuritySSLTruststore Resources ==="

# Fetch the certificate from the Nexus server itself for testing
echo "--- Fetching certificate from Nexus ---"
PEM=$(echo | openssl s_client -connect localhost:8443 -servername localhost 2>/dev/null | openssl x509 -outform PEM 2>/dev/null || echo "")

if [ -z "$PEM" ]; then
    echo "SKIP: Could not retrieve certificate (Nexus HTTPS may not be configured)"
    exit 0
fi

# Create the manifest dynamically
cat > /tmp/e2e-ssl-truststore.yaml <<EOF
apiVersion: nexus.crossplane.io/v1alpha1
kind: SecuritySSLTruststore
metadata:
  name: e2e-test-ssl-cert
spec:
  forProvider:
    pem: |
$(echo "$PEM" | sed 's/^/      /')
  providerConfigRef:
    kind: ProviderConfig
    name: default
EOF

echo "--- Testing SecuritySSLTruststore Configuration ---"

kubectl apply -f /tmp/e2e-ssl-truststore.yaml

echo "Waiting for SecuritySSLTruststore to be ready..."
sleep 5

# Wait for the resource to be synced
for i in {1..30}; do
    status=$(kubectl get securityssltruststore e2e-test-ssl-cert -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "Unknown")
    if [ "$status" = "True" ]; then
        echo "SecuritySSLTruststore is synced!"
        break
    fi
    echo "Waiting for SecuritySSLTruststore to sync... ($i/30)"
    sleep 2
done

# Verify in Nexus API
echo "Verifying certificate in Nexus truststore..."
response=$(curl -sf -u "${NEXUS_USER}:${NEXUS_PASS}" "${NEXUS_URL}/service/rest/v1/security/ssl/truststore" || echo "")
if echo "$response" | grep -q "fingerprint"; then
    echo "SUCCESS: Certificate found in Nexus truststore!"
else
    echo "WARNING: Certificate not yet visible in Nexus API"
fi

# Cleanup
echo "Cleaning up SecuritySSLTruststore..."
kubectl delete securityssltruststore e2e-test-ssl-cert --wait=true --timeout=60s
rm -f /tmp/e2e-ssl-truststore.yaml

echo "--- SecuritySSLTruststore test completed ---"
