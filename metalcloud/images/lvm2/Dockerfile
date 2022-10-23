FROM debian:bullseye

RUN apt-get update && apt-get install --yes lvm2

# Changes from "stock":
# obtain_device_list_from_udev = 0 because we're not running udev
COPY lvm.conf /etc/lvm/lvm.conf
