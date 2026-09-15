#!/usr/bin/env sh
# Build release archives for every supported OS/arch into dist/release/.
#
#   scripts/build.sh <version>          e.g. scripts/build.sh 0.1.0
#
# Produces:
#   dist/release/folder-inspect_<version>_windows_amd64.zip
#   dist/release/folder-inspect_<version>_linux_amd64.tar.gz
#   dist/release/SHA256SUMS
# Each archive holds the executable, README.md, LICENSE and docs/folder-inspect.example.yml.
# The same script runs in CI (ci.yml) and on release tags (release.yml). Needs: go, zip, tar.
set -eu

version="${1:?usage: scripts/build.sh <version>}"
cd "$(dirname "$0")/.."

out="dist/release"
rm -rf "$out"
mkdir -p "$out"

ldflags="-s -w -X main.version=${version}"
extras="README.md LICENSE docs/folder-inspect.example.yml"

# make_zip <folder under dist/stage> <target zip>: zip on Linux/macOS runners; on a Windows
# dev host (Git Bash has no zip) fall back to 7z or PowerShell's Compress-Archive.
make_zip() {
  folder="$1"; target="$2"
  if command -v zip >/dev/null 2>&1; then
    (cd dist/stage && zip -q -r "../../${target}" "$folder")
  elif command -v 7z >/dev/null 2>&1; then
    (cd dist/stage && 7z a -tzip -bso0 -bsp0 "../../${target}" "$folder")
  elif command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -Command       "Compress-Archive -Path 'dist/stage/${folder}' -DestinationPath '${target}' -Force"
  else
    echo "no zip, 7z or powershell available to create ${target}" >&2
    exit 1
  fi
}

build() {
  goos="$1"; goarch="$2"; ext="$3"
  name="folder-inspect_${version}_${goos}_${goarch}"
  stage="dist/stage/${name}"
  rm -rf "$stage"
  mkdir -p "$stage"
  echo "building ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$ldflags" \
    -o "${stage}/folder-inspect${ext}" ./cmd/folder-inspect
  cp $extras "$stage/"
  if [ "$goos" = "windows" ]; then
    make_zip "$name" "${out}/${name}.zip"
  else
    tar -czf "${out}/${name}.tar.gz" -C dist/stage "$name"
  fi
}

build windows amd64 .exe
build linux amd64 ""

rm -rf dist/stage
(cd "$out" && sha256sum ./* | sed 's#\./##' > SHA256SUMS)
echo "done:"
ls -l "$out"
