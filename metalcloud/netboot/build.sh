#!/bin/bash

rm -rf initrd/
mkdir initrd/

#cp -r packages/kernel/lib initrd/lib/

# TODO: Delete modules we definitely don't need (e.g. graphics).  We can load some later.

pushd initrd/
tar xvf ../bootos.tar usr/lib/modules
popd

tar xvf bootos.tar vmlinuz boot/
