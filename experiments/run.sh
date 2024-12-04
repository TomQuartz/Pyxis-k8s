#! /usr/bin/env bash

BASE_DIR=`realpath $(dirname $0)`
ROOT_DIR=$BASE_DIR/..

cd $ROOT_DIR

# Usage: run.sh RUN
RUN=${1:-"0"}
rm -rf experiments/$RUN
mkdir -p experiments/$RUN

./scripts/deploy.sh clean
sleep 10
./scripts/deploy.sh compute 4 4
./scripts/deploy.sh storage 1 4
sleep 10

KAYAK_MAXOUTS=(2 4 8 12 16)
PYXIS_MAXOUTS=(2 4 8 16 24 32 48)
mkdir -p experiments/$RUN/results

# Kayak
arbiter="kayak"
for maxout in ${KAYAK_MAXOUTS[@]}; do
    ./scripts/deploy.sh client -arbiter=$arbiter -maxout=$maxout -time=60 > experiments/$RUN/results/$arbiter-$maxout.log
    sleep 10
done

# Pyxis
arbiter="pyxis"
for maxout in ${PYXIS_MAXOUTS[@]}; do
    ./scripts/deploy.sh client -arbiter=$arbiter -maxout=$maxout -time=60 > experiments/$RUN/results/$arbiter-$maxout.log
    sleep 10
done

./scripts/deploy.sh clean
./experiments/plot.py $RUN
