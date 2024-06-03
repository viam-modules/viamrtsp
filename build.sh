#!/bin/bash
set -e

sudo apt-get update
sudo apt-get install -y pkg-config

make module
