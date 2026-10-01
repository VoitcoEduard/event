# Teams module

Implemented as an event activity for manually scoring games played outside the platform.

## Running an activity

1. Add a **Teams** module to an event. Configure new teams or an earlier Teams activity, formation method, count/target size, capacity, and award presets.
2. Open **Admin** for the module. Initialize it, or explicitly import legacy event teams. Importing/reusing teams does not enter absent members or copy scores.
3. Activate the module in the event so it appears on player phones and the projector. Players enter the activity using their existing device identity. Admin can also add event participants.
4. Form teams during setup. Random distribution preserves assignments; reshuffle explicitly replaces them. Team names, colors, and capacities can be edited in the admin interface.
5. **Start activity** locks player selection and enables awards. Admin can move people or add late arrivals. Each member receives the full award.
6. **Complete activity** preserves scores and standings. Reopen explicitly for corrections. The event Data view includes individual contributions, team standings, members, award history, and separate manual adjustments.

## Scoring and persistence

- One `teams_activities` MongoDB document per event/module stores configuration, entrants, reusable source invitations, teams, awards, lifecycle, and revision. A unique event/module index prevents duplicate runtimes.
- Compare-and-swap mutations revalidate lifecycle, roster, and capacity against the committed revision. This requires no MongoDB replica set or multi-document transactions.
- An award stores the exact participant IDs receiving it, admin identity, reason, timestamp, and request ID. Repeated requests return the existing contribution; conflicting request reuse fails.
- Undo appends one opposite contribution for the original recipients. Moving a person, disconnecting, completing, or reopening never changes past awards.
- Event scoring derives one team entry per award and one individual entry per recipient. These entries are never duplicated into `event_scores`. Award entry IDs are deterministic.
- Scored modules and teams cannot be deleted. Unscored module deletion writes a tombstone through the same revision boundary as scoring, preventing a concurrent first award from disappearing.
- Whole awards/penalties are nonzero and limited to ±1,000,000 points; negative totals are supported. Up to 200 teams and 12 presets are supported.
- Legacy manual team points remain team-only because original recipients are unknown. Live device IDs are preserved and preferred over the older participant collection; matching by display name is never used.

## API

Base: `/events/{eventID}/modules/{moduleID}/teams`.

| Method/path | Access | Purpose |
| --- | --- | --- |
| GET `/state` | Player device header optional | Public activity state and caller's personal totals |
| GET `/projector` | Public | Display-safe standings and awards |
| GET `/admin` | Admin session | Runtime, participant candidates, and legacy teams |
| GET `/stream` | Public | SSE change notices and reconnect signal |
| POST `/join` | Player device header | Enter/reconnect with `{name}` |
| POST `/choose` | Player device header | Select own team with `{teamId}`; empty ID leaves during setup |
| POST `/actions` | Admin session + CSRF | Initialize/configure, save/delete team, add/assign participant, distribute, start/complete/reopen, award, undo |

The `X-Teams-Player` header uses the app's existing unguessable device user ID. Public rosters expose names, not participant/device IDs, profile fields, recipient IDs, or admin audit details. Administrative routes retain the app's session and CSRF checks. Player-controlled assignment is restricted to the caller during setup with choice formation.

For award/undo actions, clients retain a `requestId` across ambiguous failures. Notifications contain event ID, module ID, and revision only. Clients fetch authoritative state, queue a second read if a notice arrives during a fetch, and poll every three seconds as fallback. Initial/reconnect award history never replays celebrations.

## Verification

From `backend`:

```sh
GOCACHE=/private/tmp/teams-go-cache go test -race ./...
GOCACHE=/private/tmp/teams-go-cache go test -tags integration -run 'TestTeamsActivityHTTPAndPersistence|TestEventScoringLatestAndManual' . ./integration/eventscoring
```

Integration tests require Docker and create isolated MongoDB containers.

From `frontend`:

```sh
npm test
npx vitest run tests/teamsActivity.spec.js tests/teamsActivityPolling.spec.js --environment jsdom
npm run build
```

Covered cases include balancing/capacity, roster reuse, full individual awards, retries, original-recipient reversals, concurrent membership/award/completion/deletion, HTTP access boundaries, legacy import, durable reload, event totals, component states, and reconnect delivery. Visual verification used a separate local fixture database at 390px mobile, desktop admin, and 1920×1080 projector sizes with ten teams and long names.
