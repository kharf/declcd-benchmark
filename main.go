package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"golang.org/x/sync/errgroup"
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered")
		}

		runCmd(
			"repository",
			"sh",
			"-c",
			"kind delete cluster --name declcd-benchmark",
		)
	}()

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

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait --for=condition=Available deploy/metrics-server --timeout=60s",
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

	eg := errgroup.Group{}
	done := make(chan bool)
	eg.Go(func() error {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-done:
				return nil
			case <-ticker.C:
				_ = runCmdWithErr(
					"",
					"sh",
					"-c",
					"kubectl -n declcd-system top pod gitops-controller-0",
				)
			}
		}
	})

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait -n declcd-system --for=condition=Ready pod/gitops-controller-0 --timeout=60s",
	)

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait -n declcd-system --for=condition=Running gitopsprojects.gitops.declcd.io/benchmark --timeout=60s",
	)

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait -n declcd-system --for=condition=Finished gitopsprojects.gitops.declcd.io/benchmark --timeout=600s",
	)

	done <- true
	err = eg.Wait()
	assertNilErr(err)

	fmt.Println("==================================================")

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl describe gitopsprojects.gitops.declcd.io/benchmark -n declcd-system | grep \"Last Transition Time\"",
	)

	fmt.Println("\n==================================================")
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

func runCmdWithErr(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir

	return cmd.Run()
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
