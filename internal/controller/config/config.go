// Package config provides a controller for ProviderConfig resources.
package config

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
)

// Setup adds controllers that reconcile ProviderConfigs by
// accounting for their current usage.
func Setup(mgr ctrl.Manager, opts controller.Options) error {
	err := setupNamespacedProviderConfig(mgr, opts)
	if err != nil {
		return err
	}

	return setupClusterProviderConfig(mgr, opts)
}

// setupNamespacedProviderConfig sets up the ProviderConfig controller.
func setupNamespacedProviderConfig(mgr ctrl.Manager, opts controller.Options) error {
	return setupProviderConfig(
		mgr,
		opts,
		v1alpha1.ProviderConfigGroupKind,
		resource.ProviderConfigKinds{
			Config:    v1alpha1.ProviderConfigGroupVersionKind,
			Usage:     v1alpha1.ProviderConfigUsageGroupVersionKind,
			UsageList: v1alpha1.ProviderConfigUsageListGroupVersionKind,
		},
		&v1alpha1.ProviderConfig{},
		&v1alpha1.ProviderConfigUsage{},
	)
}

// setupClusterProviderConfig sets up the ClusterProviderConfig controller.
func setupClusterProviderConfig(mgr ctrl.Manager, opts controller.Options) error {
	return setupProviderConfig(
		mgr,
		opts,
		v1alpha1.ClusterProviderConfigGroupKind,
		resource.ProviderConfigKinds{
			Config:    v1alpha1.ClusterProviderConfigGroupVersionKind,
			Usage:     v1alpha1.ClusterProviderConfigUsageGroupVersionKind,
			UsageList: v1alpha1.ClusterProviderConfigUsageListGroupVersionKind,
		},
		&v1alpha1.ClusterProviderConfig{},
		&v1alpha1.ClusterProviderConfigUsage{},
	)
}

// setupProviderConfig wires up a providerconfig controller and reconciler.
func setupProviderConfig(
	mgr ctrl.Manager,
	opts controller.Options,
	groupKind string,
	kinds resource.ProviderConfigKinds,
	config client.Object,
	usage client.Object,
) error {
	name := providerconfig.ControllerName(groupKind)

	reconciler := providerconfig.NewReconciler(mgr, kinds,
		providerconfig.WithLogger(opts.Logger.WithValues("controller", name)),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		For(config).
		Watches(usage, &resource.EnqueueRequestForProviderConfig{}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}
