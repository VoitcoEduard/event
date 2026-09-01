# Neo-Brutalist Redesign — Player Flow

Adapt the React "ГЕО АТЛАС" example design to our Vue 3 player-facing flow. Visual language: dark `neutral-950` background, hard 2px borders, zero border-radius, 4px offset solid shadows, monospace uppercase badges, per-context accent colors (cyan / fuchsia / lime / amber / rose / indigo).

## Decisions (locked)

| Decision | Choice |
|---|---|
| Scope | **Player flow only**: `MainScreen`, `EventScreen`, `QuizPlayerScreen`. Admin, host, projector keep current look. |
| Styling | **CSS design system** (`src/styles/brutalist.css` with tokens + utility classes), no Tailwind. |
| Fidelity | **Full neo-brutalist** — removes rounded corners, soft glows, and the EventScreen theme toggle. |
| Icons | **`lucide-vue-next`** (replaces emoji in player flow). |
| Mobile | **Port `visualViewport` height-tracking** to a Vue composable so footer buttons clear the virtual keyboard. |

## Hard constraints (do not break)

- **No logic changes**: all WebSocket state machine (`useQuizStateMachine`), router, services, i18n keys, and data flow stay untouched. This is a re-skin: template classes + `<style>` blocks only.
- All user-facing strings keep going through `$t()` / `vue-i18n` (en/ru/ro). The example's hardcoded Russian must NOT be copied.
- Admin/host/projector screens import their own styles; shared global CSS must not visually break them (scope new global rules to a `.brutalist` root class on the player screens only).

## Design tokens (to define in `src/styles/brutalist.css`)

```
--b-bg: #0a0a0a (neutral-950)      --b-surface: #171717 (neutral-900)
--b-border: #262626 (neutral-800)  --b-text: #fafafa
--b-accent-cyan / -fuchsia / -lime / -amber / -rose / -indigo
shadow: 4px 4px 0 0 <accent>       border: 2px solid
radius: 0                          fonts: Inter (body) + monospace (badges/labels)
```

Utility classes: `.b-card`, `.b-btn`, `.b-btn-accent`, `.b-badge`, `.b-input`, `.b-panel`, `.b-progress-block`, `.b-shadow-<accent>`.

## Task list (ordered)

1. **Foundation**
   - `npm i lucide-vue-next` in `frontend/`.
   - Create `frontend/src/styles/brutalist.css` with tokens + utility classes above; import it in `main.js`.
   - Create `frontend/src/composables/useVisualViewportHeight.js`: tracks `window.visualViewport.height` (resize + scroll listeners, cleanup on unmount), returns a `ref` height string. Mobile-only application (width < 768), mirroring the example's `MobileContainer`.

2. **MainScreen.vue (PIN entry)** — highest-traffic screen, do first
   - Wrap root in `.brutalist`; replace rounded card with `.b-card` (hard border + cyan offset shadow).
   - PIN input: `.b-input` — 2px border, monospace, focus state = cyan border + offset shadow; keep `inputmode="numeric"`, maxlength 6, auto-connect logic.
   - Connect button: `.b-btn-accent` (cyan bg, black text, uppercase, arrow icon via `lucide-vue-next` `ArrowRight`).
   - Brand block: replace ⚡ emoji + glow with bordered square icon box (`Zap` icon) + uppercase black-weight title, mono subtitle.
   - Keep language switcher; restyle select to hard-border dark box.
   - Delete the indigo glow / pulse keyframes / gradient button styles.

3. **EventScreen.vue (module hub)**
   - Remove the theme toggle button and `theme-bold` / classic split — one design only.
   - User profile header: avatar → bordered square with mono initial; points badge → `.b-badge` lime accent.
   - Module cards: `.b-card` rows, left icon box (hard border) with `lucide` icon per module type (`FileText`=form, `Gamepad2`=quiz, `Image`=wrong_pictures, `BarChart3`=poll — replaces 📝🎮🖼️📊), `ChevronRight` trailing.
   - Dynamic/polling cards: left 4px accent border (keep existing per-card color, map to accent palette), zero radius.
   - Section titles: mono uppercase tracking-widest neutral-500.
   - Keep `FormModule` child component styling as-is for now (its own scoped styles), just ensure the surrounding shell matches.

4. **QuizPlayerScreen.vue (all states)** — biggest file (~1844 lines), re-skin state-by-state; verify each against the state machine:
   - **Shell**: apply `useVisualViewportHeight` so the submit/confirm button sits above the virtual keyboard on the nickname-entry and number-input states.
   - **Nickname entry (LOBBY, !joined)**: mirror example's text-question pattern — `.b-input` + inline confirm button (`CornerDownLeft` icon) appears when non-empty; hint text "press Enter". Keep `@keyup.enter="joinLobby"`.
   - **Lobby (joined)**: centered waiting card, mono nickname display in cyan badge, replace `.loader` spinner with hard-edged pulsing blocks animation.
   - **GET_READY / QUESTION_PRESENTING**: black-weight uppercase headline + mono countdown numeral.
   - **QUESTION_ACTIVE — default options**: option rows as `.b-card` buttons: letter badge (А/Б/В/Г mono) + title + square checkbox (`Check` icon when selected); selected = fuchsia border + 4px offset shadow + 10% accent bg.
   - **QUESTION_ACTIVE — correct-picture (HorizontalCarousel)**: restyle frame/submit button only; do NOT touch carousel component internals.
   - **QUESTION_ACTIVE — wrong-pictures (MapCoordinatePicker)**: restyle submit button + header; picker internals untouched.
   - **QUESTION_ACTIVE — range / number**: range slider → hard-bordered track/thumb; number input → `.b-input` monospace.
   - **Per-question accent mapping** (replaces example's per-question colors): default=fuchsia, correct-picture=amber, wrong-pictures=rose, range/number=indigo, lobby/name=cyan, results/finish=lime.
   - **ANSWERED / RESULTS (correct/wrong)**: replace `bg-correct`/`bg-wrong` full-bleed gradients with lime (correct) / rose (wrong) bordered result card + mono verdict badge.
   - **LEADERBOARD / FINISHED**: ranking rows with mono position squares (`1`/`2`/`3` hard-border boxes), total-points badge; FINISHED gets lime "done" card + `RotateCcw` restart / `LogOut` exit buttons.
   - **Question header**: keep timer bar but square it (no radius, accent fill); question-number badge → bordered mono square.
   - **Progress blocks**: adopt the example's segmented progress bar (one block per question, accent fill for completed, pulse for current) in the header of QUESTION states.

5. **Cleanup**
   - Remove now-unused scoped CSS from the three screens (old gradients, radius, glows).
   - Remove EventScreen `toggleTheme` / `theme-*` logic **only after confirming** nothing else references it (grep `theme-bold`, `toggleTheme`).
   - Do NOT touch: `AdminScreen`, `AdminEventDetailsScreen`, `QuizHostScreen`, `ProjectorView`, `TeamsAdminScreen`, `DataLibraryScreen`, `QuizLibraryScreen`, all `components/*` internals (except player-shell-level buttons noted above), all `services/*`, `composables/useQuizStateMachine.js`.

6. **i18n check**
   - Re-skin must not add hardcoded strings. If any new label is needed (e.g. "press Enter" hint), add keys to `src/locales/{en,ru,ro}.json` in all three languages.

## Validation

- `cd frontend && npm run build` passes with no errors.
- Manual walkthrough of the player journey on a 390px-wide viewport: PIN entry → event hub → quiz lobby → each question type (default, correct-picture, wrong-pictures, range, number) → results → leaderboard → finished.
- Virtual keyboard test: nickname input and number input — footer/confirm button visible above keyboard on iOS Safari + Android Chrome (use browser devtools device emulation + real device if available).
- Verify admin/host/projector screens render unchanged (they share `main.js` global imports).
- Run a live quiz session (host + 2 players) to confirm the WebSocket state machine transitions render correctly in the new skin.

## Risks / open questions

- **Risk**: `FormModule` (embedded in EventScreen) has its own scoped styles that will clash visually with the new shell — acceptable for this pass; a follow-up plan can re-skin forms.
- **Risk**: removing the EventScreen theme toggle changes user-facing behavior — flag in PR description.
- **Out of scope**: admin/host/projector redesign, Tailwind migration, changes to quiz question content or backend.
