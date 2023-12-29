package workspace

import (
	"testing"
	"time"

	// artifactsv1 "github.com/kloudlite/operator/apis/artifacts/v1"
	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operators/project/internal/env"
	"github.com/kloudlite/operator/pkg/logging"
	. "github.com/kloudlite/operator/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var reconciler *Reconciler

var _ = BeforeSuite(func() {
	SetupKubernetes(AddToSchemes(crdsv1.AddToScheme), DefaultEnvTest)

	reconciler = &Reconciler{
		Client: nil,
		Scheme: Suite.Scheme,
		Env: &env.Env{
			ReconcilePeriod:         30 * time.Second,
			MaxConcurrentReconciles: 1,

			// ProjectCfgName:    "project-config",
			// DockerSecretName:  "harbor-docker-secret",
			// AdminRoleName:     "harbor-admin-role",
			SvcAccountName: "kloudlite-svc-account",
			// AccountRouterName: "account-router",
		},
		logger: logging.NewOrDie(&logging.Options{
			Name: "env",
			Dev:  true,
		}),
		Name:       "env",
		yamlClient: *Suite.K8sYamlClient,
	}

})
