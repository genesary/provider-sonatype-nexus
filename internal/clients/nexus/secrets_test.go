package nexus

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// countingReader counts the Gets that reach the wrapped reader, per key.
type countingReader struct {
	kubeclient.Reader

	gets map[types.NamespacedName]int
}

// newCountingReader wraps kube and counts its Gets.
func newCountingReader(kube kubeclient.Reader) *countingReader {
	return &countingReader{Reader: kube, gets: map[types.NamespacedName]int{}}
}

// Get counts the read and passes it through.
func (c *countingReader) Get(ctx context.Context, key types.NamespacedName, obj kubeclient.Object, opts ...kubeclient.GetOption) error {
	c.gets[key]++

	return c.Reader.Get(ctx, key, obj, opts...)
}

// TestMemoizeSecrets verifies that each Secret reaches the API once, that
// callers get independent copies, and that other reads are not memoized.
func TestMemoizeSecrets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	providerConfig := &v1alpha1.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: tenantNamespace}}
	counting := newCountingReader(newKube(t, providerConfig))
	memo := MemoizeSecrets(counting)

	if again := MemoizeSecrets(memo); again != memo {
		t.Error("MemoizeSecrets wrapped a memo a second time")
	}

	tenant := types.NamespacedName{Name: "nexus-creds", Namespace: tenantNamespace}
	victim := types.NamespacedName{Name: "nexus-creds", Namespace: victimNamespace}

	first := &corev1.Secret{}

	err := memo.Get(ctx, tenant, first)
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}

	first.Data["password"] = []byte("mutated")

	second := &corev1.Secret{}

	err = memo.Get(ctx, tenant, second)
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}

	if got := string(second.Data["password"]); got != "secret-"+tenantNamespace {
		t.Errorf("memoized Secret password = %q, want the stored value: callers share one object", got)
	}

	err = memo.Get(ctx, victim, &corev1.Secret{})
	if err != nil {
		t.Fatalf("Get other Secret: %v", err)
	}

	pcKey := types.NamespacedName{Name: "pc", Namespace: tenantNamespace}
	for range 2 {
		err = memo.Get(ctx, pcKey, &v1alpha1.ProviderConfig{})
		if err != nil {
			t.Fatalf("Get ProviderConfig: %v", err)
		}
	}

	want := map[types.NamespacedName]int{tenant: 1, victim: 1, pcKey: 2}
	for key, count := range want {
		if got := counting.gets[key]; got != count {
			t.Errorf("Gets of %s = %d, want %d", key, got, count)
		}
	}
}

// TestMemoizeSecretsRetriesFailedReads verifies that a failed read is not
// memoized, so a Secret created afterwards is found.
func TestMemoizeSecretsRetriesFailedReads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	kube := newKube(t)
	memo := MemoizeSecrets(kube)
	key := types.NamespacedName{Name: "late", Namespace: tenantNamespace}

	err := memo.Get(ctx, key, &corev1.Secret{})
	if err == nil {
		t.Fatal("Get of a missing Secret succeeded")
	}

	err = kube.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}})
	if err != nil {
		t.Fatalf("Create secret: %v", err)
	}

	err = memo.Get(ctx, key, &corev1.Secret{})
	if err != nil {
		t.Errorf("Get after the Secret was created: %v", err)
	}
}

// TestGetCredentialsReadsSharedSecretOnce verifies that a username and
// password stored in one Secret cost one read.
func TestGetCredentialsReadsSharedSecretOnce(t *testing.T) {
	t.Parallel()

	counting := newCountingReader(newKube(t))

	creds, err := GetCredentialsFromSpec(context.Background(), counting, tenantNamespace, newSpec())
	if err != nil {
		t.Fatalf("GetCredentialsFromSpec: %v", err)
	}

	if creds.Username != "admin-"+tenantNamespace || creds.Password != "secret-"+tenantNamespace {
		t.Errorf("credentials = %q/%q, want the values of the tenant Secret", creds.Username, creds.Password)
	}

	key := types.NamespacedName{Name: "nexus-creds", Namespace: tenantNamespace}
	if got := counting.gets[key]; got != 1 {
		t.Errorf("Gets of %s = %d, want 1", key, got)
	}
}
