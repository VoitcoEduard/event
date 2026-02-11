# Client Compute Shift - Implementation Todo

## Phase 1: Client-Side Answer Validation & Late Response Handling

### Backend Changes
- [ ] Update `QuizSession` model to store signals instead of raw answers
- [ ] Add `PlayerSignal` struct with `Status` field ("on_time", "grace_period", "missed")
- [ ] Add `QuestionWindow` struct with grace period tracking
- [ ] Add `AnswerCounts` map for real-time statistics
- [ ] Add `PlayerActivity` tracking (last signal time, questions answered/missed)
- [ ] Modify `ProcessMessage` in `quiz_hub.go` to handle `player:signal` instead of `player:answer`
- [ ] Implement `CanAcceptSignal()` with 2-second grace period logic
- [ ] Remove server-side scoring logic (trust client score)
- [ ] Add `sync:minimal` broadcast (answered count only, not full state)
- [ ] Update `host:show_results` to not calculate scores

### Frontend Changes
- [ ] Update `QuizPlayerScreen.vue` to validate answers locally
- [ ] Calculate score on client (time-based + streak bonus)
- [ ] Send `player:signal` with `{ questionIndex, answerIndex, scoreEarned, timeSpent }`
- [ ] Track local score, streak, rank in component state
- [ ] Remove dependency on server for score updates
- [ ] Set score to 0 when timer hits 0 (late response)
- [ ] Still transmit answer choice for statistics even when late

## Phase 2: Batch-Based Question Distribution

### Backend Changes
- [ ] Create `QuestionBatch` struct with batch metadata
- [ ] Implement `CalculateBatches()` function (~50KB target size)
- [ ] Add batch storage to `QuizSession`
- [ ] Add `handleBatchRequest` message handler
- [ ] Modify `host:start_quiz` to send first batch only
- [ ] Add `data:batch` message type for sending question chunks

### Frontend Changes
- [ ] Create `QuestionCache` class for client-side storage
- [ ] Modify `QuizPlayerScreen.vue` to use cached questions
- [ ] Add batch request logic (request when 2 questions remain)
- [ ] Handle `data:batch` WebSocket message
- [ ] Remove initial fetch of all questions via REST API
- [ ] Add loading states for batch fetching

## Phase 3: State Machine & Orchestration

### Backend Changes
- [ ] Create `cmd:state` message type for state transitions
- [ ] Refactor state management to send commands not full state
- [ ] Add state validation (prevent invalid transitions)
- [ ] Keep minimal session state (current index, state string only)
- [ ] Remove full session broadcast on every change

### Frontend Changes
- [ ] Create `useQuizStateMachine` composable
- [ ] Define all valid state transitions
- [ ] Implement `canTransition()` and `transition()` methods
- [ ] Handle `cmd:state` messages from server
- [ ] Add local state transitions (answered while timer running)
- [ ] Update UI to react to state changes

## Phase 4: Reconnection & State Sync

### Backend Changes
- [ ] Add `player:reconnect` handler for users returning after inactivity
- [ ] Implement `handleReconnection()` with catch-up logic
- [ ] Track missed questions per player
- [ ] Mark missed questions as "skipped" (answerIndex: -1, score: 0)
- [ ] Implement inactive detection (6+ missed questions = inactive)
- [ ] Add `sync:catchup` message type
- [ ] Calculate `batchesNeeded` for reconnected users

### Frontend Changes
- [ ] Detect when user returns after being away
- [ ] Send `player:reconnect` with `lastKnownQuestionIndex`
- [ ] Handle `sync:catchup` response
- [ ] Fetch missed batches via `player:batch_request`
- [ ] Show "You missed X questions" notification
- [ ] Continue from current question

## Phase 5: Signal Queue & Offline Support

### Backend Changes
- [ ] Add `player:signal_batch` handler for bulk signal processing
- [ ] Implement signal storage for admin statistics
- [ ] Add admin endpoint to fetch quiz statistics
- [ ] Create statistics aggregation (answer distribution, response times)

### Frontend Changes
- [ ] Create `SignalQueue` class with timeout awareness
- [ ] Track `QuestionWindow` (end time + grace period)
- [ ] Discard signals that exceed grace period
- [ ] Implement `enqueue()`, `flush()`, `onReconnect()` methods
- [ ] Add connection state monitoring
- [ ] Queue signals when offline, flush when reconnected
- [ ] Add retry logic with exponential backoff

## Phase 6: Admin Dashboard Statistics

### Backend Changes
- [ ] Create `QuizStatistics` aggregation model
- [ ] Add endpoint `GET /api/quiz/:moduleId/statistics`
- [ ] Implement real-time statistics calculation
- [ ] Add answer distribution tracking per question
- [ ] Add response time tracking

### Frontend Changes
- [ ] Update `AdminEventDetailsScreen.vue` to show quiz stats
- [ ] Create statistics display components
- [ ] Add real-time statistics polling or WebSocket updates
- [ ] Display answer distribution charts
- [ ] Show player engagement metrics

## Phase 7: Testing & Optimization

### Testing
- [ ] Test with 150 concurrent users simulation
- [ ] Measure bandwidth usage per question
- [ ] Test offline/ reconnection scenarios
- [ ] Verify state machine transitions
- [ ] Test batch distribution under load

### Optimization
- [ ] Optimize batch size calculation
- [ ] Add compression for question batches
- [ ] Implement client-side caching with localStorage
- [ ] Add request deduplication for batch requests
- [ ] Optimize WebSocket message size

## Files to Modify

### Backend
- `backend/models/quiz.go` - Add new structs
- `backend/hub/quiz_hub.go` - Update message handlers
- `backend/hub/types.go` - Add new message types
- `backend/handlers/quiz.go` - Add statistics endpoint

### Frontend
- `frontend/src/views/QuizPlayerScreen.vue` - Main player logic
- `frontend/src/views/QuizHostScreen.vue` - Host controls
- `frontend/src/services/websocket.js` - WS message handling
- `frontend/src/composables/useQuizStateMachine.js` - New composable
- `frontend/src/utils/questionCache.js` - New cache class
- `frontend/src/utils/signalQueue.js` - New queue class
- `frontend/src/views/AdminEventDetailsScreen.vue` - Statistics display

## Key Metrics to Achieve

- [ ] 95% reduction in bandwidth per question (from ~1.25MB to ~20KB)
- [ ] Support 150 concurrent users on minimal server
- [ ] Sub-100ms answer validation (client-side)
- [ ] Seamless batch preloading (no user-visible loading)
- [ ] Zero data loss on temporary disconnections
- [ ] Grace period handling (2-second window for late signals)
- [ ] Reconnection support (users can rejoin after missing up to 5 questions)
- [ ] Inactive detection (users marked inactive after 6+ missed questions)

## Late Response & Reconnection Summary

### Late Response Rules
1. **Client stops accepting clicks** when timer hits 0
2. **Score = 0** for late responses
3. **Answer still transmitted** for statistics and user feedback
4. **2-second grace period** on server - signals accepted but not scored
5. **After grace period** - signals rejected, not included in stats

### Reconnection Flow
1. User reconnects after being away
2. Client sends `player:reconnect` with last known question
3. Server marks missed questions as "skipped" (0 points)
4. Server sends `sync:catchup` with current state and missed batches
5. User can continue from current question

### Inactive Detection
- **1-5 missed questions**: User is still active, can rejoin
- **6+ missed questions**: User marked as inactive
- **Inactive users**: Excluded from statistics, final score = 0
