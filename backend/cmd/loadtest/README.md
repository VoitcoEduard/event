# Deployment load test

The complete NAS benchmark/profiling workflow is in
[backend/perf/README.md](../../perf/README.md). The tool now uses a public
projector observer, sends the required Origin header, registers participants
for personal polling, checks failures/latency gates, and writes JSON reports.

This command simulates the project's actual WebSocket protocol: an observer plus a
chosen number of players join one module, then every player submits one answer
at the same moment. It reports connection latency and the time until the host
observes the answer burst.

Run it from a laptop, workstation, or another server -- never from the NAS
being measured. Use a disposable event/module because it creates real players
and answer signals in that session.

## Before running

1. Create a dedicated test quiz with a long active question.
2. Start that question from the normal host UI; its WebSocket state must be
   `question-active`.
3. Use the public Cloudflare hostname, not the NAS LAN address, so the result
   includes the tunnel and normal Internet path.
4. Open Portainer alongside the test and watch application CPU/RAM, PostgreSQL
   CPU/RAM, network throughput, and Docker restarts/log size.

## Run

From `backend/`:

```sh
go run ./cmd/loadtest \
  -ws-url wss://quiz.example.com/ws \
  -module TEST_MODULE_ID \
  -players 100 \
  -question 0 \
  -confirm I_UNDERSTAND_THIS_WRITES_TEST_ANSWERS
```

Increase the `-players` count in steps (25, 50, 75, 100, 125) and use a fresh
test session/question for every run. A successful run means all players join,
all answers are observed by the host, and the answer burst stays comfortably
below your responsiveness target (a good initial target is one second).

## Optional polling load

The frontend normally polls an event endpoint in addition to keeping the
WebSocket open. Add `-poll-url` to reproduce that traffic. `{id}` is replaced
with a different simulated player ID for each client, which avoids accidentally
measuring only one per-user cache key.

```sh
go run ./cmd/loadtest \
  -ws-url wss://quiz.example.com/ws \
  -module TEST_MODULE_ID \
  -players 100 \
  -poll-url 'https://quiz.example.com/events/TEST_EVENT_ID?userId={id}' \
  -confirm I_UNDERSTAND_THIS_WRITES_TEST_ANSWERS
```

Add `-media-url` to download an existing media item once per player. Compare warm
and fresh items separately because Cloudflare cache warmth affects image timings.
Use `-output /tmp/loadtest.json` to retain machine-readable results, `-hold 60s`
to cover cache expiry, and `-mode idle -require-active=false -hold 30m` for an idle
soak. Tests return a nonzero exit code on failure; use a fresh unanswered question
for every burst. For LAN testing set `-origin` to the app's PUBLIC_BASE_URL.
