<template>
  <div class="projector-view">
    <div class="language-switcher-container">
      <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

    <div v-if="displayMode === 'idle'" class="projector-idle">
      <div class="projector-idle-grid">
        <div class="projector-idle-copy">
          <div class="projector-kicker">Projector View</div>
          <h1 class="projector-event-title">{{ currentEvent?.name || 'Event' }}</h1>
          <p v-if="currentEvent?.description" class="projector-event-description">
            {{ currentEvent.description }}
          </p>
          <div class="projector-join-meta">
            <div class="projector-pin-card">
              <span class="projector-pin-label">{{ $t('quiz.host.game_pin') }}</span>
              <span class="projector-pin-value">{{ pinCode }}</span>
            </div>
            <div class="projector-url">{{ joinUrl }}</div>
          </div>
        </div>

        <div class="projector-qr-shell">
          <div class="projector-qr-frame">
            <QRCodeVue :value="joinUrl" :size="360" level="H" />
          </div>
          <div class="projector-qr-caption">{{ $t('quiz.host.join_title') }}</div>
        </div>
      </div>
    </div>

    <!-- Connecting State -->
    <div v-else-if="displayMode === 'connecting'" class="connecting-container">
      <div class="loading-content">
        <h1>{{ $t('projector.connecting') }}</h1>
        <div class="loader"></div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="displayMode === 'error'" class="error-container">
      <h1>{{ $t('projector.conn_error') }}</h1>
      <p>{{ errorMessage }}</p>
      <button @click="retryConnection" class="retry-btn">{{ $t('projector.retry') }}</button>
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
            <QRCodeVue :value="joinUrl" :size="220" level="H" />
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
          <div class="timer" v-else-if="gameState === 'question-presenting'" :style="explainTimeRemaining > 0 ? '' : 'font-size: 1.5rem'">
            {{ explainTimeRemaining > 0 ? explainTimeRemaining : ($t('quiz.player.get_ready') || 'Ready!') }}
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
                 <img :src="getImageUrl(option.imageId)" class="cp-image" decoding="async" loading="eager" />
                 <div class="cp-number-badge">{{ idx + 1 }}</div>
              </div>
           </div>
           <div class="cp-question-text">
              <h2>{{ currentQuestion.text }}</h2>
           </div>
        </div>

        <!-- WRONG PICTURES LAYOUT -->
        <div v-else-if="isWrongPictures" class="wrong-picture-layout">
          <div class="question-text-overlay-wp">
             <h2>{{ currentQuestion.text }}</h2>
          </div>
          <div class="wp-image-container">
             <img :src="questionImageUrl" class="wp-big-img" decoding="async" loading="eager" />
          </div>
        </div>

        <div v-else class="question-main-content">
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" decoding="async" loading="eager" />
                <div class="image-overlay-gradient"></div>
              </div>
              <div class="question-text-overlay">
                <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
              </div>
            </div>
          </div>

          <div class="answers-column">
            <div v-if="isRange" class="range-projector-view">
              <!-- Min / Max labels -->
              <div class="sbs-label sbs-label-max">
                {{ formatNumber(currentQuestion?.rangeConfig?.maxValue) }}{{ currentQuestion?.rangeConfig?.unit }}
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
                {{ formatNumber(currentQuestion?.rangeConfig?.minValue) }}{{ currentQuestion?.rangeConfig?.unit }}
              </div>

              <div class="sbs-hint">↕ Scroll to select</div>
            </div>
            <template v-else-if="isNumber">
              <div class="number-host-view">
                <div class="nh-input-icon">🔢</div>
                <div class="nh-label">{{ $t('quiz.host.players_entering_number') }}</div>
              </div>
            </template>
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
                <img :src="getImageUrl(option.imageId)" class="cp-hero-img" decoding="async" loading="eager" />
                <div class="cp-hero-check">✓</div>
                <div class="cp-hero-label">{{ currentQuestion.text }}</div>
              </div>
            </div>
          </div>
          <!-- Wrong images shown as small thumbnails -->
          <div class="cp-wrong-thumbnails">
            <template v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
              <div v-if="!isAnswerCorrect(idx)" class="cp-thumb-card">
                <img :src="getImageUrl(option.imageId)" class="cp-thumb-img" decoding="async" loading="eager" />
                <div class="cp-thumb-count">{{ answerCounts[idx] || 0 }}</div>
              </div>
            </template>
          </div>
        </div>

        <!-- WRONG PICTURES RESULTS -->
        <div v-else-if="isWrongPictures" class="wrong-picture-results">
          <div class="question-text-overlay-wp">
            <h2>{{ currentQuestion.text }}</h2>
          </div>

          <div class="wp-result-stage">
            <div v-if="wrongPictureTargets.length" class="wp-result-progress" aria-label="Answer progress">
              <span
                v-for="(target, idx) in wrongPictureTargets"
                :key="target.id || idx"
                :class="{
                  'is-active': wpResultMode === 'targets' && idx === wpResultTargetIndex,
                  'is-complete': wpResultMode === 'overview' || idx < wpResultTargetIndex
                }"
              >
                {{ idx + 1 }}
              </span>
            </div>

            <WrongPictureTargetCrop
              v-if="questionImageUrl && activeWrongPictureTarget && wpResultMode === 'targets' && revealCorrect"
              class="wp-result-target-crop"
              :image-url="questionImageUrl"
              :target="activeWrongPictureTarget"
              :target-number="wpResultTargetIndex + 1"
              :label="activeWrongPictureTarget.label"
            />

            <div
              v-else-if="questionImageUrl && wrongPictureTargets.length && wpResultMode === 'overview'"
              class="wp-result-overview"
              :style="wrongPictureOverviewSize"
            >
              <img :src="questionImageUrl" class="wp-big-img" decoding="async" loading="eager" @load="onWrongPictureImageLoad" />
              <div
                v-for="(target, idx) in wrongPictureTargets"
                :key="target.id || idx"
                class="wp-target-area"
                :class="{ 'wp-target-reveal': revealCorrect }"
                :style="{
                  left: `${target.xPercent}%`,
                  top: `${target.yPercent}%`,
                  width: `${target.radius * 2}%`,
                  aspectRatio: '1'
                }"
              >
                <div class="wp-target-pulse"></div>
                <div class="wp-target-label">{{ target.label || idx + 1 }}</div>
              </div>
            </div>

            <div v-else-if="questionImageUrl" class="wp-result-overview" :style="wrongPictureOverviewSize">
              <img :src="questionImageUrl" class="wp-big-img" decoding="async" loading="eager" @load="onWrongPictureImageLoad" />
            </div>
          </div>
        </div>

        <!-- SLIDER RESULTS -->
        <div v-else-if="isRange" class="range-results-new">
          <!-- Left: question card as usual -->
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" decoding="async" loading="eager" />
                <div class="image-overlay-gradient"></div>
              </div>
              <div class="question-text-overlay">
                <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
              </div>
            </div>
          </div>

          <!-- Right: vertical slider timeline showing correct answer -->
          <div class="answers-column">
            <div class="range-result-timeline">
              <!-- Big correct value badge -->
              <div class="srt-correct-badge" :class="{ 'srt-reveal': revealCorrect }">
                <span class="srt-check">✓</span>
                <span class="srt-value">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.rangeConfig?.unit }}</span>
              </div>

              <div class="srt-label srt-label-max">
                {{ formatNumber(currentQuestion?.rangeConfig?.maxValue) }}{{ currentQuestion?.rangeConfig?.unit }}
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
                    {{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.rangeConfig?.unit }}
                  </div>
                </div>
              </div>

              <div class="srt-label srt-label-min">
                {{ formatNumber(currentQuestion?.rangeConfig?.minValue) }}{{ currentQuestion?.rangeConfig?.unit }}
              </div>
            </div>
          </div>
        </div>

        <!-- NUMBER RESULTS -->
        <div v-else-if="isNumber" class="number-results-host">
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" decoding="async" loading="eager" />
                <div class="image-overlay-gradient"></div>
              </div>
              <div class="question-text-overlay">
                <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
              </div>
            </div>
          </div>
          <div class="answers-column">
             <div class="number-result-card">
                <div class="nr-label">{{ $t('quiz.host.correct_answer') }}</div>
                <div class="nr-value">{{ currentQuestion?.correctNumber }}</div>
                <div class="nr-check">✓</div>
             </div>
          </div>
        </div>

        <!-- STANDARD QUESTION RESULTS (The main redesign target) -->
        <div v-else class="question-main-content">
          <!-- Image Column with Overlay -->
          <div class="question-column">
            <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
              <div v-if="questionImageUrl" class="question-image-container">
                <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image dimmed-image" decoding="async" loading="eager" />
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
            {{ $t('projector.enable_sound') }}
        </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCodeVue from 'qrcode.vue'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { preloader } from '../utils/imagePreload'
import WrongPictureTargetCrop from '../components/WrongPictureTargetCrop.vue'
import { getOverviewSize } from '../utils/wrongPictureGeometry'

const getScaledFontSize = (size) => {
  const ratio = window.innerWidth / 1920
  return size * ratio
}

const { locale } = useI18n()
const route = useRoute()
const eventId = route.params.eventId
const activeModuleId = ref('')
const currentEvent = ref(null)
const supportedProjectorTypes = new Set(['quiz', 'wrong_pictures', 'simple_quiz'])
let pollingInterval = null

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

const joinUrl = computed(() => `${window.location.origin}/?pin=${currentEvent.value?.pin || eventId}`)

// Display mode: 'idle', 'connecting', 'module', 'error'
const displayMode = ref('idle')
const errorMessage = ref('')

// Audio State
const bgMusicPlayer = ref(null)
const backgroundMusicUrl = ref('')
const audioOutput = ref('projector')
const showUnmuteButton = ref(false)
const isMusicPlaying = ref(false)

const fullMusicUrl = computed(() => {
  if (!backgroundMusicUrl.value) return ''
  return `${API_BASE_URL}${backgroundMusicUrl.value}`
})

// Quiz state
const gameState = ref('lobby')
const pinCode = computed(() => currentEvent.value?.pin || eventId)
const players = ref([])
const questions = ref([])
const currentQuestionIndex = ref(0)
const timeRemaining = ref(20)
const explainTimeRemaining = ref(0)
const answerCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 })
const playerSignals = ref([])

const totalAnswered = computed(() => {
  if (isRange.value) return playerSignals.value.length
  return Object.values(answerCounts.value).reduce((a, b) => a + b, 0)
})

const totalQuestions = computed(() => questions.value.length)
const currentQuestion = computed(() => questions.value[currentQuestionIndex.value])
const isLastQuestion = computed(() => currentQuestionIndex.value >= totalQuestions.value - 1)
const isTrueFalse = computed(() => currentQuestion.value?.type === 'true_false')
const isRange = computed(() => currentQuestion.value?.type === 'range')
const isCorrectPicture = computed(() => currentQuestion.value?.type === 'correct_picture')
const isWrongPictures = computed(() => currentQuestion.value?.type === 'wrong_pictures')
const isNumber = computed(() => currentQuestion.value?.type === 'number')
const revealCorrect = ref(false)
const wrongPictureTargets = computed(() => currentQuestion.value?.wrongPictureConfig?.targets || [])
const wrongPictureAspectRatio = ref(1)
const wrongPictureOverviewSize = computed(() => getOverviewSize(wrongPictureAspectRatio.value))
const onWrongPictureImageLoad = (event) => {
  const image = event.target
  if (image.naturalWidth && image.naturalHeight) {
    wrongPictureAspectRatio.value = image.naturalWidth / image.naturalHeight
  }
}
const wpResultTargetIndex = ref(0)
const wpResultMode = ref('targets')
const activeWrongPictureTarget = computed(() => wrongPictureTargets.value[wpResultTargetIndex.value])
let wrongPicturesResultTimers = []

const clearWrongPicturesResultTimers = () => {
  wrongPicturesResultTimers.forEach(clearTimeout)
  wrongPicturesResultTimers = []
}

const startWrongPicturesResultSequence = () => {
  clearWrongPicturesResultTimers()
  revealCorrect.value = false
  wpResultTargetIndex.value = 0
  wpResultMode.value = 'targets'

  if (!wrongPictureTargets.value.length) return

  const focusDuration = 2800
  wrongPicturesResultTimers.push(setTimeout(() => {
    revealCorrect.value = true

    wrongPictureTargets.value.forEach((_, index) => {
      wrongPicturesResultTimers.push(setTimeout(() => {
        if (index === wrongPictureTargets.value.length - 1) {
          wpResultMode.value = 'overview'
        } else {
          wpResultTargetIndex.value = index + 1
        }
      }, focusDuration * (index + 1)))
    })
  }, 300))
}

// Watch gameState to trigger animation
watch(gameState, (newVal) => {
    if (newVal === 'results') {
        revealCorrect.value = false
        if (isWrongPictures.value) {
          startWrongPicturesResultSequence()
        } else {
          setTimeout(() => {
            revealCorrect.value = true
          }, 1500)
        }
    } else {
        clearWrongPicturesResultTimers()
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

// Preload questions whenever they are loaded or updated
watch(questions, (newQuestions) => {
  if (Array.isArray(newQuestions) && newQuestions.length > 0) {
    preloader.preloadQuestions(newQuestions)
  }
}, { immediate: true })

// Lookahead: Preload upcoming question immediately with high priority when index changes
watch(currentQuestionIndex, (newIdx) => {
  if (questions.value && questions.value[newIdx + 1]) {
    preloader.preloadQuestion(questions.value[newIdx + 1])
  }
  if (gameState.value === 'results' && isWrongPictures.value) {
    startWrongPicturesResultSequence()
  }
})

const questionImageUrl = computed(() => {
  const imageId = currentQuestion.value?.imageId
  return imageId ? getImageUrl(imageId) : null
})
watch(questionImageUrl, () => { wrongPictureAspectRatio.value = 1 })

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
  if (!isRange.value) return []
  const q = currentQuestion.value
  if (!q || !q.rangeConfig) return []

  const min = q.rangeConfig.minValue
  const max = q.rangeConfig.maxValue
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
  if (!isRange.value) return 0
  const q = currentQuestion.value
  const min = q?.rangeConfig?.minValue || 0
  const max = q?.rangeConfig?.maxValue || 100
  const range = max - min
  if (range === 0) return 0
  return ((q.correctValue - min) / range) * 100
})

const isBucketCorrect = (bucketIdx) => {
  const q = currentQuestion.value
  const min = q.rangeConfig.minValue
  const max = q.rangeConfig.maxValue
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
let explainTimerInterval

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

const startExplainTimer = (duration) => {
    clearInterval(explainTimerInterval)
    explainTimeRemaining.value = duration
    if (duration <= 0) return
    explainTimerInterval = setInterval(() => {
        if (explainTimeRemaining.value > 0) {
            explainTimeRemaining.value--
        } else {
            clearInterval(explainTimerInterval)
        }
    }, 1000)
}

const applyModuleConfig = (module) => {
  backgroundMusicUrl.value = module?.data?.backgroundMusicUrl || ''
  audioOutput.value = module?.data?.audioOutput || 'projector'
  if (Array.isArray(module?.data?.questions) && module.data.questions.length > 0) {
    questions.value = module.data.questions
  }
  manageAudioPlayback()
}

const manageAudioPlayback = async () => {
  const shouldPlay =
    displayMode.value === 'module' &&
    backgroundMusicUrl.value &&
    (audioOutput.value === 'projector' || audioOutput.value === 'both') &&
    (gameState.value === 'get-ready' || gameState.value === 'question-presenting' || gameState.value === 'question-active')

  const player = bgMusicPlayer.value
  if (!player) return

  if (shouldPlay) {
    if (player.paused) {
      try {
        await player.play()
        isMusicPlaying.value = true
        showUnmuteButton.value = false
      } catch (err) {
        console.warn('Autoplay prevented:', err)
        showUnmuteButton.value = true
      }
    }
  } else if (!player.paused) {
    player.pause()
    isMusicPlaying.value = false
  }
}

const enableAudio = () => {
  const player = bgMusicPlayer.value
  if (player) {
    player.play().then(() => {
      isMusicPlaying.value = true
      showUnmuteButton.value = false
    }).catch(err => console.error('Still cannot play:', err))
  }
}

const resetModuleState = () => {
  clearInterval(timerInterval)
  clearInterval(explainTimerInterval)
  gameState.value = 'lobby'
  players.value = []
  questions.value = []
  currentQuestionIndex.value = 0
  timeRemaining.value = 20
  explainTimeRemaining.value = 0
  answerCounts.value = { 0: 0, 1: 0, 2: 0, 3: 0 }
  playerSignals.value = []
  displayedScores.value = {}
  revealCorrect.value = false
  backgroundMusicUrl.value = ''
  audioOutput.value = 'projector'
  preloader.clear()
  manageAudioPlayback()
}

const exitActiveModule = () => {
  activeModuleId.value = ''
  quizWS.disconnect()
  quizWS.clearHandlers()
  resetModuleState()
  displayMode.value = 'idle'
}

const connectToModule = (module) => {
  if (!module?.id) return

  quizWS.disconnect()
  quizWS.clearHandlers()
  resetModuleState()

  activeModuleId.value = module.id
  applyModuleConfig(module)
  displayMode.value = 'connecting'

  // Preload wrong pictures metadata if applicable
  if (module.type === 'wrong_pictures') {
    fetch(`${API_BASE_URL}/events/${eventId}/modules/${module.id}/wrongpictures/preload`)
      .then(res => (res.ok ? res.json() : null))
      .then(data => {
        if (data?.questions?.length > 0) {
          preloader.preloadQuestions(data.questions)
        }
      })
      .catch(err => console.warn('Failed to fetch wrongpictures preload metadata:', err))
  }

  quizWS.on('cmd:state', (payload) => {
    const prevState = gameState.value
    gameState.value = payload.state
    displayMode.value = 'module'

    if (payload.currentIndex !== undefined) {
      currentQuestionIndex.value = payload.currentIndex
    }

    if (payload.state === 'get-ready' || payload.state === 'question-active' || payload.state === 'question-presenting') {
      const player = bgMusicPlayer.value
      if (prevState !== 'get-ready' && prevState !== 'question-presenting' && prevState !== 'question-active') {
        if (player && backgroundMusicUrl.value && (audioOutput.value === 'projector' || audioOutput.value === 'both')) {
          player.load()
          player.play().then(() => {
            isMusicPlaying.value = true
            showUnmuteButton.value = false
          }).catch(() => {
            showUnmuteButton.value = false
          })
        }
      }

      if (payload.state === 'question-active' && prevState !== 'question-active') {
        startLocalTimer(currentQuestion.value?.timeLimit || 20)
      } else if (payload.state === 'question-presenting') {
        if (currentQuestion.value?.timeLimit) timeRemaining.value = currentQuestion.value.timeLimit
        startExplainTimer(payload.timeLimit || 5)
      }
    } else {
      clearInterval(timerInterval)
      clearInterval(explainTimerInterval)
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
    displayMode.value = 'module'

    if (session.Players || session.players) {
      const playersMap = session.Players || session.players
      players.value = Object.values(playersMap)

      players.value.forEach(p => {
        if (displayedScores.value[p.id] === undefined) {
          displayedScores.value[p.id] = p.score
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

  quizWS.on('disconnect', () => {
    if (activeModuleId.value === module.id) {
      displayMode.value = 'connecting'
      manageAudioPlayback()
    }
  })

  quizWS.connect(module.id, '', false, true, eventId)

  setTimeout(() => {
    if (displayMode.value === 'connecting' && activeModuleId.value === module.id) {
      displayMode.value = 'module'
    }
  }, 2000)
}

const retryConnection = () => {
  const module = currentEvent.value?.modules?.find(m => m.id === activeModuleId.value)
  if (module) {
    connectToModule(module)
    return
  }
  fetchEventState()
}

const fetchEventState = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}`)
    if (!res.ok) {
      throw new Error(`Failed to load event: ${res.status}`)
    }

    const eventData = await res.json()
    currentEvent.value = eventData

    const nextModule = eventData.modules?.find(m => m.isActive && supportedProjectorTypes.has(m.type))

    if (!nextModule) {
      if (activeModuleId.value) {
        exitActiveModule()
      } else {
        displayMode.value = 'idle'
      }
      return
    }

    if (activeModuleId.value !== nextModule.id || !quizWS.isConnected) {
      connectToModule(nextModule)
      return
    }

    applyModuleConfig(nextModule)
  } catch (err) {
    console.error('[Projector] Failed to fetch event state:', err)
    if (!currentEvent.value) {
      errorMessage.value = 'Failed to load projector state.'
      displayMode.value = 'error'
    }
  }
}

onMounted(() => {
  fetchEventState()
  pollingInterval = setInterval(fetchEventState, 3000)
})

onUnmounted(() => {
  clearInterval(timerInterval)
  clearInterval(explainTimerInterval)
  clearWrongPicturesResultTimers()
  if (pollingInterval) {
    clearInterval(pollingInterval)
  }
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped src="../styles/projector.css"></style>
