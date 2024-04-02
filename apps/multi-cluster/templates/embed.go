package templates

import (
	"bytes"
	"embed"
	"text/template"

	text_templates "github.com/kloudlite/operator/apps/multi-cluster/mpkg/text-template"
)

type TemplateName string

const (
	ServerConfg TemplateName = "wg-server.conf.tpl"
	ClientConfg TemplateName = "wg-client.conf.tpl"
)

//go:embed *
var templatesDir embed.FS

func ParseTemplate(name TemplateName, data any) ([]byte, error) {

	tempBytes, err := templatesDir.ReadFile(string(name))

	tmpl := text_templates.WithFunctions(template.New(string(name)))

	tmpl, err = tmpl.Parse(string(tempBytes))
	if err != nil {
		return nil, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, data)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
