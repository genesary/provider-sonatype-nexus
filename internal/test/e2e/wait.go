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

package e2e

import (
	"context"
	"fmt"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pollInterval is how often the wait helpers poll the API server.
const pollInterval = 2 * time.Second

// Conditioned is the slim contract every Crossplane managed resource
// already satisfies — a Kubernetes object whose status exposes the
// standard Crossplane condition set.
type Conditioned interface {
	client.Object
	GetCondition(ct xpv2.ConditionType) xpv2.Condition
}

// WaitForReady polls until obj reports both Synced=True and Ready=True.
// The supplied object is updated in-place with the latest observed state.
func (f *Framework) WaitForReady(ctx context.Context, obj Conditioned, timeout time.Duration) error {
	key := client.ObjectKeyFromObject(obj)
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		if err := f.Kube.Get(ctx, key, obj); err != nil {
			return false, err
		}
		synced := obj.GetCondition(xpv2.TypeSynced)
		ready := obj.GetCondition(xpv2.TypeReady)
		return synced.Status == corev1.ConditionTrue && ready.Status == corev1.ConditionTrue, nil
	})
}

// WaitForNexus polls Nexus until reached reports that it observes the state
// the spec asks for. Use it after updating a managed resource: the resource
// can still carry Synced=True and Ready=True from the reconcile that preceded
// the change, so its conditions say nothing about whether the change landed.
// The last error returned by reached, if any, is reported on timeout.
func (f *Framework) WaitForNexus(ctx context.Context, timeout time.Duration, reached func() (bool, error)) error {
	var lastErr error

	err := wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(context.Context) (bool, error) {
		ok, err := reached()
		if err != nil {
			lastErr = err

			return false, nil
		}

		return ok, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%w (last error from Nexus: %w)", err, lastErr)
	}

	return err
}

// WaitForDeletion polls until obj is no longer present in the API server.
func (f *Framework) WaitForDeletion(ctx context.Context, obj client.Object, timeout time.Duration) error {
	key := client.ObjectKeyFromObject(obj)
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		err := f.Kube.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

// SummariseConditions renders a one-line diagnostic of the standard
// Crossplane conditions, suitable for inclusion in a t.Fatalf message.
func SummariseConditions(obj Conditioned) string {
	synced := obj.GetCondition(xpv2.TypeSynced)
	ready := obj.GetCondition(xpv2.TypeReady)
	return fmt.Sprintf("Synced=%s(%s: %q) Ready=%s(%s: %q)",
		synced.Status, synced.Reason, synced.Message,
		ready.Status, ready.Reason, ready.Message,
	)
}
