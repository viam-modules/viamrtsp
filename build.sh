#!/bin/bash
set -e

make module
OS=$(uname -s)
mv ./bin/$OS/module.tar.gz .
