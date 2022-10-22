#!/bin/bash

mkdir -p packages/kernel
pushd packages/kernel
apt-get download linux-image-5.10.0-18-amd64
ar xf linux-image-5.10.0-18-amd64_5.10.140-1_amd64.deb
tar xf data.tar.xz
popd


mkdir -p bin/
GOBIN=`pwd`/bin/ go install go.universe.tf/netboot/cmd/pixiecore@latest

