package v1

import (
	rApi "github.com/kloudlite/operator/pkg/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/kloudlite/operator/pkg/constants"
)

type ResourceMeterSpec struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ResourceMeter is the Schema for the resourcemeters API
type ResourceMeter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceMeterSpec `json:"spec,omitempty"`
	Status rApi.Status       `json:"status,omitempty" graphql:"noinput"`
}

func (n *ResourceMeter) EnsureGVK() {
	if n != nil {
		n.SetGroupVersionKind(GroupVersion.WithKind("ResourceMeter"))
	}
}

func (n *ResourceMeter) GetStatus() *rApi.Status {
	return &n.Status
}

func (n *ResourceMeter) GetEnsuredLabels() map[string]string {
	return map[string]string{
		constants.ResourceMeterNameKey:     n.Name,
	}
}

func (n *ResourceMeter) GetEnsuredAnnotations() map[string]string {
	return map[string]string{
		constants.GVKKey: GroupVersion.WithKind("ResourceMeter").String(),
	}
}

//+kubebuilder:object:root=true

// ResourceMeterList contains a list of ResourceMeter
type ResourceMeterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceMeter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceMeter{}, &ResourceMeterList{})
}
