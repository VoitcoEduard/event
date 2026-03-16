# Number Question Type Implementation Plan

## Overview
Add a new question type "number" to the quiz module where users must input an exact integer. This is a simple binary scoring question (correct = full points, incorrect = 0 points) with no bonus for fast response.

## Requirements Summary
- **Question Type**: Number input (integer only)
- **Scoring**: Binary - full points for correct answer, 0 for incorrect
- **No time bonus**: Speed does not affect score
- **Mobile**: Numeric keyboard triggered on input field
- **Admin Display**: "Input a number" text instead of stacked answer cards
- **Results**: Show correct answer, no histogram

---

## Implementation Steps

### Phase 1: Backend Changes

#### 1.1 Update Quiz Models (`backend/models/quiz.go`)
- Add new constant: `QuestionTypeNumber = "number"`
- Add field `CorrectNumber int `json:"correctNumber" bson:"correctNumber"` to `QuizQuestion` struct

#### 1.2 Update Quiz Hub (`backend/hub/quiz_hub.go`)
- Ensure `QuestionTypeNumber` is handled in type checks (if any)

---

### Phase 2: Frontend - Admin Side

#### 2.1 Update QuizQuestionEditor (`frontend/src/components/QuizQuestionEditor.vue`)
- Add "number" option to type dropdown
- Add number question configuration panel:
  - Input field for correct number answer
  - Points configuration (simple maxPoints)
- Modify answer cards section:
  - Hide answer cards for "number" type
  - Show "Input a number" text instead
- Add handler for type change to initialize number config

#### 2.2 Update QuizModuleConfig (`frontend/src/components/QuizModuleConfig.vue`)
- Ensure number questions are handled in question list display

---

### Phase 3: Frontend - Player Side

#### 3.1 Update QuizPlayerScreen (`frontend/src/views/QuizPlayerScreen.vue`)
- Add computed property `isNumber`
- Add number input UI:
  - `<input type="number" inputmode="numeric">` for mobile numeric keyboard
  - `pattern="[0-9]*"` for additional mobile support
  - Integer-only validation (no decimal points)
- Add submit button
- Implement scoring logic:
  - Compare user input with `correctNumber`
  - Full points if exact match, 0 otherwise
  - No time-based scoring

---

### Phase 4: Frontend - Host/BigScreen

#### 4.1 Update QuizHostScreen (`frontend/src/views/QuizHostScreen.vue`)
- Add computed property `isNumber`
- Update answers column display:
  - Show "Input a number" placeholder instead of stacked cards (like slider)
- Update results phase:
  - Display correct number answer
  - Show answer statistics

#### 4.2 Update BigScreen (`frontend/src/views/BigScreen.vue`)
- Add computed property `isNumber`
- Mirror the same display logic as QuizHostScreen

---

### Phase 5: Translations

#### 5.1 Update Locale Files
- `frontend/src/locales/en.json`: Add "number" type label and UI strings
- `frontend/src/locales/ro.json`: Add Romanian translations
- `frontend/src/locales/ru.json`: Add Russian translations

Required translations:
- `quiz_config.question_editor.types.number`: "Number"
- `quiz_config.question_editor.number.correct_answer`: "Correct Answer"
- `quiz_config.question_editor.number.points`: "Points"
- `quiz.player.input_number`: "Enter a number..."
- Display text: "Input a number" on admin cards

---

### Phase 6: Testing & Edge Cases

#### 6.1 Edge Cases to Handle
- Empty input (treat as incorrect)
- Negative numbers (allow if valid)
- Very large numbers (use JavaScript number limits)
- Non-numeric input (prevent via input type)
- Leading/trailing whitespace in input

#### 6.2 Compatibility
- Ensure existing quiz templates still work
- Handle migration of existing questions gracefully

---

## File Changes Summary

| File | Changes |
|------|---------|
| `backend/models/quiz.go` | Add constant + CorrectNumber field |
| `frontend/src/components/QuizQuestionEditor.vue` | Add type option, config panel, hide answer cards |
| `frontend/src/views/QuizPlayerScreen.vue` | Add number input UI, scoring logic |
| `frontend/src/views/QuizHostScreen.vue` | Add isNumber, update display |
| `frontend/src/views/BigScreen.vue` | Add isNumber, update display |
| `frontend/src/locales/en.json` | Add translations |
| `frontend/src/locales/ro.json` | Add translations |
| `frontend/src/locales/ru.json` | Add translations |

---

## Implementation Order

1. Backend model changes (enables data structure)
2. Frontend admin editor (allows creating number questions)
3. Frontend player screen (allows answering)
4. Frontend host/bigscreen (allows viewing)
5. Translations (final polish)

---

## Notes

- Similar structure to slider question but simplified (no range, no tolerance)
- Input uses HTML5 `inputmode="numeric"` which triggers numeric keyboard on iOS/Android
- Scoring is binary (all or nothing) per user requirement
- No fast-answer bonus per user requirement
