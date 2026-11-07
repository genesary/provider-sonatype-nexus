// Package config provides a controller for ProviderConfig resources.
package config

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/genesary/provider-sonatype-nexus/apis/v1alpha1"
	"github.com/genesary/provider-sonatype-nexus/internal/helpers"
)

// Setup adds controllers that reconcile ProviderConfigs by
// accounting for their current usage.
func Setup(mgr ctrl.Manager, opts helpers.Options) error {
	err := setupNamespacedProviderConfig(mgr, opts)
	if err != nil {
		return err
	}

	return setupClusterProviderConfig(mgr, opts)
}

// setupNamespacedProviderConfig sets up the ProviderConfig controller.
func setupNamespacedProviderConfig(mgr ctrl.Manager, opts helpers.Options) error {
	return setupProviderConfig(mgr, opts, &v1alpha1.ProviderConfig{}, v1alpha1.ProviderConfigKind)
}

// setupClusterProviderConfig sets up the ClusterProviderConfig controller.
//
// Every managed resource of this provider is namespaced, so every usage record
// is a namespaced ProviderConfigUsage, even for a ClusterProviderConfig: the
// usage lives beside the managed resource that created it and records which
// kind of config it refers to. The reconciler counts the usages of a
// ClusterProviderConfig across all namespaces.
func setupClusterProviderConfig(mgr ctrl.Manager, opts helpers.Options) error {
	return setupProviderConfig(mgr, opts, &v1alpha1.ClusterProviderConfig{}, v1alpha1.ClusterProviderConfigKind)
}

// setupProviderConfig wires up the controller and reconciler of the config
// kind configKind. Both config kinds count the same ProviderConfigUsage type.
// The controller is only enqueued by the usages that refer to configKind, so
// that a ProviderConfig and a ClusterProviderConfig of the same name do not
// wake each other up.
func setupProviderConfig(mgr ctrl.Manager, opts helpers.Options, config client.Object, configKind string) error {
	name := providerconfig.ControllerName(schema.GroupKind{Group: v1alpha1.Group, Kind: configKind}.String())

	kinds := resource.ProviderConfigKinds{
		Config:    v1alpha1.SchemeGroupVersion.WithKind(configKind),
		Usage:     v1alpha1.ProviderConfigUsageGroupVersionKind,
		UsageList: v1alpha1.ProviderConfigUsageListGroupVersionKind,
	}

	reconciler := providerconfig.NewReconciler(mgr, kinds,
		providerconfig.WithLogger(opts.Logger.WithValues("controller", name)),
		providerconfig.WithRecorder(helpers.NewEventRecorder(mgr, name, opts.EventFilterFunctions...)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.ForControllerRuntime()).
		For(config).
		Watches(&v1alpha1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{Kind: configKind}).
		Complete(ratelimiter.NewReconciler(name, reconciler, opts.GlobalRateLimiter))
}
