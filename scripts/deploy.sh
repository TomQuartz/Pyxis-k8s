#! /usr/bin/env bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=$BASE_DIR/..

set -ex

function deploy_server {
    kubectl apply -f $ROOT_DIR/manifests/server.yaml
}

# Usage: client -arbiter=pyxis|kayak -maxout=8|16|32...
function deploy_client {
    go run $ROOT_DIR/cmd/client/main.go $@
}

case $1 in
server)
    deploy_server
    ;;
client)
    shift
    deploy_client $@
    ;;
*)
    echo "Usage: $0 {server|client}"
    exit 1
    ;;
esac