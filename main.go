package main

import (
	"context"
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

	nodeImage := "kindest/node:v1.29.2"
	localRegistryPort := 5001
	registryPort := 5000
	registryName := "declcd-registry"
	kindConfig := fmt.Sprintf(`apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:%v"]
    endpoint = ["http://%s:%v"]
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
		localRegistryPort,
		registryName,
		registryPort,
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

	runCmd(
		"",
		"sh",
		"-c",
		fmt.Sprintf(
			"docker run -d --restart=always -p \"127.0.0.1:%v:%v\" --network bridge --name \"%s\" registry:2",
			localRegistryPort,
			registryPort,
			registryName,
		),
	)

	runCmd(
		"",
		"sh",
		"-c",
		fmt.Sprintf(
			"docker network connect \"kind\" \"%s\"",
			registryName,
		),
	)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = copyImage(timeoutCtx, localRegistryPort)
	assertNilErr(err)

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

		_ = runCmdWithErr(
			"repository",
			"sh",
			"-c",
			"kind delete cluster --name declcd-benchmark",
		)

		_ = runCmdWithErr(
			"",
			"sh",
			"-c",
			fmt.Sprintf(
				"docker rm %s -f",
				registryName,
			),
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
		fmt.Sprintf(
			`cat <<EOF | kubectl apply --server-side -f-
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:%v"
    hostFromContainerRuntime: "%s:%d"
    hostFromClusterNetwork: "%s:%d"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
`,
			localRegistryPort,
			registryName,
			registryPort,
			registryName,
			registryPort,
		),
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

func copyImage(ctx context.Context, port int) error {
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
			"crane copy gcr.io/kubernetes-e2e-test-images/echoserver:2.2 localhost:%v/kubernetes-e2e-test-images/echoserver:2.2",
			port,
		),
	); err != nil {
		time.Sleep(2 * time.Second)
		return copyImage(
			ctx,
			port,
		)
	}

	runCmdWithErr(
		"",
		"sh",
		"-c",
		fmt.Sprintf(
			"docker tag gcr.io/kubernetes-e2e-test-images/echoserver:2.2 localhost:%v/echoserver:2.2",
			port,
		),
	)

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
