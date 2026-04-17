#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
./build.sh
sudo install -m 0755 bin/ranke-cli /usr/local/bin/ranke-cli
echo "installed: $(command -v ranke-cli)"
