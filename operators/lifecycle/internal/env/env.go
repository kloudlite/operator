package env

import (
	"github.com/codingconcepts/env"
)

type Env struct {
	MaxConcurrentReconciles int    `env:"MAX_CONCURRENT_RECONCILES" default:"5"`
	HelmJobRunnerImage      string `env:"HELM_JOB_RUNNER_IMAGE" required:"true"`
}

func GetEnvOrDie() *Env {
	var ev Env
	if err := env.Set(&ev); err != nil {
		panic(err)
	}
	return &ev
}
