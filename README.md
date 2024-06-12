Kernel: 6.8.11-xanmod1 \
CPU: AMD Ryzen 7 5800X (16) @ 3.800GHz \
Memory: 32GB \
K8s: 1.29.2 \
Declcd: 0.22.9
Controller Resources: 
limits: {
	memory: "1.5Gi"
	cpu:    "2000m"
}
requests: {
	memory: "1.5Gi"
	cpu:    "500m"
}

1 App contains 1 Deployment and 1 Service
1 HelmRelease App contains 1 Deployment, 1 Service and 1 ServiceAccount
3x runs.

### Results:

| Setup                          | Duration | Max Memory |
|--------------------------------|----------|------------|
| 125x Deployments 125x Services | 34s      | 47Mi       |
| 125x YAML HelmReleases         | 46s      | 56Mi       |
| 125x OCI HelmReleases          | 45s      | 53Mi       |
| 250x Deployments 250x Services | 1m9s     | 52Mi       |
| 250x YAML HelmReleases         | 1m32s    | 69Mi       |
| 250x OCI HelmReleases          | 1m32s    | 76Mi       |
| 500x Deployments 500x Services | 2m22s    | 68Mi       |
