package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Available returns a condition that indicates the resource is available for use.
func Available() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv1.ReasonAvailable,
	}
}

// Unavailable returns a condition that indicates the resource is not available.
func Unavailable() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv1.ReasonUnavailable,
	}
}

// Creating returns a condition that indicates the resource is being created.
func Creating() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv1.ReasonCreating,
	}
}

// Deleting returns a condition that indicates the resource is being deleted.
func Deleting() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv1.ReasonDeleting,
	}
}
