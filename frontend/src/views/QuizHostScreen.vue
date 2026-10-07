<template>
  <div class="quiz-host-screen">
    <div class="bg-layer"></div>
    
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
        <div class="lobby-actions">
            <button class="copy-link-btn" @click="copyProjectorLink">
                {{ $t('quiz.host.copy_link') }}
            </button>
        </div>
      </div>
      
      <div class="players-area">
        <div class="players-count">
          <span><AppIcon name="teams" :size="18" /> {{ players.length }}</span> {{ $t('quiz.host.players') }}
        </div>
        <div class="players-grid">
          <div v-for="player in players" :key="player.id" class="player-chip">
            {{ player.nickname }}
          </div>
        </div>
      </div>

      <div class="lobby-footer">
        <button class="start-btn" @click="startQuiz" :disabled="players.length === 0">
          {{ $t('quiz.host.start_btn') }}
        </button>
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

      <!-- New Two-Column Layout -->
      <div v-if="isCorrectPicture" class="correct-picture-host-layout">
          <div class="cp-host-grid">
               <div 
                 v-for="(option, idx) in currentQuestion.imageOptions" 
                 :key="idx"
                 class="cp-host-card"
               >
                  <img :src="getImageUrl(option.imageId)" class="cp-host-img" decoding="async" loading="eager" />
                  <div class="cp-host-badge">{{ idx + 1 }}</div>
               </div>
          </div>
          <div class="cp-host-text">
             <h2>{{ currentQuestion.text }}</h2>
          </div>
      </div>

      <!-- WRONG PICTURES HOST LAYOUT -->
      <div v-else-if="isWrongPictures" class="wrong-picture-host-layout">
          <div class="wp-host-header">
              <h1 class="wp-host-title">{{ $t('quiz.host.wrong_picture_title') }}</h1>
              <p class="wp-host-instruction">{{ currentQuestion.text }}</p>
          </div>
          <div class="wp-host-image-container">
              <img :src="questionImageUrl" class="wp-host-big-img" decoding="async" loading="eager" />
          </div>
      </div>

      <div v-else class="question-main-content">
        <!-- LEFT COLUMN: Question Card with Optional Image -->
        <div class="question-column">
          <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
            
            <!-- Image Background (if exists) -->
            <div v-if="questionImageUrl" class="question-image-container">
               <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" decoding="async" loading="eager" />
               <div class="image-overlay-gradient"></div>
            </div>

            <!-- Question Text Overlay -->
            <div class="question-text-overlay">
               <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
            </div>

          </div>
        </div>

        <!-- RIGHT COLUMN: Stacked Answer Cards -->
        <div class="answers-column">
          <div v-if="isRange" class="range-projector-view">
            <div class="sbs-label sbs-label-max">
              {{ formatNumber(currentQuestion?.rangeConfig?.maxValue) }}{{ currentQuestion?.rangeConfig?.unit }}
            </div>
            <div class="sbs-track">
              <div class="sbs-track-fill sbs-fill-anim"></div>
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
              <div class="nh-input-icon"><AppIcon name="hash" :size="32" /></div>
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
                <AppIcon class="answer-icon" :name="getShapeIcon(idx)" :size="28" />
                <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ answer }}</div>
            </div>
          </template>
        </div>
      </div>
      
      <!-- Small Answer Counter in Corner -->
      <div class="answer-counter-widget">
          <div class="counter-value">{{ totalAnswered }}</div>
          <div class="counter-label">Answers</div>
      </div>
      
      <div class="host-controls">
         <button v-if="timeRemaining > 0" class="control-btn" @click="endQuestionEarly">Skip Timer</button>
         <button v-else class="control-btn primary" @click="goToResults">Go to Results</button>
      </div>
    </div>

    <!-- RESULTS STATE (no leaderboard) -->
    <div v-else-if="gameState === 'results'" class="state-container results-container-new">
       <div class="results-content-wrapper">
         <div class="question-header" style="margin-bottom: 1rem;">
             <div class="results-mode-title">Question {{ currentQuestionIndex + 1 }} Results</div>
             <!-- <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div> -->
         </div>

         <!-- CORRECT PICTURE RESULTS -->
         <div v-if="isCorrectPicture" class="correct-picture-results-host">
           <!-- Correct image hero -->
           <div class="cp-host-correct-hero">
             <div v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
               <div v-if="isAnswerCorrect(idx)" class="cp-host-hero-frame">
                 <div class="cp-host-hero-glow"></div>
                 <img :src="getImageUrl(option.imageId)" class="cp-host-hero-img" decoding="async" loading="eager" />
                 <div class="cp-host-hero-check"><AppIcon name="check" :size="24" /></div>
                 <div class="cp-host-hero-label">{{ currentQuestion.text }}</div>
               </div>
             </div>
           </div>
           <!-- Wrong images as small thumbnails -->
           <div class="cp-host-wrong-thumbs">
             <template v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
               <div v-if="!isAnswerCorrect(idx)" class="cp-host-thumb-card">
                 <img :src="getImageUrl(option.imageId)" class="cp-host-thumb-img" decoding="async" loading="eager" />
                 <div class="cp-host-thumb-count">{{ answerCounts[idx] || 0 }}</div>
               </div>
             </template>
           </div>
         </div>

          <!-- WRONG PICTURES RESULTS HOST (Magnified) -->
          <div v-else-if="isWrongPictures" class="wrong-picture-results-magnified">
             <div class="wp-magnified-results-grid">
                <!-- Left Column: Magnified Crops (70% space) -->
                <div class="wp-magnified-column">
                    <div 
                      v-for="(target, idx) in currentQuestion.wrongPictureConfig?.targets" 
                      :key="'crop-' + idx"
                      class="wp-magnified-crop-card"
                    >
                        <WrongPictureTargetCrop
                            :image-url="questionImageUrl"
                            :target="target"
                            :target-number="idx + 1"
                        />
                    </div>
                </div>

                <!-- Right Column: Answer Labels (30% space) -->
                <div class="wp-labels-column">
                    <div 
                      v-for="(target, idx) in currentQuestion.wrongPictureConfig?.targets" 
                      :key="'label-' + idx"
                      class="wp-result-label-card compact"
                    >
                        <div class="wp-label-number small">{{ idx + 1 }}</div>
                        <div class="wp-label-text small">{{ target.label }}</div>
                    </div>
                </div>
             </div>
          </div>

         <!-- SLIDER RESULTS -->
          <div v-else-if="isRange" class="range-results-new">
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
              <div class="range-result-timeline">
                <div class="srt-correct-badge" :class="{ 'srt-reveal': revealCorrect }">
                  <AppIcon class="srt-check" name="check" :size="18" />
                  <span class="srt-value">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.rangeConfig?.unit }}</span>
                </div>
                <div class="srt-label srt-label-max">{{ formatNumber(currentQuestion?.rangeConfig?.maxValue) }}{{ currentQuestion?.rangeConfig?.unit }}</div>
                <div class="srt-track">
                  <div class="srt-fill" :style="{ height: revealCorrect ? correctMarkerPercentage + '%' : '0%' }"></div>
                  <div class="srt-marker" :class="{ 'srt-marker-reveal': revealCorrect }" :style="{ bottom: correctMarkerPercentage + '%' }">
                    <div class="srt-marker-line"></div>
                    <div class="srt-marker-bubble">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.rangeConfig?.unit }}</div>
                  </div>
                </div>
                <div class="srt-label srt-label-min">{{ formatNumber(currentQuestion?.rangeConfig?.minValue) }}{{ currentQuestion?.rangeConfig?.unit }}</div>
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
                  <div class="nr-check"><AppIcon name="check" :size="20" /></div>
               </div>
            </div>
          </div>

          <!-- STANDARD QUESTION RESULTS (REDESIGNED) -->
          <div v-else class="question-main-content">
            <!-- Left Column: Image + Distribution Overlay -->
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
                              <AppIcon class="dist-icon" :name="getShapeIcon(idx)" :size="20" />
                          </div>
                      </div>
                  </div>
                </div>
                
                <div class="question-text-overlay">
                  <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
                </div>
              </div>
            </div>

            <!-- Right Column: Results Answer Cards -->
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
                  <AppIcon class="answer-icon" :name="getShapeIcon(idx)" :size="28" />
                  <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ answer }}</div>
                  <AppIcon v-if="isAnswerCorrect(idx)" class="result-check-icon" name="check" :size="18" />
                </div>
            </div>
          </div>
       </div>

       <div class="host-controls">
         <button class="control-btn primary" @click="nextState">
           {{ $t('quiz.host.show_leaderboard') }}
         </button>
       </div>
    </div>

    <!-- LEADERBOARD STATE (beautiful animated) -->
    <div v-else-if="gameState === 'leaderboard'" class="state-container leaderboard-container" :key="leaderboardAnimKey">
      <div class="lb-bg-glow"></div>
      <div class="lb-header">
        <AppIcon class="lb-trophy" name="trophy" :size="54" />
        <h2 class="lb-title">{{ $t('quiz.host.leaderboard_title') }}</h2>
        <div class="lb-subtitle">{{ $t('quiz.host.question_of', { current: currentQuestionIndex + 1, total: totalQuestions }) }}</div>
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
            <AppIcon v-if="idx < 3" name="medal" :size="22" />
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
      <div class="host-controls">
        <button class="control-btn primary" @click="nextState">
          {{ isLastQuestion ? $t('quiz.host.end_quiz') : $t('quiz.host.next_question') }}
        </button>
      </div>
    </div>
    
    <!-- FINISHED STATE (final leaderboard with confetti) -->
    <div v-else-if="gameState === 'finished'" class="state-container final-container" :key="finalAnimKey">
       <!-- Confetti particles -->
       <div class="confetti-wrap">
         <div v-for="i in 20" :key="i" class="confetti-particle" :style="getConfettiStyle(i)"></div>
       </div>

       <div class="final-header">
         <AppIcon class="final-trophy" name="trophy" :size="64" />
         <h1 class="final-title">{{ $t('quiz.host.quiz_complete') }}</h1>
         <p class="final-subtitle">{{ $t('quiz.host.final_standings') }}</p>
       </div>

       <!-- Top 3 Podium -->
       <div class="final-podium">
          <div class="final-podium-place fp-second" v-if="topPlayers[1]">
            <AppIcon class="fp-avatar" name="medal" :size="32" />
            <div class="fp-name">{{ topPlayers[1].nickname }}</div>
            <div class="fp-score">{{ topPlayers[1].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-2"><span>2</span></div>
          </div>
          <div class="final-podium-place fp-first" v-if="topPlayers[0]">
            <AppIcon class="fp-crown" name="crown" :size="28" />
            <AppIcon class="fp-avatar" name="medal" :size="32" />
            <div class="fp-name">{{ topPlayers[0].nickname }}</div>
            <div class="fp-score">{{ topPlayers[0].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-1"><span>1</span></div>
          </div>
          <div class="final-podium-place fp-third" v-if="topPlayers[2]">
            <AppIcon class="fp-avatar" name="medal" :size="32" />
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

       <button class="control-btn" @click="exitQuiz" style="margin-top: 20px; z-index: 1;">{{ $t('quiz.host.end_quiz') }}</button>
    </div>

    <!-- Audio Player -->
    <audio ref="bgMusicPlayer" :src="fullMusicUrl" loop id="bg-music-player"></audio>
    
    <div v-if="showUnmuteButton" class="unmute-overlay">
        <button @click="enableAudio" class="unmute-btn">
            🔊 Enable Sound
        </button>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'
import { useI18n } from 'vue-i18n'
import { preloader } from '../utils/imagePreload'
import WrongPictureTargetCrop from '../components/WrongPictureTargetCrop.vue'
import AppIcon from '../components/AppIcon.vue'

const route = useRoute()
const router = useRouter()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

// State
const gameState = ref('lobby') 
const pinCode = ref(eventId)
const players = ref([])
const questions = ref([])
const currentQuestionIndex = ref(0)
const timeRemaining = ref(20)
const answerCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 }) 
const playerSignals = ref([]) // Store signals for slider histogram
const realTimeAnsweredCount = ref(0)
const totalAnswered = computed(() => {
    return realTimeAnsweredCount.value
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

// Preload questions whenever loaded
watch(questions, (newQuestions) => {
    if (Array.isArray(newQuestions) && newQuestions.length > 0) {
        preloader.preloadQuestions(newQuestions)
    }
}, { immediate: true })

// Lookahead: Preload next question immediately upon index change
watch(currentQuestionIndex, (newIdx) => {
    if (questions.value && questions.value[newIdx + 1]) {
        preloader.preloadQuestion(questions.value[newIdx + 1])
    }
})

// Watch gameState to trigger animation
watch(gameState, (newVal) => {
    if (newVal === 'results' && isRange.value) {
        revealCorrect.value = false
        setTimeout(() => {
            revealCorrect.value = true
        }, 1500)
    } else {
        revealCorrect.value = false
    }
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
})

const questionImageUrl = computed(() => {
    const imageId = currentQuestion.value?.imageId
    return imageId ? getImageUrl(imageId) : null
})

// Adaptive font size calculation for question text - use all available space
const questionFontSize = computed(() => {
    const text = currentQuestion.value?.text || ''
    const hasImage = !!questionImageUrl.value
    
    // Larger base font sizes - maximize use of space
    let baseSize;
    
    if (!hasImage) {
        // No image - massive text centered
        if (text.length < 30) {
            baseSize = 10.0;
        } else if (text.length < 60) {
            baseSize = 7.0;
        } else if (text.length < 100) {
            baseSize = 5.0;
        } else {
            baseSize = 3.5;
        }
    } else {
        // With image - text at bottom
        if (text.length < 30) {
            baseSize = 8.0; 
        } else if (text.length < 60) {
            baseSize = 5.0; 
        } else if (text.length < 100) {
            baseSize = 3.5; 
        } else {
            baseSize = 2.5;
        }
    }
    return `${baseSize}vw`;
})

// Calculate individual font size for each answer card
const getAnswerFontSize = (answerText) => {
    const text = answerText || ''
    const length = text.length
    
    // Calculate font size for each answer individually based on text length
    let size
    if (length < 15) {
        size = 2.4
    } else if (length < 25) {
        size = 2.0
    } else if (length < 40) {
        size = 1.6
    } else if (length < 60) {
        size = 1.3
    } else if (length < 100) {
        size = 1.0
    } else {
        size = 0.85
    }
    
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
    return {
        left: ((i * 37) % 100) + '%',
        animationDelay: ((i * 0.13) % 3) + 's',
        animationDuration: (2.5 + (i % 3) * 0.5) + 's',
        background: color,
        width: (8 + (i % 4) * 4) + 'px',
        height: (8 + (i % 4) * 4) + 'px',
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
        // assuming sig has answerValue
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
    // Highlight bucket containing correct answer?
    // Determine which bucket correctValue falls into
    // Re-use logic or duplicate lightly simple math
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
        return idx === 0 ? 'check' : 'close'
    }
    return ['triangle', 'diamond', 'circle', 'square'][idx]
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

// Timer logic
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
            // Auto-transition to Results when timer hits 0
            if (gameState.value === 'question-active') {
                goToResults()
            }
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

// Actions
const startQuiz = () => {
    quizWS.send('host:start_quiz', { moduleId })
}

const endQuestionEarly = () => {
    // This is a host-side visual skip, ideally backend handles text_question
    // But for now, we can just jump to results if we want
    quizWS.send('host:show_results', { moduleId })
}

const goToResults = () => {
    quizWS.send('host:show_results', { moduleId })
}

const nextState = () => {
    if (gameState.value === 'results') {
        // Transition to Leaderboard
        quizWS.send('host:show_leaderboard', { moduleId })
    } else if (gameState.value === 'leaderboard') {
        if (isLastQuestion.value) {
            // End quiz
            quizWS.send('host:end_quiz', { moduleId })
        } else {
            quizWS.send('host:next_question', { moduleId })
        }
    }
}

const exitQuiz = async () => {
    // Reset the quiz session before exiting
    console.log('[DEBUG] exitQuiz: Sending host:reset_quiz for module:', moduleId)
    quizWS.send('host:reset_quiz', { moduleId })
    
    // Wait a bit to ensure message is sent before disconnecting
    await new Promise(resolve => setTimeout(resolve, 300))
    console.log('[DEBUG] exitQuiz: Navigating back to admin-event-details')
    router.push({ name: 'admin-event-details', params: { id: eventId } })
    // Note: disconnect is handled by onUnmounted, don't call it here to avoid double-disconnect
}

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

// Manage Audio
const manageAudioPlayback = async () => {
    // Only play if output includes 'admin'
    const shouldPlay = 
        backgroundMusicUrl.value && 
        (audioOutput.value === 'admin' || audioOutput.value === 'both') &&
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

const { success } = useToast()
const { t } = useI18n()

const copyProjectorLink = () => {
    const url = `${window.location.origin}/projector/${eventId}`
    navigator.clipboard.writeText(url)
    success(t('quiz.host.toast.link_copied'))
}

watch(gameState, () => {
    manageAudioPlayback()
})

// WebSocket & Data Loading
let stopped = false
onMounted(async () => {
    // 1. Fetch Questions & Module Config
    try {
        const res = await fetch(`${API_BASE_URL}/events/${eventId}`)
        if (stopped) return
        const data = await res.json()
        if (stopped) return
        pinCode.value = data.pin || eventId
        const mod = data.modules?.find(m => m.id === moduleId)
        if (mod && mod.data) {
            if (mod.data.questions) {
                questions.value = mod.data.questions
            }
            // Music Config
            backgroundMusicUrl.value = mod.data.backgroundMusicUrl || ''
            audioOutput.value = mod.data.audioOutput || 'projector'
            manageAudioPlayback()
        }
    } catch (e) {
        console.error('Failed to load questions', e)
    }

    // Navigation during the fetch already ran onUnmounted. Registering or
    // connecting now would undo that disconnect on the shared socket.
    if (stopped) return

    // 2. Register handlers BEFORE connecting (server sends messages immediately)
    quizWS.on('cmd:state', (payload) => {
        // Handle state transitions from server
        const prevState = gameState.value
        gameState.value = payload.state
        
        if (payload.currentIndex !== undefined) {
            currentQuestionIndex.value = payload.currentIndex
        }
        
        // Handle transitions
        if (payload.state === 'question-active') {
            // Restart timer when entering question state
            if (prevState !== 'question-active' || payload.currentIndex !== currentQuestionIndex.value) {
                startLocalTimer(currentQuestion.value?.timeLimit || 20)
            }
        } else if (payload.state === 'question-presenting') {
            // Set time remaining to question time without starting timer
            if (currentQuestion.value?.timeLimit) {
                timeRemaining.value = currentQuestion.value.timeLimit
            }
            startExplainTimer(payload.timeLimit || 5) // note: payload.timeLimit in cmd:state is the ExplainTime
            realTimeAnsweredCount.value = 0 // Reset for new question
            answerCounts.value = { 0: 0, 1: 0, 2: 0, 3: 0 } // Reset counts to zero
            playerSignals.value = [] // Reset slider player signals
        } else if (payload.state === 'results') {
            clearInterval(timerInterval)
            clearInterval(explainTimerInterval)
        }
        
        if ((payload.state === 'get-ready' || payload.state === 'question-presenting') && 
            (prevState === 'lobby' || prevState === 'leaderboard' || prevState === 'results')) {
            const player = bgMusicPlayer.value
            if (player) {
                player.currentTime = 0
            }
        }
        
        manageAudioPlayback()
    })

    // Listen for minimal sync updates (answer counts, player info)
    quizWS.on('sync:minimal', (payload) => {
        // Ignore stats for an old question arriving during a new question's explain delay
        if (payload.questionIndex !== undefined && payload.questionIndex !== currentQuestionIndex.value) return;
        
        // Update answer counts if provided
        if (payload.answerCounts) {
            // Convert string keys from JSON to numeric keys for proper reactivity
            answerCounts.value = {
                0: payload.answerCounts["0"] || 0,
                1: payload.answerCounts["1"] || 0,
                2: payload.answerCounts["2"] || 0,
                3: payload.answerCounts["3"] || 0,
            }
        }
        if (payload.answeredCount !== undefined) {
            realTimeAnsweredCount.value = payload.answeredCount
        }
    })

    // Listen for full state updates (legacy support - includes players)
    quizWS.on('state:update', (session) => {
        // Update players
        if (session.Players || session.players) {
            const playersMap = session.Players || session.players
            players.value = Object.values(playersMap)
            
            players.value.forEach(p => {
               if (displayedScores.value[p.id] === undefined) {
                   displayedScores.value[p.id] = p.score;
               }
            })
        }
        
        // Update Player Signals (for histogram)
        if (session.PlayerSignals || session.playerSignals) {
             const signals = session.PlayerSignals || session.playerSignals
             playerSignals.value = signals.filter(s => s.questionIndex === currentQuestionIndex.value)
             realTimeAnsweredCount.value = playerSignals.value.length
        }

        // Update Answer Counts
        if (session.AnswerCounts || session.answerCounts) {
             const counts = session.AnswerCounts || session.answerCounts
             answerCounts.value = {
                0: counts["0"] || 0,
                1: counts["1"] || 0,
                2: counts["2"] || 0,
                3: counts["3"] || 0,
             }
        }
    })

    // 3. Connect WS (after handlers are registered)
    quizWS.connect(moduleId, '', true)
 })

onUnmounted(() => {
    stopped = true
    clearInterval(timerInterval)
    clearInterval(explainTimerInterval)
    // Release preloaded image references so they don't accumulate across
    // quizzes for the lifetime of the SPA session (mirrors ProjectorView).
    preloader.clear()
    quizWS.disconnect()
    quizWS.clearHandlers()
})
</script>

<style scoped src="../styles/quiz-host.css"></style>
