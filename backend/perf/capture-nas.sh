#!/bin/sh
# Run ON the NAS while a second machine generates traffic.
# Only selected inspect fields are collected; environment variables are excluded.
set -eu
if [ "$#" -lt 2 ]; then
  printf 'Usage: %s OUTPUT_DIR APP_CONTAINER [MONGO_CONTAINER] [TUNNEL_CONTAINER]\n' "$0" >&2
  exit 2
fi
capture_out=$1
shift
app_container=$1
mkdir -p "$capture_out"
capture_out=$(CDPATH= cd -- "$capture_out" && pwd)
capture_seconds=${CAPTURE_SECONDS:-60}
case "$capture_seconds" in ''|*[!0-9]*|0) printf 'CAPTURE_SECONDS must be a positive integer\n' >&2; exit 2 ;; esac
{
  date -u
  uname -a
  if [ -r /proc/loadavg ]; then cat /proc/loadavg; fi
  if [ -r /proc/meminfo ]; then cat /proc/meminfo; fi
  docker version --format '{{.Server.Version}}'
  docker inspect --format '{{.Name}} image={{.Image}} memoryLimit={{.HostConfig.Memory}} nanoCPUs={{.HostConfig.NanoCpus}} cpuQuota={{.HostConfig.CpuQuota}} cpuPeriod={{.HostConfig.CpuPeriod}} restarts={{.RestartCount}}' "$@"
} > "$capture_out/environment.txt"
(
  capture_end=$(( $(date +%s) + capture_seconds ))
  while [ "$(date +%s)" -lt "$capture_end" ]; do
    date -u +%Y-%m-%dT%H:%M:%SZ
    docker stats --no-stream --format '{{json .}}' "$@"
    if [ -r /proc/stat ]; then cat /proc/stat; fi
    if [ -r /proc/loadavg ]; then cat /proc/loadavg; fi
    # Kernel counters include quota throttling when cgroup v2 is in use. Try
    # the usual v1 path as a fallback; neither path is required on every ADM.
    docker exec "$app_container" cat /sys/fs/cgroup/cpu.stat 2>/dev/null || \
      docker exec "$app_container" cat /sys/fs/cgroup/cpu/cpu.stat 2>/dev/null || true
    if [ -r /proc/pressure/cpu ]; then cat /proc/pressure/cpu; fi
    if [ -r /proc/pressure/io ]; then cat /proc/pressure/io; fi
    if [ "$(date +%s)" -lt "$capture_end" ]; then sleep 2; fi
  done
) > "$capture_out/monitoring.log" 2>&1 &
stats_pid=$!
trap 'kill "$stats_pid" 2>/dev/null || true' EXIT HUP INT TERM
if [ "${PROFILES:-0}" = 1 ]; then
  # BusyBox wget is available in the application's Alpine runtime. The listener
  # stays on container loopback; no profiling port needs publishing.
  perf_base=http://127.0.0.1:${PERF_PORT:-6060}
  docker exec "$app_container" wget -qO- "$perf_base/debug/perf" > "$capture_out/runtime-before.json"
  case "${PROFILE_KIND:-cpu}" in
    cpu)
      docker exec "$app_container" wget -T "$((capture_seconds + 15))" -qO- \
        "$perf_base/debug/pprof/profile?seconds=$capture_seconds" > "$capture_out/cpu.pprof" ;;
    locks)
      # Enable PERF_MUTEX_FRACTION/PERF_BLOCK_RATE in the diagnostic deployment.
      sleep "$capture_seconds"
      docker exec "$app_container" wget -qO- "$perf_base/debug/pprof/mutex" > "$capture_out/mutex.pprof"
      docker exec "$app_container" wget -qO- "$perf_base/debug/pprof/block" > "$capture_out/block.pprof" ;;
    *) printf 'PROFILE_KIND must be cpu or locks\n' >&2; exit 2 ;;
  esac
  docker exec "$app_container" wget -qO- "$perf_base/debug/pprof/heap" > "$capture_out/heap.pprof"
  docker exec "$app_container" wget -qO- "$perf_base/debug/pprof/goroutine" > "$capture_out/goroutine.pprof"
  docker exec "$app_container" wget -qO- "$perf_base/debug/perf" > "$capture_out/runtime-after.json"
fi
wait "$stats_pid"
docker inspect --format '{{.Name}} restarts={{.RestartCount}}' "$@" > "$capture_out/restarts-after.txt"
trap - EXIT HUP INT TERM
printf 'Capture saved to %s\n' "$capture_out"
