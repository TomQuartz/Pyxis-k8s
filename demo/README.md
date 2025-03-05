Pyxis demo
==================================

### Deploying Pyxis demo

**requirements**
- A running Kubernetes cluster
  - tested on Ubuntu 22.04 Kubernetes v1.30.0

```shell
./scripts/deploy.sh
```

### Running demo app

```shell
./scripts/run.sh
```

### Pyxis Storage Server Memory Usage

```shell
# outputs the current usage
./scripts/metric.sh
```

### Cleaning up Pyxis

```shell
./scripts/cleanup.sh
```