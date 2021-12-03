#!/bin/sh

set -o errexit

#
# Script able to create a Kubernetes cluster using kind tool
# A private docker registry is deployed and available at localhost:5000
# The ingress to route the traffic is installed
# 2 node ports are exposed on the cluster: 30000, 31000

reg_name='kind-registry'
reg_port='5000'

read -p "Do you want to delete the kind cluster (y|n) - Default: n ? " cluster_delete
cluster_delete=${cluster_delete:-n}
read -p "Which kubernetes version should we install (1.14 .. 1.22) - Default: 1.21 ? " version
k8s_minor_version=${version:-1.21}
read -p "What logging verbosity do you want (0..9) - A verbosity setting of 0 logs only critical events - Default: 0 ? " logging_verbosity
logging_verbosity=${logging_verbosity:-0}

kindCmd="kind -v ${logging_verbosity} create cluster"

# Kind cluster config template
kindCfg=$(cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
nodes:
- role: control-plane
  extraMounts:
  - hostPath: $(pwd)/wks
    containerPath: /workspace
  - hostPath: $(pwd)/cache
    containerPath: /cache
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 30000
    hostPort: 30000
    protocol: tcp
  - containerPort: 31000
    hostPort: 31000
    protocol: tcp
EOF
)

if [ "$cluster_delete" == "y" ]; then
  echo "Deleting kind cluster ..."
  kind delete cluster
fi

# Create a kind cluster
# - Configures containerd to use the local Docker registry
# - Enables Ingress on ports 80 and 443
if [ "$k8s_minor_version" != "default" ]; then
  patch_version=$(wget -q https://registry.hub.docker.com/v1/repositories/kindest/node/tags -O - | \
  jq -r '.[].name' | grep -E "^v${k8s_minor_version}.[0-9]+$" | \
  cut -d. -f3 | sort -rn | head -1)
  k8s_version="v${k8s_minor_version}.${patch_version}"
  kindCmd+=" --image kindest/node:${k8s_version}"
else
  k8s_version=$k8s_minor_version
fi

echo "Creating a Kind cluster with Kubernetes version : ${k8s_version} and logging verbosity: ${logging_verbosity}"
echo "${kindCfg}" | ${kindCmd} --config=-

# Start a local Docker registry (unless it already exists)
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# Connect the local Docker registry with the kind network
docker network connect "kind" "${reg_name}" > /dev/null 2>&1 &

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

# Deploy the nginx Ingress controller on k8s >= 1.19
VERSION=$(curl https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/stable.txt)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/$VERSION/deploy/static/provider/kind/deploy.yaml