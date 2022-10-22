#!/bin/bash

echo "Create common script for building the image?"
exit 1

mkdir -p netboot/initrd

CGO_ENABLED=0 go build -o netboot/initrd/init ./cmd/metalagent

pushd netboot/initrd
#find . | cpio -ov --format=newc | gzip -9 >../initramfz
#find . -print0 | cpio --null -ov --format=newc | gzip -n >../initramfz
find . -print0 | cpio --null -ov --format=newc  >../initramfz
popd


#qemu-system-x86_64   -kernel netboot/packages/kernel/boot/vmlinuz-5.10.0-18-amd64 -initrd netboot/initramfz -m 4G --enable-kvm -nographic -serial mon:stdio -append 'console=ttyS0'

sudo ./netboot/bin/pixiecore boot --debug --bootmsg "booting with pixiecore" netboot/packages/kernel/boot/vmlinuz-5.10.0-18-amd64 netboot/initramfz
