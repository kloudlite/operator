package v1

import (
	"fmt"

	"operators.kloudlite.io/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rApi "operators.kloudlite.io/pkg/operator"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	AccountRef string `json:"accountRef,omitempty"`
	// DisplayName of Project
	DisplayName string `json:"displayName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".spec.accountRef",name=AccountRef,type=string
// +kubebuilder:printcolumn:JSONPath=".status.isReady",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec `json:"spec,omitempty"`
	Status rApi.Status `json:"status,omitempty"`
}

func (p *Project) LogRef() string {
	return fmt.Sprintf("%s/%s/%s", p.Namespace, p.Kind, p.Name)
}

func (p *Project) GetStatus() *rApi.Status {
	return &p.Status
}

func (p *Project) GetEnsuredLabels() map[string]string {
	return map[string]string{constants.ProjectNameKey: p.Name}
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
