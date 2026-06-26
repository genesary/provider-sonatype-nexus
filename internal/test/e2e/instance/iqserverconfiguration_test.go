//go:build e2e

package instance_test

import (
	"testing"
	"time"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancev1alpha1 "github.com/genesary/provider-sonatype-nexus/apis/instance/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/test/e2e"
)

func TestIQServerConfigurationUpdate(t *testing.T) {
	t.Parallel()

	f := e2e.New(t)

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
				Enabled:  false,
				ShowLink: false,
			},
		},
	}

	f.CreateAndWaitForReady(t, cr, 2*time.Minute)
	e2e.AssertReady(t, cr)
	e2e.AssertSynced(t, cr)
}
