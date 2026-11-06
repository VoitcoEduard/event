# Plan to fix all 27 confirmed findings

Prepared 5 November 2026 against commit `5a014509`. Implementation began 5 November and the changes are described in the commit that accompanies this plan. The delivery and rollout gates below remain useful for deployment.

Scope: the 27 confirmed findings in [CLAIM_VERIFICATION.md](</Users/eduard/go_server v8/CLAIM_VERIFICATION.md>). Original claim numbers are retained. Partial and rejected claims are outside this plan except where a dependency must be addressed to complete a confirmed fix.

## Implementation and verification status

The eight code batches are implemented in this change. Quiz answers are scored and acknowledged by the server, public quiz and Simple Quiz views filter private data, participant linking requires a secret, and published Simple Quiz snapshots contribute to event scores. The frontend handles reconnects, stale asynchronous work, and the listed edge cases. The backend query, cache, error, and port fixes are included.

Automated verification: frontend tests and production build pass; backend short tests pass under the race detector; `go vet ./...` passes; PostgreSQL integration tests compile. The PostgreSQL integration suite, live browser game walkthroughs, database migration on an existing copy, and deployment gates below still require an environment with Docker/PostgreSQL and a running application. This workspace has no working Docker daemon or PostgreSQL executable, so those runtime gates could not be exercised here. Existing participant rows without a join secret cannot prove ownership automatically; the event screen registers a new identity after an ownership conflict and informs the player. Historical scores remain under the old identity until an administrator reviews any intended merge.

## Intended behavior and defaults

- Public players/projectors receive only the question, public presentation state, permitted aggregate statistics, and deliberately published results. Administrator-only details require server-side authorization. A player's private answers require that player's reconnect credential.
- Correct answers, numeric targets, and Wrong Pictures target coordinates remain server-side until the question's explicit reveal/results phase. Public preload responses may carry media references but no hidden answer information. Future questions remain unrevealed.
- The server determines answer acceptance, elapsed scoring time, correctness, streaks, and awarded points. Existing formulas, defaults, grace-period behavior, one-answer rules, and integer rounding remain the baseline. Any unavoidable timing difference from removing browser clock drift is documented and tested.
- Simple Quiz contributes its latest explicitly published snapshot to event totals. Grading edits affect totals only after Update Published. Reset removes that module's contribution; contributions remain after completion/restart; repeated reads never add points again. These defaults match the existing mutable session/reset model and do not introduce historical-run accumulation.
- Reconnecting does not award points based on an untrusted offline timestamp. Answers arriving after the permitted server deadline/grace period get an explicit outcome. An answer awaiting acknowledgement is shown as pending, rather than receiving a permanent local score.
- Equal minimum/maximum ranges are supported as a fixed-value range. Invalid/reversed/nonfinite ranges are rejected on save and rendered safely if encountered in legacy data.

## Delivery order

Each batch is a separate reviewable change, with its own relevant validation. Batches 1–2 are the first release candidates. Batch 3 needs a coordinated server/frontend release; batch 5 builds on its answer acknowledgements. Batch 6 implements optimized score reads after batch 4 defines Simple Quiz contributions. UI/resource work can be developed independently, but delivery remains incremental.

### 1. Restrict private quiz views — #1, #3, #4

**Work**

- Require an active administrator session for Simple Quiz host state and host results. An unauthenticated `view=host` must return an authorization error, never select a privileged projection by query string alone.
- Require the matching Simple Quiz player ID and secret for personal state/results. Send secrets in a request header or suitable private credential channel, not URL parameters. Keep intentionally published leaderboard data accessible separately.
- Define explicit host, player, and projector response shapes for HTTP and WebSockets. Remove raw QuizSession broadcasts to anonymous clients, including joins, reconnects, results/leaderboard broadcasts, and UpdateSession paths. Projectors retain aggregate counts and the configured public leaderboard, not individual answer history/private IDs.
- Preserve judge-token behavior and verify that judge-only data does not appear in other projections. Set private credential-bearing responses to no-store and keep credentials out of request logs/cache keys that could be exposed.
- Immediately remove raw playerSignals from public sockets. Finish removal of hidden answer fields from question batches together with batch 3, because current browser scoring depends on those fields. Do not mark #3 fully complete while batches still expose answers.

**Main files**: `backend/handlers/simple_quiz.go`, `simple_quiz_view.go`, `backend/hub/hub_projector.go`, `hub_player.go`, `hub_host.go`, `quiz_hub.go`, `frontend/src/services/simpleQuiz.js`, related player/projector views.

**Acceptance**: route-level anonymous/admin/judge/player credential matrix; wrong-player and wrong-module secrets fail; host controls still work; projector/player messages contain no other player's private answers. Exercise real registered routes, not only the projection helper. Reuse the disclosure reproductions as regression tests with the expectations inverted.

### 2. Stop player-flow failures and next-question corruption — #15, #16, #39

**Work**

- Separate authoritative server-state synchronization from local player actions. A valid server snapshot can restore lobby, presenting, active, results, leaderboard, or finished directly; local submit still obeys a constrained transition table.
- Apply state with a public opaque run identifier and question identity/revision. Ignore obsolete responses, preserve already-answered state when resynchronizing the same question, and reset timers/drafts only for a real question/run change. Reconnecting to active must hydrate the correct question and deadline even if presenting was missed.
- In Simple Quiz submit, capture identity, question index, draft key, and answer ID before awaiting. Clear only the submitted question's matching draft. Apply `submitted` and clear the visible editor only if that request still belongs to the visible question. Do not let an old completion clear a new pending answer/request or a newer edited draft.
- Add one shared UUID generator using randomUUID when available and getRandomValues otherwise. Keep existing stored IDs; use the helper from both shared identity and Simple Quiz answer creation. Make storage failures recoverable rather than preventing screen setup.

**Main files**: `frontend/src/composables/useQuizStateMachine.js`, `views/QuizPlayerScreen.vue`, `views/SimpleQuizPlayerScreen.vue`, `utils/index.js`, `services/simpleQuiz.js`; server state/reconnect payload builders for the run/question identifiers.

**Acceptance**: mid-question join/reconnect, results-to-next-question, missed presenting, reset/new run, repeated same-state messages, and out-of-order responses behave correctly. A deferred Q0 submit resolving after Q1 begins leaves Q1's draft and submitted flag intact. UUID generation works when randomUUID is absent on LAN HTTP and preserves valid UUID syntax.

### 3. Move scoring to the server, then remove answer leaks — #21, #38, #55; completes #3

**Work**

- Implement Go scoring for classic/default choice, true/false, multi-select, correct-picture, number, range, and Wrong Pictures. Use existing scoring behavior as fixtures: configured maxPoints/defaults, time decay, streak updates/reset, exact numeric/range answers including zero tolerance, aspect-correct geometry, one-to-one target matching, and integer remainder allocation.
- Store server-owned streak/scoring state per run/player. Derive elapsed time and grace acceptance from the actual active-question window and server receipt time. Ignore browser-supplied score, correctness, streak, target ID/match/distance, aspect ratio, and elapsed scoring time.
- Snapshot Wrong Pictures image dimensions from trusted media metadata when a run starts. Handle legacy missing dimensions through server-side metadata repair/image inspection before the run can score; never silently substitute a player-supplied aspect ratio.
- Add an answer submission ID and a private acknowledgement with run/question identity, accepted/rejected/pending outcome, awarded points, and authoritative total. Duplicate submissions return the original outcome without scoring twice; preserve enough per-run outcome information to acknowledge replay after reconnect. Reveal target geometry/correct answers only when the question's server-controlled result policy permits it.
- Remove browser correctness/point calculation from production submission paths. Treat any feedback before acknowledgement as pending. Build post-reveal review from permitted server data rather than complete answer keys downloaded in advance.
- Replace public question serialization with an allowlist projection across event/module APIs, templates where public, HTTP quiz state/results/preload APIs, and every WebSocket batch/broadcast. Cover both quiz and wrong_pictures module types. Strip correctIndex, correctIndices, correctNumber, correctValue, hidden target geometry, notes, and nested equivalents until authorized/revealed; do not mutate stored/admin templates while projecting.
- Guard remaining image onload work by question/image generation and unmount state. Older image loads must not alter current presentation or review dimensions. This can land as a small early fix while server scoring is implemented.
- Use server-provided serverNow/start/deadline metadata to render remaining time from elapsed time, not interval tick counts. Synchronize offset on state/reconnect; recompute on visibility/focus. Presentation, answer, and explanation timers share this approach. The server remains authoritative if browser time changes.

**Main files**: new backend scoring module; `backend/hub/hub_player.go`, `quiz_timer.go`, `models/quiz.go`, `handlers/public_data.go` and all question-delivery paths; `frontend/src/views/QuizPlayerScreen.vue`, `QuizHostScreen.vue`, `ProjectorView.vue` and shared timing utilities.

**Acceptance**: backend scoring fixtures cover all supported types, wrong answers, malformed/nonfinite/out-of-range inputs, duplicate/replayed submissions, streak reset, and grace/deadline boundaries. Forged client points do not affect totals. Inspect serialized HTTP and socket payloads for answer-bearing fields before reveal and for future questions after reveal. Slow old image loads cannot change the current question. Simulated suspended tabs recover the correct countdown immediately. Complete host/player/projector games for every question type using only sanitized payloads.

**Release constraint**: the scorer, acknowledgement handling, and sanitized frontend ship together after compatibility tests. Use an explicit protocol version; an incompatible old tab receives a clear reload prompt rather than a hidden fallback that trusts client scores. Schedule cutover between active games. Rollback must target a release that preserves authorization/public filtering; do not restore exposed answers merely to keep an old frontend running.

### 4. Include Simple Quiz in event scoring — #2

**Work**

- Add a persisted event-scoped participant reference to Simple Quiz players while retaining their existing player IDs and reconnect secrets. New joins from an event must bind to that event's participant identity through a verifiable registration/reconnect credential; a freely supplied ID or nickname is not ownership proof. Add the minimal event enrollment/reconnect credential if none exists.
- Use an additive migration and transactional join/link behavior. Already issued Simple Quiz secrets continue reconnecting the same Simple Quiz identity. Do not merge legacy players into event participants by matching names. Preserve unlinked historical players as separate, clearly labeled score rows; provide explicit administrator-assisted linking for an intended merge and recompute contributions after linking.
- Query the stored Published snapshot, not working grades or the mutable player.Score field. Adapt it into one module contribution in the event score read model. Avoid parallel ledger writes that can drift from the session snapshot or be counted twice.
- Update LoadEventScores, PlayerTotal, participant status, and frontend score tables/module classification so game, manual, and activity points combine consistently. Keep any existing manual-score editing semantics explicit rather than accidentally enabling editing of published Simple Quiz grades from another endpoint.
- Invalidate relevant event/participant score response caches on publish, update-published, reset, link changes, and module deletion. Load already published snapshots automatically after deployment; no manual republish is required for compatible stored results.

**Main files**: `backend/models/simple_quiz.go`, `postgres/simplequiz.go`, a new additive migration, `handlers/simple_quiz.go`, `database/event_scoring.go`, `database/repositories.go`, participant join/reconnect boundary, `frontend/src/utils/eventScores.js` and Simple Quiz join UI/API.

**Acceptance**: PostgreSQL integration tests for publish, correction without republish, update-published, completion, restart, reset, deleted module, repeated publication/reads, and mixed module totals. No working scores appear early; latest published version replaces the prior contribution. A linked participant has one event row; legacy unlinked scores remain visible without speculative merging. Failed persistence changes no visible total. Test unauthorized cross-event/other-player linking.

### 5. Make reconnect and offline submission reliable — #12, #13

**Depends on**: batch 2 run/question synchronization and batch 3 submission IDs/acknowledgements.

**Work**

- Retry unexpected disconnections continuously using capped exponential backoff and jitter. Add an online-event retry and a visible connection/pending-answer indicator. Intentional navigation/disconnect and authorization rejection stop retrying; stale socket callbacks cannot reconnect an abandoned session.
- Replace the ten-second drop rule with an acknowledged pending queue scoped to player, module, run, question, and answer ID. Sending alone does not dequeue an item. Retry with the same ID; bound pending entries to the current run and explicitly settle obsolete/rejected entries.
- On reconnect, restore membership/run/question state before replay. Reconcile stored outcomes first; resend eligible unanswered submissions and receive explicit deadline/wrong-run rejection when necessary. Never convert a rejected offline answer into earned points using its local timestamp.
- Keep pending submissions through transient disconnects; deliberately clear them on leaving the run/identity, with an appropriate visible outcome. Page-reload persistence is not required for this scoped fix; a reload must still reconcile already accepted server answers.

**Main files**: `frontend/src/services/websocket.js`, `utils/signalQueue.js`, player status UI, backend answer/reconnect acknowledgement paths.

**Acceptance**: outages longer than five retries recover automatically; no overlapping retry chains; auth rejection stops retrying. Verify answer sent before disconnect but acknowledgement lost, answer unsent for more than ten seconds during a long question, server deadline passed while offline, new run/module, and duplicate delivery. Every answer ends accepted once or visibly rejected; client total matches server total.

### 6. Bound backend resource use and improve scoring reads/errors — #19, #20, #24, #29, #50

**Work**

- #19: bulk-load modules for the selected event IDs in one additional query, preserving event/module ordering. A database request per event must disappear.
- #20: bulk-load all players for the selected quiz runs in one query. Add a participant-only total query/read path that includes the exact same latest-game, published Simple Quiz, manual, and activity contribution rules as LoadEventScores. Avoid loading the full event and every player merely to display one participant's number.
- Add indexes only where query plans/representative fixture measurements show they are needed. Prefer set-based SQL over an additional cross-request score cache until profiling justifies one.
- #24: translate known CHECK-constraint errors into a validation-domain error and a safe 400/422 response. Use named constraints/explicit mappings for user-correctable failures; retain logs and internal-error handling for unexpected invariant failures. Do not expose SQL or raw database error details.
- #29: replace the permanently growing per-module mutex map with a fixed-size set of hash-selected mutexes. Every read/restore/set/update/delete for a module selects the same mutex. Do not delete a mutex entry while waiters might still hold it; that could introduce two independent locks for one module. Check collision contention before choosing the final stripe count.
- #50: add an idempotent cache Close lifecycle that stops and joins the janitor. Give handler caches an explicit owner, close them on shutdown, and register cleanup for dynamically created/test caches. Preserve safe concurrent use and the documented byte-ownership contract.

**Main files**: `backend/postgres/events.go`, `scoring.go`, `errors.go`, `database/event_scoring.go`, `repositories.go`, `hub/simple_quiz.go`, `cache/cache.go`, handler dependency construction and `main.go`.

**Acceptance**: query count is bounded as event/run counts increase; participant totals agree with full event totals on mixed fixtures including Simple Quiz corrections/reset. Inspect representative PostgreSQL query plans. Concurrent same-module operations remain serialized under the race detector; distinct-ID churn does not grow lock count. Cache Close is repeatable, stops its worker deterministically, and is wired to owner shutdown. Known invalid scoring inputs return actionable safe errors; unexpected failures are still logged.

### 7. Correct asynchronous UI cleanup and edge cases — #31, #32, #33, #34, #35, #57, #58, #68

**Work**

- #31: use a single in-flight event refresh or chained polling after completion, AbortController on unmount/event change, and a response generation check before applying state. Clear delayed completion callbacks when their screen ends.
- #32: consolidate drag cleanup in an idempotent endDrag invoked on mouseup/touchend, touchcancel, blur, and unmount. Remove the exact installed listener references and options. Ensure repeated drags do not accumulate listeners.
- #33: track leaderboard requestAnimationFrame IDs; cancel on state exit/unmount and replace prior animation generations on reentry. Final displayed values come from the current authoritative scores.
- #34: handle empty image sets and zero/nonfinite track dimensions before carousel division/emission. Clamp a valid index when images change and disable submission with no valid option. Recalculate after layout becomes measurable; guard the actual submit handler as well as the button.
- #35: return a single finite tick and fixed value for min==max, disable meaningless dragging, and reject reversed/nonfinite configurations. Keep normal positive ranges unchanged.
- #57: disable map placement until an image is loaded and has nonzero measurable bounds, guard image-error/dimension paths, and track/cancel the hint timeout on replacement/unmount. Show loading/failure state rather than silently ignoring an enabled action.
- #58: own reveal timeouts by run/question/state generation and cancel them on state exit, question change, reset, and unmount. Cover both range host and non-WrongPictures projector paths; preserve the already tracked Wrong Pictures sequence.
- #68: safely parse and validate stored currentEvent data, clearing only invalid event data and falling back to the URL event ID or event-entry screen. Handle unavailable storage without throwing during setup.

**Main files**: `frontend/src/views/EventScreen.vue`, `QuizHostScreen.vue`, `ProjectorView.vue`; `components/RangeComponent.vue`, `GuessMarker.vue`, `HorizontalCarousel.vue`, `MapCoordinatePicker.vue`.

**Acceptance**: mount/unmount component tests with deferred requests, fake timers, rAF control, and instrumented global listeners. Test unmount during drag, rapid leaderboard/results/question switching, slow/stale polling, image failure/delayed load, empty/resized carousel, equal/reversed range, malformed JSON, structurally invalid stored objects, and denied storage. No stale callback changes current screen state and no invalid selection/value reaches submission.

### 8. Fix session retry/navigation and server configuration — #56, #40

**Work**

- #56: mark a session load complete only after a successful parsed/validated response or a definitive 401. Transient HTTP, network, and malformed-body failures remain retryable and do not become a cached signed-out state. Deduplicate concurrent loads or sequence them so an old response cannot overwrite newer login/logout state.
- Preserve a validated internal return path through login and pending-account flows. Support both development login and the Google OAuth round trip; carry the destination in the existing server-held login transaction/one-time state, not an unrestricted redirect URL. Default to /admin if missing/invalid. Continue enforcing current status/role checks on the destination.
- #40: parse PORT with a documented default of 8080; reject invalid/non-numeric/out-of-range configured values with a clear startup error. Build the HTTP listener address from the validated value. Keep existing Compose mappings and health checks aligned; changing a container's internal port still requires the corresponding mapping change.

**Main files**: `frontend/src/session.js`, `router/index.js`, `views/LoginScreen.vue` and pending UI; `backend/auth/service.go`, login transaction storage and an additive migration if needed; `backend/main.go`, environment documentation.

**Acceptance**: 503/network/invalid-JSON session failure followed by a successful retry; definitive 401; concurrent login/logout/load. Protected deep links survive both login modes and pending approval. External/protocol-relative/malformed return targets cannot redirect outside the application. Backend starts on a configured free test port, defaults to 8080, and rejects invalid ports.

## Coverage checklist

| Batch | Confirmed claim IDs | Count |
|---|---|---:|
| 1 — Private views | #1, #3, #4 (#3 completed in batch 3) | 3 |
| 2 — Player flow | #15, #16, #39 | 3 |
| 3 — Server scoring/public answers/time | #21, #38, #55 | 3 |
| 4 — Simple Quiz event totals | #2 | 1 |
| 5 — Reconnect/answer delivery | #12, #13 | 2 |
| 6 — Backend queries/resources/errors | #19, #20, #24, #29, #50 | 5 |
| 7 — UI lifecycle/edge cases | #31, #32, #33, #34, #35, #57, #58, #68 | 8 |
| 8 — Session/navigation/port | #56, #40 | 2 |
| **Total unique confirmed findings** | | **27** |

## Validation and rollout gates

1. Before each fix, turn the relevant verified reproduction into a focused behavioral regression. For security, scoring, and asynchronous lifecycle changes, test the externally observable behavior rather than checking for source-code strings. Wire component tests into the test command/CI, not only a manually invoked script. Small port/storage changes need focused checks, not a new broad testing framework.
2. Run relevant frontend tests and production build, backend unit/race tests, and vet/format checks for touched code. Run PostgreSQL integration tests for authorization, participant binding, score projection/migrations, and query/error behavior; these were not exercised by the original verification. Match the existing CI commands: `npm test`, `npm run build`, `go test -race -short ./...`, `go test -race -p 2 -tags integration -count=1 ./...`, and `go vet ./...` from their respective directories. Docker is needed for the existing integration fixtures.
3. Exercise actual HTTP/WebSocket flows: host starts, player joins, player answers, host advances/reveals, projector renders, player reconnects, quiz finishes, results persist and appear exactly once in event totals. Include a long offline interval, rapid question advance, and a hidden/throttled browser tab. Verify all supported question types and Simple Quiz judge/publication flow.
4. Migrate a copy of an existing database as well as a fresh database. Preserve old published snapshots and reconnect identities; legacy ambiguous identities remain separate until explicitly linked. Check event totals before/after migration and explain each intended new Simple Quiz contribution. Avoid destructive down migrations for rollback.
5. Deploy the protocol/scoring change between games, with reload guidance for open old tabs. Run image/build/fresh-install/recreation/restore smoke checks when migrations or image/runtime configuration change. Do not enable a partially upgraded client/server protocol in a live game.
6. Observe authorization failures, answer acknowledgements/rejections, reconnect recovery, score agreement, query counts, and bounded resource counts on staging. Keep logs free of secrets/raw private answers. Production deployment is a later concrete action; this plan does not deploy anything.
7. Finish by rerunning the original 27 checks and updating CLAIM_VERIFICATION.md with the fixing change and validation evidence for each. #3 and #21 close only once every public answer-delivery path is sanitized and the complete game still functions without those hidden fields.
