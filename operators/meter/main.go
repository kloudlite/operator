package main

import (
	wgv1 "github.com/kloudlite/operator/apis/meter/v1"
	"github.com/kloudlite/operator/operator"

	rmeter "github.com/kloudlite/operator/operators/meter/internal/controllers/resource-meter"
	"github.com/kloudlite/operator/operators/meter/internal/env"
)

func main() {
	ev := env.GetEnvOrDie()

	mgr := operator.New("meter")
	mgr.AddToSchemes(wgv1.AddToScheme)

	mgr.RegisterControllers(
		&rmeter.Reconciler{Name: "ResourceMeter", Env: ev},
	)

	mgr.Start()
}
