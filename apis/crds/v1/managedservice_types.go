package v1

import (
	ct "operators.kloudlite.io/apis/common-types"
	"operators.kloudlite.io/lib/constants"
	rApi "operators.kloudlite.io/lib/operator"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rawJson "operators.kloudlite.io/lib/raw-json"
)

type msvcKind struct {
	APIVersion string `json:"apiVersion"`
	// +kubebuilder:default=Service
	// +kubebuilder:validation:Optional
	Kind string `json:"kind"`
}

// ManagedServiceSpec defines the desired state of ManagedService
type ManagedServiceSpec struct {
	CloudProvider ct.CloudProvider `json:"cloudProvider"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	MsvcKind     msvcKind            `json:"msvcKind"`
	Inputs       rawJson.KubeRawJson `json:"inputs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ManagedService is the Schema for the managedservices API
type ManagedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedServiceSpec `json:"spec,omitempty"`
	Status rApi.Status        `json:"status,omitempty"`
}

func (m *ManagedService) GetStatus() *rApi.Status {
	return &m.Status
}

func (m *ManagedService) GetEnsuredLabels() map[string]string {
	return map[string]string{
		"kloudlite.io/msvc.name": m.Name,
	}
}

func (m *ManagedService) GetEnsuredAnnotations() map[string]string {
	return map[string]string{
		constants.AnnotationKeys.GroupVersionKind: GroupVersion.WithKind("ManagedService").String(),
	}
}

func (m *ManagedService) GetWatchLabels() map[string]string {
	return map[string]string{
		"msvc.kloudlite.io/ref": m.Name,
	}
}

// +kubebuilder:object:root=true

// ManagedServiceList contains a list of ManagedService
type ManagedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedService{}, &ManagedServiceList{})
}
