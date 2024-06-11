Kernel: 6.8.11-xanmod1 \
CPU: AMD Ryzen 7 5800X (16) @ 3.800GHz \
Memory: 32GB \
K8s: 1.29.2 \
Declcd: 0.22.9

3x runs.

### Results:

| Setup                          | Duration | Max Memory |
|--------------------------------|----------|------------|
| 125x Deployments 125x Services | 34s      | 47Mi       |
| 125x YAML HelmReleases         | 46s      | 56Mi       |
| 125x OCI HelmReleases          | 45s      | 53Mi       |
