#!/bin/bash

set -e
set -x

mkdir -p netboot/initrd

CGO_ENABLED=0 go build -o netboot/initrd/init ./cmd/metalinit
CGO_ENABLED=0 go build -o netboot/initrd/sbin/metalagent ./cmd/metalagent

rm -rf netboot/initrd/bin/
mkdir -p netboot/initrd/bin/
cp netboot/busybox/busybox netboot/initrd/bin/busybox

cp netboot/metalagent.service netboot/initrd/metalagent.service

mkdir -p netboot/initrd/usr/share/udhcpc
cp netboot/busybox/examples/udhcp/simple.* netboot/initrd/usr/share/udhcpc/

cp netboot/bin/crane netboot/initrd/bin/crane

mkdir -p netboot/initrd/etc/ssl/certs/
cp /etc/ssl/certs/ca-certificates.crt netboot/initrd/etc/ssl/certs/

pushd netboot/initrd
#find . | cpio -ov --format=newc | gzip -9 >../initramfz
#find . -print0 | cpio --null -ov --format=newc | gzip -n >../initramfz
find . -print0 | cpio --null -ov --format=newc  > ../initramfz
popd

