package v1

import (
	"fmt"

	"github.com/kloudlite/operator/pkg/constants"
	"github.com/kloudlite/operator/toolkit/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnvironmentRouting struct {
	Mode                EnvironmentRoutingMode `json:"mode,omitempty"`
	PublicIngressClass  string                 `json:"publicIngressClass,omitempty" graphql:"noinput"`
	PrivateIngressClass string                 `json:"privateIngressClass,omitempty" graphql:"noinput"`
}

type EnvironmentRoutingMode string

const (
	EnvironmentRoutingModePublic  EnvironmentRoutingMode = "public"
	EnvironmentRoutingModePrivate EnvironmentRoutingMode = "private"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	TargetNamespace string `json:"targetNamespace,omitempty"`

	Routing *EnvironmentRouting `json:"routing,omitempty"`

	Suspend bool `json:"suspend,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:JSONPath=".spec.targetNamespace",name="target-ns",type=string
//+kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Seen,type=date
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
//+kubebuilder:printcolumn:JSONPath=".spec.suspend",name=Suspend,type=boolean
//+kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string
//+kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// Environment is the Schema for the environments API
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status reconciler.Status `json:"status,omitempty" graphql:"noinput"`
}

func (e *Environment) EnsureGVK() {
	if e != nil {
		e.SetGroupVersionKind(GroupVersion.WithKind("Environment"))
	}
}

func (e *Environment) GetIngressClassName() string {
	if e.Spec.Routing == nil {
		return string(EnvironmentRoutingModePrivate)
	}

	if e.Spec.Routing.Mode == EnvironmentRoutingModePublic {
		return string(e.Spec.Routing.PublicIngressClass)
	}

	return string(e.Spec.Routing.PrivateIngressClass)
}

func (e *Environment) GetStatus() *reconciler.Status {
	return &e.Status
}

func (e *Environment) GetEnsuredLabels() map[string]string {
	return map[string]string{
		constants.TargetNamespaceKey: e.Spec.TargetNamespace,
	}
}

func (e *Environment) GetEnsuredAnnotations() map[string]string {
	m := map[string]string{}
	if e.Spec.Routing != nil {
		m["kloudlite.io/environment.routing"] = fmt.Sprintf("%s (%s)", e.Spec.Routing.Mode, func() string {
			if e.Spec.Routing.Mode == EnvironmentRoutingModePublic {
				return e.Spec.Routing.PublicIngressClass
			}
			return e.Spec.Routing.PrivateIngressClass
		}())
	}

	return m
}

//+kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
