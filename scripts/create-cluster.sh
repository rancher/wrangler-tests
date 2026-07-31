#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="wrangler-integration"

k3d cluster list | grep ${CLUSTER_NAME} || \
k3d cluster create ${CLUSTER_NAME} --agents 1 --wait

kubectl config use-context k3d-${CLUSTER_NAME}

kubectl get namespace wrangler-tests >/dev/null 2>&1 || \
kubectl create namespace wrangler-tests