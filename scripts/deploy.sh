#! /usr/bin/env bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=$BASE_DIR/..
cd $ROOT_DIR

set -ex

# Usage: compute|storage num_nodes workers_per_node
function deploy_server {
    server=$1
    replicas=$2
    workers=$3
    # update the configmap
    kubectl delete configmap $server-config --ignore-not-found
    kubectl create configmap $server-config \
        --from-literal=WORKERS=$workers
    # delete to force reload the configmap
    kubectl delete -f $ROOT_DIR/manifests/$server.yaml --ignore-not-found
    kubectl apply -f $ROOT_DIR/manifests/$server.yaml
    # scale the deployment
    kubectl scale deployment pyxis-$server --replicas=$replicas
}

# Usage: client [-debug] -arbiter=pyxis|kayak -maxout=8|16|32...
function deploy_client {
    go run $ROOT_DIR/cmd/client/main.go $@
}

case $1 in
compute|storage)
    deploy_server $@
    ;;
clean)
    kubectl delete -f $ROOT_DIR/manifests/compute.yaml --ignore-not-found
    kubectl delete -f $ROOT_DIR/manifests/storage.yaml --ignore-not-found
    ;;
client)
    shift
    deploy_client $@
    ;;
*)
    echo "Usage: $0 {compute|storage|client}"
    exit 1
    ;;
esac