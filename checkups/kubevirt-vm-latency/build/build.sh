#!/bin/bash

set -ex

SCRIPT_PATH=$(dirname "$(realpath "$0")")
PROJECT_PATH="$(realpath ${SCRIPT_PATH}/..)"

CRI="${CRI:-podman}"

IMAGE="${IMAGE:-kubevirt-latency-check}"

TEMPLATE="$SCRIPT_PATH/Dockerfile.in"
BASE_IMAGE="alpine:3.14"
BIN="latencycheck"

build_dir=$(mktemp -d "/tmp/build.XXXX")
trap "rm -rf $build_dir" EXIT

# build go binray
go build -o "$build_dir/$BIN" $PROJECT_PATH/cmd/*.go

# generate Dockerfile from template
(
    export BASE_IMAGE=$BASE_IMAGE BIN=$BIN
    envsubst < "$TEMPLATE" > "$build_dir/Dockerfile"
)

pushd "$build_dir"
    # build
    $CRI build . -t "$IMAGE"
popd
