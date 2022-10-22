#!/bin/bash

rm -rf initrd/
mkdir initrd/

cp -r packages/kernel/lib initrd/lib/

# TODO: Delete modules we definitely don't need (e.g. graphics).  We can load some later.


mkdir -p bin/
GOBIN=`pwd`/bin/ go install go.universe.tf/netboot/cmd/pixiecore@latest

