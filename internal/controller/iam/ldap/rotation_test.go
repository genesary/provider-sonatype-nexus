package ldap

import (
	"context"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	iammocks "github.com/genesary/provider-sonatype-nexus/test/mocks/iam"
)

// TestAuthPasswordRotation verifies that rotating or repointing the bind
// password Secret is reported as drift and re-applied to Nexus, even though
// Nexus never returns the bind password.
func TestAuthPasswordRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	newSecret := func(name, password string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Data:       map[string][]byte{"password": []byte(password)},
		}
	}

	cr := newTestLDAP("corp-ldap", "ldap.example.com")
	cr.Namespace = "default"
	cr.Spec.ForProvider.AuthPasswordSecretRef = &xpv2.LocalSecretKeySelector{
		LocalSecretReference: xpv2.LocalSecretReference{Name: "bind-v1"},
		Key:                  "password",
	}

	secret := newSecret("bind-v1", "old-password")
	kube := fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(cr, secret, newSecret("bind-v2", "repointed-password")).
		WithStatusSubresource(cr).
		Build()

	var applied []string

	mc := iammocks.NewMockLDAPClient()
	mc.CreateFn = func(ldap security.LDAP) error {
		applied = append(applied, ldap.AuthPassword)

		return nil
	}
	mc.UpdateFn = func(_ string, ldap security.LDAP) error {
		applied = append(applied, ldap.AuthPassword)

		return nil
	}
	mc.GetFn = func(_ string) (*security.LDAP, error) {
		return &security.LDAP{
			Name: "corp-ldap", Protocol: "ldap", Host: "ldap.example.com", Port: 389,
			SearchBase: "dc=example,dc=com", AuthSchema: "none", UserBaseDN: "ou=people,dc=example,dc=com",
		}, nil
	}

	e := &external{client: mc, kube: kube}

	assertUpToDate := func(step string, want bool) {
		t.Helper()

		obs, err := e.Observe(ctx, cr)
		if err != nil {
			t.Fatalf("%s: Observe() unexpected error: %v", step, err)
		}

		if obs.ResourceUpToDate != want {
			t.Fatalf("%s: ResourceUpToDate = %v, want %v", step, obs.ResourceUpToDate, want)
		}
	}

	_, err := e.Create(ctx, cr)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	assertUpToDate("after create", true)

	secret.Data["password"] = []byte("rotated-password")

	err = kube.Update(ctx, secret)
	if err != nil {
		t.Fatalf("rotating secret: %v", err)
	}

	assertUpToDate("after rotation", false)

	_, err = e.Update(ctx, cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	assertUpToDate("after update", true)

	cr.Spec.ForProvider.AuthPasswordSecretRef.Name = "bind-v2"
	assertUpToDate("after repointing", false)

	want := []string{"old-password", "rotated-password"}
	if len(applied) != len(want) || applied[0] != want[0] || applied[1] != want[1] {
		t.Fatalf("applied passwords = %v, want %v", applied, want)
	}
}
