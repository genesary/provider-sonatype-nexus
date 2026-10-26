package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CredentialsFingerprint returns a fingerprint of the credentials a managed
// resource applies to Nexus, or an empty string when it applies none.
//
// Nexus never returns credentials, so drift detection cannot see a rotated
// Secret or a repointed reference. Controllers record this fingerprint in
// status when they apply credentials and compare it on every Observe instead.
//
// The fingerprint covers the identity and resourceVersion of every Secret the
// selectors reference, never the Secret values, so publishing it in status
// gives readers nothing to test password guesses against. Any change to a
// referenced Secret therefore re-applies the credentials once, even if the
// selected key is unchanged. spec carries the non-secret configuration that
// is applied alongside the credentials but not reported back by Nexus; it is
// JSON-encoded into the fingerprint and may be nil. namespace binds the
// selectors as in GetSecretValue.
func CredentialsFingerprint(
	ctx context.Context,
	kube kubeclient.Client,
	namespace string,
	spec any,
	selectors ...*xpv2.SecretKeySelector,
) (string, error) {
	type secretIdentity struct {
		Namespace       string `json:"namespace"`
		Name            string `json:"name"`
		Key             string `json:"key"`
		UID             string `json:"uid"`
		ResourceVersion string `json:"resourceVersion"`
	}

	secrets := make([]*secretIdentity, len(selectors))
	referenced := spec != nil

	for index, selector := range selectors {
		if selector == nil {
			continue
		}

		key, err := secretObjectKey(namespace, selector)
		if err != nil {
			return "", err
		}

		secret := &corev1.Secret{}

		err = kube.Get(ctx, key, secret)
		if err != nil {
			return "", errors.Wrap(err, "cannot get secret")
		}

		secrets[index] = &secretIdentity{
			Namespace:       key.Namespace,
			Name:            key.Name,
			Key:             selector.Key,
			UID:             string(secret.UID),
			ResourceVersion: secret.ResourceVersion,
		}
		referenced = true
	}

	if !referenced {
		return "", nil
	}

	encoded, err := json.Marshal(struct {
		Secrets []*secretIdentity `json:"secrets"`
		Spec    any               `json:"spec,omitempty"`
	}{Secrets: secrets, Spec: spec})
	if err != nil {
		return "", errors.Wrap(err, "cannot encode credentials fingerprint")
	}

	sum := sha256.Sum256(encoded)

	return hex.EncodeToString(sum[:]), nil
}

// PatchStatus applies mutate to obj and persists the resulting status change.
//
// The managed reconciler overwrites in-memory status changes made during
// Create when it records the external-name annotation, so Create must persist
// a recorded fingerprint itself. Changes made during Observe and Update are
// persisted by the reconciler and need no patch.
func PatchStatus[T kubeclient.Object](ctx context.Context, kube kubeclient.Client, obj T, mutate func(T)) error {
	before, ok := obj.DeepCopyObject().(kubeclient.Object)
	if !ok {
		return errors.New("cannot copy object")
	}

	mutate(obj)

	return errors.Wrap(kube.Status().Patch(ctx, obj, kubeclient.MergeFrom(before)), "cannot patch status")
}
