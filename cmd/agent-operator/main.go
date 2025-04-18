package main

import (
	"github.com/kloudlite/operator/toolkit/operator"

	lifecycle "github.com/kloudlite/operator/operators/lifecycle/controller"

	resourceWatcher "github.com/kloudlite/operator/operators/resource-watcher/controller"

	workmachine "github.com/kloudlite/operator/operators/workmachine/register"
	workspace "github.com/kloudlite/operator/operators/workspace/register"
)

func main() {
	mgr := operator.New("agent-operator")

	// kloudlite resources
	// app.RegisterInto(mgr)
	// project.RegisterInto(mgr)
	// helmCharts.RegisterInto(mgr)
	// routers.RegisterInto(mgr)

	// kloudlite managed services
	// msvcAndMres.RegisterInto(mgr)

	// msvcMongo.RegisterInto(mgr)
	// msvcRedis.RegisterInto(mgr)
	// msvcMysql.RegisterInto(mgr)
	// msvcPostgres.RegisterInto(mgr)

	lifecycle.RegisterInto(mgr)

	// kloudlite resource status updates
	resourceWatcher.RegisterInto(mgr)

	// distribution.RegisterInto(mgr)

	// networkingv1.RegisterInto(mgr)
	// serviceIntercept.RegisterInto(mgr)
	workmachine.RegisterInto(mgr)
	workspace.RegisterInto(mgr)

	// pluginMongoDB.RegisterInto(mgr)
	// pluginHelmChart.RegisterInto(mgr)

	mgr.Start()
}
