package user

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

// TestPasswordRotation verifies that rotating or repointing the password
// Secret is reported as drift and re-applied to Nexus, even though Nexus
// never returns the password.
func TestPasswordRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	newSecret := func(name, password string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Data:       map[string][]byte{defaultPasswordKey: []byte(password)},
		}
	}

	cr := newTestUser("alice", "Alice", "Smith")
	cr.Namespace = "default"
	cr.Spec.ForProvider.PasswordSecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{Name: "alice-v1", Namespace: "default"},
		Key:             defaultPasswordKey,
	}

	secret := newSecret("alice-v1", "old-password")
	kube := fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(cr, secret, newSecret("alice-v2", "repointed-password")).
		WithStatusSubresource(cr).
		Build()

	var applied []string

	mc := iammocks.NewMockUserClient()
	mc.CreateFn = func(user security.User) error {
		applied = append(applied, user.Password)

		return nil
	}
	mc.UpdateFn = func(_ string, _ security.User) error { return nil }
	mc.ChangePasswordFn = func(_, password string) error {
		applied = append(applied, password)

		return nil
	}
	mc.GetFn = func(_ string) (*security.User, error) {
		return &security.User{
			UserID: "alice", FirstName: "Alice", LastName: "Smith",
			EmailAddress: "alice@example.com", Status: "active", Source: "default",
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

	secret.Data[defaultPasswordKey] = []byte("rotated-password")

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

	cr.Spec.ForProvider.PasswordSecretRef.Name = "alice-v2"
	assertUpToDate("after repointing", false)

	_, err = e.Update(ctx, cr)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	assertUpToDate("after second update", true)

	want := []string{"old-password", "rotated-password", "repointed-password"}
	if len(applied) != len(want) {
		t.Fatalf("applied passwords = %v, want %v", applied, want)
	}

	for i := range want {
		if applied[i] != want[i] {
			t.Fatalf("applied passwords = %v, want %v", applied, want)
		}
	}

	// A User being deleted must not need its Secret, which a namespace
	// deletion may already have removed.
	cr.Spec.ForProvider.PasswordSecretRef.Name = "deleted"
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)
	assertUpToDate("while deleting", true)
}
