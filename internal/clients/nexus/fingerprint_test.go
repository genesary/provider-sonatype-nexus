package nexus

import (
	"context"
	"strings"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestCredentialsFingerprint verifies that the fingerprint changes whenever
// the credentials a managed resource applies could have changed.
func TestCredentialsFingerprint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	kube := newKube(t)
	password := newSelector(tenantNamespace, "password")

	fingerprint := func(spec any, selectors ...*xpv2.SecretKeySelector) string {
		t.Helper()

		got, err := CredentialsFingerprint(ctx, kube, tenantNamespace, spec, selectors...)
		if err != nil {
			t.Fatalf("CredentialsFingerprint() unexpected error: %v", err)
		}

		return got
	}

	if got := fingerprint(nil); got != "" {
		t.Errorf("fingerprint without credentials = %q, want empty", got)
	}

	if got := fingerprint(nil, nil, nil); got != "" {
		t.Errorf("fingerprint with unset selectors = %q, want empty", got)
	}

	base := fingerprint(nil, password)
	if base == "" {
		t.Fatal("fingerprint with a selector is empty")
	}

	if strings.Contains(base, "secret-"+tenantNamespace) {
		t.Fatal("fingerprint contains the secret value")
	}

	if got := fingerprint(nil, password); got != base {
		t.Errorf("fingerprint is not stable: %q then %q", base, got)
	}

	if got := fingerprint(nil, newSelector(tenantNamespace, "username")); got == base {
		t.Error("fingerprint did not change when the selector key changed")
	}

	if got := fingerprint(nil, nil, password); got == base {
		t.Error("fingerprint did not change when the selector moved to another field")
	}

	if got := fingerprint(map[string]string{"type": "username"}, password); got == base {
		t.Error("fingerprint did not change when the spec changed")
	}

	secret := &corev1.Secret{}

	err := kube.Get(ctx, types.NamespacedName{Name: "nexus-creds", Namespace: tenantNamespace}, secret)
	if err != nil {
		t.Fatalf("Get secret: %v", err)
	}

	secret.Data["password"] = []byte("rotated")

	err = kube.Update(ctx, secret)
	if err != nil {
		t.Fatalf("Update secret: %v", err)
	}

	if got := fingerprint(nil, password); got == base {
		t.Error("fingerprint did not change when the secret was rotated")
	}
}

// TestCredentialsFingerprintErrors verifies that a missing or cross-namespace
// Secret is reported instead of being fingerprinted as absent.
func TestCredentialsFingerprintErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		selector *xpv2.SecretKeySelector
		wantErr  string
	}{
		"CrossNamespace": {selector: newSelector(victimNamespace, "password"), wantErr: ErrSecretNamespaceMismatch},
		"MissingSecret": {
			selector: &xpv2.SecretKeySelector{SecretReference: xpv2.SecretReference{Name: "missing"}, Key: "password"},
			wantErr:  "cannot get secret",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := CredentialsFingerprint(context.Background(), newKube(t), tenantNamespace, nil, tc.selector)
			assertSecretResult(t, "CredentialsFingerprint", got, err, "", tc.wantErr)
		})
	}
}
