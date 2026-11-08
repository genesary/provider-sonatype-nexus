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

package instance_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	v395 "github.com/sonatype-nexus-community/nexus-repo-api-client-go/v395"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

// TestTaskLifecycle checks that a Task creates a weekly blob store
// compaction in Nexus, corrects drift made outside the provider, including a
// property and an alert email added behind its back, and deletes the task
// once deleted.
func TestTaskLifecycle(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const taskName = "e2e-compact-default"

	start := metav1.NewTime(time.Now().Add(24 * time.Hour).UTC().Truncate(time.Minute))
	wantProperties := map[string]string{"blobstoreName": "default"}

	cr := &instancev1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-compact-default", Namespace: "default"},
		Spec: instancev1alpha1.TaskSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.TaskParameters{
				Name:    taskName,
				Type:    "blobstore.compact",
				Enabled: new(true),
				Schedule: instancev1alpha1.TaskSchedule{
					Type:          instancev1alpha1.TaskScheduleWeekly,
					StartDate:     &start,
					RecurringDays: []int32{1, 7},
				},
				Properties: wantProperties,
			},
		},
	}

	matchesSpec := func(got *v395.TaskXO) bool {
		days := slices.Clone(got.RecurringDays)
		slices.Sort(days)

		return got.GetEnabled() &&
			got.GetAlertEmail() == "" &&
			got.GetSchedule() == "weekly" &&
			got.StartDate != nil && got.StartDate.Equal(start.Time) &&
			slices.Equal(days, []int32{1, 7}) &&
			maps.Equal(got.GetProperties(), wantProperties)
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)
	e2e.AssertExternalName(t, cr, taskName)

	got, err := f.FetchTask(taskName)
	if err != nil {
		t.Fatalf("fetching task from Nexus: %v", err)
	}

	if got == nil {
		t.Fatalf("task %q not found in Nexus", taskName)
	}

	if !matchesSpec(got) {
		t.Fatalf("Nexus task = %+v, want an enabled weekly compaction of the default blob store", got)
	}

	if cr.Status.AtProvider.ID != got.GetId() {
		t.Errorf("AtProvider.ID = %q, want %q", cr.Status.AtProvider.ID, got.GetId())
	}

	// Drift made directly in Nexus must be reverted to the spec, and the
	// property and alert email added out of band removed: the spec is the
	// complete set.
	err = f.UpdateTask(got.GetId(), v395.UpdateTasksRequest{
		Name:                  taskName,
		Enabled:               false,
		AlertEmail:            new("drift@example.com"),
		NotificationCondition: "FAILURE",
		Frequency:             v395.FrequencyXO{Schedule: "manual"},
		Properties:            &map[string]string{"blobstoreName": "default", "blobsOlderThan": "7"},
	})
	if err != nil {
		t.Fatalf("injecting task drift: %v", err)
	}

	err = f.Nudge(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	err = f.WaitForNexus(context.Background(), 2*time.Minute, func() (bool, error) {
		observed, fetchErr := f.FetchTask(taskName)
		if fetchErr != nil || observed == nil {
			return false, fetchErr
		}

		return matchesSpec(observed), nil
	})
	if err != nil {
		observed, _ := f.FetchTask(taskName)
		t.Fatalf("task drift was not corrected: %v\n  nexus reports: %+v", err, observed)
	}

	f.Delete(t, cr)

	err = f.WaitForDeletion(context.Background(), cr, e2e.DefaultDeleteTimeout)
	if err != nil {
		t.Fatalf("waiting for Task deletion: %v", err)
	}

	got, err = f.FetchTask(taskName)
	if err != nil {
		t.Fatalf("fetching task from Nexus after deletion: %v", err)
	}

	if got != nil {
		t.Errorf("task %q still exists in Nexus after deletion", taskName)
	}
}
