# Initial bottleneck audit

These are measurements on an **Apple M1 Pro workstation**, Go 1.25.5,
darwin/arm64, taken October 28, 2026. They are not AS6704T measurements and
establish no safe production player count. Timings are a short 100 ms/case
screening run; repeat on the NAS with the kit before making capacity decisions.
Scoring algorithms and production handlers were not optimized for this audit.

## Measured growth

| Real code path / fixture | Result on workstation | Interpretation |
| --- | --- | --- |
| 150-answer burst, 0 past questions | 0.87 ms per whole burst | Handler CPU with no network/DB/log output |
| Same burst, 25 past questions | 3.42 ms | Cost grows with history |
| Same burst, 100 past questions | 10.58 ms | About 12x the first-question cost |
| Full state, 150 players, 0 past questions | 10,354 bytes, 0.039 ms encoding | Initial state is small |
| Full state, 150 players, 25 past questions | 690,129 bytes, 1.13 ms encoding | Roughly 104 MB if sent to 150 recipients |
| Full state, 150 players, 100 past questions | 2,733,955 bytes, 4.23 ms encoding | Roughly 410 MB if sent to 150 recipients |
| Simple Quiz full BSON, 150 players, 1 question | 55 KB, 0.30 ms encoding | Excludes DB write |
| Simple Quiz full BSON, 150 players, 100 questions | 4.30 MB, 19.68 ms encoding | Whole-document replacement per submission/grade grows expensive |
| Team award projection, 500 awards, 15 recipients | 1.26 ms, 4.79 MB allocated per call | Rebuilding every recipient entry creates garbage |
| Team award projection, 2,000 awards, 15 recipients | 7.22 ms, 22.92 MB allocated per call | Stress case, not assumed normal workload |
| One player total across 25 modules / 150 players | 0.027 ms, zero allocations | In-memory sum is relatively small; DB materialization is excluded |

Full-state sizes depend on the synthetic fixture. Actual questions, guesses,
IDs and history can change them. Network figures are uncompressed application
payload totals, not measured wire traffic. The server serializes each broadcast
once, but queues/sends that payload separately to each matching connection.

A separate two-second CPU profile of the late-game burst attributed about
**80% of samples to the duplicate-check loop** in `hub/hub_player.go`, including
struct copies and string equality. Routine logs were sent to a discard writer;
logging output and network are deliberately excluded. This is a confirmed
hotspot in that particular fixture, not 80% of production CPU.

## Priority list

1. **Full session broadcasts.** `hub/hub_host.go` broadcasts the whole session
   on results, leaderboard and finish; `hub/quiz_hub.go` marshals and fans it out
   while holding the shared hub lock. Session history remains part of the JSON.
   Test late-game payload sizes and tunnel throughput. A compact role-specific
   state/leaderboard message would address both encoding and transfer cost.
2. **History scans per answer.** `hub/hub_player.go:162` loops over every past
   signal. This is the largest CPU hotspot in the profiled burst. A per-player
   answered-question marker would make duplicate checks independent of history.
3. **Database work under the shared quiz lock.** `processHostEndQuiz` calls
   `saveQuizResults` while `ProcessMessage` holds `h.mu`. A slow final save can
   delay messages for every game, even when CPU is low. Capture mutex/block
   profiles while finishing one quiz and answering in another.
4. **Personal event polling rebuilds event-wide scores on cache miss.**
   `database/event_repository.go` calls `LoadEventScores`, which aggregates latest
   quiz runs, reads all event entries and team activities, and materializes team
   award entries before summing one player. The 15-second cache is per player;
   synchronized expiry can repeat the same event-wide work for many players.
   Test registered identities, realistic history, cache expiry and MongoDB CPU.
   Consider a shared event scoring snapshot if measurements confirm the cost.
5. **Simple Quiz persistence.** `hub/simple_quiz.go` replaces the full session
   document on join, submission and grading. Encoding alone grows with every
   stored answer, before database/storage cost. Benchmark a real later-game
   submission/grade burst separately before approving a capacity limit.
6. **Team projection and whole-activity updates.** Team awards replay to per-team
   and per-recipient entries; optimistic writes replace the activity document.
   Polling and multiple administrators can amplify allocations/retries. Measure
   realistic award history, then consider incremental projections if warranted.
7. **Upload conversion and cold media.** `utils/image.go` invokes cwebp and uses
   temporary files. That CPU belongs to a child process. `handlers/media.go`
   caches downloads in RAM (256 MB), but concurrent misses can still hit GridFS.
   Test an upload during play and a fresh-image download storm; distinguish app,
   MongoDB, tunnel and host/storage pressure.
8. **Small repeated costs.** Every answer scans all hub clients for observer
   delivery, serializes debug payload data even in release builds, and emits
   ordinary logs. These are secondary to the measured history loop at 100
   questions, but become visible in large bursts or across simultaneous games.

The likely first improvements are a compact state broadcast and constant-time
duplicate checks, followed by moving final persistence outside the shared lock.
Confirm polling/database costs on the AS6704T before choosing caching/index
changes. Avoid making a server-scoring migration responsible for unrelated
network/history growth.

## What remains to measure on the AS6704T

- Installed RAM, Docker resource limits, storage layout and background NAS load.
- Repeated LAN and tunnel runs at expected peak plus 1.5x, including late-game
  history and 30-minute connection/polling soak.
- Whole answer-burst delay, join/poll percentiles, missing answers, disconnects,
  memory trend, app/Mongo/tunnel CPU, disk I/O and restarts.
- A final save during a second game's answer burst; Simple Quiz grading/submission
  persistence; an upload during play. These UI-driven deployment scenarios are
  not automated by the regular-quiz load generator.
- Query plans for the event scoring aggregation. Existing `quiz_results` indexes
  cover event/module filtering but not the `finishedAt` sort in that aggregation;
  an explain on representative data is needed before choosing an index.

See [README.md](README.md) for build/run commands and limitations.
