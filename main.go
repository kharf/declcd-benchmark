package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/app.cue
var appTemplate string

func main() {
	var appCount int
	flag.IntVar(&appCount, "app-count", 1, "")
	flag.Parse()

	wd, err := os.Getwd()
	assertNilErr(err)

	repositoryDir := filepath.Join(wd, "repository")
	appsDir := filepath.Join(repositoryDir, "apps")
	infraDir := filepath.Join(repositoryDir, "infrastructure")

	rmAll(appsDir)
	rmAll(infraDir)

	mkDirAll(appsDir)
	mkDirAll(infraDir)

	for i := range appCount {
		appName := fmt.Sprintf("app%v", i)
		appDir := filepath.Join(appsDir, appName)

		mkDirAll(appDir)
		file, err := os.Create(
			filepath.Join(appDir, fmt.Sprintf("%s.cue", appName)),
		)
		assertNilErr(err)

		tmpl, err := template.New("app").Parse(appTemplate)
		assertNilErr(err)

		err = tmpl.Execute(file, map[string]interface{}{
			"Package":   appName,
			"App":       appName,
			"Namespace": "alpha",
		})
		assertNilErr(err)
	}

}

func rmAll(dir string) {
	err := os.RemoveAll(dir)
	assertNilErr(err)
}

func mkDirAll(dir string) {
	err := os.MkdirAll(dir, 0777)
	assertNilErr(err)
}

func assertNilErr(err error) {
	if err != nil {
		panic(err)
	}
}
