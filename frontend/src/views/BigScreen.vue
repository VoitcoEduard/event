<template>
  <div class="big-screen">
    <div class="language-switcher-container">
      <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>
    <!-- Screen Manager Overrides -->
    <div v-if="overrideScreenState?.currentView === 'welcome'" class="screensaver-container welcome-slide-view">
      <div class="welcome-center">
        <h1 class="welcome-title">{{ $t('quiz.host.join_title') }}</h1>
        <div v-if="overrideScreenState.showWelcomeQR" class="qr-container-big">
          <QRCodeVue :value="joinUrl" :size="400" level="H" />
        </div>
        <div class="pin-display-big" :class="{ 'no-qr': !overrideScreenState.showWelcomeQR }">
          {{ $t('quiz.host.game_pin') }} <span>{{ eventId }}</span>
        </div>
        <div class="join-url">{{ joinUrl }}</div>
      </div>
    </div>

    <div v-else-if="overrideScreenState?.currentView === 'custom_text'" class="screensaver-container custom-text-view">
       <h1 class="custom-text-display">{{ overrideScreenState.customText }}</h1>
    </div>

    <!-- New Slide View Mode -->
    <div v-else-if="overrideScreenState?.currentView === 'slide'" class="slide-bigscreen-container" :style="{ backgroundColor: overrideScreenState.activeSlide?.backgroundColor || '#000000' }">
      <!-- Background Image -->
      <img v-if="overrideScreenState.activeSlide?.backgroundImage" :src="getImageUrl(overrideScreenState.activeSlide.backgroundImage)" class="slide-bg-image" />
      
      <!-- Canvas Elements -->
      <div 
        v-for="element in overrideScreenState.activeSlide?.elements || []" 
        :key="element.id"
        class="slide-element"
        :style="{
          left: element.x + '%',
          top: element.y + '%',
          width: element.width + '%',
          height: element.height + '%',
          color: element.color || '#ffffff',
          fontSize: getScaledFontSize(element.fontSize) + 'px',
          textAlign: element.textAlign || 'center',
          fontWeight: element.fontWeight || 'normal'
        }"
      >
        <div v-if="element.type === 'text'" class="slide-text">{{ element.content }}</div>
        <img v-else-if="element.type === 'image'" :src="getImageUrl(element.content)" class="slide-image" />
        <div v-else-if="element.type === 'qr'" class="slide-qr">
           <QRCodeVue :value="joinUrl" :size="300" level="H" />
        </div>
      </div>
    </div>

    <div v-else-if="overrideScreenState?.currentView === 'screensaver'" class="screensaver-container">
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

    <!-- Original Display Modes (if screen state is module) -->
    <div v-else-if="displayMode === 'screensaver'" class="screensaver-container">
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

      <!-- QUESTION STATE (Active & Presenting) -->
      <div v-else-if="gameState === 'question-active' || gameState === 'question-presenting'" class="state-container question-container">
        <div class="question-header">
          <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }" v-if="gameState === 'question-active'">
            {{ timeRemaining }}
          </div>
          <div class="timer" v-else-if="gameState === 'question-presenting'" style="font-size: 1.5rem">
            {{ $t('quiz.player.get_ready') || 'Ready!' }}
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
            <div v-if="isSlider" class="slider-bigscreen-view">
              <!-- Min / Max labels -->
              <div class="sbs-label sbs-label-max">
                {{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}{{ currentQuestion?.sliderConfig?.unit }}
              </div>

              <!-- The track -->
              <div class="sbs-track">
                <div class="sbs-track-fill sbs-fill-anim"></div>
                <!-- Animated pulsing thumb -->
                <div class="sbs-thumb sbs-thumb-pulse">
                  <div class="sbs-thumb-lines"><span></span><span></span><span></span></div>
                </div>
              </div>

              <div class="sbs-label sbs-label-min">
                {{ formatNumber(currentQuestion?.sliderConfig?.minValue) }}{{ currentQuestion?.sliderConfig?.unit }}
              </div>

              <div class="sbs-hint">↕ Scroll to select</div>
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
      <div v-else-if="gameState === 'results'" class="state-container results-container-new">
        <div class="question-header">
           <!-- Reuse header but with Results title -->
           <div class="results-mode-title">{{ $t('quiz.host.results_title', { num: currentQuestionIndex + 1 }) }}</div>
           <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
        </div>

        <!-- CORRECT PICTURE RESULTS -->
        <div v-if="isCorrectPicture" class="correct-picture-results">
          <!-- Correct image hero -->
          <div class="cp-correct-hero">
            <div v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
              <div v-if="isAnswerCorrect(idx)" class="cp-hero-frame">
                <div class="cp-hero-glow"></div>
                <img :src="getImageUrl(option.imageId)" class="cp-hero-img" />
                <div class="cp-hero-check">✓</div>
                <div class="cp-hero-label">{{ currentQuestion.text }}</div>
              </div>
            </div>
          </div>
          <!-- Wrong images shown as small thumbnails -->
          <div class="cp-wrong-thumbnails">
            <template v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
              <div v-if="!isAnswerCorrect(idx)" class="cp-thumb-card">
                <img :src="getImageUrl(option.imageId)" class="cp-thumb-img" />
                <div class="cp-thumb-count">{{ answerCounts[idx] || 0 }}</div>
              </div>
            </template>
          </div>
        </div>

        <!-- SLIDER RESULTS -->
        <div v-else-if="isSlider" class="slider-results-new">
          <!-- Left: question card as usual -->
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

          <!-- Right: vertical slider timeline showing correct answer -->
          <div class="answers-column">
            <div class="slider-result-timeline">
              <!-- Big correct value badge -->
              <div class="srt-correct-badge" :class="{ 'srt-reveal': revealCorrect }">
                <span class="srt-check">✓</span>
                <span class="srt-value">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.sliderConfig?.unit }}</span>
              </div>

              <div class="srt-label srt-label-max">
                {{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}{{ currentQuestion?.sliderConfig?.unit }}
              </div>

              <!-- Track with animated marker -->
              <div class="srt-track">
                <!-- Fill up to correct value -->
                <div 
                  class="srt-fill"
                  :style="{ height: revealCorrect ? correctMarkerPercentage + '%' : '0%' }"
                ></div>
                <!-- Correct answer thumb marker -->
                <div 
                  class="srt-marker"
                  :class="{ 'srt-marker-reveal': revealCorrect }"
                  :style="{ bottom: correctMarkerPercentage + '%' }"
                >
                  <div class="srt-marker-line"></div>
                  <div class="srt-marker-bubble">
                    {{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.sliderConfig?.unit }}
                  </div>
                </div>
              </div>

              <div class="srt-label srt-label-min">
                {{ formatNumber(currentQuestion?.sliderConfig?.minValue) }}{{ currentQuestion?.sliderConfig?.unit }}
              </div>
            </div>
          </div>
        </div>

        <!-- STANDARD QUESTION RESULTS (The main redesign target) -->
        <div v-else class="question-main-content">
          <!-- Image Column with Overlay -->
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image dimmed-image" />
                <div class="image-overlay-gradient"></div>
                
                <!-- DISTRIBUTION OVERLAY -->
                <div class="distribution-overlay-container">
                    <div class="dist-chart">
                        <div 
                            v-for="(count, idx) in (isTrueFalse ? 2 : 4)"
                            :key="idx"
                            class="dist-bar-col"
                        >
                            <div class="dist-bar-track">
                                <div 
                                    class="dist-bar-fill"
                                    :class="getAnswerColorClass(idx)"
                                    :style="{ height: getBarHeight(idx) + '%' }"
                                ></div>
                            </div>
                            <div class="dist-count">{{ answerCounts[idx] || 0 }}</div>
                            <div class="dist-icon">{{ getShapeIcon(idx) }}</div>
                        </div>
                    </div>
                </div>
              </div>
              <div class="question-text-overlay">
                <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
              </div>
            </div>
          </div>

          <!-- Answers Column -->
          <div class="answers-column">
             <div 
                v-for="(answer, idx) in currentQuestion?.answers" 
                :key="idx" 
                class="answer-card-stacked result-card"
                :class="[
                    getAnswerColorClass(idx),
                    isAnswerCorrect(idx) ? 'result-correct' : 'result-wrong'
                ]"
                v-show="!isTrueFalse || idx < 2"
              >
                <div class="answer-icon">{{ getShapeIcon(idx) }}</div>
                <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ answer }}</div>
                <div v-if="isAnswerCorrect(idx)" class="result-check-icon">✓</div>
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
                  :style="{ width: getScoreBarWidth(displayedScores[player.id] || 0) + '%' }"
                ></div>
              </div>
              <div class="lb-score-value" style="position: relative;">
                {{ (displayedScores[player.id] || 0).toLocaleString() }}
                <span v-if="player.showDiff" class="lb-diff-anim" :style="{ animationDelay: (((topPlayers.length - 1 - idx) * 0.12) + 0.5) + 's' }">+{{ player.diffValue }}</span>
              </div>
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
import QRCodeVue from 'qrcode.vue'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { getScreensavers } from '../services/media'

const getScaledFontSize = (size) => {
    // 1920 is the base width for big-screen
    const ratio = window.innerWidth / 1920;
    return size * ratio;
}

const { locale } = useI18n()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

// Screen Manager State override
const overrideScreenState = ref(null)
const joinUrl = computed(() => {
  return `${window.location.origin}/?pin=${eventId}`
})

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
        
        topPlayers.value.forEach((p, idx) => {
            const startScore = displayedScores.value[p.id] || 0
            const targetScore = p.score
            const pointsGained = targetScore - startScore
            
            p.showDiff = pointsGained > 0
            p.diffValue = pointsGained
            
            displayedScores.value[p.id] = startScore
            
            const duration = 1500
            const delay = ((topPlayers.value.length - 1 - idx) * 120)
            const startTime = performance.now() + delay
            
            const animate = (currentTime) => {
                if (currentTime < startTime) {
                    requestAnimationFrame(animate)
                    return
                }
                const elapsed = currentTime - startTime
                const progress = Math.min(elapsed / duration, 1)
                const easeOut = 1 - Math.pow(1 - progress, 3) 
                
                displayedScores.value[p.id] = Math.floor(startScore + (pointsGained * easeOut))
                
                if (progress < 1) {
                    requestAnimationFrame(animate)
                } else {
                    displayedScores.value[p.id] = targetScore
                }
            }
            requestAnimationFrame(animate)
        })
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
const displayedScores = ref({})

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

const isAnswerCorrect = (idx) => {
  const q = currentQuestion.value
  if (!q) return false
  
  if (q.type === 'multi_select' && Array.isArray(q.correctIndices)) {
      return q.correctIndices.includes(idx)
  }
  return q.correctIndex === idx
}

// Timer
let timerInterval
const startLocalTimer = (duration) => {
  clearInterval(timerInterval)
  timeRemaining.value = duration
  // Only start countdown if state is active
  if (gameState.value !== 'question-active') return
  
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

    // Try to autoplay music when quiz starts (get-ready or question-active or question-presenting)
    if (payload.state === 'get-ready' || payload.state === 'question-active' || payload.state === 'question-presenting') {
      displayMode.value = 'quiz'
      // Try to autoplay when quiz starts - reload audio first
      const player = bgMusicPlayer.value
      // only reload if this is a new 'start' phase
      if (prevState !== 'get-ready' && prevState !== 'question-presenting' && prevState !== 'question-active') {
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
      }
      
      if (payload.state === 'question-active' && (prevState !== 'question-active' || payload.currentIndex !== currentQuestionIndex.value)) {
        startLocalTimer(currentQuestion.value?.timeLimit || 20)
      } else if (payload.state === 'question-presenting') {
        // Set timer value but don't count down
        if (currentQuestion.value?.timeLimit) timeRemaining.value = currentQuestion.value.timeLimit
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
      
      players.value.forEach(p => {
         if (displayedScores.value[p.id] === undefined) {
             displayedScores.value[p.id] = p.score;
         }
      })
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
  
  // Set up Screen State SSE
  const sse = new EventSource(`${API_BASE_URL}/events/${eventId}/screen-stream`)
  sse.onmessage = (event) => {
    try {
      const state = JSON.parse(event.data)
      overrideScreenState.value = state
      
      // If it forces screensaver, refresh screensaver image
      if (state.currentView === 'screensaver') {
        loadScreensaverConfig() 
      }
    } catch (e) {
      console.error('Failed to parse screen state', e)
    }
  }

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
/* Newly added Slide Viewer mode */
.slide-bigscreen-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  width: 100vw;
  position: relative;
  overflow: hidden;
}

.slide-bg-image {
  position: absolute;
  top: 0; left: 0; width: 100%; height: 100%;
  object-fit: cover;
  z-index: 0;
}

.slide-element {
  position: absolute;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.slide-text {
  width: 100%;
  word-wrap: break-word;
}

.slide-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.slide-qr {
  background: white;
  padding: 10px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

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
}

.lb-diff-anim {
  color: #32D74B;
  font-size: 1.5rem;
  margin-left: 10px;
  font-weight: 900;
  opacity: 0;
  position: absolute;
  right: -80px;
  top: 0;
  animation: floatUpDiff 2s forwards ease-out;
  text-shadow: 0 2px 10px rgba(50,215,75,0.6);
}

@keyframes floatUpDiff {
  0% { opacity: 0; transform: translateY(10px) scale(0.8); }
  15% { opacity: 1; transform: translateY(0) scale(1.1); }
  25% { transform: scale(1); }
  80% { opacity: 1; transform: translateY(-15px); }
  100% { opacity: 0; transform: translateY(-25px); }
}

/* Animations based on key */
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
/* ===== CORRECT PICTURE RESULTS — NEW HERO LAYOUT ===== */
.correct-picture-results {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
    padding: 20px;
}

/* Hero: big correct image */
.cp-correct-hero {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
}

.cp-hero-frame {
    position: relative;
    border-radius: 24px;
    overflow: hidden;
    max-height: 70vh;
    max-width: 70vw;
    width: 100%;
    aspect-ratio: 4/3;
    animation: cpHeroReveal 0.7s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
    border: 4px solid #32D74B;
    box-shadow:
        0 0 0 8px rgba(50, 215, 75, 0.25),
        0 0 60px rgba(50, 215, 75, 0.7),
        0 0 120px rgba(50, 215, 75, 0.3),
        0 30px 80px rgba(0,0,0,0.6);
}

@keyframes cpHeroReveal {
    from { opacity: 0; transform: scale(0.7); }
    to   { opacity: 1; transform: scale(1); }
}

.cp-hero-glow {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse at center, rgba(50,215,75,0.15) 0%, transparent 70%);
    z-index: 1;
    pointer-events: none;
}

.cp-hero-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
}

.cp-hero-check {
    position: absolute;
    top: 18px;
    right: 18px;
    background: #32D74B;
    color: white;
    width: 64px;
    height: 64px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2.4rem;
    font-weight: 900;
    box-shadow: 0 4px 20px rgba(0,0,0,0.4);
    z-index: 5;
    animation: cpCheckPop 0.5s 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

@keyframes cpCheckPop {
    from { opacity: 0; transform: scale(0); }
    to   { opacity: 1; transform: scale(1); }
}

.cp-hero-label {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 24px 20px 20px;
    background: linear-gradient(to top, rgba(0,0,0,0.85) 0%, transparent 100%);
    color: #fff;
    font-size: 2vw;
    font-weight: 700;
    text-align: center;
    z-index: 5;
    text-shadow: 0 2px 8px rgba(0,0,0,0.7);
}

/* Wrong image thumbnails row */
.cp-wrong-thumbnails {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
}

.cp-thumb-card {
    position: relative;
    width: 130px;
    border-radius: 12px;
    overflow: hidden;
    border: 2px solid rgba(255,255,255,0.1);
    opacity: 0.22;
    filter: grayscale(100%) brightness(0.6);
    animation: cpThumbFade 0.5s ease forwards;
}

@keyframes cpThumbFade {
    from { opacity: 0.5; transform: scale(0.85); }
    to   { opacity: 0.22; transform: scale(1); }
}

.cp-thumb-img {
    width: 100%;
    aspect-ratio: 4/3;
    object-fit: cover;
    display: block;
}

.cp-thumb-count {
    position: absolute;
    bottom: 4px;
    right: 6px;
    background: rgba(0,0,0,0.7);
    color: #fff;
    font-size: 0.85rem;
    font-weight: bold;
    padding: 2px 6px;
    border-radius: 8px;
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

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }
.color-true { background-color: #0A84FF; }
.color-false { background-color: #FF3B30; }

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
  justify-content: center; /* Center content vertically to reduce dead space */
  padding: 2vw;
  height: 100%;
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
  margin-bottom: 4vh; /* Slightly more space */
  animation: popIn 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
  transform-origin: center center;
  /* Make it pop more */
  transform: scale(1.1);
}

.correct-answer-display {
  font-size: 3.5vw; /* Larger font */
  padding: 1.5vw 4vw;
  border-radius: 2vw;
  display: inline-flex;
  align-items: center;
  gap: 1.5vw;
  box-shadow: 0 0 40px rgba(255,255,255,0.2);
  border: 2px solid rgba(255,255,255,0.3);
  background-clip: padding-box;
}

.correct-icon { font-size: 2.5vw; }
.correct-text { font-weight: 700; }

.answers-stats {
  width: 90%; /* Wider */
  margin: 2vh 0;
  flex: 1; /* Take up available space */
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.stats-grid {
  display: flex;
  gap: 3vw; /* More gap */
  height: 100%; /* Fill container */
  max-height: 50vh; /* Limit max height */
  min-height: 30vh;
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

.stat-col-correct {
  position: relative;
  animation: correctColReveal 0.5s 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

@keyframes correctColReveal {
  from { transform: scale(0.92); }
  to   { transform: scale(1); }
}

.stat-col-wrong {
  opacity: 0.15; /* Much dimmer */
  filter: grayscale(100%); /* Fully grayscale */
  transform: scale(0.95); /* Slightly smaller */
  transition: opacity 0.6s ease, filter 0.6s ease, transform 0.6s ease;
}

.stat-bar-correct {
  box-shadow: 0 0 50px rgba(50, 215, 75, 0.8), 0 0 15px rgba(255,255,255,0.6);
  border: 3px solid rgba(50, 215, 75, 1);
  position: relative;
  z-index: 10;
}

.stat-icon-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.correct-check-badge {
  position: absolute;
  top: -10px;
  right: -12px;
  background: #32D74B;
  color: white;
  font-size: 1vw;
  font-weight: 900;
  width: 1.8vw;
  height: 1.8vw;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 10px rgba(50, 215, 75, 0.7);
  animation: badgePop 0.4s 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

@keyframes badgePop {
  from { transform: scale(0); }
  to   { transform: scale(1); }
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

/* ===== NEW RESULTS LAYOUT ===== */
.results-container-new {
    display: flex;
    flex-direction: column;
    padding: 2vw;
    box-sizing: border-box;
}

.results-mode-title {
    font-size: 3vw;
    font-weight: bold;
    color: #fff;
    margin-right: auto;
}

/* Dimmed Image for Results */
.dimmed-image {
    transition: filter 0.5s ease;
    filter: brightness(0.4) blur(2px);
}

/* Distribution Overlay */
.distribution-overlay-container {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 10;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 2vw; /* Space from top */
    background: rgba(0,0,0,0.3);
}

.dist-chart {
    display: flex;
    gap: 3vw;
    width: 80%;
    height: 50%; /* Occupy top 50% of image */
    align-items: flex-start;
    justify-content: center;
}

.dist-bar-col {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
}

.dist-bar-track {
    width: 100%;
    flex: 1;
    display: flex;
    align-items: flex-start;
    /* background: rgba(255,255,255,0.05); */
    border-radius: 0 0 8px 8px;
}

.dist-bar-fill {
    width: 100%;
    border-radius: 0 0 8px 8px;
    min-height: 4px; /* Ensure visibility even if 0 */
    transition: height 1s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 0 10px rgba(0,0,0,0.5);
    opacity: 0.9;
}

.dist-count {
    font-size: 2.5vw;
    font-weight: 900;
    color: white;
    text-shadow: 0 2px 4px rgba(0,0,0,0.8);
    margin-top: 5px;
}

.dist-icon {
    font-size: 2vw;
    margin-top: 5px;
    text-shadow: 0 2px 4px rgba(0,0,0,0.8);
}

/* Result Answer Cards */
.result-card {
    transition: all 0.5s ease;
    border: 3px solid transparent; /* Prepare for border */
}

/* Wrong Answers: Dimmed */
.result-wrong {
    opacity: 0.3;
    filter: grayscale(100%);
    transform: scale(0.98);
}

/* Correct Answer: Highlighted */
.result-correct {
    opacity: 1;
    transform: scale(1.02);
    box-shadow: 0 0 30px rgba(50, 215, 75, 0.6);
    border-color: #32D74B;
    z-index: 5;
    background: linear-gradient(135deg, #2e7d32, #43a047) !important; /* Override gradient */
}

.result-check-icon {
    font-size: 3vw;
    color: #fff;
    font-weight: 900;
    margin-left: 1vw;
    filter: drop-shadow(0 0 5px rgba(0,0,0,0.5));
    animation: popIn 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

/* ── Slider BigScreen View (question-active) ──────────────────── */
.slider-bigscreen-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 10px;
  padding: 16px 0;
}
.sbs-track {
  position: relative;
  width: 10px;
  flex: 1;
  min-height: 0;
  background: rgba(255,255,255,0.12);
  border-radius: 8px;
  overflow: visible;
}
.sbs-track-fill {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  border-radius: 8px;
  background: linear-gradient(to top, #0A84FF, #64D2FF);
}
@keyframes sbsFillLoop {
  0%   { height: 0%; }
  100% { height: 85%; }
}
.sbs-fill-anim {
  animation: sbsFillLoop 4s ease-in-out infinite alternate;
}

@keyframes sbsThumbMove {
  0%   { bottom: 0%; }
  100% { bottom: 85%; }
}

.sbs-thumb {
  position: absolute;
  left: 50%;
  width: 44px;
  height: 44px;
  background: #ffffff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 3px 14px rgba(0,0,0,0.35);
  bottom: 85%;
  transform: translate(-50%, 50%);
  animation: sbsThumbMove 4s ease-in-out infinite alternate,
             sbsThumbPulse 1.8s ease-in-out infinite;
}
@keyframes sbsThumbPulse {
  0%, 100% { box-shadow: 0 3px 14px rgba(0,0,0,0.35), 0 0 0 0 rgba(10,132,255,0); }
  50%       { box-shadow: 0 3px 14px rgba(0,0,0,0.35), 0 0 0 10px rgba(10,132,255,0.3); }
}
.sbs-thumb-lines {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.sbs-thumb-lines span {
  display: block;
  width: 12px;
  height: 2px;
  border-radius: 1px;
  background: rgba(0,0,0,0.22);
}
.sbs-label {
  font-size: 1.3vw;
  font-weight: 700;
  color: rgba(255,255,255,0.55);
  text-align: center;
}
.sbs-hint {
  font-size: 1.1vw;
  font-weight: 600;
  color: rgba(255,255,255,0.35);
  letter-spacing: 0.05em;
  margin-top: 4px;
}

/* ── Slider Results — New Vertical Timeline ───────────────────── */
.slider-results-new {
  display: flex;
  flex-direction: row;
  width: 100%;
  height: 100%;
}
.slider-result-timeline {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 8px;
  padding: 16px 0;
}
.srt-correct-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #30D158;
  border-radius: 14px;
  padding: 10px 22px;
  opacity: 0;
  transform: translateY(-12px) scale(0.9);
  transition: opacity 0.5s ease, transform 0.5s cubic-bezier(0.34,1.56,0.64,1);
}
.srt-correct-badge.srt-reveal {
  opacity: 1;
  transform: translateY(0) scale(1);
}
.srt-check {
  font-size: 1.8vw;
  font-weight: 900;
  color: #000;
}
.srt-value {
  font-size: 2.4vw;
  font-weight: 900;
  color: #000;
  letter-spacing: -0.01em;
}
.srt-label {
  font-size: 1.1vw;
  font-weight: 600;
  color: rgba(255,255,255,0.45);
}
.srt-track {
  position: relative;
  width: 10px;
  flex: 1;
  min-height: 0;
  background: rgba(255,255,255,0.1);
  border-radius: 8px;
  overflow: visible;
}
.srt-fill {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  border-radius: 8px;
  background: linear-gradient(to top, #30D158, #86efac);
  transition: height 1.0s cubic-bezier(0.22, 1, 0.36, 1);
  box-shadow: 0 0 16px rgba(48,209,88,0.5);
}
.srt-marker {
  position: absolute;
  left: 50%;
  transform: translate(-50%, 50%);
  display: flex;
  align-items: center;
  gap: 0;
  opacity: 0;
  transition: bottom 1.0s cubic-bezier(0.22,1,0.36,1), opacity 0.4s ease;
}
.srt-marker.srt-marker-reveal { opacity: 1; }
.srt-marker-line {
  width: 32px;
  height: 4px;
  background: #30D158;
  border-radius: 2px;
  box-shadow: 0 0 10px rgba(48,209,88,0.8);
}
.srt-marker-bubble {
  background: #30D158;
  color: #000;
  font-size: 1.0vw;
  font-weight: 800;
  padding: 4px 10px;
  border-radius: 6px;
  white-space: nowrap;
  box-shadow: 0 0 12px rgba(48,209,88,0.5);
}

/* Screen Manager Additions */
.welcome-slide-view {
  background: linear-gradient(135deg, #1A1A1D 0%, #000000 100%);
  display: flex;
  justify-content: center;
  align-items: center;
}
.welcome-center {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 30px;
}
.welcome-title {
  font-size: 5rem;
  font-weight: 800;
  text-shadow: 2px 2px 8px rgba(0,0,0,0.3);
  margin: 0;
}
.qr-container-big {
  background: white;
  padding: 20px;
  border-radius: 20px;
  box-shadow: 0 10px 30px rgba(0,0,0,0.5);
  display: inline-block;
}
.pin-display-big {
  font-size: 3rem;
  font-weight: 600;
  color: #ddd;
}
.pin-display-big span {
  font-size: 5rem;
  font-weight: 800;
  color: #FF9500;
  margin-left: 20px;
  letter-spacing: 5px;
}
.pin-display-big.no-qr {
  font-size: 5rem;
}
.pin-display-big.no-qr span {
  font-size: 10rem;
}
.join-url {
  font-size: 2.5rem;
  color: #aaa;
  margin-top: 10px;
  font-weight: 500;
}

.custom-text-view {
  background: #121212;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 50px;
}
.custom-text-display {
  font-size: 8rem;
  font-weight: 800;
  text-align: center;
  color: white;
  text-shadow: 2px 4px 12px rgba(0,0,0,0.5);
  line-height: 1.2;
}

</style>
