package v1

import (
	ct "github.com/kloudlite/operator/apis/common-types"
	"github.com/kloudlite/operator/pkg/constants"
	rApi "github.com/kloudlite/operator/pkg/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StandaloneServiceSpec defines the desired state of StandaloneService
type StandaloneServiceSpec struct {
	ct.NodeSelectorAndTolerations `json:",inline"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	ReplicaCount int          `json:"replicaCount,omitempty"`
	Resources    ct.Resources `json:"resources"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="ReplicaCount",type="integer",JSONPath=".spec.replicaCount"
// +kubebuilder:printcolumn:JSONPath=".status.isReady",name=Ready,type=boolean
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// StandaloneService is the Schema for the standaloneservices API
type StandaloneService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneServiceSpec `json:"spec"`
	Status rApi.Status           `json:"status,omitempty"`

  Output ct.ManagedServiceOutput `json:"output,omitempty"`
}

func (s *StandaloneService) EnsureGVK() {
	if s != nil {
		s.SetGroupVersionKind(GroupVersion.WithKind("StandaloneService"))
	}
}

func (s *StandaloneService) GetStatus() *rApi.Status {
	return &s.Status
}

func (s *StandaloneService) GetEnsuredLabels() map[string]string {
	return map[string]string{
		constants.MsvcNameKey: s.Name,
	}
}

func (s *StandaloneService) GetEnsuredAnnotations() map[string]string {
	return map[string]string{
		constants.AnnotationKeys.GroupVersionKind: GroupVersion.WithKind("StandaloneService").String(),
	}
}

// +kubebuilder:object:root=true

// StandaloneServiceList contains a list of StandaloneService
type StandaloneServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StandaloneService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StandaloneService{}, &StandaloneServiceList{})
}
