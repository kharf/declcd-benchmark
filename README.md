Kernel: 6.8.11-xanmod1 \
CPU: AMD Ryzen 7 5800X (16) @ 3.800GHz \
Memory: 32GB

1 App contains 1 Deployment and 1 Service \
1 HelmRelease App contains 1 Deployment, 1 Service and 1 ServiceAccount \
3x runs.

------------------------------------------------------------------

K8s: 1.29 \
Declcd: 0.24.9


### Results:

| Setup                            | Duration | Max Memory |
|----------------------------------|----------|------------|
| 125x Deployments 125x Services   | 16s      | 27Mi       |
| 125x YAML HelmReleases           | 29s      | 48Mi       |
| 125x OCI HelmReleases            | 28s      | 50Mi       |
| 250x Deployments 250x Services   | 32s      | 34Mi       |
| 250x YAML HelmReleases           | 56s      | 52Mi       |
| 250x OCI HelmReleases            | 54s      | 56Mi       |
| 500x Deployments 500x Services   | 1m05s    | 39Mi       |
| 500x YAML HelmReleases           | 1m45s    | 61Mi       |
| 500x OCI HelmReleases            | 1m53s    | 84Mi       |
| 1000x Deployments 1000x Services | 2m11s    | 50Mi       |
| 1000x YAML HelmReleases          | 3m48s    | 69Mi       |
| 1000x OCI HelmReleases           | 3m46s    | 137Mi      |

------------------------------------------------------------------

K8s: 1.29 \
Declcd: 0.23.1


### Results:

| Setup                            | Duration | Max Memory |
|----------------------------------|----------|------------|
| 125x Deployments 125x Services   | 18s      | 40Mi       |
| 125x YAML HelmReleases           | 28s      | 49Mi       |
| 125x OCI HelmReleases            | 28s      | 50Mi       |
| 250x Deployments 250x Services   | 30s      | 32Mi       |
| 250x YAML HelmReleases           | 54s      | 53Mi       |
| 250x OCI HelmReleases            | 55s      | 55Mi       |
| 500x Deployments 500x Services   | 1m01s    | 51Mi       |
| 500x YAML HelmReleases           | 1m49s    | 63Mi       |
| 500x OCI HelmReleases            | 1m49s    | 90Mi       |
| 1000x Deployments 1000x Services | 2m03s    | 58Mi       |
| 1000x YAML HelmReleases          | 3m43s    | 71Mi       |
| 1000x OCI HelmReleases           | 3m39s    | 135Mi      |

------------------------------------------------------------------

K8s: 1.29 \
Declcd: 0.22.9


### Results:

| Setup                            | Duration | Max Memory |
|----------------------------------|----------|------------|
| 125x Deployments 125x Services   | 34s      | 47Mi       |
| 125x YAML HelmReleases           | 46s      | 56Mi       |
| 125x OCI HelmReleases            | 45s      | 53Mi       |
| 250x Deployments 250x Services   | 1m9s     | 52Mi       |
| 250x YAML HelmReleases           | 1m32s    | 69Mi       |
| 250x OCI HelmReleases            | 1m32s    | 76Mi       |
| 500x Deployments 500x Services   | 2m26s    | 69Mi       |
| 500x YAML HelmReleases           | 3m09s    | 84Mi       |
| 500x OCI HelmReleases            | 3m11s    | 108Mi      |
| 1000x Deployments 1000x Services | 5m05s    | 85Mi       |
| 1000x YAML HelmReleases          | 6m39s    | 107Mi      |
| 1000x OCI HelmReleases           | 6m33s    | 169Mi      |
