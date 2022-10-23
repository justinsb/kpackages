#!/bin/bash

set -e
set -x

./build.sh

sudo ./netboot/bin/pixiecore boot --debug --bootmsg "booting with pixiecore" netboot/packages/kernel/boot/vmlinuz-5.10.0-18-amd64 netboot/initramfz
