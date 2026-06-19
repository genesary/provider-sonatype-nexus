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
	"k8s.io/utils/ptr"

	iamv1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/iam/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestUserCRUD(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const (
		crName     = "e2e-user-crud"
		userID     = "e2e-user-crud"
		secretName = "e2e-user-crud-password"
	)

	pwSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"password": "E2ePassword123!"},
	}
	if err := f.Kube.Create(context.Background(), pwSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating password secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), pwSecret)
	})

	user := &iamv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{Name: crName, Namespace: "default"},
		Spec: iamv1alpha1.UserSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: iamv1alpha1.UserParameters{
				UserID:       userID,
				FirstName:    "E2E",
				LastName:     "Test",
				EmailAddress: "e2e-user-crud@example.com",
				Status:       ptr.To("active"),
				PasswordSecretRef: &xpv2.SecretKeySelector{
					Key: "password",
					SecretReference: xpv2.SecretReference{
						Name:      secretName,
						Namespace: "default",
					},
				},
			},
		},
	}

	f.CreateAndWaitForReady(t, user, 2*time.Minute)
	e2e.AssertReady(t, user)
	e2e.AssertSynced(t, user)

	got, err := f.FetchUser(userID)
	if err != nil {
		t.Fatalf("fetching user from Nexus: %v", err)
	}
	if got == nil {
		t.Fatalf("user %q not found in Nexus", userID)
	}
	if got.UserID != userID {
		t.Errorf("user ID = %q, want %q", got.UserID, userID)
	}
}
