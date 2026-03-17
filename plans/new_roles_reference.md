# New Roles Reference

## Purpose

This document defines the simplified role model for the next-stage platform design. It is a reference for product, backend, frontend, and permission decisions. It is not an implementation plan.

## Role Model

The platform uses three roles:

- `platform_admin`
- `game_admin`
- `game_user`

The model is intentionally simple. The goal is to support big events, single-game sessions, and solo play without introducing a large multi-tier permission system.

---

## 1. Platform Admin

### Definition

`platform_admin` is the full-trust administrative role.

This role is intended for internal operators or fully trusted organization owners who manage the system as a whole.

### Scope

Global platform scope.

### Responsibilities

- manage reusable templates and content library
- manage media and supporting assets
- create and manage big event experiences
- create and manage single-game experiences
- publish and manage solo-play catalog items
- generate temporary access for delegated game admins
- access analytics across all experiences
- manage platform-level settings and operational controls

### Key Principle

`platform_admin` owns the system, the content layer, and the experience layer.

---

## 2. Game Admin

### Definition

`game_admin` is a limited administrative role for operating a specific simple game experience.

This role is intended for users who do not need full event creation or full platform access. Its main purpose is to support the lightweight “create and run one game” use case.

### Scope

Scoped to one assigned simple game resource.

It must not imply access to the full platform or to unrelated games.

### Responsibilities

- access one assigned simple game admin surface
- configure lightweight game-specific settings
- launch, run, pause, stop, or reset that game
- share the player access link or join code for that game
- view player activity and results for that game

### Restrictions

`game_admin` must not:

- create or manage full event experiences
- manage the global template library broadly
- manage users or platform settings
- access other admins’ games by default
- publish solo catalog content globally

### Key Principle

`game_admin` operates a game instance, not the entire platform.

---

## 3. Game User

### Definition

`game_user` is the participant role.

This is the default player-facing role for people who join a hosted session or launch a solo play attempt.

### Scope

Scoped to participation in a session or solo attempt.

### Responsibilities

- join and play games
- submit answers or interactions
- receive results, feedback, and personal progress information
- optionally replay a solo experience when allowed

### Restrictions

`game_user` has no administrative authority.

### Key Principle

`game_user` interacts with the runtime layer only as a participant.

---

## Access Model

The platform distinguishes between global authority and scoped authority.

- `platform_admin` has global authority
- `game_admin` has scoped authority over a single simple game
- `game_user` has runtime participation only

This keeps the model small while still protecting the platform from over-broad access.

---

## Authentication Shape

The role model assumes different authentication patterns for different roles.

### Platform Admin Authentication

`platform_admin` should authenticate through a standard admin login flow and receive a secure session.

### Game Admin Authentication

`game_admin` can authenticate in two valid ways:

1. through a normal authenticated admin account with scoped permissions
2. through a temporary expirable access link that grants scoped access to one simple game

The temporary link flow is especially important for delegated game operation without requiring full account onboarding.

### Game User Authentication

`game_user` does not need full admin authentication. A lightweight player identity is sufficient.

---

## Temporary Game Admin Access

The simplified model includes delegated access for `game_admin` through expirable links.

### Reference Design

- a `platform_admin` creates an access grant for one simple game
- the grant is delivered as a secure expirable link
- when the recipient opens the link, the backend validates the grant
- the backend establishes a restricted authenticated session
- that session gives `game_admin` access only to the assigned game

### Constraints

- access is time-limited
- access is revocable
- access is scoped to one simple game resource
- access must not escalate into platform-wide privileges

---

## Relationship to Product Modes

### Big Event Mode

Primarily managed by `platform_admin`.

### Single Game Session Mode

May be created by `platform_admin` and operated by either `platform_admin` or delegated `game_admin`.

### Solo Mode

Published and managed by `platform_admin`, consumed by `game_user`.

---

## Design Intent

This role model is intentionally minimal.

The system should avoid introducing many fine-grained administrative tiers until there is a clear operational need. The immediate goal is to support new product flows with a small, understandable permission model that matches the platform’s next-stage architecture.
