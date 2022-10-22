#!/bin/bash

set -e
set -x

mkdir -p netboot/initrd

CGO_ENABLED=0 go build -o netboot/initrd/init ./cmd/metalagent

rm -rf netboot/initrd/bin/
mkdir -p netboot/initrd/bin/
cp netboot/busybox/busybox netboot/initrd/bin/busybox


mkdir -p netboot/initrd/usr/share/udhcpc
cp netboot/busybox/examples/udhcp/simple.* netboot/initrd/usr/share/udhcpc/

pushd netboot/initrd
#find . | cpio -ov --format=newc | gzip -9 >../initramfz
#find . -print0 | cpio --null -ov --format=newc | gzip -n >../initramfz
find . -print0 | cpio --null -ov --format=newc  >../initramfz
popd


qemu-system-x86_64  \
  -kernel netboot/packages/kernel/boot/vmlinuz-5.10.0-18-amd64 \
  -initrd netboot/initramfz \
  -netdev user,id=net0,net=192.168.76.0/24,dhcpstart=192.168.76.9 \
  -device rtl8139,netdev=net0 \
  -m 4G --enable-kvm \
  -nographic -serial mon:stdio -append 'console=ttyS0'
