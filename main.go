package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/sync/errgroup"
)

//go:embed templates/app.cue
var appTemplate string

//go:embed templates/helm-app.cue
var helmAppTemplate string

//go:embed templates/ns.cue
var nsTemplate string

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	var appCount, helmAppCount int
	flag.IntVar(&appCount, "app-count", 1, "")
	flag.IntVar(&helmAppCount, "helm-app-count", 0, "")
	flag.Parse()

	nodeImage := "kindest/node:v1.29.2"
	localRegistryPort := 5000

	cmd := exec.Command(
		"sh",
		"-c",
		fmt.Sprintf("kubectl port-forward svc/twuni-docker-registry %v", localRegistryPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered")
		}
		done <- true

		fmt.Println(cmd.Process.Kill())

		_ = runCmdWithErr(
			"repository",
			"sh",
			"-c",
			"kind delete cluster --name declcd-benchmark",
		)
	}()

	wd, err := os.Getwd()
	assertNilErr(err)

	repositoryDir := filepath.Join(wd, "repository")
	appsDir := filepath.Join(repositoryDir, "apps")
	infraDir := filepath.Join(repositoryDir, "infrastructure")

	kindConfig := fmt.Sprintf(`apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
  - role: control-plane
    image: %s
    extraMounts:
      - hostPath: %s
        containerPath: /repository
  - role: worker
    image: %s
    extraMounts:
      - hostPath: %s
        containerPath: /repository`,
		nodeImage,
		repositoryDir,
		nodeImage,
		repositoryDir,
	)

	fmt.Println(kindConfig)

	rmAll(appsDir)
	rmAll(infraDir)
	rmAll(filepath.Join(repositoryDir, ".git"))

	mkDirAll(appsDir)
	mkDirAll(infraDir)

	makeFile(appsDir, "alpha", nsTemplate, map[string]interface{}{
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

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if appCount != 0 {
		err = copyImage(
			timeoutCtx,
			"gcr.io/kubernetes-e2e-test-images/echoserver",
			"2.2",
			fmt.Sprintf("localhost:%v/kubernetes-e2e-test-images/echoserver", localRegistryPort),
		)
		assertNilErr(err)
	}

	chartsDir := filepath.Join(wd, "charts")
	mkDirAll(chartsDir)
	// defer rmAll(chartsDir)

	err = os.WriteFile(filepath.Join(wd, "kind-config.yaml"), []byte(kindConfig), 0666)
	assertNilErr(err)

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
		"helm repo add twuni https://helm.twun.io",
	)

	runCmd(
		"",
		"sh",
		"-c",
		"helm install twuni twuni/docker-registry --set persistence.enabled=true",
	)

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait deploy twuni-docker-registry --for=condition=Available --timeout=90s",
	)

	pEG := errgroup.Group{}
	pEG.Go(func() error {
		return cmd.Run()
	})

	for i := range helmAppCount {
		appName := fmt.Sprintf("helmapp%v", i)
		helmAppDir := filepath.Join(infraDir, appName)
		chartName := fmt.Sprintf("fakeapp%v", i)

		mkDirAll(helmAppDir)
		makeFile(helmAppDir, appName, helmAppTemplate, map[string]interface{}{
			"Package":   appName,
			"HelmApp":   appName,
			"Namespace": "alpha",
			"ChartName": chartName,
		})

		runCmd(
			"charts",
			"sh",
			"-c",
			fmt.Sprintf(
				"helm create fakeapp%v",
				i,
			),
		)

		runCmd(
			"charts",
			"sh",
			"-c",
			fmt.Sprintf(
				"helm package ./fakeapp%v",
				i,
			),
		)

		runCmd(
			"charts",
			"sh",
			"-c",
			fmt.Sprintf(
				"helm push fakeapp%v-0.1.0.tgz oci://localhost:%v/charts",
				i,
				localRegistryPort,
			),
		)
	}

	runCmd(
		"",
		"sh",
		"-c",
		"kubectl wait --for=condition=Available deploy/metrics-server --timeout=60s",
	)

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

func copyImage(
	ctx context.Context,
	image string,
	version string,
	targetImage string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := runCmdWithErr(
		"",
		"sh",
		"-c",
		fmt.Sprintf(
			"crane copy %s:%s %s:%s",
			image,
			version,
			targetImage,
			version,
		),
	); err != nil {
		time.Sleep(2 * time.Second)
		return copyImage(
			ctx,
			image,
			version,
			targetImage,
		)
	}

	return nil
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
