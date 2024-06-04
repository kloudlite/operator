package templates

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatewayDeploymentArgs struct {
	metav1.ObjectMeta

	ServiceAccountName string

	GatewayWgConfigURI     string
	GatewayWgConfigHashURI string
	GatewayAdminAPIImage   string
	WebhookServerImage     string

	GatewayWgSecretName string
	GatewayGlobalIP     string

	ClusterCIDR string
	ServiceCIDR string

	IPManagerConfigName      string
	IPManagerConfigNamespace string
}

type WebhookTemplateArgs struct {
	NamePrefix      string
	Namespace       string
	OwnerReferences []metav1.OwnerReference

	ServiceName string

	WebhookServerImage string
}

type GatewayRBACTemplateArgs struct {
	metav1.ObjectMeta
}