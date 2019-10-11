SHELL:=/bin/bash
GOROOT=
GO111MODULE=on

KIND?=kind
KUBECTL?=kubectl
SDK?=operator-sdk

REGISTRY_REPOSITORY?=theperiklis

CLUSTER_NAME=nop-operator-cluster
CLUSTER_VERSION=v1.14.6
KUBECONFIG_PATH=$(shell kind get kubeconfig-path --name ${CLUSTER_NAME})

OPERATOR_REV=$(shell git rev-parse --short HEAD)
OPERATOR_POD_NAME=$(shell KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) get pod -l name=nop-operator -o json | jq -r '.items[0].metadata.name')

build:
	$(SDK) build $(REGISTRY_REPOSITORY)/nop-operator:$(OPERATOR_REV)

test:
	$(SDK) test local ./...

cluster-create:
	$(KIND) create cluster --name $(CLUSTER_NAME) --image kindest/node:$(CLUSTER_VERSION)

cluster-delete: cluster-status
	$(KIND) delete cluster --name $(CLUSTER_NAME)

cluster-deploy: cluster-status
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/service_account.yaml
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/role.yaml
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/role_binding.yaml
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/crds/operators.nefeli.eu_nopoperators_crd.yaml
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/operator.yaml
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) apply -f deploy/crds/operators.nefeli.eu_v1alpha1_nopoperator_cr.yaml

cluster-status:
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) cluster-info

operator-status:
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) get pod -l name=nop-operator

operator-port-forward:
	KUBECONFIG=$(KUBECONFIG_PATH) $(KUBECTL) port-forward pod/$(OPERATOR_POD_NAME) 8686:8686

all: test build
.PHONY: all build test
