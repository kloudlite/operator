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
	Resources                     ct.Resources `json:"resources"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	ReplicaCount int `json:"replicaCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Last_Reconciled_At,type=date
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name=Age,type=date

// StandaloneService is the Schema for the standaloneservices API
type StandaloneService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StandaloneServiceSpec `json:"spec"`
	Status rApi.Status           `json:"status,omitempty"`

	Output ct.ManagedServiceOutput `json:"output"`
}

func (ss *StandaloneService) EnsureGVK() {
	if ss != nil {
		ss.SetGroupVersionKind(GroupVersion.WithKind("StandaloneService"))
	}
}

func (ss *StandaloneService) GetStatus() *rApi.Status {
	return &ss.Status
}

func (ss *StandaloneService) GetEnsuredLabels() map[string]string {
	return map[string]string{constants.MsvcNameKey: ss.Name}
}

func (ss *StandaloneService) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
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
