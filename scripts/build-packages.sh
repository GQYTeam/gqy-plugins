#!/bin/sh
# 构建两个官方插件的确定性 tar.gz 与 checksums.txt；产物在 dist/，不提交到 Git。
set -eu
cd "$(dirname "$0")/.."
make package
cd dist
if command -v shasum >/dev/null 2>&1; then
  shasum -a 256 show-images-default-1.0.0.tar.gz emote-pack-default-1.0.0.tar.gz > checksums.txt
else
  sha256sum show-images-default-1.0.0.tar.gz emote-pack-default-1.0.0.tar.gz > checksums.txt
fi
cat checksums.txt
