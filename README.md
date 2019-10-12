# nop-operator

The nop-operator is a prototype implementation of an k8s operator to enable zero operations on a k8s cluster. The operator aims to reconcile other third-party operators or controllers based upon release channels. A release channel represents the location and its metadata (e.g. operator name and version) to retrieve the third-party manifests. The current state of this implementation is alpha and **not** supposed to be used on a production cluster.

The reconciliation loop handles only the most basic resources that comprise a third party controller/operator namely `core/v1.ServiceAccount`, `rbac/v1.Role`, `rbac/v1.RoleBinding` and `apps/v1.Deployment`. In addition, the channels are processed in a sequential manner in an all-or-nothing approach for the sake of simplicity. For what is worth the implementation is by far not complete to handle more complex lifecycle scenarios beyond simple deployments (See [Limitations](#Limitations))

## Prerequisites

- [go](https://golang.org/) >= 1.13
- [docker](https://www.docker.com/)
- [operator-sdk](https://github.com/operator-framework/operator-sdk/commit/40b81381884a6c5536a8f97505b7ed680690fb81) >= 40b81381 (Due to go 1.13.x issues)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [kind](https://github.com/kubernetes-sigs/kind)

## How to run

The project's `Makefile` captures the basic commands for a local cluster lifecycle based on `sigs.k8s.io/kind`.

### Build

``` shell
make build
```

### Test

``` shell
make test
```

### Deploy operator in local kind cluster

``` shell
make cluster-create cluster-deploy
```

### Reset the local kind cluster

``` shell
make cluster-reset
```

### Get operator logs

``` shell
make operator-logs
```

## Limitations

TBD
