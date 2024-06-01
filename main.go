package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

//go:embed templates/app.cue
var appTemplate string

//go:embed templates/ns.cue
var nsTemplate string

func main() {
	var appCount int
	flag.IntVar(&appCount, "app-count", 1, "")
	flag.Parse()

	wd, err := os.Getwd()
	assertNilErr(err)

	repositoryDir := filepath.Join(wd, "repository")
	appsDir := filepath.Join(repositoryDir, "apps")
	infraDir := filepath.Join(repositoryDir, "infrastructure")

	kindConfig := fmt.Sprintf(`apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
  - role: control-plane
    extraMounts:
      - hostPath: %s
        containerPath: /repository`,
		repositoryDir,
	)

	fmt.Println(kindConfig)

	rmAll(appsDir)
	rmAll(infraDir)
	rmAll(filepath.Join(repositoryDir, ".git"))

	mkDirAll(appsDir)
	mkDirAll(infraDir)

	makeFile(appsDir, "alpha.cue", nsTemplate, map[string]interface{}{
		"Package":   "apps",
		"Namespace": "alpha",
	})

	for i := range appCount {
		appName := fmt.Sprintf("app%v", i)
		appDir := filepath.Join(appsDir, appName)

		mkDirAll(appDir)
		makeFile(appDir, appName, appTemplate, map[string]interface{}{
			"Package":   appName,
			"App":       appName,
			"Namespace": "alpha",
		})
	}

	err = os.WriteFile(filepath.Join(wd, "kind-config.yaml"), []byte(kindConfig), 0666)
	assertNilErr(err)

	runCmd(
		"repository",
		"git",
		"init",
	)

	runCmd(
		"repository",
		"git",
		"add",
		".",
	)

	runCmd(
		"repository",
		"git",
		"commit",
		"-m",
		"\"Init\"",
	)

	runCmd(
		"",
		"kind",
		"create",
		"cluster",
		"--config",
		"kind-config.yaml",
		"--name",
		"declcd-benchmark",
		"--wait",
		"5m",
	)
	defer runCmd("kind", "delete", "cluster", "--name", "declcd-benchmark")

	runCmd(
		"",
		"sh",
		"-c",
		"helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/",
	)

	runCmd(
		"",
		"sh",
		"-c",
		"helm install metrics-server metrics-server/metrics-server --set args=\"{--kubelet-insecure-tls}\"",
	)

	err = os.Setenv("CUE_EXPERIMENT", "modules")
	assertNilErr(err)

	runCmd(
		"repository",
		"declcd",
		"install",
		"-u",
		"/repository",
		"-b",
		"main",
		"--name",
		"benchmark",
		"-i",
		"3600",
	)

}

func makeFile(dir string, name string, templateName string, data map[string]interface{}) {
	file, err := os.Create(
		filepath.Join(dir, fmt.Sprintf("%s.cue", name)),
	)
	assertNilErr(err)

	tmpl, err := template.New("").Parse(templateName)
	assertNilErr(err)

	err = tmpl.Execute(file, data)
	assertNilErr(err)
}

func runCmd(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	err := cmd.Run()
	assertNilErr(err)
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
