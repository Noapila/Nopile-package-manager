#!/bin/bash
echo "==> Installing Nopile"
mkdir -p /var/lib/nopile || exit 1
mkdir -p /var/tmp/nopile || exit 1
mkdir -p /var/cache/nopile || exit 1
cp ./nopile /usr/bin/nopile || exit 1
echo "Nopile installed Succesfully !"
