package nexus

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// secretMemo is a kubeclient.Reader that fetches each Secret at most once and
// passes every other read through.
type secretMemo struct {
	kubeclient.Reader

	secrets map[types.NamespacedName]*corev1.Secret
}

// MemoizeSecrets returns a reader that fetches each Secret from kube at most
// once and returns a copy of it on later reads.
//
// With the Secret cache disabled every Secret read is an API request, and one
// operation often reads the same Secret more than once: for its fingerprint
// and then for its value, or for two keys of the same Secret. Wrapping kube
// once per operation makes those reads cost one request, and ensures a
// fingerprint and the values read with it come from the same Secret version.
//
// The returned reader is not safe for concurrent use and never sees Secret
// changes made after its first read, so it must not outlive one operation.
func MemoizeSecrets(kube kubeclient.Reader) kubeclient.Reader {
	if _, ok := kube.(*secretMemo); ok {
		return kube
	}

	return &secretMemo{Reader: kube, secrets: map[types.NamespacedName]*corev1.Secret{}}
}

// Get reads obj, serving Secrets from the memo once they have been fetched.
// Failed reads are not memoized.
func (m *secretMemo) Get(ctx context.Context, key types.NamespacedName, obj kubeclient.Object, opts ...kubeclient.GetOption) error {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return m.Reader.Get(ctx, key, obj, opts...)
	}

	if cached, found := m.secrets[key]; found {
		cached.DeepCopyInto(secret)

		return nil
	}

	err := m.Reader.Get(ctx, key, secret, opts...)
	if err != nil {
		return err
	}

	m.secrets[key] = secret.DeepCopy()

	return nil
}
