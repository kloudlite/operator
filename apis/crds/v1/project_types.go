package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kloudlite/operator/pkg/constants"
	rApi "github.com/kloudlite/operator/pkg/operator"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	TargetNamespace string `json:"targetNamespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".spec.targetNamespace",name="target-namespace",type=string
// +kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Last_Reconciled_At,type=date
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec `json:"spec"`
	Status rApi.Status `json:"status,omitempty" graphql:"noinput"`
}

func (p *Project) EnsureGVK() {
	if p != nil {
		p.SetGroupVersionKind(GroupVersion.WithKind("Project"))
	}
}

func (p *Project) GetStatus() *rApi.Status {
	return &p.Status
}

func (p *Project) GetEnsuredLabels() map[string]string {
	labels := map[string]string{
		constants.ProjectNameKey:     p.Name,
		constants.TargetNamespaceKey: p.Spec.TargetNamespace,
	}
	return labels
}

func (p *Project) GetEnsuredAnnotations() map[string]string {
	return map[string]string{
		constants.AnnotationKeys.GroupVersionKind: GroupVersion.WithKind("Project").String(),
	}
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
