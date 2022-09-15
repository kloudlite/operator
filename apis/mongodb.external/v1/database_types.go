package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rApi "operators.kloudlite.io/lib/operator"
)

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	DbName        string `json:"dbName,omitempty"`
	ConnectionUri string `json:"connectionUri,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec `json:"spec,omitempty"`
	Status rApi.Status  `json:"status,omitempty"`
}

func (db *Database) GetStatus() *rApi.Status {
	return &db.Status
}

func (db *Database) GetEnsuredLabels() map[string]string {
	return map[string]string{}
}

func (db *Database) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}