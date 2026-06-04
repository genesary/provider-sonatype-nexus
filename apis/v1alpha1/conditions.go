package v1alpha1

import (
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Available returns a condition that indicates the resource is
// available for use.
func Available() xpv2.Condition {
	return xpv2.Condition{
		Type:               xpv2.TypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv2.ReasonAvailable,
	}
}

// Unavailable returns a condition that indicates the resource is not available.
func Unavailable() xpv2.Condition {
	return xpv2.Condition{
		Type:               xpv2.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv2.ReasonUnavailable,
	}
}

// Creating returns a condition that indicates the resource is being created.
func Creating() xpv2.Condition {
	return xpv2.Condition{
		Type:               xpv2.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv2.ReasonCreating,
	}
}

// Deleting returns a condition that indicates the resource is being deleted.
func Deleting() xpv2.Condition {
	return xpv2.Condition{
		Type:               xpv2.TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             xpv2.ReasonDeleting,
	}
}
