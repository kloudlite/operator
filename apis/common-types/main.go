package common_types

import (
	"github.com/kloudlite/operator/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:object:generate=true

type ResourceRef struct {
	metav1.TypeMeta `json:",inline"`
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:printcolumn:JSONPath=".status.isReady",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

type Reconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error
	GetName() string
}
