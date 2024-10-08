package types

import (
	"strings"

	"github.com/kloudlite/operator/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceUpdate struct {
	Object *unstructured.Unstructured `json:"object"`
}

type ResourceStatus string

func (rs ResourceStatus) String() string {
	return string(rs)
}

const (
	ResourceStatusUpdated  ResourceStatus = "updated"
	ResourceStatusDeleting ResourceStatus = "deleting"
	ResourceStatusDeleted  ResourceStatus = "deleted"
)

const (
	ResourceStatusKey string = "resource-watcher-resource-status"
)

func HasOtherKloudliteFinalizers(obj client.Object) bool {
	hasOtherKloudliteFinalizers := false

	for _, f := range obj.GetFinalizers() {
		if strings.Contains(f, "kloudlite.io") {
			if f == constants.StatusWatcherFinalizer {
				continue
			}
			hasOtherKloudliteFinalizers = true
		}
	}

	return hasOtherKloudliteFinalizers
}

var SecretWatchingAnnotation = map[string]string{
	"kloudlite.io/watch-secret": "true",
}

var ConfigWatchingAnnotation = map[string]string{
	"kloudlite.io/watch-configmap": "true",
}

const (
	KeyClusterManagedSvcSecret = "resource-watcher-cmsvc-secret"
	KeyProjectManagedSvcSecret = "resource-watcher-pmsvc-secret"
	KeyManagedResSecret        = "resource-watcher-mres-secret"

	KeyVPNDeviceConfig = "resource-watcher-wireguard-config"

	KeyGlobalVPNWgParams = "resource-watcher-gvpn-wg-params"
	KeyGatewayWgParams   = "resource-watcher-gateway-wg-params"
)
