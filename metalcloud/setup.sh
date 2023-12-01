#!/bin/bash

set -e
set -x

export METAL_HOST=10.78.79.73

go run ./cmd/metaldo ping
go run ./cmd/metaldo exec -- /bin/modprobe t10_pi
go run ./cmd/metaldo exec -- /bin/modprobe nvme_core
go run ./cmd/metaldo exec -- /bin/modprobe nvme
go run ./cmd/metaldo exec -- /bin/modprobe dm_mod

go run ./cmd/metaldo exec -- /bin/modprobe crc16

# Workaround for module aliasing (?) https://www.spinics.net/lists/linux-ext4/msg60651.html
go run ./cmd/metaldo exec -- /bin/modprobe crc32c-intel

go run ./cmd/metaldo exec -- /bin/modprobe mbcache
go run ./cmd/metaldo exec -- /bin/modprobe md_mod
go run ./cmd/metaldo exec -- /bin/modprobe linear
go run ./cmd/metaldo exec -- /bin/modprobe jbd2
go run ./cmd/metaldo exec -- /bin/modprobe ext4

go run ./cmd/metaldo exec -- /bin/mkdir /sys
go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /sys

go run ./cmd/metaldo exec -- /bin/mkdir /run
go run ./cmd/metaldo exec -- /bin/mount -t tmpfs tmpfs /run

go run ./cmd/metaldo exec -- /bin/dmesg

go run ./cmd/metaldo ping
go run ./cmd/metaldo exec -- /bin/mkdir /lvm2
go run ./cmd/metaldo exec -- /bin/crane export justinsb/contained-lvm2 /lvm2/image.tar
go run ./cmd/metaldo exec -- /bin/mkdir /lvm2/rootfs
go run ./cmd/metaldo exec -- /bin/tar -xf /lvm2/image.tar -C /lvm2/rootfs
go run ./cmd/metaldo exec -- /bin/mount -t proc procfs /lvm2/rootfs/proc
go run ./cmd/metaldo exec -- /bin/mount -t devtmpfs udev /lvm2/rootfs/dev
go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /lvm2/rootfs/sys
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/lvs --report json

if false; then
    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan --mknodes


    go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap
    go run ./cmd/metaldo exec -- /bin/crane export justinsb/contained-debootstrap /debootstrap/image.tar
    go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap/rootfs
    go run ./cmd/metaldo exec -- /bin/tar -xf /debootstrap/image.tar -C /debootstrap/rootfs
    go run ./cmd/metaldo exec -- /bin/mount -t proc procfs /debootstrap/rootfs/proc
    go run ./cmd/metaldo exec -- /bin/mount -t devtmpfs udev /debootstrap/rootfs/dev
    go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /debootstrap/rootfs/sys

    go run ./cmd/metaldo exec --chroot /debootstrap/rootfs -- /sbin/debootstrap --help
    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/lvs --reportformat json
    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgs --reportformat json
    # We need --zero n or else we need udev: https://serverfault.com/questions/827251/cannot-do-lvcreate-not-found-device-not-cleared-on-centos
    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/lvcreate --zero n -L 10G -n root-b pool

    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan --mknodes
    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/mkfs.ext4 /dev/pool/root-a

    go run ./cmd/metaldo exec -- /bin/ls /dev/pool

    go run ./cmd/metaldo exec -- /bin/mkdir -p /debootstrap/rootfs/mnt
    go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root-a /debootstrap/rootfs/mnt


    go run ./cmd/metaldo exec --chroot /debootstrap/rootfs -- /usr/sbin/debootstrap --include=apparmor,ca-certificates,systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye /mnt
    #sudo /sbin/debootstrap --cache-dir=`pwd`/cache/debootstrap --include=systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye initrd
    go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root-a /mnt


    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /bin/mount -t ext4 /dev/pool/root-a /mnt

    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/switch_root /mnt /sbin/init

    go run ./cmd/metaldo exec -- /bin/switch_root /mnt /sbin/init
fi


go run ./cmd/metaldo exec -- /bin/mkdir -p /mnt
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgchange -a y
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan -vvvv --mknodes
go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root-a /mnt

if false; then
    go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap
    go run ./cmd/metaldo exec -- /bin/crane export justinsb/contained-debootstrap /debootstrap/image.tar
    go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap/rootfs
    go run ./cmd/metaldo exec -- /bin/tar -xf /debootstrap/image.tar -C /debootstrap/rootfs
    go run ./cmd/metaldo exec -- /bin/mount -t proc procfs /debootstrap/rootfs/proc
    go run ./cmd/metaldo exec -- /bin/mount -t devtmpfs udev /debootstrap/rootfs/dev
    go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /debootstrap/rootfs/sys

    go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root-a /debootstrap/rootfs/mnt

    go run ./cmd/metaldo exec --chroot /debootstrap/rootfs -- /usr/sbin/debootstrap --include=systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye /mnt
fi


go run ./cmd/metaldo exec -- /bin/ls /mnt
go run ./cmd/metaldo exec -- /bin/ls /mnt/sbin

# remove password on root account
go run ./cmd/metaldo exec --chroot /mnt -- /usr/bin/passwd -d root

go run ./cmd/metaldo exec -- /bin/cp /init /mnt/sbin/metalagent
# TODO: make is possible to just write a file?
go run ./cmd/metaldo exec -- /bin/mkdir -p /etc/systemd/system/
go run ./cmd/metaldo exec -- /bin/cp /metalagent.service /mnt/etc/systemd/system/metalagent.service
#go run ./cmd/metaldo exec -- /bin/chmod 644 /mnt/etc/systemd/system/metalagent.service
go run ./cmd/metaldo exec --chroot /mnt -- /usr/bin/systemctl enable metalagent.service

sleep 10; go run ./cmd/metaldo exec --replace -- /bin/switch_root /mnt /sbin/init
