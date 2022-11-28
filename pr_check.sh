#!/bin/bash
set -exv

BASE_IMG="gabi"

IMG="${BASE_IMG}:check"

docker login quay.io -u ${QUAY_USER} -p ${QUAY_TOKEN}

./integ.sh

BUILD_CMD="docker build" IMG="$IMG" make docker-build
