package controller

import (
	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	redisMsvcv1 "github.com/kloudlite/operator/apis/redis.msvc/v1"
	"github.com/kloudlite/operator/operator"
	"github.com/kloudlite/operator/operators/msvc-redis/internal/controllers/prefix"
	"github.com/kloudlite/operator/operators/msvc-redis/internal/controllers/standalone"
	"github.com/kloudlite/operator/operators/msvc-redis/internal/env"
)

func RegisterInto(mgr operator.Operator) {
	ev := env.GetEnvOrDie()
	mgr.AddToSchemes(redisMsvcv1.AddToScheme, crdsv1.AddToScheme)
	mgr.RegisterControllers(
		&standalone.ServiceReconciler{Name: "msvc-redis:standalone-svc", Env: ev},
		&prefix.Reconciler{Name: "msvc-redis:prefix", Env: ev},
		// &aclaccount.Reconciler{Name: "msvc-redis:acl-account", Env: ev},
		// &acl_configmap.Reconciler{Name: "msvc-redis:acl-configmap", Env: ev},
	)
}
