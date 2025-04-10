package templates

import (
	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkspaceTemplateArgs struct {
	Metadata           metav1.ObjectMeta
	WorkMachineName    string
	IsOn               bool
	ServiceAccountName string

	ImageInitContainer string
	ImageSSH           string

	EnableTTYD bool
	ImageTTYD  string

	EnableJupyterNotebook bool
	ImageJupyterNotebook  string

	EnableCodeServer bool
	ImageCodeServer  string

	EnableVSCodeServer bool
	ImageVscodeServer  string

	ImagePullPolicy string

	KloudliteDeviceFQDN string

	RouterSpec crdsv1.RouterSpec
}
