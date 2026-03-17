# Event Module Orchestration Architecture

## Purpose

This document describes the recommended top-level architecture for event-only multi-module experiences where player identity, shared data, and scoring persist across the entire event.

It is intended as a future reference for implementation planning.

---

## Problem Statement

In events, the platform needs to support a a combination of difren modules in different order.

The important requirement is that participant identity must persist across all modules so the system can:

- keep the same user name throughout the whole event
- track cumulative score per user
- support team assignment and team-based rounds
- allow one module to generate data needed by another module
- let admins configure these relationships explicitly

This is an event-specific orchestration requirement, not a generic requirement for every product mode.

---

## Current Situation

The current event model in [`backend/models/event.go`](../backend/models/event.go) stores an [`Event`](../backend/models/event.go#L68) with a flat [`Modules`](../backend/models/event.go#L75) array and generic module [`Data`](../backend/models/event.go#L20).

That is enough for isolated modules, but not enough for:

- persistent participant identity across modules
- shared runtime state across modules
- explicit module dependencies
- cumulative scoring across the event
- end-of-event leaderboard and prizes

So the architecture must evolve from a simple “event contains modules” model into an “event orchestrates a shared runtime” model.

---

## Architectural Goal

For events, treat the event as a container for a coordinated participant journey.

The event should provide:

1. an ordered module flow
2. a participant registry
3. a shared event data layer
4. per-module runtime state
5. cumulative scoring


In short:

- the event defines the experience
- the event runtime coordinates the modules
- the participant identity persists across modules
- modules exchange data through explicit bindings
- scores are accumulated at event level

---

## Core Domain Concepts

### 1. Event Definition

This is the admin-authored configuration.

It should contain:

- event metadata
- ordered module flow
- participant policy
- scoring policy
- reward policy
- branding / presentation configuration

This is close to the current [`Event`](../backend/models/event.go#L68), but conceptually it should be treated as configuration rather than the live runtime.

### 2. Event Session

This is the live runtime execution of an event.

It should contain:

- current event status
- active module
- joined participants
- shared event state
- module runtime states
- score entries or leaderboard snapshots

This should be a separate runtime concept even if the first implementation keeps some state near the event.

### 3. Participant

A participant is the persistent identity inside one event session.

This is the key entity required for cross-module continuity.

Each participant joins once and is reused throughout the whole event.

### 4. Team

If the event supports team games, teams should be first-class entities, not ad hoc attributes on participants.

### 5. Module Runtime State

Each configured module should also have a live runtime state.

This tracks whether the module is pending, active, completed, or blocked, along with outputs it generates during execution.

### 6. Shared Event State

This is the controlled storage layer used for cross-module data sharing.

This is where a module writes outputs that another module can read.

### 7. Score Entries

Scores should be stored as append-only entries rather than only a mutable running total.

This makes aggregation, ranking, and auditing easier.

---

## Recommended Type Model

Below is a high-level starter model for future implementation.

### Participant

```ts
Participant {
  id: string
  eventSessionId: string
  joinKey: string
  displayName: string
  normalizedName: string
  teamId?: string
  profile: Record<string, any>
  status: 'joined' | 'active' | 'left' | 'disqualified'
  createdAt: Date
  updatedAt: Date
}
```

#### Notes

- [`displayName`](../plans/event_module_orchestration_architecture.md#participant) is the visible player name
- [`normalizedName`](../plans/event_module_orchestration_architecture.md#participant) helps search or deduplication
- [`joinKey`](../plans/event_module_orchestration_architecture.md#participant) supports reconnect logic
- [`profile`](../plans/event_module_orchestration_architecture.md#participant) stores mapped fields from forms

### Team

```ts
Team {
  id: string
  eventSessionId: string
  name: string
  color?: string
  memberParticipantIds: string[]
  metadata: Record<string, any>
}
```

### EventSession

```ts
EventSession {
  id: string
  eventId: string
  status: 'draft' | 'waiting' | 'live' | 'paused' | 'completed'
  activeModuleId?: string
  moduleOrder: string[]
  participants: Participant[]
  teams: Team[]
  sharedState: Record<string, any>
  scoreBoard: ScoreEntry[]
  moduleStates: ModuleRuntimeState[]
  startedAt?: Date
  endedAt?: Date
}
```

### ModuleRuntimeState

```ts
ModuleRuntimeState {
  moduleId: string
  status: 'pending' | 'ready' | 'active' | 'completed' | 'failed' | 'skipped'
  startedAt?: Date
  endedAt?: Date
  outputs: Record<string, any>
  participantProgress: Record<string, ModuleParticipantState>
}
```

### ModuleParticipantState

```ts
ModuleParticipantState {
  participantId: string
  status: 'not_started' | 'in_progress' | 'submitted' | 'completed'
  scoreDelta?: number
  resultData?: Record<string, any>
}
```

### ScoreEntry

```ts
ScoreEntry {
  id: string
  eventSessionId: string
  participantId?: string
  teamId?: string
  moduleId: string
  category: 'base' | 'bonus' | 'penalty' | 'adjustment'
  points: number
  reason?: string
  createdAt: Date
}
```

### LeaderboardEntry

```ts
LeaderboardEntry {
  participantId?: string
  teamId?: string
  totalScore: number
  moduleBreakdown: Record<string, number>
  rank: number
}
```

---

## Shared Data Model

Cross-module data sharing should not rely on one module directly reading another module’s internal storage.

Instead, the event runtime should use explicit shared state entries.

### SharedStateValue

```ts
SharedStateValue {
  key: string
  scope: 'event' | 'participant' | 'team' | 'module'
  producerModuleId?: string
  consumerModuleIds?: string[]
  valueType: 'string' | 'number' | 'boolean' | 'object' | 'array'
  value: any
  updatedAt: Date
}
```

### Example keys

- `participant.displayName`
- `participant.company`
- `participant.teamId`
- `event.secretWord`
- `module.welcome.validatedParticipants`
- `team.assignment.round1`

### Sharing principle

- one module produces output
- output is persisted into shared state
- downstream modules declare which shared keys they consume
- admin configures these connections

This makes data sharing understandable and maintainable.

---

## Module Contracts

To support orchestration, modules need explicit contracts.

The current generic [`Module`](../backend/models/event.go#L15) with [`Data`](../backend/models/event.go#L20) should remain flexible, but future architecture should add formal orchestration metadata.

### EventModuleDefinition

```ts
EventModuleDefinition {
  id: string
  type: 'form' | 'quiz' | 'wrong_pictures' | 'team_game' | string
  name: string
  config: Record<string, any>
  sequenceIndex: number
  identityBehavior: IdentityBehavior
  inputs: ModuleInputBinding[]
  outputs: ModuleOutputDefinition[]
  scoringPolicy?: ScoringPolicy
  activationRules?: ActivationRule[]
}
```

### IdentityBehavior

```ts
IdentityBehavior {
  requiresParticipant: boolean
  canCreateParticipant: boolean
  canUpdateParticipantProfile: boolean
  canAssignTeam: boolean
}
```

### ModuleInputBinding

```ts
ModuleInputBinding {
  targetKey: string
  sourceType: 'shared_state' | 'participant_profile' | 'team' | 'module_output' | 'static'
  sourceKey: string
  required: boolean
}
```

### ModuleOutputDefinition

```ts
ModuleOutputDefinition {
  key: string
  scope: 'participant' | 'team' | 'event' | 'module'
  valueType: 'string' | 'number' | 'boolean' | 'object' | 'array'
  persist: boolean
}
```

### Why these contracts matter

They allow scenarios like:

- welcome form creates participant profile
- another module reads participant display name
- team assignment module outputs `participant.teamId`
- team game consumes `participant.teamId`
- final reward module reads cumulative score totals

---

## Welcome Form as Registration Module

The welcome form should become the canonical participant registration module.

Its role is not only to collect a submission, but to map selected fields into participant identity and profile data.

### FormFieldBinding

```ts
FormFieldBinding {
  formFieldId: string
  targetType: 'participant.displayName' | 'participant.profile' | 'team.name' | 'shared_state'
  targetKey?: string
  required: boolean
}
```

### Example

- field `name` -> `participant.displayName`
- field `company` -> `participant.profile.company`
- field `nickname` -> `participant.profile.nickname`

This is how the participant becomes persistent for all later modules.

---

## Participant Policy

Because identity persistence is central to this feature, the event should define explicit participant rules.

### ParticipantPolicy

```ts
ParticipantPolicy {
  identityMode: 'anonymous' | 'named' | 'named_unique' | 'external_id'
  requiredFields: string[]
  dedupeStrategy: 'none' | 'name' | 'email' | 'phone' | 'external_id'
  reconnectStrategy: 'device_session' | 'magic_code' | 'name_plus_pin'
}
```

### Recommendation

Start with event-scoped identity rather than global platform accounts.

That keeps the player flow simple while still making names persistent inside one event.

---

## Scoring Policy

Scoring should be managed at event level, not only inside each module.

### EventScoringPolicy

```ts
EventScoringPolicy {
  aggregation: 'sum' | 'weighted_sum' | 'best_of_modules'
  moduleWeights: Record<string, number>
  rankingScope: 'individual' | 'team' | 'both'
  tieBreakerRules: string[]
}
```

### Principle

- modules produce score entries
- event scoring policy aggregates them
- leaderboard is derived from aggregated entries

This supports final rankings and prize decisions cleanly.

---

## Reward Policy

If the product needs prize logic, that should also be event-level configuration.

### RewardPolicy

```ts
RewardPolicy {
  rewardBasis: 'final_rank' | 'score_threshold' | 'module_achievement'
  rewards: RewardDefinition[]
}
```

This allows rules such as:

- top 3 participants receive prizes
- highest team score wins
- any player above a threshold gets a reward

---

## Cross-Module Dependencies

Modules may depend on:

- participant registration existing
- team assignment existing
- another module being completed
- a specific shared value being available

These dependencies should be explicit.

### ActivationRule

```ts
ActivationRule {
  type: 'module_completed' | 'shared_value_exists' | 'participant_field_exists' | 'team_assigned'
  sourceModuleId?: string
  key?: string
  operator?: 'exists' | 'equals' | 'not_equals' | 'in'
  value?: any
}
```

### Examples

- team game requires `participant.teamId`
- final game requires welcome module completed
- bonus module requires `event.secretWord`

This is preferable to hidden special-case logic.

---

## Admin Configuration Model

Admins should configure the event as a flow rather than as a flat unordered list.

The current module list in [`frontend/src/views/AdminEventDetailsScreen.vue`](../frontend/src/views/AdminEventDetailsScreen.vue) is a good starting point, but future event orchestration needs a richer configuration model.

### EventFlowConfig

```ts
EventFlowConfig {
  modules: EventModuleDefinition[]
  participantPolicy: ParticipantPolicy
  scoringPolicy: EventScoringPolicy
  rewardPolicy?: RewardPolicy
}
```

### Minimum admin controls

The admin should eventually be able to configure:

1. module order
2. activation prerequisites
3. participant identity rules
4. score aggregation rules
5. module input/output mappings
6. reward or prize rules

---

## Persistence Recommendation

For implementation, persist these concerns separately:

### 1. Event definition

Long-lived admin-authored configuration.

### 2. Event session runtime

Live mutable state for the running event.

### 3. Event participants

Persistent participant identities within the event session.

### 4. Module attempts / submissions

Per participant or per team outputs from each module.

### 5. Score entries

Append-only scoring records.

### Suggested collections or tables

- `events`
- `event_sessions`
- `event_participants`
- `module_attempts`
- `event_scores`

This can be introduced incrementally even if the existing [`Event`](../backend/models/event.go#L68) stays as the primary definition object for now.

---

## Runtime Flow

Recommended event runtime progression:

1. event waiting for participants
2. welcome form or registration module
3. first module active
4. next modules activated in sequence
5. scores accumulated throughout runtime
6. event completed
7. final leaderboard and reward outcome generated

Recommended participant lifecycle:

1. joined
2. registered
3. active
4. completed or left

---

## Architectural Boundaries

To keep the system clean, future implementation should separate these type groups.

### Authoring types

- event definition
- event module definition
- participant policy
- scoring policy
- reward policy

### Runtime types

- event session
- participant
- team
- module runtime state
- shared state

### Result types

- module attempt
- score entry
- leaderboard entry

### Mapping / contract types

- module input binding
- module output definition
- form field binding
- activation rule

This separation will reduce confusion between configuration, runtime, and results.

---

## Recommended Evolution Path

### Phase 1

Keep the current [`Event`](../backend/models/event.go#L68) as the definition object, but add architecture for:

- ordered module flow
- participant policy
- score policy
- module input/output mappings

Also add separate persistence for participants, module attempts, and score entries.

### Phase 2

Introduce an explicit `EventSession` runtime model.

### Phase 3

Formalize module contracts and shared-state bindings.

### Phase 4

Add richer orchestration such as optional branches, advanced team logic, and automatic reward calculations.

---

## Final Recommendation

The recommended event-only architecture is:

- use the event as the admin-authored orchestrated experience definition
- create a dedicated event session runtime for live execution
- create participants once and reuse them across all modules
- make teams first-class when team gameplay exists
- use shared state for cross-module data transfer
- use explicit input/output bindings for module interoperability
- store scores as append-only score entries
- compute leaderboard and prize outcomes from aggregated scores

This provides a clean foundation for:

- persistent user identity across modules
- shared data between modules
- cumulative event scoring
- team-based rounds
- admin-configurable orchestration rules
- final prize or ranking logic

It should be used as the future reference architecture for event module orchestration.
