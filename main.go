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
	var appCount, helmAppCount int
	flag.IntVar(&appCount, "app-count", 1, "")
	flag.IntVar(&helmAppCount, "helm-app-count", 0, "")
	flag.Parse()

	err := run(appCount, helmAppCount)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Finished")
}

func run(appCount int, helmAppCount int) error {
	nodeImage := "kindest/node:v1.29.2"
	localRegistryPort := 5000

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	cmd := exec.Command(
		"sh",
		"-c",
		fmt.Sprintf("kubectl port-forward svc/twuni-docker-registry %v", localRegistryPort),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println(err)
		}
		done <- true
	}()

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

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

	err = rmAll(appsDir)
	if err != nil {
		return err
	}
	err = rmAll(infraDir)
	if err != nil {
		return err
	}
	err = rmAll(filepath.Join(repositoryDir, ".git"))
	if err != nil {
		return err
	}

	err = mkDirAll(appsDir)
	if err != nil {
		return err
	}
	err = mkDirAll(infraDir)
	if err != nil {
		return err
	}

	err = makeFile(appsDir, "alpha", nsTemplate, map[string]interface{}{
		"Package":   "apps",
		"Namespace": "alpha",
	})
	if err != nil {
		return err
	}

	for i := range appCount {
		appName := fmt.Sprintf("app%v", i)
		appDir := filepath.Join(appsDir, appName)

		err = mkDirAll(appDir)
		if err != nil {
			return err
		}
		err = makeFile(appDir, appName, appTemplate, map[string]interface{}{
			"Package":   appName,
			"App":       appName,
			"Namespace": "alpha",
		})
		if err != nil {
			return err
		}
	}

	chartsDir := filepath.Join(wd, "charts")
	err = mkDirAll(chartsDir)
	if err != nil {
		return err
	}
	defer rmAll(chartsDir)

	err = os.WriteFile(filepath.Join(wd, "kind-config.yaml"), []byte(kindConfig), 0666)
	if err != nil {
		return err
	}

	err = runCmd(
		"",
		"kind create cluster --config kind-config.yaml --name declcd-benchmark --wait 5m",
	)
	if err != nil {
		return err
	}
	defer runCmd(
		"repository",
		"kind delete cluster --name declcd-benchmark",
	)

	err = runCmd(
		"",
		"helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/",
	)
	if err != nil {
		return err
	}

	err = runCmd(
		"",
		"helm install metrics-server metrics-server/metrics-server --set args=\"{--kubelet-insecure-tls}\"",
	)
	if err != nil {
		return err
	}

	err = runCmd(
		"",
		"helm repo add twuni https://helm.twun.io",
	)
	if err != nil {
		return err
	}

	err = runCmd(
		"",
		"helm install twuni twuni/docker-registry --set persistence.enabled=true",
	)
	if err != nil {
		return err
	}

	err = runCmd(
		"",
		"kubectl wait deploy twuni-docker-registry --for=condition=Available --timeout=90s",
	)
	if err != nil {
		return err
	}

	pEG := errgroup.Group{}
	pEG.Go(func() error {
		return cmd.Run()
	})

	for i := range helmAppCount {
		appName := fmt.Sprintf("helmapp%v", i)
		helmAppDir := filepath.Join(infraDir, appName)
		chartName := fmt.Sprintf("fakeapp%v", i)

		err = mkDirAll(helmAppDir)
		if err != nil {
			return err
		}
		err = makeFile(helmAppDir, appName, helmAppTemplate, map[string]interface{}{
			"Package":   appName,
			"HelmApp":   appName,
			"Namespace": "alpha",
			"ChartName": chartName,
		})
		if err != nil {
			return err
		}

		err = runCmd(
			"charts",
			fmt.Sprintf(
				"helm create fakeapp%v",
				i,
			),
		)
		if err != nil {
			return err
		}

		err = runCmd(
			"charts",
			fmt.Sprintf(
				"helm package ./fakeapp%v",
				i,
			),
		)
		if err != nil {
			return err
		}

		err = runCmd(
			"charts",
			fmt.Sprintf(
				"helm push fakeapp%v-0.1.0.tgz oci://localhost:%v/charts",
				i,
				localRegistryPort,
			),
		)
		if err != nil {
			return err
		}
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
		if err != nil {
			return err
		}
	}

	err = runCmd(
		"",
		"kubectl wait --for=condition=Available deploy/metrics-server --timeout=60s",
	)
	if err != nil {
		return err
	}

	err = runDeclcd(done)
	if err != nil {
		return err
	}

	return nil
}

func runDeclcd(done chan bool) error {
	if err := runCmd(
		"repository",
		"git init",
	); err != nil {
		return err
	}

	if err := runCmd(
		"repository",
		"git add .",
	); err != nil {
		return err
	}

	if err := runCmd(
		"repository",
		"git commit -m \"Init\"",
	); err != nil {
		return err
	}

	if err := os.Setenv("CUE_EXPERIMENT", "modules"); err != nil {
		return err
	}

	if err := runCmd(
		"repository",
		"declcd install -u /repository -b main --name benchmark -i 3600",
	); err != nil {
		return err
	}

	eg := &errgroup.Group{}
	eg.Go(func() error {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-done:
				return nil
			case <-ticker.C:
				_ = runCmd(
					"",
					"kubectl -n declcd-system top pod gitops-controller-0",
				)
			}
		}
	})

	if err := runCmd(
		"",
		"kubectl wait -n declcd-system --for=condition=Ready pod/gitops-controller-0 --timeout=60s",
	); err != nil {
		return err
	}

	if err := runCmd(
		"",
		"kubectl wait -n declcd-system --for=condition=Running gitopsprojects.gitops.declcd.io/benchmark --timeout=60s",
	); err != nil {
		return err
	}

	if err := runCmd(
		"",
		"kubectl wait -n declcd-system --for=condition=Finished gitopsprojects.gitops.declcd.io/benchmark --timeout=600s",
	); err != nil {
		return err
	}

	fmt.Println("==================================================")

	if err := runCmd(
		"",
		"kubectl describe gitopsprojects.gitops.declcd.io/benchmark -n declcd-system | grep \"Last Transition Time\"",
	); err != nil {
		return err
	}

	fmt.Println("\n==================================================")

	done <- true
	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
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

	if err := runCmd(
		"",
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

func makeFile(dir string, name string, templateName string, data map[string]interface{}) error {
	file, err := os.Create(
		filepath.Join(dir, fmt.Sprintf("%s.cue", name)),
	)
	if err != nil {
		return err
	}

	tmpl, err := template.New("").Parse(templateName)
	if err != nil {
		return err
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return err
	}

	return nil
}

func runCmd(dir string, cmdString string) error {
	cmd := exec.Command("sh", "-c", cmdString)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

func rmAll(dir string) error {
	return os.RemoveAll(dir)
}

func mkDirAll(dir string) error {
	return os.MkdirAll(dir, 0777)
}
