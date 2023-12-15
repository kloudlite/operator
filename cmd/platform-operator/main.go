package main

import (
	"github.com/kloudlite/operator/operator"
	// app "github.com/kloudlite/operator/operators/app-n-lambda/controller"
	resourceWatcher "github.com/kloudlite/operator/operators/resource-watcher/controller"
)

func main() {
	mgr := operator.New("platform-operator")

	// kloudlite resources
	// app.RegisterInto(mgr)
	// project.RegisterInto(mgr)
	// helmCharts.RegisterInto(mgr)
	// routers.RegisterInto(mgr)
	//
	// // kloudlite managed services
	// msvcAndMres.RegisterInto(mgr)
	// msvcMongo.RegisterInto(mgr)
	// msvcRedis.RegisterInto(mgr)
	// msvcRedpanda.RegisterInto(mgr)
	//
	// // kloudlite cluster management
	// clusters.RegisterInto(mgr)
	// nodepool.RegisterInto(mgr)
	//
	// // kloudlite resource status updates
	resourceWatcher.RegisterInto(mgr, false)

	mgr.Start()
}
