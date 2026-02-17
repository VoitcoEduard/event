<template>
  <div class="big-screen">
    <!-- Screensaver Mode -->
    <div v-if="displayMode === 'screensaver'" class="screensaver-container">
      <img 
        v-if="screensaverImageUrl" 
        :src="screensaverImageUrl" 
        alt="Screensaver" 
        class="screensaver-image"
        @load="onImageLoad"
      />
      <div v-else class="screensaver-placeholder">
        <p>No screensaver configured</p>
        <p class="hint">Ask admin to set up a screensaver image</p>
      </div>
    </div>

    <!-- Connecting State -->
    <div v-else-if="displayMode === 'connecting'" class="connecting-container">
      <div class="loading-content">
        <h1>Connecting...</h1>
        <div class="loader"></div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="displayMode === 'error'" class="error-container">
      <h1>Connection Error</h1>
      <p>{{ errorMessage }}</p>
      <button @click="retryConnection" class="retry-btn">Retry</button>
    </div>

    <!-- Quiz Display States -->
    <template v-else>
      <!-- LOBBY STATE -->
      <div v-if="gameState === 'lobby'" class="state-container lobby-container">
        <div class="lobby-header">
          <h1 class="game-title">Join the Quiz!</h1>
          <div class="pin-display">
            <span class="pin-label">Game PIN:</span>
            <span class="pin-code">{{ pinCode }}</span>
          </div>
          <div class="qr-container">
            <div class="qr-placeholder">QR Code</div>
          </div>
        </div>
        
        <div class="players-area">
          <div class="players-count">
            <span>👥 {{ players.length }}</span> Players
          </div>
          <div class="players-grid">
            <div v-for="player in players" :key="player.id" class="player-chip">
              {{ player.nickname }}
            </div>
          </div>
        </div>
      </div>

      <!-- GET READY STATE -->
      <div v-else-if="gameState === 'get-ready'" class="state-container get-ready-container">
        <div class="loading-content">
          <h1>Get Ready!</h1>
          <div class="loader"></div>
        </div>
      </div>

      <!-- QUESTION STATE -->
      <div v-else-if="gameState === 'question-active'" class="state-container question-container">
        <div class="question-header">
          <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }">
            {{ timeRemaining }}
          </div>
          <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
        </div>

        <div class="question-main-content">
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" />
                <div class="image-overlay-gradient"></div>
              </div>
              <div class="question-text-overlay">
                <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
              </div>
            </div>
          </div>

          <div class="answers-column">
            <div v-if="isSlider" class="slider-instruction-view">
              <h2 class="slider-instruction-text">Slide to answer</h2>
              <div class="slider-demo-track">
                <div class="slider-demo-thumb"></div>
              </div>
            </div>
            <template v-else>
              <div 
                v-for="(answer, idx) in currentQuestion?.answers" 
                :key="idx" 
                class="answer-card-stacked"
                :class="getAnswerColorClass(idx)"
                v-show="!isTrueFalse || idx < 2"
              >
                <div class="answer-icon">{{ getShapeIcon(idx) }}</div>
                <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ answer }}</div>
              </div>
            </template>
          </div>
        </div>
        
        <div class="answer-counter-widget">
          <div class="counter-value">{{ totalAnswered }}</div>
          <div class="counter-label">Answers</div>
        </div>
      </div>

      <!-- RESULTS/LEADERBOARD STATE -->
      <div v-else-if="gameState === 'results' || gameState === 'leaderboard'" class="state-container results-container">
        <h2 class="results-title">{{ gameState === 'results' ? 'Question Results' : 'Leaderboard' }}</h2>
        
        <div v-if="gameState === 'results' && !isSlider" class="correct-answer-reveal">
          <p>Correct Answer:</p>
          <div class="correct-answer-display" :class="getAnswerColorClass(currentQuestion?.correctIndex)">
            {{ getShapeIcon(currentQuestion?.correctIndex) }} 
            {{ currentQuestion?.answers[currentQuestion?.correctIndex] }}
          </div>
        </div>

        <div v-if="gameState === 'results' && !isSlider" class="answers-stats results-mode">
          <div class="stats-grid">
            <div v-for="(count, idx) in (isTrueFalse ? 2 : 4)" :key="idx" class="stat-column">
              <div class="stat-bar-container">
                <div 
                  class="stat-bar" 
                  :class="getAnswerColorClass(idx)"
                  :style="{ height: getBarHeight(idx) + '%' }"
                ></div>
              </div>
              <div class="stat-count">{{ answerCounts[idx] || 0 }}</div>
              <div class="stat-icon">{{ getShapeIcon(idx) }}</div>
            </div>
          </div>
        </div>

        <!-- Slider Histogram Results (New Minimal Design) -->
        <div v-if="gameState === 'results' && isSlider" class="answers-stats results-mode slider-results-view">
          <div class="slider-result-container">
            <div class="slider-result-track"></div>
            <div class="slider-distribution-overlay">
              <div 
                v-for="(count, idx) in histogramBuckets" 
                :key="idx"
                class="distribution-bar-wrapper"
                :style="{ left: (idx / histogramBuckets.length * 100) + '%', width: (100 / histogramBuckets.length) + '%' }"
              >
                <div 
                  class="distribution-bar"
                  :style="{ height: getHistogramBarHeight(count) + '%' }"
                  :class="{ 'has-votes': count > 0 }"
                >
                  <span class="bar-tooltip" v-if="count > 0">{{ count }}</span>
                </div>
              </div>
            </div>
            <div 
              class="correct-answer-marker" 
              :class="{ 'reveal-marker': revealCorrect }"
              :style="{ left: correctMarkerPercentage + '%' }"
            >
              <div class="marker-line"></div>
              <div class="marker-label">
                <span class="marker-title">Correct</span>
                <span class="marker-value">{{ formatNumber(currentQuestion?.correctValue) }}</span>
              </div>
            </div>
            <div class="slider-labels">
              <span class="label-min">{{ formatNumber(currentQuestion?.sliderConfig?.minValue) }}</span>
              <span class="label-max">{{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}</span>
            </div>
          </div>
        </div>

        <div class="leaderboard-list">
          <div v-for="(player, idx) in topPlayers" :key="player.id" class="leaderboard-item">
            <div class="rank">{{ idx + 1 }}</div>
            <div class="name">{{ player.nickname }}</div>
            <div class="score">{{ player.score }} pts</div>
          </div>
        </div>
      </div>
      
      <!-- FINISHED STATE -->
      <div v-else-if="gameState === 'finished'" class="state-container finished-container">
        <h1>Quiz Finished!</h1>
        <div class="podium">
          <div class="podium-place second" v-if="topPlayers[1]">
            <div class="podium-bar">
              <span class="podium-rank">2</span>
              <span class="podium-name">{{ topPlayers[1].nickname }}</span>
              <span class="podium-score">{{ topPlayers[1].score }}</span>
            </div>
          </div>
          <div class="podium-place first" v-if="topPlayers[0]">
            <div class="podium-bar">
              <span class="podium-rank">1</span>
              <span class="podium-name">{{ topPlayers[0].nickname }}</span>
              <span class="podium-score">{{ topPlayers[0].score }}</span>
            </div>
          </div>
          <div class="podium-place third" v-if="topPlayers[2]">
            <div class="podium-bar">
              <span class="podium-rank">3</span>
              <span class="podium-name">{{ topPlayers[2].nickname }}</span>
              <span class="podium-score">{{ topPlayers[2].score }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { getScreensavers } from '../services/media'

const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

// Display mode: 'connecting', 'screensaver', 'quiz', 'error'
const displayMode = ref('connecting')
const errorMessage = ref('')
const screensaverImageUrl = ref('')

// Quiz state
const gameState = ref('lobby') 
const pinCode = ref(eventId)
const players = ref([])
const questions = ref([])
const currentQuestionIndex = ref(0)
const timeRemaining = ref(20)
const answerCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 })
const playerSignals = ref([])

const totalAnswered = computed(() => {
  if (isSlider.value) return playerSignals.value.length
  return Object.values(answerCounts.value).reduce((a, b) => a + b, 0)
})

const totalQuestions = computed(() => questions.value.length)
const currentQuestion = computed(() => questions.value[currentQuestionIndex.value])
const isLastQuestion = computed(() => currentQuestionIndex.value >= totalQuestions.value - 1)
const isTrueFalse = computed(() => currentQuestion.value?.type === 'true_false')
const isSlider = computed(() => currentQuestion.value?.type === 'slider')
const revealCorrect = ref(false)

// Watch gameState to trigger animation
watch(gameState, (newVal) => {
    if (newVal === 'results' && isSlider.value) {
        revealCorrect.value = false
        setTimeout(() => {
            revealCorrect.value = true
        }, 1500) // 1.5s delay before revealing correct answer
    } else {
        revealCorrect.value = false
    }
})

const questionImageUrl = computed(() => {
  const imageId = currentQuestion.value?.imageId
  return imageId ? getImageUrl(imageId) : null
})

const questionFontSize = computed(() => {
  const text = currentQuestion.value?.text || ''
  let baseSize
  if (text.length < 30) {
    baseSize = 7.0
  } else if (text.length < 60) {
    baseSize = 5.5
  } else if (text.length < 100) {
    baseSize = 4.0
  } else {
    baseSize = 3.0
  }
  return `${baseSize}vw`
})

const getAnswerFontSize = (answerText) => {
  const text = answerText || ''
  const length = text.length
  let size
  if (length < 15) size = 2.4
  else if (length < 25) size = 2.0
  else if (length < 40) size = 1.6
  else if (length < 60) size = 1.3
  else if (length < 100) size = 1.0
  else size = 0.85
  return `${size}vw`
}

const formatNumber = (num, decimals = 0) => {
  if (num === undefined || num === null) return ''
  return Number(num).toFixed(decimals).replace(/\.0+$/, '')
}

const topPlayers = computed(() => {
  return [...players.value].sort((a, b) => b.score - a.score).slice(0, 5)
})

const getBarHeight = (idx) => {
  const counts = Object.values(answerCounts.value)
  const max = Math.max(...counts, 1)
  const count = answerCounts.value[idx] || 0
  return (count / max) * 100
}

const histogramBuckets = computed(() => {
  if (!isSlider.value) return []
  const q = currentQuestion.value
  if (!q || !q.sliderConfig) return []

  const min = q.sliderConfig.minValue
  const max = q.sliderConfig.maxValue
  const bucketCount = 20
  const buckets = new Array(bucketCount).fill(0)
  const range = max - min
  if (range === 0) return buckets

  playerSignals.value.forEach(sig => {
    if (sig.questionIndex !== currentQuestionIndex.value) return
    const val = sig.answerValue !== undefined ? sig.answerValue : (sig.payload?.answerValue || 0)
    let bucketIdx = Math.floor(((val - min) / range) * bucketCount)
    if (bucketIdx < 0) bucketIdx = 0
    if (bucketIdx >= bucketCount) bucketIdx = bucketCount - 1
    buckets[bucketIdx]++
  })
  return buckets
})

const getHistogramBarHeight = (count) => {
  const max = Math.max(...histogramBuckets.value, 1)
  return (count / max) * 100
}

const correctMarkerPercentage = computed(() => {
  if (!isSlider.value) return 0
  const q = currentQuestion.value
  const min = q?.sliderConfig?.minValue || 0
  const max = q?.sliderConfig?.maxValue || 100
  const range = max - min
  if (range === 0) return 0
  return ((q.correctValue - min) / range) * 100
})

const isBucketCorrect = (bucketIdx) => {
  const q = currentQuestion.value
  const min = q.sliderConfig.minValue
  const max = q.sliderConfig.maxValue
  const bucketCount = 20
  const range = max - min
  const val = q.correctValue
  let correctIdx = Math.floor(((val - min) / range) * bucketCount)
  if (correctIdx < 0) correctIdx = 0
  if (correctIdx >= bucketCount) correctIdx = bucketCount - 1
  return bucketIdx === correctIdx
}

const getShapeIcon = (idx) => {
  if (isTrueFalse.value) {
    return idx === 0 ? '✔️' : '❌'
  }
  return ['▲', '◆', '●', '■'][idx]
}

const getAnswerColorClass = (idx) => {
  if (isTrueFalse.value) {
    return idx === 0 ? 'color-true' : 'color-false'
  }
  return 'color-' + idx
}

// Timer
let timerInterval
const startLocalTimer = (duration) => {
  clearInterval(timerInterval)
  timeRemaining.value = duration
  timerInterval = setInterval(() => {
    if (timeRemaining.value > 0) {
      timeRemaining.value--
    } else {
      clearInterval(timerInterval)
    }
  }, 1000)
}

// Load screensaver config
const loadScreensaverConfig = async () => {
  try {
    // First try to load event-specific screensaver
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screensaver`)
    if (res.ok) {
      const data = await res.json()
      if (data.imageId) {
        screensaverImageUrl.value = getImageUrl(data.imageId)
        return
      }
    }
    
    // Fallback: Load from global screensaver library
    const screensavers = await getScreensavers()
    if (screensavers.length > 0) {
      // Use the first screensaver from the library as default
      screensaverImageUrl.value = screensavers[0].thumbnailUrl
    }
  } catch (e) {
    console.error('Failed to load screensaver config', e)
    // Try to load from library as last resort
    try {
      const screensavers = await getScreensavers()
      if (screensavers.length > 0) {
        screensaverImageUrl.value = screensavers[0].thumbnailUrl
      }
    } catch (err) {
      console.error('Failed to load screensaver from library', err)
    }
  }
}

// Connect to WebSocket
const connect = () => {
  displayMode.value = 'connecting'
  
  // Register handlers before connecting
  quizWS.on('cmd:state', (payload) => {
    const prevState = gameState.value
    gameState.value = payload.state
    
    if (payload.currentIndex !== undefined) {
      currentQuestionIndex.value = payload.currentIndex
    }
    
    if (payload.state === 'question-active') {
      if (prevState !== 'question-active' || payload.currentIndex !== currentQuestionIndex.value) {
        startLocalTimer(currentQuestion.value?.timeLimit || 20)
      }
      displayMode.value = 'quiz'
    } else if (payload.state === 'results') {
      clearInterval(timerInterval)
      displayMode.value = 'quiz'
    } else if (payload.state === 'finished') {
      // Show screensaver when quiz is finished
      displayMode.value = 'screensaver'
    } else if (payload.state === 'lobby') {
      // Show screensaver when quiz is in lobby (not started)
      displayMode.value = 'screensaver'
    } else if (payload.state === 'get-ready') {
      displayMode.value = 'quiz'
    } else if (payload.state === 'leaderboard') {
      displayMode.value = 'quiz'
    }
  })

  quizWS.on('sync:minimal', (payload) => {
    if (payload.answerCounts) {
      answerCounts.value = {
        0: payload.answerCounts["0"] || 0,
        1: payload.answerCounts["1"] || 0,
        2: payload.answerCounts["2"] || 0,
        3: payload.answerCounts["3"] || 0,
      }
    }
  })

  quizWS.on('state:update', (session) => {
    if (session.Players || session.players) {
      const playersMap = session.Players || session.players
      players.value = Object.values(playersMap)
    }
    
    if (session.PlayerSignals || session.playerSignals) {
      const signals = session.PlayerSignals || session.playerSignals
      playerSignals.value = signals.filter(s => s.questionIndex === currentQuestionIndex.value)
    }
  })

  quizWS.on('data:batch', (batch) => {
    if (batch.questions) {
      questions.value = batch.questions
    }
  })
  
  // Handle disconnection - show screensaver
  quizWS.on('disconnect', () => {
    displayMode.value = 'screensaver'
  })

  // Connect as bigscreen
  quizWS.connect(moduleId, '', false, true)
}

const retryConnection = () => {
  connect()
}

const onImageLoad = () => {
  // Image loaded successfully
}

onMounted(async () => {
  // Load screensaver config first
  await loadScreensaverConfig()
  
  // Show screensaver first (before connecting to WebSocket)
  displayMode.value = 'screensaver'
  
  // Only connect to WebSocket if we have a moduleId (quiz mode)
  if (moduleId) {
    // Connect to WebSocket after a brief delay to allow screensaver to render
    setTimeout(() => {
      connect()
    }, 500)
  }
  // If no moduleId, just show screensaver (static display mode)
})

onUnmounted(() => {
  clearInterval(timerInterval)
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped>
.big-screen {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background-color: #121212;
  color: white;
}

/* Screensaver Styles */
.screensaver-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #000;
}

.screensaver-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  animation: fadeIn 1s ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.screensaver-placeholder {
  text-align: center;
  color: #666;
}

.screensaver-placeholder .hint {
  font-size: 1.2rem;
  margin-top: 1rem;
}

/* Connecting State */
.connecting-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-content {
  text-align: center;
}

.loader {
  border: 4px solid #333;
  border-top: 4px solid #4CAF50;
  border-radius: 50%;
  width: 50px;
  height: 50px;
  animation: spin 1s linear infinite;
  margin: 20px auto;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Error State */
.error-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.retry-btn {
  margin-top: 20px;
  padding: 12px 24px;
  background-color: #4CAF50;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1.2rem;
}

/* Quiz State Styles - Reuse from QuizHostScreen */
.state-container {
  width: 100%;
  height: 100%;
  padding: 2vw;
  box-sizing: border-box;
}

/* Lobby */
.lobby-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.lobby-header {
  text-align: center;
  margin-bottom: 3vh;
}

.game-title {
  font-size: 5vw;
  margin-bottom: 2vh;
}

.pin-display {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1vw;
  margin: 2vh 0;
}

.pin-label {
  font-size: 2vw;
  color: #888;
}

.pin-code {
  font-size: 4vw;
  font-weight: bold;
  color: #4CAF50;
  background: #1e1e1e;
  padding: 1vw 2vw;
  border-radius: 1vw;
}

.qr-placeholder {
  width: 15vw;
  height: 15vw;
  background: #1e1e1e;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 1vw;
  margin: 0 auto;
}

.players-area {
  text-align: center;
  width: 80%;
}

.players-count {
  font-size: 2vw;
  margin-bottom: 2vh;
}

.players-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 1vw;
}

.player-chip {
  background: #333;
  padding: 0.5vw 1vw;
  border-radius: 2vw;
  font-size: 1.5vw;
}

/* Question */
.question-container {
  display: flex;
  flex-direction: column;
}

.question-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2vh;
}

.timer {
  font-size: 8vw;
  font-weight: bold;
  color: #4CAF50;
}

.timer.timer-low {
  color: #f44336;
  animation: pulse 0.5s infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.question-counter {
  font-size: 3vw;
  color: #888;
}

.question-main-content {
  display: flex;
  gap: 2vw;
  flex: 1;
  min-height: 0;
}

.question-column {
  flex: 1;
  display: flex;
}

.question-card {
  background: #1e1e1e;
  border-radius: 2vw;
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.question-image-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
}

.question-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-overlay-gradient {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50%;
  background: linear-gradient(to top, #1e1e1e, transparent);
}

.question-text-overlay {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  height: 100%;
  padding: 2vw;
}

.question-text {
  text-align: center;
  margin: 0;
}

.answers-column {
  width: 40%;
  display: flex;
  flex-direction: column;
  gap: 1vw;
}

.answer-card-stacked {
  flex: 1;
  background: #1e1e1e;
  border-radius: 1vw;
  display: flex;
  align-items: center;
  padding: 0 2vw;
  gap: 1vw;
}

.answer-icon {
  font-size: 3vw;
}

.answer-text {
  flex: 1;
}

.color-0 { background: linear-gradient(135deg, #e91e63, #f06292); }
.color-1 { background: linear-gradient(135deg, #2196f3, #64b5f6); }
.color-2 { background: linear-gradient(135deg, #ff9800, #ffb74d); }
.color-3 { background: linear-gradient(135deg, #4caf50, #81c784); }
.color-true { background: linear-gradient(135deg, #4caf50, #81c784); }
.color-false { background: linear-gradient(135deg, #f44336, #e57373); }

.answer-counter-widget {
  position: fixed;
  bottom: 2vw;
  right: 2vw;
  background: #1e1e1e;
  padding: 1vw 2vw;
  border-radius: 1vw;
  text-align: center;
}

.counter-value {
  font-size: 3vw;
  font-weight: bold;
}

.counter-label {
  font-size: 1vw;
  color: #888;
}

/* Slider Instruction View */
.slider-instruction-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 2vw;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 16px;
  border: 1px solid rgba(255,255,255,0.1);
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.95); }
  to { opacity: 1; transform: scale(1); }
}

.slider-instruction-text {
  font-size: 3vw;
  font-weight: 800;
  color: #fff;
  margin-bottom: 2vh;
  text-shadow: 0 2px 4px rgba(0,0,0,0.5);
}

.slider-demo-track {
  width: 80%;
  height: 4vh;
  background: rgba(255,255,255,0.2);
  border-radius: 2vh;
  position: relative;
  overflow: visible;
}

.slider-demo-thumb {
  position: absolute;
  top: 50%;
  left: 0;
  transform: translate(-50%, -50%);
  width: 5vh;
  height: 5vh;
  background: #FF9500;
  border-radius: 50%;
  box-shadow: 0 0 20px rgba(255, 149, 0, 0.5);
  animation: slide-demo 2s infinite ease-in-out;
}

@keyframes slide-demo {
  0% { left: 0%; opacity: 0; }
  10% { opacity: 1; }
  50% { left: 100%; opacity: 1; }
  90% { opacity: 1; }
  100% { left: 100%; opacity: 0; }
}

/* Slider Results View */
.slider-results-view {
  width: 90%;
  margin: 2vh 0;
}

.slider-result-container {
  position: relative;
  height: 40vh;
  padding: 2vw;
}

.slider-result-track {
  position: absolute;
  top: 50%;
  left: 5%;
  right: 5%;
  height: 8px;
  background: #333;
  transform: translateY(-50%);
  border-radius: 4px;
}

.slider-distribution-overlay {
  position: absolute;
  top: 10%;
  left: 5%;
  right: 5%;
  bottom: 20%;
  display: flex;
  align-items: flex-end;
}

.distribution-bar-wrapper {
  position: absolute;
  bottom: 0;
  height: 100%;
  display: flex;
  align-items: flex-end;
}

.distribution-bar {
  width: 80%;
  margin: 0 auto;
  background: #555;
  border-radius: 4px 4px 0 0;
  transition: height 0.3s ease;
  position: relative;
}

.distribution-bar.has-votes {
  background: linear-gradient(to top, #FF9500, #FFB74D);
}

.bar-tooltip {
  position: absolute;
  top: -2.5vh;
  left: 50%;
  transform: translateX(-50%);
  font-size: 1.2vw;
  font-weight: bold;
  color: #FF9500;
}

.correct-answer-marker {
  position: absolute;
  top: 5%;
  transform: translateX(-50%);
  z-index: 10;
  opacity: 0;
  transition: all 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.correct-answer-marker.reveal-marker {
  opacity: 1;
}

.marker-line {
  width: 4px;
  height: 45vh;
  background: #4CAF50;
  margin: 0 auto;
}

.marker-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: #4CAF50;
  padding: 1vh 1.5vw;
  border-radius: 8px;
  white-space: nowrap;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.marker-title {
  font-size: 1vw;
  font-weight: bold;
  color: white;
}

.marker-value {
  font-size: 1.5vw;
  font-weight: bold;
  color: white;
}

.slider-labels {
  position: absolute;
  bottom: 0;
  left: 5%;
  right: 5%;
  display: flex;
  justify-content: space-between;
  font-size: 1.5vw;
  color: #888;
}

/* Results */
.results-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2vw;
}

.results-title {
  font-size: 4vw;
  margin-bottom: 2vh;
}

.correct-answer-reveal {
  text-align: center;
  margin-bottom: 2vh;
}

.correct-answer-display {
  font-size: 3vw;
  padding: 1vw 2vw;
  border-radius: 1vw;
  display: inline-block;
}

.answers-stats {
  width: 80%;
  margin: 2vh 0;
}

.stats-grid {
  display: flex;
  gap: 2vw;
  height: 30vh;
}

.stat-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-bar-container {
  flex: 1;
  width: 100%;
  background: #1e1e1e;
  border-radius: 1vw;
  display: flex;
  align-items: flex-end;
  overflow: hidden;
}

.stat-bar {
  width: 100%;
  border-radius: 1vw;
  transition: height 0.3s;
}

.stat-count {
  font-size: 2vw;
  margin: 1vh 0;
}

.stat-icon {
  font-size: 2vw;
}

/* Slider Histogram */
.slider-stats {
  width: 80%;
}

.histogram-container {
  height: 30vh;
}

.histogram-track-host {
  height: 80%;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  background: #1e1e1e;
  border-radius: 1vw;
  padding: 1vw;
}

.histogram-bar-host {
  flex: 1;
  background: #555;
  border-radius: 4px 4px 0 0;
  position: relative;
}

.histogram-bar-host.bar-correct {
  background: #4caf50;
}

.bar-count {
  position: absolute;
  top: -2vh;
  left: 50%;
  transform: translateX(-50%);
  font-size: 1vw;
}

.histogram-axis {
  display: flex;
  justify-content: space-between;
  padding: 1vw;
  font-size: 1.5vw;
}

.correct-marker-label {
  position: absolute;
  transform: translateX(-50%);
  color: #4caf50;
  font-weight: bold;
}

.leaderboard-list {
  width: 60%;
}

.leaderboard-item {
  display: flex;
  align-items: center;
  background: #1e1e1e;
  padding: 1vw 2vw;
  border-radius: 1vw;
  margin-bottom: 1vh;
}

.rank {
  font-size: 3vw;
  font-weight: bold;
  width: 10%;
  color: #4CAF50;
}

.name {
  flex: 1;
  font-size: 2vw;
}

.score {
  font-size: 2vw;
  color: #ff9800;
}

/* Finished */
.finished-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.finished-container h1 {
  font-size: 5vw;
  margin-bottom: 3vh;
}

.podium {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 2vw;
}

.podium-place {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.podium-bar {
  background: #1e1e1e;
  padding: 2vw;
  border-radius: 1vw;
  text-align: center;
  display: flex;
  flex-direction: column;
}

.podium-rank {
  font-size: 4vw;
  font-weight: bold;
}

.podium-place.first .podium-rank { color: #ffd700; }
.podium-place.second .podium-rank { color: #c0c0c0; }
.podium-place.third .podium-rank { color: #cd7f32; }

.podium-name {
  font-size: 2vw;
  margin: 1vh 0;
}

.podium-score {
  font-size: 1.5vw;
  color: #ff9800;
}

.podium-place.first { height: 30vh; }
.podium-place.first .podium-bar { height: 100%; }
.podium-place.second { height: 25vh; }
.podium-place.second .podium-bar { height: 80%; }
.podium-place.third { height: 20vh; }
.podium-place.third .podium-bar { height: 60%; }
</style>
