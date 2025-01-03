package templates

import (
	"embed"
	"path/filepath"

	"github.com/kloudlite/operator/pkg/templates"
)

//go:embed *
var templatesDir embed.FS

type templateFile string

const (
	HelmMongoDBCluster        templateFile = "./helm-mongodb-cluster.yml.tpl"
	HelmMongoDBStandalone     templateFile = "./helm-mongodb-standalone.yml.tpl"
	HelmMongoDBStandaloneAuth templateFile = "./helm-mongodb-standalone-auth.yml.tpl"
	JobCreateDBUser           templateFile = "./job-create-db-user.yml.tpl"

	DBLifecycleTemplate templateFile = "./db-lifecycle.yml.tpl"

	BackupCronJobTemplate       templateFile = "./backup-cronjob.yml.tpl"
	BackupLifecycleSpecTemplate templateFile = "./backup-lifecycle-spec.yml.tpl"
)

func Read(t templateFile) ([]byte, error) {
	return templatesDir.ReadFile(filepath.Join(string(t)))
}

var ParseBytes = templates.ParseBytes
