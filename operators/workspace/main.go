package main

import (
	crdsv1 "github.com/kloudlite/operator/apis/crds/v1"
	"github.com/kloudlite/operator/operator"

	"github.com/kloudlite/operator/operators/workspace/internal/env"
)

func main() {
	ev := env.GetEnvOrDie()
	mgr := operator.New("workspace")
	mgr.AddToSchemes(crdsv1.AddToScheme)

	mgr.RegisterControllers()

	mgr.Start()
}
