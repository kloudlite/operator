package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"operators.kloudlite.io/lib/errors"
	"operators.kloudlite.io/lib/templates"
)

var (
	//go:embed templates
	templatesFS embed.FS
)

func fileExists(absPath string) bool {
	_, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return true
}

func main() {
	t := template.New("klop")
	t = templates.WithFunctions(t)

	app := cli.NewApp()
	app.Name = "klop"
	app.Commands = []*cli.Command{
		{
			Name: "controller",
			Subcommands: []*cli.Command{
				{
					Name: "create",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "debug"},
						&cli.StringFlag{Name: "package, pkg", Required: true},
						&cli.StringFlag{Name: "kind", Required: true},
						&cli.StringFlag{Name: "kind-pkg", Required: true},
						&cli.StringFlag{Name: "kind-plural", Required: true},
						&cli.StringFlag{Name: "api-group", Required: true},
						&cli.StringFlag{Name: "out"},
					},
					Action: func(cctx *cli.Context) error {
						tName := "templates/controller.go.tpl"
						if _, err := t.ParseFS(templatesFS, tName); err != nil {
							return err
						}

						out := new(bytes.Buffer)

						if err := t.ExecuteTemplate(
							out, strings.Split(tName, "/")[1], map[string]any{
								"package":     cctx.String("package"),
								"kind":        cctx.String("kind"),
								"kind-pkg":    cctx.String("kind-pkg"),
								"kind-plural": cctx.String("kind-plural"),
								"api-group":   cctx.String("api-group"),
							},
						); err != nil {
							return err
						}

						isDebug := cctx.Bool("debug")
						outputFile := cctx.String("out")

						if !isDebug {
							if outputFile == "" {
								return errors.Newf("flag out should be present, in case debug mode is off")
							}
							dir, err := os.Getwd()
							if err != nil {
								return err
							}

							outputFile = filepath.Join(dir, outputFile)
						}

						if !isDebug {
							if fileExists(outputFile) {
								return errors.Newf("filepath: %s already exists", outputFile)
							}
							if err := os.WriteFile(outputFile, out.Bytes(), 0644); err != nil {
								return err
							}
						} else {
							fmt.Println(out.String())
						}
						return nil
					},
				},
			},
		},
		{
			Name: "msvc-controller",
			Subcommands: []*cli.Command{
				{
					Name: "create",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "debug"},
						&cli.StringFlag{Name: "package, pkg", Required: true},
						&cli.StringFlag{Name: "kind", Required: true},
						&cli.StringFlag{Name: "kind-pkg", Required: true},
						&cli.StringFlag{Name: "kind-plural", Required: true},
						&cli.StringFlag{Name: "api-group", Required: true},
						&cli.StringFlag{Name: "out"},
					},
					Action: func(cctx *cli.Context) error {
						tName := "templates/msvc-controller.go.tpl"
						if _, err := t.ParseFS(templatesFS, tName); err != nil {
							return err
						}

						out := new(bytes.Buffer)

						if err := t.ExecuteTemplate(
							out, strings.Split(tName, "/")[1], map[string]any{
								"package":     cctx.String("package"),
								"kind":        cctx.String("kind"),
								"kind-pkg":    cctx.String("kind-pkg"),
								"kind-plural": cctx.String("kind-plural"),
								"api-group":   cctx.String("api-group"),
							},
						); err != nil {
							return err
						}

						isDebug := cctx.Bool("debug")
						outputFile := cctx.String("out")

						if !isDebug {
							if outputFile == "" {
								return errors.Newf("flag out should be present, in case debug mode is off")
							}
							dir, err := os.Getwd()
							if err != nil {
								return err
							}

							outputFile = filepath.Join(dir, outputFile)
						}

						if !isDebug {
							if fileExists(outputFile) {
								return errors.Newf("filepath: %s already exists", outputFile)
							}
							if err := os.WriteFile(outputFile, out.Bytes(), 0644); err != nil {
								return err
							}
						} else {
							fmt.Println(out.String())
						}
						return nil
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}