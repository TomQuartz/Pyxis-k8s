#!/bin/bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/..`
cd $ROOT_DIR

kubectl apply -f manifests/compute.yaml
kubectl apply -f manifests/storage.yaml

sleep 10
