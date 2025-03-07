#!/bin/bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/..`
cd $ROOT_DIR

kubectl delete -f manifests/compute.yaml
kubectl delete -f manifests/storage.yaml
