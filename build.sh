#!/bin/bash
set -ex
export GOPATH=$PWD/vendor
export CGO_ENABLED=0
go fmt
go build
strip solar-proxy
