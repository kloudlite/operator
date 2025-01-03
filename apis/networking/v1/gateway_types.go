package v1

import (
	ct "github.com/kloudlite/operator/apis/common-types"
	rApi "github.com/kloudlite/operator/toolkit/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PeerVisibility string

const (
	PeerVisibilityPublic  PeerVisibility = "public"
	PeerVisibilityPrivate PeerVisibility = "private"
)

type Peer struct {
	DNSHostname string `json:"dnsHostname,omitempty"`
	Comments    string `json:"comments,omitempty"`

	PublicKey      string  `json:"publicKey"`
	PublicEndpoint *string `json:"publicEndpoint,omitempty"`
	IP             *string `json:"ip,omitempty"`

	DNSSuffix  *string  `json:"dnsSuffix,omitempty"`
	AllowedIPs []string `json:"allowedIPs,omitempty"`

	Visibility PeerVisibility `json:"visibility,omitempty"`
}

type GatewayLoadBalancer struct {
	Hosts []string `json:"hosts"`
	Port  int32    `json:"port"`
}

type GatewayServiceType string

const (
	GatewayServiceTypeLoadBalancer GatewayServiceType = "LoadBalancer"
	GatewayServiceTypeNodePort     GatewayServiceType = "NodePort"
)

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	TargetNamespace string `json:"targetNamespace,omitempty"`

	GlobalIP string `json:"globalIP"`

	ClusterCIDR string `json:"clusterCIDR"`
	SvcCIDR     string `json:"svcCIDR"`

	DNSSuffix string `json:"dnsSuffix"`

	Peers []Peer `json:"peers,omitempty"`

	// secret's data must be serializable into LocalOverrides
	LocalOverrides *ct.SecretRef `json:"localOverrides,omitempty"`

	//+kubebuilder:default=LoadBalancer
	ServiceType GatewayServiceType `json:"serviceType,omitempty"`

	// it will be filled by resource controller
	LoadBalancer *GatewayLoadBalancer `json:"loadBalancer,omitempty"`
	NodePort     *int32               `json:"nodePort,omitempty"`

	// secret's data will be unmarshalled into WireguardKeys
	WireguardKeysRef ct.LocalObjectReference `json:"wireguardKeysRef,omitempty"`
}

type WireguardKeys struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type LocalOverrides struct {
	Peers []Peer `json:"peers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:JSONPath=".status.lastReconcileTime",name=Seen,type=date
// +kubebuilder:printcolumn:JSONPath=".spec.globalIP",name=GlobalIP,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.clusterCIDR",name=ClusterCIDR,type=string
// +kubebuilder:printcolumn:JSONPath=".spec.svcCIDR",name=ServiceCIDR,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.checks",name=Checks,type=string
// +kubebuilder:printcolumn:JSONPath=".metadata.annotations.kloudlite\\.io\\/operator\\.resource\\.ready",name=Ready,type=string

// Gateway is the Schema for the gateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewaySpec `json:"spec,omitempty"`
	Status rApi.Status `json:"status,omitempty"`
}

func (p *Gateway) EnsureGVK() {
	if p != nil {
		p.SetGroupVersionKind(GroupVersion.WithKind("Gateway"))
	}
}

func (p *Gateway) GetStatus() *rApi.Status {
	return &p.Status
}

func (p *Gateway) GetEnsuredLabels() map[string]string {
	return map[string]string{}
}

func (p *Gateway) GetEnsuredAnnotations() map[string]string {
	return map[string]string{}
}

//+kubebuilder:object:root=true

// GatewayList contains a list of Gateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
