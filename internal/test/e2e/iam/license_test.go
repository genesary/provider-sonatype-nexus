//go:build e2e

/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iam_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestLicenseCR verifies that a License CR can be created and reconciled by
// the controller. A placeholder (invalid) license is used, so we only assert
// that the controller attempts reconciliation (Synced condition is set), not
// that the license is actually installed in Nexus.
//
// This test is NOT parallel since License is a cluster-scoped singleton.
func TestLicenseCR(t *testing.T) {
	f := e2e.New(t)

	const (
		crName     = "e2e-test-license"
		secretName = "e2e-license-file"
	)

	licenseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"license.lic": []byte("placeholder-license-data"),
		},
	}
	if err := f.Kube.Create(context.Background(), licenseSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating license secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), licenseSecret)
	})

	license := &iamv1alpha1.License{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: iamv1alpha1.LicenseSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ClusterProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.LicenseParameters{
				LicenseSecretRef: &xpv2.SecretKeySelector{
					Key: "license.lic",
					SecretReference: xpv2.SecretReference{
						Name:      secretName,
						Namespace: "default",
					},
				},
			},
		},
	}

	f.Apply(t, license)
	t.Cleanup(func() {
		f.Delete(t, license)
	})

	// Wait for the Synced condition to be set (any status) — the placeholder
	// license is invalid so Ready=True is not expected.
	ctx := context.Background()
	key := types.NamespacedName{Name: crName}
	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := f.Kube.Get(ctx, key, license); err != nil {
			return false, err
		}
		synced := license.GetCondition(xpv2.TypeSynced)
		return synced.Status != corev1.ConditionUnknown && synced.Status != "", nil
	})
	if err != nil {
		t.Fatalf("waiting for License Synced condition: %v", err)
	}

	// The Synced condition must be set — its value (True or False) depends on
	// whether the placeholder was accepted by Nexus.
	synced := license.GetCondition(xpv2.TypeSynced)
	if synced.Status == "" {
		t.Error("License Synced condition is unset after reconciliation")
	}
	t.Logf("License Synced=%s (reason=%s): %s", synced.Status, synced.Reason, synced.Message)
}
