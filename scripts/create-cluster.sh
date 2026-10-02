#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="wrangler-integration"

if ! k3d cluster list | grep -q "${CLUSTER_NAME}"; then
  k3d cluster create "${CLUSTER_NAME}" --agents 1 --wait
fi

kubectl config use-context "k3d-${CLUSTER_NAME}"

mkdir -p "${HOME}/.kube"
k3d kubeconfig get "${CLUSTER_NAME}" > "${HOME}/.kube/config"
chmod 600 "${HOME}/.kube/config"

kubectl get namespace wrangler-tests >/dev/null 2>&1 || \
  kubectl create namespace wrangler-tests