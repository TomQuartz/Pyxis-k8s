#!/bin/bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=`realpath $BASE_DIR/..`
cd $ROOT_DIR

./bin/demo -hops=3 -keys=5

echo -e "\n\n################### Storage-side Logs ###################\n"
storage_pod=$(kubectl get pods -l app=pyxis-storage -o jsonpath='{.items[0].metadata.name}')
kubectl logs $storage_pod
