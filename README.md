# wrangler-playground

A playground for exploring and developing an integration testing framework for Rancher Wrangler controllers.

This project uses a k3d cluster to validate Wrangler informers, controllers, and event propagation behavior.

---

## What this project is

This repository is an **integration testing harness** for Wrangler.

It focuses on validating Kubernetes behavior such as:

- Informer event delivery (CREATE / UPDATE / DELETE)
- Controller lifecycle startup and shutdown
- Cache synchronization behavior
- Interaction between Kubernetes API server and Wrangler shared informers

Unlike unit tests or envtest-based setups, this project runs against a **k3d cluster**.

---

## Requirements

- Docker
- Go 1.26.3
- k3d
- kubectl

---

## Getting started

### 1. Create a cluster

```bash
./scripts/create-cluster.sh
```

### 2. Run Tests

```bash
go test ./tests/integration/... -v --kubeconfig ~/.kube/config
```

## What is being tested?

This project currently validates Wrangler behavior using ConfigMap as the test resource.

### Current test coverage

#### Informer event lifecycle
```
CREATE → informer receives object

UPDATE → informer receives modified object

DELETE → informer receives tombstone event
```

#### Controller behavior
```
Wrangler factory initialization

Controller startup and shutdown lifecycle

Cache synchronization before event handling
```