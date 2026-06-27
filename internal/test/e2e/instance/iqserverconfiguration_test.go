//go:build e2e

package instance_test

import (
	"context"
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestIQServerConfigurationUpdate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

	const secretName = "e2e-iq-credentials"

	iqSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": "e2e-iq-user",
			"password": "E2ePassword123!",
		},
	}
	if err := f.Kube.Create(context.Background(), iqSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating IQ credentials secret: %v", err)
	}
	t.Cleanup(func() {
		_ = f.Kube.Delete(context.Background(), iqSecret)
	})

	cr := &instancev1alpha1.IQServerConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-iq-connection", Namespace: "default"},
		Spec: instancev1alpha1.IQServerConfigurationSpec{
			ManagedResourceSpec: xpv2.ManagedResourceSpec{
				ProviderConfigReference: &xpv2.ProviderConfigReference{
					Kind: "ProviderConfig",
					Name: f.ProviderConfigName,
				},
			},
			ForProvider: instancev1alpha1.IQServerConfigurationParameters{
				Enabled:            ptr.To(false),
				ShowLink:           ptr.To(false),
				URL:                "http://iq-server:8070",
				AuthenticationType: ptr.To("USER"),
				UsernameSecretRef: &xpv2.SecretKeySelector{
					Key: "username",
					SecretReference: xpv2.SecretReference{
						Name:      secretName,
						Namespace: "default",
					},
				},
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

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)
}
