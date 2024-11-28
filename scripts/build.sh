#! /usr/bin/env bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=$BASE_DIR/..

set -ex

REGISTRY=shengqipku
TAG=${TAG:-"latest"}

export DOCKER_BUILDKIT=1

function build {
    targets=("compute" "storage")
    if [ "$#" -gt 0 ]; then
        targets=("$@")
    fi
    echo "Building: ${targets[@]}"

    cd $ROOT_DIR
    for target in ${targets[@]}; do
        image=$REGISTRY/pyxis-$target:$TAG
        docker build --platform linux/amd64 --build-arg TARGET=$target \
            -f $ROOT_DIR/build/Dockerfile.server \
            -t $image .
        docker push $image
    done
}

build $@
