# AS6704T performance suite

This suite identifies capacity limits before changing scoring. It does not
change the game's scoring rules. The AS6704T uses an Intel N5105 with four CPU
cores and the Linux `amd64` target. Installed RAM, Docker limits, storage and
other NAS tasks still affect the result. Record those with every baseline.

There are three complementary measurements:

1. **Code benchmarks on the NAS:** repeatable CPU/allocation measurements using
   the real Go handlers and data models, without a database or network.
2. **Deployment load from another machine:** real WebSockets, participant
   registration, event polling, media downloads and a simultaneous answer burst.
3. **NAS capture:** app/MongoDB/tunnel CPU, memory, network and disk counters,
   plus optional Go CPU, heap, goroutine and contention profiles.

Do not equate a microbenchmark with production capacity. Run deployment tests
against a disposable event in a separate development deployment/database. Tests
register real participants and append real quiz answers. Delete the disposable
event through the application when finished; the tool does not erase data.

## Build once on your workstation

From the repository root, with the project's Go version installed:

```sh
sh backend/perf/build-kit.sh
```

The output is `backend/perf-kit/linux-amd64/`. Copy this directory to the NAS.
It includes static Linux binaries, scripts and a build manifest. Go, Node and
MongoDB do not need to be installed on the NAS for the code benchmarks. The
`server` binary is retained for matching profile symbols; the script does not
replace or restart the deployed app. Rebuild the normal Docker image to enable
the optional diagnostics code in a deployment.

If ADM does not allow executing binaries from the selected folder, mount the
kit into an Alpine container and run there:

```sh
docker run --rm --network none \
  -v "$PWD/linux-amd64:/perf" -w /perf alpine:3.22 \
  sh /perf/run-benchmarks.sh /perf/results/container-baseline
```

The container writes results as its user; choose the output location accordingly.
Use the same Docker CPU/memory limits as the app if you have configured limits.

## Code benchmarks on the NAS

Inside the copied directory:

```sh
sh run-benchmarks.sh ./results/baseline
```

Defaults: 500 ms per case, three repetitions, `GOMAXPROCS` 1 and 4. This takes
several minutes. For a quick check:

```sh
BENCHTIME=100ms COUNT=1 sh run-benchmarks.sh ./results/quick
```

Each operation in `BenchmarkQuizAnswerBurst` is a **whole question's burst**
(50/150/300 answers), including ordinary in-memory reset work. Fixtures seed
0, 25 or 100 previous questions. The other benchmarks isolate:

| Benchmark | What it measures |
| --- | --- |
| `QuizFullStateJSON` | Full session encoding, including history; reports bytes per recipient |
| `QuizObserverBroadcast` | Scan of 150/600/1,500 connections for a small observer update |
| `QuizConcurrentGames` | Parallel handlers sharing the hub mutex; includes per-answer fixture reset and channel scheduling |
| `SimpleQuizWorkingResults` | Grading projection and leaderboard sorting |
| `SimpleQuizPersistenceBSON` | Encoding the full document replaced on each submission/grade; excludes MongoDB/disk |
| `TeamsScoreProjection` | Replay/materialization of 50/500/2,000 awards with 15 recipients each |
| `EventPlayerTotal` | In-memory total scan across 5/25/100 modules; excludes query/aggregation |
| `EventCacheHitParallel`, `MediaCacheHitParallel` | Hot-cache lock/lookup contention |

Handler logging is discarded in code benchmarks to isolate CPU. Real log
formatting/output, database activity and connection fan-out are captured in the
deployment test. Concurrent-games benchmarks use no network clients and do not
measure broadcast fan-out. Do not interpret their ns/op as end-to-end latency.

To profile a code benchmark separately from the baseline:

```sh
PROFILE=cpu COUNT=1 sh run-benchmarks.sh ./results/cpu
PROFILE=locks COUNT=1 sh run-benchmarks.sh ./results/locks
```

Optional upload-conversion measurement requires `cwebp` on the host/container:

```sh
IMAGE_BENCH=1 COUNT=1 sh run-benchmarks.sh ./results/images
```

This converts a synthetic noisy 4K JPEG through the real temporary-file/cwebp
path, three times. A missing converter produces a SKIP, not a successful image
capacity measurement. Child-process CPU is visible in Docker/OS counters,
not in the parent Go CPU profile. A host run can use a different temp disk from
the app container; use the app's storage layout for meaningful comparisons.

## Deployment scenarios from your laptop

Run the load generator on your laptop, not the measured NAS. On macOS use
`go run ./cmd/loadtest` from `backend/`; the kit's `loadtest` is a Linux binary.
An administrator starts the disposable quiz from the ordinary UI. The tool
joins as a public projector observer, so it needs no administrator credentials.

Create a quiz with a sufficiently long active question (e.g. 300 seconds).
Start question zero and run:

```sh
go run ./cmd/loadtest \
  -ws-url wss://quiz.example.com/ws \
  -module TEST_MODULE_ID -question 0 -players 150 \
  -poll-url 'https://quiz.example.com/events/TEST_EVENT_ID?userId={id}' \
  -hold 60s -output /tmp/quiz-150.json \
  -confirm I_UNDERSTAND_THIS_WRITES_TEST_ANSWERS
```

Personal polling automatically registers each participant first using the event's
`/participants/join` route. The actual returned participant ID is used for both
WebSocket play and polling. The tester checks that personal event responses
contain `participantStatus`; an HTTP 200 that silently omitted score aggregation
is a failure. Join timing includes registration when enabled. `-register-url`
overrides the inferred join endpoint.

Use `-media-url https://quiz.example.com/media/EXISTING_MEDIA_ID` to include one
simultaneous download per player. Keep the real media URL: cache-busting would
test a different workload. Test a fresh media item and a warm item separately.

Reports contain p50/p95/p99/max joins and polling, media request timings,
whole-burst observer delay, failures, disconnects and generator queue drops.
HTTP timing includes reading the response body. Redirects/HTML fallbacks fail.
The generator fails with a nonzero exit code when a threshold or completeness
check fails. Default gates are a 1 s burst, 5 s join p95, 1 s polling p95 and
zero HTTP failures/disconnects/queue drops. Override gates with `-max-burst`,
`-max-join-p95`, `-max-poll-p95` to match your required responsiveness.

**A fresh unanswered question is required for every burst.** Reset the test game
or advance to the next question and update `-question`. Reused questions are
rejected before load generation; old answer counts must not create a false pass.
Only the generator and intended host/projector should use this disposable module.
Burst timing is time until the observer sees all answers, not per-player score
confirmation latency (that message does not exist in the current protocol).
The preflight also rejects a question with insufficient scoring time remaining;
registration/media-download time can still consume the window, so use a long
question and inspect completeness failures rather than retrying on the same one.

Recommended matrix:

| Scenario | Configuration/purpose |
| --- | --- |
| Join storm | 25, 50, 100, 150, then 225/300 players; default simultaneous joins |
| Normal arrivals | Add `-join-spread 30s` to the same workload |
| Poll/cache expiry | `-hold 60s`; spans multiple 15-second cache TTLs |
| Soak | `-mode idle -require-active=false -hold 30m`; lobby or active game; connections/polling only |
| Late-game burst | Use real accumulated history; run later questions in the UI with new generator runs |
| Multiple games | Run two/four generator processes against separate modules simultaneously; compare with one game |
| Media storm | Add an existing `-media-url`; compare warm vs fresh item |
| Network/tunnel | Repeat on LAN and public hostname; keep `-origin` equal to PUBLIC_BASE_URL |
| Background interference | Repeat with normal backup/indexing/other NAS activity |

For LAN tests, for example `-ws-url ws://NAS_IP:8082/ws`, explicitly set
`-origin https://your-configured-public-origin.example`. The WebSocket server
requires that Origin even when testing a direct address. Use matching LAN
HTTP poll/media URLs to isolate the tunnel. A long-lived idle soak does not
simulate automatic question transitions, grading, uploads or repeated answers.
Drive those through the normal test UI or run separate burst tests.

## Capture resource usage on the NAS

While the laptop generates traffic, run on the NAS (substitute container names):

```sh
CAPTURE_SECONDS=90 sh capture-nas.sh ./results/deployment app mongo tunnel
```

This captures selected containers, restart counts, host memory and Linux CPU/I/O
pressure where available, host CPU counters and application quota-throttling
counters when readable. It does not collect container environment variables,
credentials or application logs. Container CPU can exceed 100%: one saturated
core is roughly 100%, so the four-core host can approach 400% before accounting
for limits/other tasks. Inspect app **and** MongoDB **and** tunnel, and the NAS
host's overall load; the app alone does not tell the whole story.

For deeper diagnosis add these environment variables to a test app deployment:

```yaml
PERF_ADDR: 127.0.0.1:6060
```

Restart/redeploy it through your usual process. No profile port needs publishing.
Diagnostics use a separate private listener; the public API does not register
these routes. Profiling is off by default. Capture during a load/soak window:

```sh
PROFILES=1 PROFILE_KIND=cpu CAPTURE_SECONDS=30 \
  sh capture-nas.sh ./results/profile-cpu app mongo tunnel
```

For a separate contention run enable:

```yaml
PERF_ADDR: 127.0.0.1:6060
PERF_MUTEX_FRACTION: "5"
PERF_BLOCK_RATE: "1000000"
```

Then capture with `PROFILES=1 PROFILE_KIND=locks`. Mutex/block captures are
cumulative since sampling was enabled; restart the diagnostic deployment before
comparing scenarios. Run CPU and contention profiling separately. Remove the
sampling variables for baseline measurements; sampling itself adds overhead.
The runtime snapshots report actual Go parallelism, heap, allocations, GC cycles
and goroutines. `PERF_PORT` changes the capture port if you changed PERF_ADDR.

Copy results and the matching `server` binary back to your workstation:

```sh
go tool pprof -top ./server ./results/profile-cpu/cpu.pprof
go tool pprof -top -alloc_space ./server ./results/profile-cpu/heap.pprof
go tool pprof -top ./server ./results/profile-locks/mutex.pprof
```

Use the exact deployed binary/image build for symbolization. For code benchmark
profiles use `hub.test` instead of `server`. Retain both old/new raw benchmark
files and compare repetitions on the same NAS with the same limits/background
load; the local workstation's timings cannot establish NAS capacity.

## Capacity decision

Start with the expected peak and a 1.5x overload rehearsal. Select the production
cap below the first level that violates your latency gates, loses answers or
connections, steadily grows memory, swaps, or saturates CPU/disk. Keep headroom
for NAS background work. A single passing run is not a capacity guarantee: repeat
each important scenario at least three times and include the soak and later-game
history. Do not extrapolate a safe player count from microbenchmarks alone.

For suspected database bottlenecks inspect the event-score aggregation's query
plan and the `quiz_results` indexes on a representative test dataset. This suite
does not automatically run MongoDB explain/profiling commands or seed old runs.
Populate a representative disposable event, including previous runs and team
awards, before measuring polling. Simple Quiz submission/grade persistence,
team mutations and uploads need separate UI-driven deployment scenarios; their
code benchmarks isolate costs but do not measure MongoDB round trips.

Reference: [Go diagnostics](https://go.dev/doc/diagnostics),
[Docker stats](https://docs.docker.com/reference/cli/docker/container/stats/),
[AS6704T specifications](https://www.asustor.com/en/product/spec?p_id=77).
