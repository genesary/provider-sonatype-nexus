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

package config_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	nexusv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// inUseHoldPeriod is how long a deleted ClusterProviderConfig must survive
// while a managed resource still uses it.
const inUseHoldPeriod = 20 * time.Second

// TestClusterProviderConfigInUseBlocksDeletion checks that a
// ClusterProviderConfig used by a managed resource records that usage and is
// only removed once the managed resource is gone.
func TestClusterProviderConfigInUseBlocksDeletion(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)
	ctx := context.Background()

	// Copy the shared ClusterProviderConfig, so that this test can delete its
	// own without affecting the others.
	shared := &nexusv1alpha1.ClusterProviderConfig{}
	err := f.Kube.Get(ctx, client.ObjectKey{Name: f.ProviderConfigName}, shared)
	if err != nil {
		t.Fatalf("getting ClusterProviderConfig %q: %v", f.ProviderConfigName, err)
	}

	cpc := &nexusv1alpha1.ClusterProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-in-use"},
		Spec:       shared.Spec,
	}
	f.Apply(t, cpc)
	t.Cleanup(func() {
		f.Delete(t, cpc)

		waitErr := f.WaitForDeletion(context.Background(), cpc, time.Minute)
		if waitErr != nil {
			t.Logf("cleanup: waiting for deletion of ClusterProviderConfig %q: %v", cpc.GetName(), waitErr)
		}
	})

	role := &iamv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-cpc-in-use", Namespace: "default"},
		Spec: iamv1alpha1.RoleSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: nexusv1alpha1.ClusterProviderConfigKind,
					Name: cpc.GetName(),
				},
			},
			ForProvider: iamv1alpha1.RoleParameters{
				ID:   "e2e-cpc-in-use",
				Name: "E2E ClusterProviderConfig in use",
			},
		},
	}
	f.CreateAndWaitForReady(t, role, 2*time.Minute)

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		getErr := f.Kube.Get(ctx, client.ObjectKeyFromObject(cpc), cpc)

		return getErr == nil && cpc.Status.Users == 1, getErr
	})
	if err != nil {
		t.Fatalf("waiting for ClusterProviderConfig to record 1 user: %v (users = %d)", err, cpc.Status.Users)
	}

	f.Delete(t, cpc)

	// The ClusterProviderConfig must outlive its deletion while in use.
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, inUseHoldPeriod, true, func(ctx context.Context) (bool, error) {
		return false, f.Kube.Get(ctx, client.ObjectKeyFromObject(cpc), cpc)
	})
	if !wait.Interrupted(err) {
		t.Fatalf("ClusterProviderConfig in use was removed: %v", err)
	}

	f.Delete(t, role)

	err = f.WaitForDeletion(ctx, role, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for Role deletion: %v", err)
	}

	err = f.WaitForDeletion(ctx, cpc, time.Minute)
	if err != nil {
		t.Fatalf("waiting for the unused ClusterProviderConfig to be removed: %v", err)
	}
}
