package v1

import (
	rApi "github.com/kloudlite/operator/toolkit/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AWSMachineConfig struct {
	Region           string `json:"region" graphql:"noinput"`
	AvailabilityZone string `json:"availabilityZone"`

	AMI          string `json:"ami"`
	InstanceType string `json:"instanceType"`

	PublicSubnetID  string `json:"publicSubnetId" graphql:"noinput"`
	SecurityGroupID string `json:"SecurityGroupID" graphql:"noinput"`

	RootVolumeType string `json:"rootVolumeType" graphql:"noinput"`
	RootVolumeSize int    `json:"rootVolumeSize" graphql:"noinput"`

	ExternalVolumeType string `json:"externalVolumeType" graphql:"noinput"`
	ExternalVolumeSize string `json:"externalVolumeSize"`

	IAMInstanceProfileRole *string `json:"iamInstanceProfileRole,omitempty" graphql:"noinput"`
}

// +kubebuilder:validation:Enum=ON;OFF;
// +kubebuilder:default=ON
type WorkMachineState string

const (
	WorkMachineStateOn  WorkMachineState = "ON"
	WorkMachineStateOff WorkMachineState = "OFF"
)

// WorkMachineSpec defines the desired state of WorkMachine
type WorkMachineSpec struct {
	State WorkMachineState `json:"state"`

	SSHPublicKeys []string `json:"sshPublicKeys"`

	AWSMachineConfig `json:"aws"`
}

type WorkMachineStatus struct {
	rApi.Status        `json:"status,omitempty"`
	MachinePulicSSHKey string `json:"machineSSHKey,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WorkMachine is the Schema for the workmachines API
type WorkMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkMachineSpec   `json:"spec,omitempty"`
	Status WorkMachineStatus `json:"status,omitempty"`
}

func (r *WorkMachine) EnsureGVK() {
	if r != nil {
		r.SetGroupVersionKind(GroupVersion.WithKind("Workspace"))
	}
}

func (w *WorkMachine) GetStatus() *rApi.Status {
	return &w.Status.Status
}

func (w *WorkMachine) GetEnsuredLabels() map[string]string {
	return map[string]string{
		"kloudlite.io/workmachine.name": w.Name,
	}
}

func (w *WorkMachine) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

// +kubebuilder:object:root=true

// WorkMachineList contains a list of WorkMachine
type WorkMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkMachine{}, &WorkMachineList{})
}
