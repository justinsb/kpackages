#!/bin/bash

set -e
set -x

export METAL_HOST=10.78.79.73

go run ./cmd/metaldo ping

# For module loading, at least
go run ./cmd/metaldo exec -- /bin/ln -sf usr/lib /lib



# go run ./cmd/metaldo exec -- /bin/modprobe t10_pi
# go run ./cmd/metaldo exec -- /bin/modprobe nvme_core
# go run ./cmd/metaldo exec -- /bin/modprobe nvme
go run ./cmd/metaldo exec -- /bin/modprobe dm_mod

# go run ./cmd/metaldo exec -- /bin/modprobe crc16

# # Workaround for module aliasing (?) https://www.spinics.net/lists/linux-ext4/msg60651.html
# go run ./cmd/metaldo exec -- /bin/modprobe crc32c-intel

# go run ./cmd/metaldo exec -- /bin/modprobe mbcache
# go run ./cmd/metaldo exec -- /bin/modprobe md_mod
# go run ./cmd/metaldo exec -- /bin/modprobe linear
# go run ./cmd/metaldo exec -- /bin/modprobe jbd2
go run ./cmd/metaldo exec -- /bin/modprobe ext4

# go run ./cmd/metaldo exec -- /bin/mkdir /sys
# go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /sys

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

# Repartition nvme0n1 and run LVM on it
if false; then
go run ./cmd/metaldo exec -- /bin/mkdir /fdisk
go run ./cmd/metaldo exec -- /bin/crane export justinsb/contained-fdisk /fdisk/image.tar
go run ./cmd/metaldo exec -- /bin/mkdir /fdisk/rootfs
go run ./cmd/metaldo exec -- /bin/tar -xf /fdisk/image.tar -C /fdisk/rootfs
go run ./cmd/metaldo exec -- /bin/mount -t proc procfs /fdisk/rootfs/proc
go run ./cmd/metaldo exec -- /bin/mount -t devtmpfs udev /fdisk/rootfs/dev
go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /fdisk/rootfs/sys
go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/fdisk -l
go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/sfdisk --delete /dev/nvme0n1 2
go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/sfdisk --delete /dev/nvme0n1 3
go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/sfdisk --delete /dev/nvme0n1
go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/sfdisk --dump /dev/nvme0n1 << EOF
label: gpt
EOF

go run ./cmd/metaldo exec --chroot /fdisk/rootfs -- /sbin/sfdisk /dev/nvme0n1 << EOF
,1024M
;
EOF
   #go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgremove pool -f


    go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgcreate pool /dev/nvme0n1p2

   go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan --mknodes
fi


go run ./cmd/metaldo exec -- /bin/mkdir -p /mnt
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgchange -a y
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan --mknodes
#go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root0 /mnt

# Create root0 LV
if false; then
  go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/lvs --reportformat json
  go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgs --reportformat json

  # We need --zero n or else we need udev: https://serverfault.com/questions/827251/cannot-do-lvcreate-not-found-device-not-cleared-on-centos
  go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/lvcreate --zero n -L 10G -n root0 pool

  go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan --mknodes
  go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/mkfs.ext4 /dev/pool/root0
fi

# Install OS into root0
if false; then
  go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap
  go run ./cmd/metaldo exec -- /bin/crane export justinsb/contained-debootstrap /debootstrap/image.tar
  go run ./cmd/metaldo exec -- /bin/mkdir /debootstrap/rootfs
  go run ./cmd/metaldo exec -- /bin/tar -xf /debootstrap/image.tar -C /debootstrap/rootfs
  go run ./cmd/metaldo exec -- /bin/mount -t proc procfs /debootstrap/rootfs/proc
  go run ./cmd/metaldo exec -- /bin/mount -t devtmpfs udev /debootstrap/rootfs/dev
  go run ./cmd/metaldo exec -- /bin/mount -t sysfs sysfs /debootstrap/rootfs/sys

# TODO: Who did mkdir on /debootstrap/rootfs/mnt ??
  go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root0 /debootstrap/rootfs/mnt

    # TODO: Upgrade to bookworm
  go run ./cmd/metaldo exec --chroot /debootstrap/rootfs -- /usr/sbin/debootstrap --include=systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye /mnt
# go run ./cmd/metaldo exec --chroot /debootstrap/rootfs -- /usr/sbin/debootstrap --include=apparmor,ca-certificates,systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye /mnt

  go run ./cmd/metaldo exec -- /bin/umount  /debootstrap/rootfs/mnt
fi





    #sudo /sbin/debootstrap --cache-dir=`pwd`/cache/debootstrap --include=systemd-sysv,linux-image-amd64,iproute2,isc-dhcp-client,openssh-server,openssl,curl,lvm2 --variant=minbase bullseye initrd
    #go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root0 /mnt


    #go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /bin/mount -t ext4 /dev/pool/root0 /mnt

    #go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/switch_root /mnt /sbin/init

    #go run ./cmd/metaldo exec -- /bin/switch_root /mnt /sbin/init

# Mount root0 on /mnt
go run ./cmd/metaldo exec -- /bin/mount -t ext4 /dev/pool/root0 /mnt
go run ./cmd/metaldo exec -- /bin/ls /mnt
go run ./cmd/metaldo exec -- /bin/ls /mnt/sbin

# Copy modules so we have the right modules
# TODO: Should we bother installing the kernel?
go run ./cmd/metaldo exec -- /bin/cp -r /usr/lib/modules/6.1.0-13-amd64/ /mnt/usr/lib/modules/

# remove password on root account
go run ./cmd/metaldo exec --chroot /mnt -- /usr/bin/passwd -d root

go run ./cmd/metaldo exec -- /bin/cp /init /mnt/sbin/metalagent
# TODO: make it possible to just write a file?
go run ./cmd/metaldo exec -- /bin/mkdir -p /etc/systemd/system/
go run ./cmd/metaldo exec -- /bin/cp /metalagent.service /mnt/etc/systemd/system/metalagent.service
#go run ./cmd/metaldo exec -- /bin/chmod 644 /mnt/etc/systemd/system/metalagent.service
go run ./cmd/metaldo exec --chroot /mnt -- /usr/bin/systemctl enable metalagent.service

go run ./cmd/metaldo exec -- /bin/hostname lenovo920q-1

sleep 10; go run ./cmd/metaldo exec --replace -- /bin/switch_root /mnt /sbin/init

# Set ssh key
# TODO: Use cp command?
go run ./cmd/metaldo exec mkdir /root/.ssh/
cat ~/.ssh/id_ed25519.pub | go run ./cmd/metaldo exec tee /root/.ssh/authorized_keys

# Set hostname
echo "lenovo920q-1" | go run ./cmd/metaldo exec tee /etc/hostname
go run ./cmd/metaldo exec -- hostname -F /etc/hostname

# Turn off swap
# TODO: Is there any swap?  if so, should we repartition?
go run ./cmd/metaldo exec -- swapon --summary
go run ./cmd/metaldo exec -- swapoff --all

# Switch to the correct kernel
# # TODO: Maybe easier to just install a bootloader?
# go run ./cmd/metaldo exec -- apt-get install --yes kexec-tools
# go run ./cmd/metaldo exec -- /sbin/kexec -l /boot/vmlinuz-5.10.0-26-amd64 --initrd=/boot/initrd.img-5.10.0-26-amd64
#go run ./cmd/metaldo exec -- apt-get install --yes grub-efi
# go run ./cmd/metaldo exec -- apt-get install --yes grub
# go run ./cmd/metaldo exec -- grub2-mkconfig -o /boot/grub2/grub.cfg
# go run ./cmd/metaldo exec -- grub2-install /dev/nvme0


# Create thinpool for volumes
#	// Must precreate thinpool with: lvcreate -L 200G -T pool/thinpool
go run ./cmd/metaldo exec -- lvcreate -L 200G -T pool/thinpool
#	// Can extend with e.g. /sbin/lvextend -L 20G pool/thinpool
go run ./cmd/metaldo exec -- /sbin/vgscan --mknodes
go run ./cmd/metaldo exec --chroot /lvm2/rootfs -- /sbin/vgscan  --mknodes
# must run vgchange -a y ?
# get error: /usr/sbin/thin_check: execvp failed: No such file or directory
# go run ./cmd/metaldo exec -- apt install --yes thin-provisioning-tools

# We need wget (or curl) for kOps
go run ./cmd/metaldo exec -- apt-get install --yes wget

# We need reasonably accurate time for certificate sigining etc
# chrony vs systemd-timesyncd: systemd-timesyncd is smaller/faster, but less accurate.
# kOps already uses chrony, so we want to stick with that for now.
go run ./cmd/metaldo exec -- apt-get install --yes chrony
go run ./cmd/metaldo exec -- systemctl status chrony
# Maybe we need to set the time once in case of too much drift?


# Add keys?
#cat ~/.ssh/authorized_keys | ssh root@10.78.79.73 tee -a /root/.ssh/authorized_keys