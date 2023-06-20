package operator

import (
	"github.com/kloudlite/operator/logging"
	rawJson "github.com/kloudlite/operator/pkg/raw-json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:object:generate=true

type Check struct {
	Status     bool   `json:"status"`
	Message    string `json:"message,omitempty"`
	Generation int64  `json:"generation,omitempty"`
}

type ResourceRef struct {
	metav1.TypeMeta `json:",inline"`
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:printcolumn:JSONPath=".status.isReady",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

type Status struct {
	// +kubebuilder:validation:Optional
	IsReady   bool             `json:"isReady"`
	Resources []ResourceRef    `json:"resources,omitempty"`
	Message   *rawJson.RawJson `json:"message,omitempty"`

	Checks            map[string]Check `json:"checks,omitempty"`
	LastReconcileTime *metav1.Time     `json:"lastReconcileTime,omitempty"`
}

type Reconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager, logger logging.Logger) error
	GetName() string
}
