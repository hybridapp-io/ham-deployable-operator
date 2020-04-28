# Deployable Operator

[![Build](http://prow.purple-chesterfield.com/badge.svg?jobs=images-ham-deployable-operator-amd64-postsubmit)](http://prow.purple-chesterfield.com/?job=images-ham-deployable-operator-amd64-postsubmit)
[![GoDoc](https://godoc.org/github.com/IBM/hybriddeploayble-operator?status.svg)](https://godoc.org/github.com/IBM/ham-deployable-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/IBM/ham-deployable-operator)](https://goreportcard.com/report/github.com/IBM/ham-deployable-operator)
[![Code Coverage](https://codecov.io/gh/IBM/ham-deployable-operator/branch/master/graphs/badge.svg?branch=master)](https://codecov.io/gh/IBM/ham-deployable-operator?branch=master)
[![License](https://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
[![Container](https://quay.io/repository/multicloudlab/hybriddeployable-operator/status)](https://quay.io/repository/multicloudlab/hybriddeployable-operator?tab=tags)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [What is the Deployable Operator](#what-is-the-deployable-operator)
- [Community, discussion, contribution, and support](#community-discussion-contribution-and-support)
- [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Quick Start](#quick-start)
        - [Clone Deployable Operator Repository](#clone-hybriddeployable-operator-repository)
        - [Build Deployable Operator](#build-hybriddeployable-operator)
        - [Install Deployable Operator](#install-hybriddeployable-operator)
        - [Uninstall Deployable Operator](#uninstall-hybriddeployable-operator)
    - [Troubleshooting](#troubleshooting)
- [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## What is the Deployable Operator

The hybridDeployable resource is introduced to handle deployable components running on non-kubernetes platform(s). This operator is intended to work as part of collection of operators for the HybridApplication.  See [References](#hybridApplication-references) for additional information.

## Community, discussion, contribution, and support

Check the [CONTRIBUTING Doc](CONTRIBUTING.md) for how to contribute to the repo.

------

## Getting Started

### Prerequisites

- git v2.18+
- Go v1.13.4+
- operator-sdk v0.15.1
- Kubernetes v1.14+
- kubectl v1.14+

Check the [Development Doc](docs/development.md) for how to contribute to the repo.

### Quick Start

#### Clone Deployable Operator Repository

```shell
$ mkdir -p "$GOPATH"/src/github.com/hybridapp-io
$ cd "$GOPATH"/src/github.com/hybridapp-io
$ git clone https://github.com/hybridapp-io/ham-deployable-operator.git
$ cd "$GOPATH"/src/github.com/hybridapp-io/ham-deployable-operator
```

#### Build Deployable Operator

Build the ham-deployable-operator and push it to a registry.  Modify the example below to reference a container reposistory you have access to.

```shell
$ operator-sdk build quay.io/<user>/ham-deployable-operator:v0.1.0
$ sed -i 's|REPLACE_IMAGE|quay.io/johndoe/ham-deployable-operator:v0.1.0|g' deploy/operator.yaml
$ docker push quay.io/johndoe/ham-deployable-operator:v0.1.0
```

#### Install Deployable Operator

Register the CRD.

```shell
$ kubectl create -f deploy/crds/app.cp4mcm.ibm.com_hybriddeployables_crd.yaml
```

Setup RBAC and deploy.

```shell
$ kubectl create -f deploy/service_account.yaml
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
$ kubectl create -f deploy/operator.yaml
```

Verify ham-deployable-operator is up and running.

```shell
$ kubectl get deployment
NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
ham-deployable-operator   1/1     1            1           2m20s
```

Create the sample CR.

```shell
$ kubectl create -f deploy/crds/app.cp4mcm.ibm.com_hybriddeployables_cr.yaml
NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
ham-deployable-operator   1/1     1            1           2m20s
$ kubectl get hybriddeployables
NAME     AGE
simple   11s
```

#### Uninstall Deployable Operator

Remove all resources created.

```shell
$ kubectl delete -f deploy
$ kubectl delete -f deploy/crds
```

### Troubleshooting

Please refer to [Troubleshooting documentation](docs/trouble_shooting.md) for further info.

## References
