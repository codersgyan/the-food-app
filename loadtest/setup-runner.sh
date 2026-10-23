#!/usr/bin/env bash
# setup-runner.sh installs what the test runner needs: k6, which sends the
# load, and jq. k6 comes from its GitHub release, checked against the
# release's checksums. Run it on the test runner (Ubuntu, ARM or x86).
set -euo pipefail
K6_VERSION=${K6_VERSION:-v2.3.0}

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq jq > /dev/null

if ! k6 version 2> /dev/null | grep -q "$K6_VERSION"; then
  arch=$(dpkg --print-architecture) # arm64 or amd64
  name="k6-$K6_VERSION-linux-$arch"
  base="https://github.com/grafana/k6/releases/download/$K6_VERSION"
  cd /tmp
  curl -sSLO "$base/$name.tar.gz"
  curl -sSL "$base/k6-$K6_VERSION-checksums.txt" -o k6-checksums.txt
  grep "$name.tar.gz" k6-checksums.txt | sha256sum -c -
  tar xzf "$name.tar.gz"
  sudo install "$name/k6" /usr/local/bin/k6
fi
k6 version
