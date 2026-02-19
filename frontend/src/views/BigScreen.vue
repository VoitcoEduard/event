<template>
  <div class="big-screen">
    <div class="language-switcher-container">
      <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

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
        <p>{{ $t('big_screen.no_screensaver') }}</p>
        <p class="hint">{{ $t('big_screen.screensaver_hint') }}</p>
      </div>
    </div>

    <!-- Connecting State -->
    <div v-else-if="displayMode === 'connecting'" class="connecting-container">
      <div class="loading-content">
        <h1>{{ $t('big_screen.connecting') }}</h1>
        <div class="loader"></div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="displayMode === 'error'" class="error-container">
      <h1>{{ $t('big_screen.conn_error') }}</h1>
      <p>{{ errorMessage }}</p>
      <button @click="retryConnection" class="retry-btn">{{ $t('big_screen.retry') }}</button>
    </div>

    <!-- Quiz Display States -->
    <template v-else>
      <!-- LOBBY STATE -->
      <div v-if="gameState === 'lobby'" class="state-container lobby-container">
        <div class="lobby-header">
          <h1 class="game-title">{{ $t('quiz.host.join_title') }}</h1>
          <div class="pin-display">
            <span class="pin-label">{{ $t('quiz.host.game_pin') }}</span>
            <span class="pin-code">{{ pinCode }}</span>
          </div>
          <div class="qr-container">
            <div class="qr-placeholder">{{ $t('quiz.host.qr_placeholder') }}</div>
          </div>
        </div>
        
        <div class="players-area">
          <div class="players-count">
            <span>👥 {{ players.length }}</span> {{ $t('quiz.host.players') }}
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
          <h1>{{ $t('quiz.player.get_ready') }}</h1>
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

        <!-- CORRECT PICTURE LAYOUT -->
        <div v-if="isCorrectPicture" class="correct-picture-layout">
           <div class="cp-images-grid">
              <div 
                v-for="(option, idx) in currentQuestion.imageOptions" 
                :key="idx"
                class="cp-image-card"
              >
                 <img :src="getImageUrl(option.imageId)" class="cp-image" />
                 <div class="cp-number-badge">{{ idx + 1 }}</div>
              </div>
           </div>
           <div class="cp-question-text">
              <h2>{{ currentQuestion.text }}</h2>
           </div>
        </div>

        <div v-else class="question-main-content">
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
              <h2 class="slider-instruction-text">{{ $t('quiz.host.slide_to_answer') }}</h2>
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
          <div class="counter-label">{{ $t('quiz.host.answers_label') }}</div>
        </div>
      </div>

      <!-- RESULTS STATE (no leaderboard here) -->
      <div v-else-if="gameState === 'results'" class="state-container results-container">
        <div class="results-header">
          <span class="results-label">{{ $t('quiz.host.results_title', { num: currentQuestionIndex + 1 }) }}</span>
        </div>

        <!-- CORRECT PICTURE RESULTS -->
        <div v-if="isCorrectPicture" class="correct-picture-results">
            <div class="cp-results-grid">
              <div 
                v-for="(option, idx) in currentQuestion.imageOptions" 
                :key="idx"
                class="cp-result-card"
                :class="{ 
                    'is-correct': idx === currentQuestion.correctIndex,
                    'is-wrong': idx !== currentQuestion.correctIndex
                }"
              >
                 <div class="cp-image-wrapper">
                    <img :src="getImageUrl(option.imageId)" class="cp-image" />
                    <div v-if="idx === currentQuestion.correctIndex" class="cp-correct-badge">✓</div>
                 </div>
                 
                 <div class="cp-stats-bar">
                    <div class="cp-bar-fill" :style="{ width: getBarHeight(idx) + '%' }"></div> <!-- repurposing getBarHeight logic but for width -->
                    <span class="cp-count">{{ answerCounts[idx] || 0 }}</span>
                 </div>
              </div>
           </div>
           <div class="cp-question-text results-mode">
              <h2>{{ currentQuestion.text }}</h2>
           </div>
        </div>

        <div v-else-if="!isSlider" class="correct-answer-reveal">
          <p class="correct-label">{{ $t('quiz.host.correct_answer') }}</p>
          <div class="correct-answer-display" :class="getAnswerColorClass(currentQuestion?.correctIndex)">
            <span class="correct-icon">{{ getShapeIcon(currentQuestion?.correctIndex) }}</span>
            <span class="correct-text">{{ currentQuestion?.answers[currentQuestion?.correctIndex] }}</span>
          </div>
        </div>

        <div v-if="!isSlider" class="answers-stats results-mode">
          <div class="stats-grid">
            <div v-for="(count, idx) in (isTrueFalse ? 2 : 4)" :key="idx" class="stat-column">
              <div class="stat-bar-container">
                <div 
                  class="stat-bar" 
                  :class="[getAnswerColorClass(idx), idx === currentQuestion?.correctIndex ? 'stat-bar-correct' : '']"
                  :style="{ height: getBarHeight(idx) + '%' }"
                ></div>
              </div>
              <div class="stat-count">{{ answerCounts[idx] || 0 }}</div>
              <div class="stat-icon">{{ getShapeIcon(idx) }}</div>
            </div>
          </div>
        </div>

        <!-- Slider Histogram Results -->
        <div v-if="isSlider" class="answers-stats results-mode slider-results-view">
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
                <span class="marker-title">{{ $t('quiz.host.correct_label') }}</span>
                <span class="marker-value">{{ formatNumber(currentQuestion?.correctValue) }}</span>
              </div>
            </div>
            <div class="slider-labels">
              <span class="label-min">{{ formatNumber(currentQuestion?.sliderConfig?.minValue) }}</span>
              <span class="label-max">{{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- LEADERBOARD STATE (beautiful animated) -->
      <div v-else-if="gameState === 'leaderboard'" class="state-container leaderboard-container" :key="leaderboardAnimKey">
        <div class="lb-bg-glow"></div>
        <div class="lb-header">
          <div class="lb-trophy">🏆</div>
          <h2 class="lb-title">{{ $t('quiz.host.leaderboard_title') }}</h2>
          <div class="lb-subtitle">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
        </div>
        <div class="lb-list">
          <div 
            v-for="(player, idx) in topPlayers" 
            :key="player.id" 
            class="lb-row"
            :class="['lb-rank-' + (idx + 1)]"
            :style="{ animationDelay: ((topPlayers.length - 1 - idx) * 0.12) + 's' }"
          >
            <div class="lb-rank-badge" :class="'lb-badge-' + (idx + 1)">
              <span v-if="idx === 0">🥇</span>
              <span v-else-if="idx === 1">🥈</span>
              <span v-else-if="idx === 2">🥉</span>
              <span v-else>{{ idx + 1 }}</span>
            </div>
            <div class="lb-player-name">{{ player.nickname }}</div>
            <div class="lb-score-area">
              <div class="lb-score-bar-wrap">
                <div 
                  class="lb-score-bar" 
                  :class="'lb-bar-' + (idx + 1)"
                  :style="{ width: getScoreBarWidth(player.score) + '%' }"
                ></div>
              </div>
              <div class="lb-score-value">{{ player.score.toLocaleString() }}</div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- FINISHED STATE (final leaderboard with confetti) -->
      <div v-else-if="gameState === 'finished'" class="state-container final-container" :key="finalAnimKey">
        <!-- Confetti particles -->
        <div class="confetti-wrap">
          <div v-for="i in 30" :key="i" class="confetti-particle" :style="getConfettiStyle(i)"></div>
        </div>

        <div class="final-header">
          <div class="final-trophy">🏆</div>
          <h1 class="final-title">{{ $t('quiz.host.quiz_complete') }}</h1>
          <p class="final-subtitle">{{ $t('quiz.host.final_standings') }}</p>
        </div>

        <!-- Top 3 Podium -->
        <div class="final-podium">
          <div class="final-podium-place fp-second" v-if="topPlayers[1]">
            <div class="fp-avatar">🥈</div>
            <div class="fp-name">{{ topPlayers[1].nickname }}</div>
            <div class="fp-score">{{ topPlayers[1].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-2"><span>2</span></div>
          </div>
          <div class="final-podium-place fp-first" v-if="topPlayers[0]">
            <div class="fp-crown">👑</div>
            <div class="fp-avatar">🥇</div>
            <div class="fp-name">{{ topPlayers[0].nickname }}</div>
            <div class="fp-score">{{ topPlayers[0].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-1"><span>1</span></div>
          </div>
          <div class="final-podium-place fp-third" v-if="topPlayers[2]">
            <div class="fp-avatar">🥉</div>
            <div class="fp-name">{{ topPlayers[2].nickname }}</div>
            <div class="fp-score">{{ topPlayers[2].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-3"><span>3</span></div>
          </div>
        </div>

        <!-- Ranks 4-6 -->
        <div class="final-rest" v-if="topPlayers.length > 3">
          <div 
            v-for="(player, idx) in topPlayers.slice(3)" 
            :key="player.id" 
            class="final-rest-row"
            :style="{ animationDelay: (0.8 + idx * 0.15) + 's' }"
          >
            <span class="final-rest-rank">{{ idx + 4 }}</span>
            <span class="final-rest-name">{{ player.nickname }}</span>
            <span class="final-rest-score">{{ player.score.toLocaleString() }}</span>
          </div>
        </div>
      </div>
    </template>
    <!-- Audio Player -->
    <audio ref="bgMusicPlayer" :src="fullMusicUrl" loop id="bg-music-player"></audio>
    
    <div v-if="showUnmuteButton" class="unmute-overlay">
        <button @click="enableAudio" class="unmute-btn">
            {{ $t('big_screen.enable_sound') }}
        </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { getScreensavers } from '../services/media'

const { locale } = useI18n()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

// Display mode: 'connecting', 'screensaver', 'quiz', 'error'
const displayMode = ref('connecting')
const errorMessage = ref('')
const screensaverImageUrl = ref('')

// Audio State
const bgMusicPlayer = ref(null)
const backgroundMusicUrl = ref('')
const audioOutput = ref('bigscreen')
const showUnmuteButton = ref(false)
const isMusicPlaying = ref(false)

const fullMusicUrl = computed(() => {
    if (!backgroundMusicUrl.value) return ''
    return `${API_BASE_URL}${backgroundMusicUrl.value}`
})

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
const isCorrectPicture = computed(() => currentQuestion.value?.type === 'correct_picture')
const revealCorrect = ref(false)

// Watch gameState to trigger animation
watch(gameState, (newVal) => {
    if (newVal === 'results' && isSlider.value) {
        revealCorrect.value = false
        setTimeout(() => {
            revealCorrect.value = true
        }, 1500)
    } else {
        revealCorrect.value = false
    }

    // Re-trigger leaderboard animation each time we enter leaderboard state
    if (newVal === 'leaderboard') {
        leaderboardAnimKey.value++
    }
    if (newVal === 'finished') {
        finalAnimKey.value++
    }
    
    // Manage Audio Playback based on state
    manageAudioPlayback()
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
  return [...players.value].sort((a, b) => b.score - a.score).slice(0, 6)
})

const leaderboardAnimKey = ref(0)
const finalAnimKey = ref(0)

const getScoreBarWidth = (score) => {
  const max = topPlayers.value[0]?.score || 1
  return Math.max(5, (score / max) * 100)
}

const getConfettiStyle = (i) => {
  const colors = ['#FFD700', '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FFEAA7', '#DDA0DD', '#98D8C8']
  const color = colors[i % colors.length]
  const left = ((i * 37) % 100)
  const delay = ((i * 0.13) % 3)
  const duration = 2.5 + (i % 3) * 0.5
  const size = 8 + (i % 4) * 4
  return {
    left: left + '%',
    animationDelay: delay + 's',
    animationDuration: duration + 's',
    background: color,
    width: size + 'px',
    height: size + 'px',
    borderRadius: i % 3 === 0 ? '50%' : '2px',
  }
}

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

// Load Module Config (for music)
const loadModuleConfig = async () => {
    if (!moduleId) return
    try {
        const res = await fetch(`${API_BASE_URL}/events/${eventId}/modules/${moduleId}`)
        if (res.ok) {
            const mod = await res.json()
            if (mod.data) {
                backgroundMusicUrl.value = mod.data.backgroundMusicUrl || ''
                audioOutput.value = mod.data.audioOutput || 'bigscreen'
                manageAudioPlayback()
            }
        }
    } catch (e) {
        console.error("Failed to load module config for audio", e)
    }
}

const manageAudioPlayback = async () => {
    // Determine if we should play
    // Only play music during quiz (not in screensaver mode after quiz ends)
    const shouldPlay = 
        backgroundMusicUrl.value && 
        (audioOutput.value === 'bigscreen' || audioOutput.value === 'both') &&
        displayMode.value === 'quiz'

    const player = bgMusicPlayer.value
    if (!player) return

    if (shouldPlay) {
        if (player.paused) {
            try {
                await player.play()
                isMusicPlaying.value = true
                showUnmuteButton.value = false
            } catch (err) {
                console.warn("Autoplay prevented:", err)
                showUnmuteButton.value = true
            }
        }
    } else {
        if (!player.paused) {
            player.pause()
            isMusicPlaying.value = false
        }
    }
}

const enableAudio = () => {
    const player = bgMusicPlayer.value
    if (player) {
        player.play().then(() => {
            isMusicPlaying.value = true
            showUnmuteButton.value = false
        }).catch(err => console.error("Still cannot play:", err))
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

    // Try to autoplay music when quiz starts (get-ready or question-active)
    if (payload.state === 'get-ready' || payload.state === 'question-active') {
      displayMode.value = 'quiz'
      // Try to autoplay when quiz starts - reload audio first
      const player = bgMusicPlayer.value
      if (player && backgroundMusicUrl.value && (audioOutput.value === 'bigscreen' || audioOutput.value === 'both')) {
        player.load()
        player.play().then(() => {
          isMusicPlaying.value = true
          showUnmuteButton.value = false
        }).catch(() => {
          // Silently fail - don't show button on big screen
          showUnmuteButton.value = false
        })
      }
      if (payload.state === 'question-active' && (prevState !== 'question-active' || payload.currentIndex !== currentQuestionIndex.value)) {
        startLocalTimer(currentQuestion.value?.timeLimit || 20)
      }
    } else if (payload.state === 'results') {
      clearInterval(timerInterval)
      displayMode.value = 'quiz'
    } else if (payload.state === 'finished') {
      // Show screensaver when quiz is finished
      displayMode.value = 'screensaver'
    } else if (payload.state === 'lobby') {
      // Show screensaver when quiz is in lobby (not started)
      displayMode.value = 'screensaver'
    } else if (payload.state === 'leaderboard') {
      displayMode.value = 'quiz'
    }

    manageAudioPlayback()
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
    manageAudioPlayback()
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
  await loadModuleConfig()
  
  // Show screensaver first (before connecting to WebSocket)
  displayMode.value = 'screensaver'
  manageAudioPlayback()
  
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
  position: relative;
}

.language-switcher-container {
    position: absolute;
    top: 20px;
    right: 20px;
    z-index: 1000;
}

.lang-select {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.2);
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    outline: none;
    backdrop-filter: blur(10px);
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

/* CORRECT PICTURE STYLES */
.correct-picture-layout {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 20px;
}

.cp-images-grid {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;
    padding: 20px;
    align-items: center;
    justify-content: center;
}

.cp-image-card {
    position: relative;
    height: 100%;
    max-height: 50vh;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #1C1C1E;
    border-radius: 12px;
    overflow: hidden;
    border: 2px solid #333;
}

.cp-image {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
}

.cp-number-badge {
    position: absolute;
    top: 10px;
    left: 10px;
    width: 40px;
    height: 40px;
    background: rgba(0,0,0,0.7);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 900;
    font-size: 1.5rem;
    border: 2px solid white;
}

.cp-question-text {
    flex: 0 0 15%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0,0,0,0.5);
    border-radius: 12px;
    margin-top: 20px;
    padding: 20px;
}

.cp-question-text h2 {
    font-size: 3rem;
    text-align: center;
    margin: 0;
}

/* CP RESULTS */
.correct-picture-results {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.cp-results-grid {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    padding: 20px;
}

.cp-result-card {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.cp-image-wrapper {
    flex: 1;
    position: relative;
    border-radius: 12px;
    overflow: hidden;
    border: 2px solid #333;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #1C1C1E;
}

.cp-result-card.is-wrong .cp-image-wrapper {
    filter: grayscale(100%);
    opacity: 0.5;
}

.cp-result-card.is-correct .cp-image-wrapper {
    border-color: #32D74B;
    box-shadow: 0 0 30px rgba(50, 215, 75, 0.4);
}

.cp-correct-badge {
    position: absolute;
    bottom: 10px;
    right: 10px;
    background: #32D74B;
    color: white;
    width: 50px;
    height: 50px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2rem;
    font-weight: 900;
    box-shadow: 0 4px 10px rgba(0,0,0,0.3);
}

.cp-stats-bar {
    height: 40px;
    background: #333;
    margin-top: 10px;
    border-radius: 20px;
    position: relative;
    overflow: hidden;
}

.cp-bar-fill {
    height: 100%;
    background: #0A84FF;
    transition: width 1s ease-out;
}

.cp-result-card.is-correct .cp-bar-fill {
    background: #32D74B;
}

.cp-count {
    position: absolute;
    right: 15px;
    top: 50%;
    transform: translateY(-50%);
    font-weight: bold;
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
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

/* ===== RESULTS STATE ===== */
.results-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2vw;
  background: radial-gradient(ellipse at top, #1a1a2e 0%, #0d0d0d 70%);
}

.results-header {
  margin-bottom: 2vh;
}

.results-label {
  font-size: 1.8vw;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.15em;
  font-weight: 600;
}

.correct-label {
  font-size: 1.5vw;
  color: #aaa;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: 1vh;
}

.correct-answer-reveal {
  text-align: center;
  margin-bottom: 2.5vh;
  animation: popIn 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.correct-answer-display {
  font-size: 2.8vw;
  padding: 1.2vw 3vw;
  border-radius: 1.5vw;
  display: inline-flex;
  align-items: center;
  gap: 1vw;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
}

.correct-icon { font-size: 2.5vw; }
.correct-text { font-weight: 700; }

.answers-stats {
  width: 80%;
  margin: 1vh 0;
}

.stats-grid {
  display: flex;
  gap: 2vw;
  height: 32vh;
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
  background: rgba(255,255,255,0.05);
  border-radius: 1vw;
  display: flex;
  align-items: flex-end;
  overflow: hidden;
  border: 1px solid rgba(255,255,255,0.08);
}

.stat-bar {
  width: 100%;
  border-radius: 1vw;
  transition: height 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.stat-bar-correct {
  box-shadow: 0 0 20px rgba(255,255,255,0.3);
  position: relative;
}

.stat-count {
  font-size: 2vw;
  margin: 1vh 0;
  font-weight: 700;
}

.stat-icon {
  font-size: 2vw;
}

/* ===== LEADERBOARD STATE ===== */
.leaderboard-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2vw 4vw;
  position: relative;
  overflow: hidden;
  background: radial-gradient(ellipse at 50% 0%, #1a0a3e 0%, #0a0a0a 70%);
}

.lb-bg-glow {
  position: absolute;
  top: -10vh;
  left: 50%;
  transform: translateX(-50%);
  width: 60vw;
  height: 40vh;
  background: radial-gradient(ellipse, rgba(138, 43, 226, 0.15) 0%, transparent 70%);
  pointer-events: none;
}

.lb-header {
  text-align: center;
  margin-bottom: 3vh;
  animation: fadeSlideDown 0.6s ease-out;
}

.lb-trophy {
  font-size: 4vw;
  margin-bottom: 0.5vh;
  filter: drop-shadow(0 0 20px rgba(255, 215, 0, 0.6));
  animation: trophyPulse 2s ease-in-out infinite;
}

@keyframes trophyPulse {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 20px rgba(255, 215, 0, 0.6)); }
  50% { transform: scale(1.05); filter: drop-shadow(0 0 30px rgba(255, 215, 0, 0.9)); }
}

.lb-title {
  font-size: 3.5vw;
  font-weight: 900;
  background: linear-gradient(135deg, #FFD700, #FFA500);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0;
  letter-spacing: 0.05em;
}

.lb-subtitle {
  font-size: 1.4vw;
  color: #888;
  margin-top: 0.5vh;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

.lb-list {
  width: 100%;
  max-width: 70vw;
  display: flex;
  flex-direction: column;
  gap: 1.2vh;
}

.lb-row {
  display: flex;
  align-items: center;
  gap: 2vw;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 1.2vw;
  padding: 1.2vh 2vw;
  animation: slideInFromBottom 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  backdrop-filter: blur(10px);
  transition: transform 0.2s;
}

.lb-rank-1 { border-color: rgba(255, 215, 0, 0.4); background: rgba(255, 215, 0, 0.08); }
.lb-rank-2 { border-color: rgba(192, 192, 192, 0.3); background: rgba(192, 192, 192, 0.05); }
.lb-rank-3 { border-color: rgba(205, 127, 50, 0.3); background: rgba(205, 127, 50, 0.05); }

@keyframes slideInFromBottom {
  from { opacity: 0; transform: translateY(4vh) scale(0.95); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

.lb-rank-badge {
  width: 4vw;
  height: 4vw;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 2.2vw;
  font-weight: 900;
  flex-shrink: 0;
  background: rgba(255,255,255,0.1);
  color: #ccc;
}

.lb-badge-1 { background: rgba(255, 215, 0, 0.2); font-size: 2.5vw; }
.lb-badge-2 { background: rgba(192, 192, 192, 0.2); font-size: 2.5vw; }
.lb-badge-3 { background: rgba(205, 127, 50, 0.2); font-size: 2.5vw; }

.lb-player-name {
  flex: 1;
  font-size: 2.2vw;
  font-weight: 700;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lb-score-area {
  display: flex;
  align-items: center;
  gap: 1.5vw;
  min-width: 30%;
}

.lb-score-bar-wrap {
  flex: 1;
  height: 0.8vh;
  background: rgba(255,255,255,0.1);
  border-radius: 1vw;
  overflow: hidden;
}

.lb-score-bar {
  height: 100%;
  border-radius: 1vw;
  transition: width 1s cubic-bezier(0.34, 1.56, 0.64, 1);
  background: linear-gradient(90deg, #8B5CF6, #A78BFA);
}

.lb-bar-1 { background: linear-gradient(90deg, #FFD700, #FFA500); }
.lb-bar-2 { background: linear-gradient(90deg, #C0C0C0, #E8E8E8); }
.lb-bar-3 { background: linear-gradient(90deg, #CD7F32, #E8A87C); }

.lb-score-value {
  font-size: 2vw;
  font-weight: 800;
  color: #FFD700;
  min-width: 8vw;
  text-align: right;
}

/* ===== FINAL LEADERBOARD (FINISHED) ===== */
.final-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1.5vw 3vw;
  position: relative;
  overflow: hidden;
  background: radial-gradient(ellipse at 50% 0%, #1a0a3e 0%, #0a0a0a 70%);
}

/* Confetti */
.confetti-wrap {
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  pointer-events: none;
  overflow: hidden;
}

.confetti-particle {
  position: absolute;
  top: -20px;
  animation: confettiFall linear infinite;
  opacity: 0.85;
}

@keyframes confettiFall {
  0% { transform: translateY(-20px) rotate(0deg); opacity: 1; }
  100% { transform: translateY(110vh) rotate(720deg); opacity: 0; }
}

.final-header {
  text-align: center;
  margin-bottom: 2vh;
  animation: fadeSlideDown 0.7s ease-out;
  z-index: 1;
}

.final-trophy {
  font-size: 4vw;
  filter: drop-shadow(0 0 30px rgba(255, 215, 0, 0.8));
  animation: trophyPulse 2s ease-in-out infinite;
}

.final-title {
  font-size: 4vw;
  font-weight: 900;
  background: linear-gradient(135deg, #FFD700, #FF6B6B, #4ECDC4);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0.5vh 0;
  letter-spacing: 0.05em;
}

.final-subtitle {
  font-size: 1.5vw;
  color: #aaa;
  text-transform: uppercase;
  letter-spacing: 0.15em;
  margin: 0;
}

/* Final Podium */
.final-podium {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 2vw;
  margin-bottom: 2vh;
  z-index: 1;
}

.final-podium-place {
  display: flex;
  flex-direction: column;
  align-items: center;
  animation: podiumRise 0.8s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.fp-first { animation-delay: 0.5s; }
.fp-second { animation-delay: 0.2s; }
.fp-third { animation-delay: 0.35s; }

@keyframes podiumRise {
  from { opacity: 0; transform: translateY(5vh) scale(0.8); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

.fp-crown {
  font-size: 3vw;
  animation: crownFloat 2s ease-in-out infinite;
  filter: drop-shadow(0 0 10px rgba(255, 215, 0, 0.8));
}

@keyframes crownFloat {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-0.5vh); }
}

.fp-avatar {
  font-size: 3.5vw;
  margin-bottom: 0.5vh;
}

.fp-name {
  font-size: 1.8vw;
  font-weight: 700;
  color: #fff;
  text-align: center;
  max-width: 15vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 0.3vh;
}

.fp-score {
  font-size: 1.5vw;
  font-weight: 800;
  margin-bottom: 0.5vh;
}

.fp-first .fp-score { color: #FFD700; }
.fp-second .fp-score { color: #C0C0C0; }
.fp-third .fp-score { color: #CD7F32; }

.fp-block {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.8vw 0.8vw 0 0;
  font-size: 2.5vw;
  font-weight: 900;
  color: rgba(0,0,0,0.5);
}

.fp-block-1 {
  width: 14vw;
  height: 9vh;
  background: linear-gradient(180deg, #FFD700, #B8860B);
  box-shadow: 0 0 30px rgba(255, 215, 0, 0.5);
}

.fp-block-2 {
  width: 12vw;
  height: 6vh;
  background: linear-gradient(180deg, #C0C0C0, #808080);
  box-shadow: 0 0 20px rgba(192, 192, 192, 0.3);
}

.fp-block-3 {
  width: 12vw;
  height: 4.5vh;
  background: linear-gradient(180deg, #CD7F32, #8B4513);
  box-shadow: 0 0 20px rgba(205, 127, 50, 0.3);
}

/* Ranks 4-6 */
.final-rest {
  display: flex;
  flex-direction: column;
  gap: 0.8vh;
  width: 50vw;
  z-index: 1;
}

.final-rest-row {
  display: flex;
  align-items: center;
  gap: 2vw;
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 1vw;
  padding: 0.8vh 2vw;
  animation: slideInFromBottom 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.final-rest-rank {
  font-size: 1.8vw;
  font-weight: 900;
  color: #888;
  width: 3vw;
  text-align: center;
}

.final-rest-name {
  flex: 1;
  font-size: 1.8vw;
  font-weight: 600;
  color: #ddd;
}

.final-rest-score {
  font-size: 1.8vw;
  font-weight: 700;
  color: #aaa;
}

/* Shared animations */
@keyframes fadeSlideDown {
  from { opacity: 0; transform: translateY(-2vh); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes popIn {
  0% { opacity: 0; transform: scale(0.7); }
  70% { transform: scale(1.05); }
  100% { opacity: 1; transform: scale(1); }
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

/* Audio Controls */
.unmute-overlay {
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 1000;
}

.unmute-btn {
    background-color: rgba(255, 255, 255, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.4);
    color: white;
    padding: 10px 16px;
    border-radius: 20px;
    cursor: pointer;
    font-size: 1rem;
    backdrop-filter: blur(10px);
    transition: all 0.2s;
}

.unmute-btn:hover {
    background-color: rgba(255, 255, 255, 0.3);
    transform: scale(1.05);
}
</style>
