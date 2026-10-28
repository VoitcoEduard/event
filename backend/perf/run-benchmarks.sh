#!/bin/sh
set -eu
kit_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
result_dir=${1:-"$kit_dir/results/$(date -u +%Y%m%dT%H%M%SZ)"}
mkdir -p "$result_dir"
result_dir=$(CDPATH= cd -- "$result_dir" && pwd)
{
  date -u
  uname -a
  if [ -r /proc/cpuinfo ]; then cat /proc/cpuinfo; fi
  if [ -r /proc/meminfo ]; then cat /proc/meminfo; fi
  if [ -f "$kit_dir/build-info.txt" ]; then cat "$kit_dir/build-info.txt"; fi
  printf 'BENCHTIME=%s COUNT=%s BENCH_CPU=%s\n' "${BENCHTIME:-500ms}" "${COUNT:-3}" "${BENCH_CPU:-1,4}"
} > "$result_dir/environment.txt"
for package in hub models database cache; do
  printf 'Benchmarking %s (results: %s)\n' "$package" "$result_dir/$package.txt"
  "$kit_dir/$package.test" -test.run='^$' -test.bench='Benchmark(Quiz|SimpleQuiz|Teams|Event|MediaCache)' \
    -test.benchmem -test.benchtime="${BENCHTIME:-500ms}" -test.count="${COUNT:-3}" \
    -test.cpu="${BENCH_CPU:-1,4}" > "$result_dir/$package.txt" 2>&1
  cat "$result_dir/$package.txt"
done
# Profiles are separate from the baseline because sampling changes performance.
case "${PROFILE:-none}" in
  none) ;;
  cpu)
    "$kit_dir/hub.test" -test.run='^$' -test.bench='BenchmarkQuizAnswerBurst/players=150/history=100$' \
      -test.benchtime=5s -test.cpuprofile="$result_dir/hub.cpu.pprof" \
      -test.memprofile="$result_dir/hub.heap.pprof" > "$result_dir/profile.txt" 2>&1 ;;
  locks)
    "$kit_dir/hub.test" -test.run='^$' -test.bench='BenchmarkQuizConcurrentGames' \
      -test.benchtime=5s -test.cpu=4 -test.mutexprofile="$result_dir/hub.mutex.pprof" \
      -test.mutexprofilefraction=5 -test.blockprofile="$result_dir/hub.block.pprof" \
      -test.blockprofilerate=1000000 > "$result_dir/profile.txt" 2>&1 ;;
  *) printf 'PROFILE must be none, cpu, or locks\n' >&2; exit 2 ;;
esac
if [ "${IMAGE_BENCH:-0}" = 1 ]; then
  "$kit_dir/utils.test" -test.run='^$' -test.bench=BenchmarkImageConversion \
    -test.benchmem -test.benchtime=3x -test.count=1 > "$result_dir/images.txt" 2>&1
  cat "$result_dir/images.txt"
fi
printf 'Results saved to %s\n' "$result_dir"
