#!/bin/sh
# Build on a workstation; execute the resulting static test binaries on the NAS.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
backend_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
kit_arch=${KIT_ARCH:-amd64}
kit_out=${1:-"$backend_dir/perf-kit/linux-$kit_arch"}
mkdir -p "$kit_out"
kit_out=$(CDPATH= cd -- "$kit_out" && pwd)
cd "$backend_dir"
for package in hub models database cache utils; do
  printf 'Building %s for linux/%s\n' "$package" "$kit_arch"
  CGO_ENABLED=0 GOOS=linux GOARCH="$kit_arch" go test -c -o "$kit_out/$package.test" "./$package"
done
CGO_ENABLED=0 GOOS=linux GOARCH="$kit_arch" go build -trimpath -o "$kit_out/loadtest" ./cmd/loadtest
# Keep the matching server binary for profiler symbolization and optional deployment.
CGO_ENABLED=0 GOOS=linux GOARCH="$kit_arch" go build -trimpath -o "$kit_out/server" .
cp "$script_dir/run-benchmarks.sh" "$script_dir/capture-nas.sh" "$script_dir/README.md" "$script_dir/FINDINGS.md" "$kit_out/"
{
  date -u
  go version
  printf 'target=linux/%s\n' "$kit_arch"
  git rev-parse HEAD
  git status --short
} > "$kit_out/build-info.txt"
chmod +x "$kit_out/"*.sh
printf 'Ready: %s\n' "$kit_out"
