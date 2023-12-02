#!/bin/bash

ssh root@${METAL_HOST} mkdir -p /etc/metal/pki/
scp ${MACHINE_NAME}/ca.crt root@${METAL_HOST}:/etc/metal/pki/ca.crt

scp ${MACHINE_NAME}/client-ca.crt root@${METAL_HOST}:/etc/metal/pki/client-ca.crt

scp ${MACHINE_NAME}/server.crt root@${METAL_HOST}:/etc/metal/pki/server.crt
scp ${MACHINE_NAME}/server.key root@${METAL_HOST}:/etc/metal/pki/server.key

