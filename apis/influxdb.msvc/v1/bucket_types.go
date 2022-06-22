package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rApi "operators.kloudlite.io/lib/operator"
	rawJson "operators.kloudlite.io/lib/raw-json"
)

type BucketSpec struct {
	ManagedSvcName string              `json:"managedSvcName"`
	Inputs         rawJson.KubeRawJson `json:"inputs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Bucket is the Schema for the buckets API
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec  `json:"spec,omitempty"`
	Status rApi.Status `json:"status,omitempty"`
}

func (b *Bucket) GetStatus() *rApi.Status {
	return &b.Status
}

func (b *Bucket) GetEnsuredLabels() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}