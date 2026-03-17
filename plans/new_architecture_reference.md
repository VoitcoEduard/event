# New Architecture Reference

## Purpose

This document describes the target high-level architecture for evolving the platform beyond the current event-centric model. It is a design reference, not an implementation plan.

## Design Goal

The platform must support three product modes using the same core content model:

1. big event experiences
2. single-game sessions
3. solo play experiences

The system should reuse templates across all of these modes without forcing everything into one event-only structure.

---

## Core Shift

The system should evolve from:

- events containing modules as the primary operating model

to:

- reusable templates powering multiple experience types

This means event remains an important product mode, but it is no longer the only top-level organizing concept.

---

## Architectural Layers

The target design is based on three core layers plus one supporting outcomes layer.

### 1. Content Layer

The content layer stores reusable playable definitions.

It includes:

- game templates
- question sets and game configuration
- media assets
- content metadata such as language, duration, tags, or difficulty

### Role of the Layer

The content layer defines what can be played.

Content must be reusable and independent from any specific event or runtime session.

---

### 2. Experience Layer

The experience layer packages reusable content into concrete product use cases.

It includes three primary experience types:

- `event_experience`
- `quick_game_experience`
- `solo_publication`

### Role of the Layer

The experience layer defines how content is delivered.

It is the product-facing layer that differentiates between a complex hosted event, a lightweight single-game flow, and an always-available solo play item.

---

### 3. Runtime Layer

The runtime layer manages active play execution.

It includes:

- hosted sessions
- solo attempts
- player presence
- live state transitions
- answer or signal collection
- scoring and runtime feedback

### Role of the Layer

The runtime layer defines what is happening now.

It should be separated from template authoring and from long-lived experience configuration.

---

### 4. Results and Analytics Layer

The results layer stores outcomes produced by runtime activity.

It includes:

- session results
- player attempts
- aggregate statistics
- leaderboards where applicable

### Role of the Layer

The results layer defines what happened.

It must remain distinct from the reusable content layer so gameplay outcomes do not mutate master templates.

---

## Target Domain Model

The high-level model should revolve around four concepts:

- `template`
- `experience`
- `session` or `attempt`
- `result`

### Template

A reusable game definition.

Examples:

- quiz template
- wrong pictures template
- future mini-game template

### Experience

A packaged use case built from one or more templates.

Examples:

- a big event made of multiple modules
- a simple one-game setup for quick launch
- a published solo item in a player library

### Session or Attempt

An active execution instance.

- hosted experiences create sessions
- solo experiences create individual attempts

### Result

A persisted gameplay outcome used for analytics, reporting, or leaderboards.

---

## Product Modes in the New Architecture

## Big Event Mode

Big event mode remains a premium orchestrated experience.

### Characteristics

- multiple modules or game segments
- host-controlled flow
- optional big screen and screen manager
- event-specific branding, slides, or presentation controls

### Architectural Position

Represented as an `event_experience` that references reusable templates and creates one or more hosted sessions.

---

## Single-Game Session Mode

This is the simplified admin-facing product flow.

### Characteristics

- exactly one selected template
- minimal setup and minimal admin UI
- direct launch and share flow
- no need to build a full event

### Architectural Position

Represented as a `quick_game_experience` that references one template and creates a hosted session.

### Design Intent

This mode exists to remove friction for customers who only need one activity and do not need the full event workflow.

---

## Solo Mode

This is the self-serve player-facing product surface.

### Characteristics

- no live host required
- players browse available small games
- each player launches an independent play attempt
- content may be replayable or always available

### Architectural Position

Represented as a `solo_publication` that references a reusable template and creates independent player attempts.

### Design Intent

Solo mode is not just a one-player hosted session. It is a published playable item that can be launched repeatedly by many users at different times.

---

## Template Reuse Principle

The same template should be usable across different product modes.

For example, one quiz template may be:

- included inside a big event
- launched as a quick single-game session
- published into the solo game catalog

This is a key architectural requirement.

---

## Configuration Layers

The system should distinguish between three kinds of configuration.

### Template Configuration

Defines the reusable default content and rules.

### Experience Configuration

Defines how the template is packaged for a specific use case.

Examples:

- branding
- timing policy
- presentation mode
- visibility settings

### Runtime State

Defines the live execution state.

Examples:

- current question
- joined players
- active timer
- current score state

### Key Principle

Runtime state must not directly rewrite the master template.

---

## Access and Administration in the New Design

The simplified role model aligns with the new architecture as follows:

- `platform_admin` manages templates, experiences, and publishing
- `game_admin` operates one assigned `quick_game_experience`
- `game_user` participates in sessions or solo attempts

This keeps permissions aligned with the architecture without adding unnecessary role complexity.

---

## Authentication Shape

The new architecture assumes two different administrative access paths.

### Platform Administrative Access

Persistent authenticated access for full administrators.

### Delegated Game Administrative Access

Scoped temporary access for simple game operation, ideally granted through an expirable access link that establishes a restricted session.

### Player Access

Lightweight session participation for gameplay.

---

## Why This Architecture Fits the Current System

The current system already contains the foundations for this evolution:

- reusable templates exist
- event structures exist
- module runtime concepts exist
- quiz session runtime concepts exist

The new architecture does not reject the current system. Instead, it reframes it:

- event becomes one experience type
- templates become the central reusable asset
- quick game becomes a first-class lightweight experience
- solo mode becomes a published content surface

---

## Design Intent Summary

The target architecture is a reusable play platform built around templates, not just an event builder.

### Core model

- templates define playable content
- experiences package that content for different use cases
- sessions or attempts execute the experience
- results preserve outcomes and analytics

### Product outcome

This structure allows the same content foundation to support:

- large hosted events
- simple one-game admin flows
- self-serve solo game libraries

without forcing all workflows through the same heavy event setup.
