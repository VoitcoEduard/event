# Quiz Results Persistence Plan

## Overview
Add database persistence for quiz session results when admin ends the quiz. The current implementation stores all quiz data in-memory only and clears it when the session ends.

## Data Structure to Save

**QuizResult document (new "quiz_results" collection):**
```go
type QuizResult struct {
    EventID       string         `json:"eventId" bson:"eventId"`
    ModuleID      string         `json:"moduleId" bson:"moduleId"`
    FinishedAt    time.Time      `json:"finishedAt" bson:"finishedAt"`
    
    Players       []PlayerResult `json:"players" bson:"players"`
    PlayerSignals []SignalResult `json:"playerSignals" bson:"playerSignals"`
}

type PlayerResult struct {
    ID       string `json:"id" bson:"id"`
    Nickname string `json:"nickname" bson:"nickname"`
    Score    int    `json:"score" bson:"score"`
}

type SignalResult struct {
    PlayerID      string  `json:"playerId" bson:"playerId"`
    QuestionIndex int     `json:"questionIndex" bson:"questionIndex"`
    AnswerValue   float64 `json:"answerValue" bson:"answerValue"`   // For slider questions
    ScoreEarned   int     `json:"scoreEarned" bson:"scoreEarned"`
    TimeSpent     int     `json:"timeSpent" bson:"timeSpent"`     // milliseconds
}
```

**NOT saved:** state, answerIndex, answerIndices, timestamp, status

## Implementation Steps

### Step 1: Create QuizResult Model
**File:** `backend/models/quiz.go`
- Add `QuizResult`, `PlayerResult`, `SignalResult` structs

### Step 2: Add Database Save Function
**File:** `backend/hub/quiz_hub.go`
- Add `saveQuizResults()` function that writes to new "quiz_results" collection

### Step 3: Modify host:end_quiz Handler
**File:** `backend/hub/quiz_hub.go`
- In the `host:end_quiz` case (line 669):
  1. Call save function to persist results BEFORE clearing
  2. Keep existing state change to "finished" 
  3. Keep existing broadcast to clients

### Step 4: Update Frontend - Replace Exit with End Quiz
**File:** `frontend/src/views/QuizHostScreen.vue`
- Line 189: Change "Exit" button text to "End Quiz"
- Modify `exitQuiz()` function to call end_quiz instead of reset_quiz

## Current Flow vs New Flow

**Current:**
1. Host clicks "End Quiz" → sends `host:end_quiz` → sets state to "finished" (no DB write)
2. Host clicks "Exit" → sends `host:reset_quiz` → clears all session data

**New:**
1. Host clicks "End Quiz" → sends `host:end_quiz` → saves results to DB → sets state to "finished"
2. Host clicks "Exit" → sends `host:reset_quiz` → clears session (results already saved)

## WebSocket Safety
- Save happens synchronously within the lock, before broadcast
- State changes happen first, then save, then clients are notified
- This ensures WebSocket state remains consistent

## Files to Modify
1. `backend/models/quiz.go` - Add QuizResult, PlayerResult, SignalResult models
2. `backend/hub/quiz_hub.go` - Add save function, modify host:end_quiz handler
3. `frontend/src/views/QuizHostScreen.vue` - Update Exit button behavior
