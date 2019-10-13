# nop-operator

The nop-operator is a prototype implementation of a k8s operator to enable zero-operations on a cluster. The operator aims to reconcile other third-party operators and/or controllers based upon release channels. A release channel represents the location and its metadata (e.g. operator name and version) a released operator/controller to retrieve its manifests. The current state of implementation is alpha and **not** supposed to be used on a production cluster.

The reconciliation loop handles only the most basic resources that comprise a third party controller/operator namely `core/v1.ServiceAccount`, `rbac/v1.Role`, `rbac/v1.RoleBinding` and `apps/v1.Deployment`. In addition, the channels are processed in a sequential manner in an all-or-nothing approach for the sake of simplicity. For what is worth the implementation is by far not complete to handle more complex lifecycle scenarios beyond simple deployments (See [Limitations](#Limitations))

## Prerequisites

- [go](https://golang.org/) >= 1.13
- [docker](https://www.docker.com/)
- [operator-sdk](https://github.com/operator-framework/operator-sdk/commit/40b81381884a6c5536a8f97505b7ed680690fb81) >= 40b81381 (Due to go 1.13.x issues)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [kind](https://github.com/kubernetes-sigs/kind)
- (Optional) [direnv](https://direnv.net/)

## How to run

The `Makefile` captures the basic commands for a local cluster based on `sigs.k8s.io/kind`.

*Hint:* To access the kind cluster via `kubectl` directly, remember to `export KUBECONFIG=$(kind get kubeconfig-path --name nop-operator-cluster)` or use direnv.

### Build

``` shell
make build
```

### Test

``` shell
make test
```

### Deploy operator in local cluster

``` shell
make cluster-create cluster-deploy
```

### Delete the local cluster

``` shell
make cluster-delete
```

### Reset the local cluster

Resetting the cluster means a complete operator re-build, re-publish, cluster delete, create and re-deploy of the operator. For publishing checkout the configuration hint below.

``` shell
make cluster-reset
```

### Get operator logs

``` shell
make operator-logs
```

### Publishing images from a fork for local testing

To develop, test and publish docker images for this project the `Makefile` variable `REGISTRY_REPOSITORY` needs to be set to a repository the fork maintainer has access rights to push images.

## Limitations

As mentioned in the introduction section the current implementation is not complete and suffers from the following limitations. However, these limitations represent more or less implementation challenges towards more robustness and completeness. The author does not intend to support or provide solutions for these topics in the future:
- Sequential reconciliation of all channels endangers an unnecessary coupling between operator deployments. The channel list could be handled with concurrency primitives reporting the reconciliation status back to the nop-operator.
- Missing handlers for physical/cluster or hierarchical dependencies across reconcilable resources. The are three basic categories on how reconciliation success can be assessed on k8s resources. First, basic RBAC style resources follow a hierarchical approach `Role <--- RoleBinding --> ServiceAccount`. In this scenario after a miss or failure the nop-operator needs to re-apply all of them. Second, resources that depend on cluster/physical resources like PV/PVCs, CNI, etc. can be reconciled by other k8s controllers. Third and finally, "aggregating" resources like `Deployment` or `StatefulSet` can be reconciled independently by k8s controllers, however their success/failure states differ a lot from each other.
- Missing backward compatibility safety measures for operator/controller deployments. Although, each release channel declares a version number the nop-operator does not check by any means that the deployment could break the current state of the cluster. The reconciliation is missing safety procedure handling for breaking changes (e.g. semver checks).
- Off-loading resource version handling to client-go. The current approach to handle each resource Group-Kind-Version independently in the code base does not scale well with kubernetes versioning and deprecation policy. A more robust approach would offload the work to client-go using `Dynamic.Interface` and `unstructured.Unstructured{}` (See [client-go/examples](https://github.com/kubernetes/client-go/tree/master/examples/dynamic-create-update-delete-deployment)).
