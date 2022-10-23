#!/bin/bash

set -e
set -x

./build.sh


qemu-system-x86_64  \
  -kernel netboot/packages/kernel/boot/vmlinuz-5.10.0-18-amd64 \
  -initrd netboot/initramfz \
  -netdev user,id=net0,net=192.168.76.0/24,dhcpstart=192.168.76.9 \
  -device rtl8139,netdev=net0 \
  -m 4G --enable-kvm \
  -nographic -serial mon:stdio -append 'console=ttyS0'
