#!/bin/bash
set -exv

BASE_IMG="gabi"

IMG="${BASE_IMG}:check"

BUILD_CMD="docker build" IMG="$IMG" make docker-build
