<template>
  <div class="quiz-host-screen">
    <div class="bg-layer"></div>
    
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
        <div class="lobby-actions">
            <button class="copy-link-btn" @click="copyBigScreenLink">
                {{ $t('quiz.host.copy_link') }}
            </button>
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

      <div class="lobby-footer">
        <button class="start-btn" @click="startQuiz" :disabled="players.length === 0">
          Start Quiz
        </button>
      </div>
    </div>

    <!-- GET READY STATE -->
    <div v-else-if="gameState === 'get-ready'" class="state-container get-ready-container">
      <div class="loading-content">
        <h1>Get Ready!</h1>
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
                  <img :src="getImageUrl(option.imageId)" class="cp-host-img" />
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
              <h1 class="wp-host-title">What is wrong with the picture?</h1>
              <p class="wp-host-instruction">{{ currentQuestion.text }}</p>
          </div>
          <div class="wp-host-image-container">
              <img :src="questionImageUrl" class="wp-host-big-img" />
          </div>
      </div>

      <div v-else class="question-main-content">
        <!-- LEFT COLUMN: Question Card with Optional Image -->
        <div class="question-column">
          <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
            
            <!-- Image Background (if exists) -->
            <div v-if="questionImageUrl" class="question-image-container">
               <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || 'Question image'" class="question-image" />
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
          <div v-if="isSlider" class="slider-bigscreen-view">
            <div class="sbs-label sbs-label-max">
              {{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}{{ currentQuestion?.sliderConfig?.unit }}
            </div>
            <div class="sbs-track">
              <div class="sbs-track-fill sbs-fill-anim"></div>
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
                 <img :src="getImageUrl(option.imageId)" class="cp-host-hero-img" />
                 <div class="cp-host-hero-check">✓</div>
                 <div class="cp-host-hero-label">{{ currentQuestion.text }}</div>
               </div>
             </div>
           </div>
           <!-- Wrong images as small thumbnails -->
           <div class="cp-host-wrong-thumbs">
             <template v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
               <div v-if="!isAnswerCorrect(idx)" class="cp-host-thumb-card">
                 <img :src="getImageUrl(option.imageId)" class="cp-host-thumb-img" />
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
                        <div class="wp-crop-container square">
                            <div 
                                class="wp-crop-image" 
                                :style="{ 
                                    backgroundImage: `url(${questionImageUrl})`,
                                    backgroundPosition: `${target.xPercent}% ${target.yPercent}%`,
                                    backgroundSize: `${100 / (target.radius * 0.15)}%` 
                                }"
                            ></div>
                            <div class="wp-crop-badge">{{ idx + 1 }}</div>
                        </div>
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
          <div v-else-if="isSlider" class="slider-results-new">
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
              <div class="slider-result-timeline">
                <div class="srt-correct-badge" :class="{ 'srt-reveal': revealCorrect }">
                  <span class="srt-check">✓</span>
                  <span class="srt-value">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.sliderConfig?.unit }}</span>
                </div>
                <div class="srt-label srt-label-max">{{ formatNumber(currentQuestion?.sliderConfig?.maxValue) }}{{ currentQuestion?.sliderConfig?.unit }}</div>
                <div class="srt-track">
                  <div class="srt-fill" :style="{ height: revealCorrect ? correctMarkerPercentage + '%' : '0%' }"></div>
                  <div class="srt-marker" :class="{ 'srt-marker-reveal': revealCorrect }" :style="{ bottom: correctMarkerPercentage + '%' }">
                    <div class="srt-marker-line"></div>
                    <div class="srt-marker-bubble">{{ formatNumber(currentQuestion?.correctValue) }}{{ currentQuestion?.sliderConfig?.unit }}</div>
                  </div>
                </div>
                <div class="srt-label srt-label-min">{{ formatNumber(currentQuestion?.sliderConfig?.minValue) }}{{ currentQuestion?.sliderConfig?.unit }}</div>
              </div>
            </div>
          </div>

          <!-- STANDARD QUESTION RESULTS (REDESIGNED) -->
          <div v-else class="question-main-content">
            <!-- Left Column: Image + Distribution Overlay -->
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
                  <div class="answer-icon">{{ getShapeIcon(idx) }}</div>
                  <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ answer }}</div>
                  <div v-if="isAnswerCorrect(idx)" class="result-check-icon">✓</div>
                </div>
            </div>
          </div>
       </div>

       <div class="host-controls">
         <button class="control-btn primary" @click="nextState">
           Show Leaderboard
         </button>
       </div>
    </div>

    <!-- LEADERBOARD STATE (beautiful animated) -->
    <div v-else-if="gameState === 'leaderboard'" class="state-container leaderboard-container" :key="leaderboardAnimKey">
      <div class="lb-bg-glow"></div>
      <div class="lb-header">
        <div class="lb-trophy">🏆</div>
        <h2 class="lb-title">Leaderboard</h2>
        <div class="lb-subtitle">Question {{ currentQuestionIndex + 1 }} of {{ totalQuestions }}</div>
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
      <div class="host-controls">
        <button class="control-btn primary" @click="nextState">
          {{ isLastQuestion ? 'End Quiz' : 'Next Question' }}
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
         <div class="final-trophy">🏆</div>
         <h1 class="final-title">Quiz Complete!</h1>
         <p class="final-subtitle">Final Standings</p>
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

       <button class="control-btn" @click="exitQuiz" style="margin-top: 20px; z-index: 1;">End Quiz</button>
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

// State
const gameState = ref('lobby') 
const pinCode = ref(eventId) // Using eventID as PIN for simplicity
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
const isSlider = computed(() => currentQuestion.value?.type === 'slider')
const isCorrectPicture = computed(() => currentQuestion.value?.type === 'correct_picture')
const isWrongPictures = computed(() => currentQuestion.value?.type === 'wrong_pictures')
const revealCorrect = ref(false)

// Watch gameState to trigger animation
import { watch } from 'vue'
watch(gameState, (newVal) => {
    if (newVal === 'results' && isSlider.value) {
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
    if (!isSlider.value) return 0
    const q = currentQuestion.value
    const min = q?.sliderConfig?.minValue || 0
    const max = q?.sliderConfig?.maxValue || 100
    const range = max - min
    if (range === 0) return 0
    return ((q.correctValue - min) / range) * 100
})

const isBucketCorrect = (bucketIdx) => {
    // Highlight bucket containing correct answer?
    // Determine which bucket correctValue falls into
    // Re-use logic or duplicate lightly simple math
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
const audioOutput = ref('bigscreen')
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

const copyBigScreenLink = () => {
    const url = `${window.location.origin}/bigscreen/${eventId}/${moduleId}`
    navigator.clipboard.writeText(url)
    success(t('quiz.host.toast.link_copied'))
}

watch(gameState, () => {
    manageAudioPlayback()
})

// WebSocket & Data Loading
onMounted(async () => {
    // 1. Fetch Questions & Module Config
    try {
        const res = await fetch(`${API_BASE_URL}/events/${eventId}`)
        const data = await res.json()
        const mod = data.modules?.find(m => m.id === moduleId)
        if (mod && mod.data) {
            if (mod.data.questions) {
                questions.value = mod.data.questions
            }
            // Music Config
            backgroundMusicUrl.value = mod.data.backgroundMusicUrl || ''
            audioOutput.value = mod.data.audioOutput || 'bigscreen'
            manageAudioPlayback()
        }
    } catch (e) {
        console.error('Failed to load questions', e)
    }

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
    clearInterval(timerInterval)
    clearInterval(explainTimerInterval)
    quizWS.disconnect()
    quizWS.clearHandlers()
})
</script>

<style scoped>
.quiz-host-screen {
    width: 100vw;
    height: 100vh;
    background-color: #121212;
    color: white;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    font-family: 'Inter', sans-serif;
}

.language-switcher-container {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 100;
}

.lang-select {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.2);
    padding: 6px 10px;
    border-radius: 8px;
    font-size: 12px;
    cursor: pointer;
    outline: none;
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
    padding: 8px 12px;
    border-radius: 20px;
    cursor: pointer;
    font-size: 0.9rem;
    backdrop-filter: blur(10px);
    transition: all 0.2s;
}

.unmute-btn:hover {
    background-color: rgba(255, 255, 255, 0.3);
}

/* Dynamic background */
.bg-layer {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    background: radial-gradient(circle at center, #2c2c2e 0%, #000 100%);
    z-index: 0;
}

.language-switcher-container {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 100;
}

.lang-select {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.2);
    padding: 6px 10px;
    border-radius: 8px;
    font-size: 12px;
    cursor: pointer;
    outline: none;
}

.state-container {
    z-index: 1;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    padding: 20px;
}

/* LOBBY */
.lobby-header {
    text-align: center;
    margin-top: 40px;
}

.game-title { font-size: 3rem; margin-bottom: 20px; text-shadow: 0 4px 10px rgba(0,0,0,0.5); }

.pin-display {
    background: white;
    color: black;
    padding: 10px 30px;
    border-radius: 8px;
    font-size: 2rem;
    font-weight: bold;
    display: flex;
    gap: 10px;
    justify-content: center;
    align-items: center;
}

.qr-container {
    margin-top: 20px;
    width: 150px;
    height: 150px;
    background: white;
    margin-left: auto;
    margin-right: auto;
    color: black;
}

.lobby-actions {
    margin-top: 20px;
    display: flex;
    justify-content: center;
}

.copy-link-btn {
    background-color: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid rgba(255, 255, 255, 0.3);
    padding: 10px 20px;
    border-radius: 8px;
    cursor: pointer;
    font-size: 1rem;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s;
}

.copy-link-btn:hover {
    background-color: rgba(255, 255, 255, 0.2);
    transform: translateY(-2px);
}

.players-area {
    margin-top: 40px;
    width: 100%;
    max-width: 1000px;
}

.players-count {
    font-size: 1.5rem;
    margin-bottom: 20px;
    text-align: center;
}

.players-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    justify-content: center;
}

.player-chip {
    background-color: rgba(255,255,255,0.1);
    padding: 10px 20px;
    border-radius: 20px;
    font-weight: 600;
    animation: popIn 0.3s ease-out;
}

@keyframes popIn {
    0% { transform: scale(0); opacity: 0; }
    80% { transform: scale(1.1); }
    100% { transform: scale(1); opacity: 1; }
}

.lobby-footer {
    margin-top: auto;
    margin-bottom: 40px;
}

.start-btn {
    background-color: #32D74B;
    color: white;
    border: none;
    font-size: 1.5rem;
    padding: 15px 40px;
    border-radius: 8px;
    cursor: pointer;
    font-weight: bold;
    transition: transform 0.1s;
}
.start-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.start-btn:hover:not(:disabled) { transform: scale(1.05); }

/* QUESTION */
.question-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    padding: 2vw;
    box-sizing: border-box;
}

.question-header {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 10vh;
}

.timer {
    font-size: 2.5rem;
    font-weight: 900;
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: rgba(255,255,255,0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    border: 4px solid #0A84FF;
}

.timer-low { border-color: #FF453A; color: #FF453A; animation: pulse 1s infinite; }

@keyframes pulse {
    0% { transform: scale(1); }
    50% { transform: scale(1.1); }
    100% { transform: scale(1); }
}

/* New Two-Column Layout */
.question-main-content {
    flex: 1;
    display: flex;
    gap: 30px;
    margin-top: 20px;
    min-height: 0;
    width: 100%;
}

/* LEFT COLUMN: Question Card - 60% */
.question-column {
    flex: 0 0 60%;
    display: flex;
    flex-direction: column;
    min-width: 0;
}

.question-card {
    background: linear-gradient(135deg, #1C1C1E 0%, #2C2C2E 100%);
    border-radius: 16px;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-shadow: 0 8px 32px rgba(0,0,0,0.4);
    border: 1px solid rgba(255,255,255,0.1);
    position: relative;
    overflow: hidden; /* Ensures image and overlay stay inside */
}

/* If no image, center text vertically */
.question-card:not(.has-image) {
    justify-content: center;
    align-items: center;
    padding: 30px;
}

/* IMAGE CONTAINER (Background) */
.question-image-container {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    z-index: 1;
}

.question-image {
    width: 100%;
    height: 100%;
    object-fit: cover; /* Smart cropping */
}

/* Gradient Overlay for text readability */
.image-overlay-gradient {
    position: absolute;
    bottom: 0; left: 0; right: 0;
    height: 75%; /* Increased from 60% */
    background: linear-gradient(to bottom, rgba(0,0,0,0) 0%, rgba(0,0,0,0.8) 50%, rgba(0,0,0,0.95) 100%);
    z-index: 2;
}

/* TEXT OVERLAY AREA */
.question-text-overlay {
    position: absolute;
    bottom: 0; left: 0; right: 0;
    padding: 30px 40px;
    z-index: 3;
    display: flex;
    justify-content: center;
    align-items: flex-end; /* Text sits at bottom */
    text-align: center;
    background: transparent;
}

/* Center text when no image */
.question-card:not(.has-image) .question-text-overlay {
    height: 100%;
    align-items: center;
    justify-content: center;
    padding: 20px;
}

/* Slider Instruction */
.slider-instruction-view {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 16px;
    padding: 20px;
    border: 1px solid rgba(255,255,255,0.1);
}

.slider-instruction-text {
    font-size: 2.5rem;
    font-weight: 800;
    margin-bottom: 40px;
    text-align: center;
    color: #fff;
    text-shadow: 0 2px 4px rgba(0,0,0,0.5);
}

.slider-demo-track {
    width: 80%;
    height: 10px;
    background: rgba(255,255,255,0.2);
    border-radius: 5px;
    position: relative;
    overflow: visible;
}

.slider-demo-thumb {
    width: 40px;
    height: 40px;
    background: #0A84FF;
    border-radius: 50%;
    position: absolute;
    top: 50%;
    transform:translate(-50%, -50%);
    box-shadow: 0 0 15px rgba(10, 132, 255, 0.6);
    animation: slide-demo 2s infinite ease-in-out;
}

@keyframes slide-demo {
    0% { left: 0%; opacity: 0; }
    10% { opacity: 1; }
    50% { left: 100%; opacity: 1; }
    90% { opacity: 1; }
    100% { left: 100%; opacity: 0; }
}

/* Slider Results - New Design */
.slider-results-view {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.slider-result-container {
    width: 90%;
    height: 300px;
    position: relative;
    display: flex;
    align-items: flex-end;
    margin-bottom: 50px; /* Space for labels below */
}

/* Track Line */
.slider-result-track {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: rgba(255, 255, 255, 0.3);
    border-radius: 2px;
}

/* Distribution Bars */
.slider-distribution-overlay {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 4px; /* Sit on track */
    display: flex;
    align-items: flex-end;
}

.distribution-bar-wrapper {
    position: absolute;
    bottom: 0;
    height: 100%;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    /* padding: 0 2px; gap between bars */ 
}

.distribution-bar {
    width: 80%;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 4px 4px 0 0;
    transition: height 1s ease-out;
    position: relative;
    min-height: 0; /* Allow 0 height */
}

.distribution-bar.has-votes {
    background: rgba(255, 255, 255, 0.6);
    box-shadow: 0 0 10px rgba(255,255,255,0.2);
}

.bar-tooltip {
    position: absolute;
    top: -25px;
    left: 50%;
    transform: translateX(-50%);
    font-size: 0.9rem;
    font-weight: bold;
    color: white;
}

/* Correct Marker */
.correct-answer-marker {
    position: absolute;
    bottom: 0;
    height: 120%; /* Go taller than bars */
    width: 0; /* Just a line position */
    z-index: 10;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    opacity: 0; /* Hidden initially */
    transform: translateY(20px);
    transition: all 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.correct-answer-marker.reveal-marker {
    opacity: 1;
    transform: translateY(0);
}

.marker-line {
    width: 4px;
    height: 100%;
    background: #32D74B; /* Neon Green */
    box-shadow: 0 0 15px #32D74B;
    border-radius: 4px;
}

.marker-label {
    position: absolute;
    top: -60px; /* Float above everything */
    background: #32D74B;
    color: black;
    padding: 8px 16px;
    border-radius: 8px;
    font-weight: 800;
    display: flex;
    flex-direction: column;
    align-items: center;
    box-shadow: 0 4px 15px rgba(0,0,0,0.5);
    white-space: nowrap;
    animation: popIn 0.5s cubic-bezier(0.175, 0.885, 0.32, 1.275) 1s backwards; /* Delay appearance */
}

.marker-label::after {
    content: '';
    position: absolute;
    bottom: -6px;
    left: 50%;
    transform: translateX(-50%);
    border-width: 6px 6px 0;
    border-style: solid;
    border-color: #32D74B transparent transparent transparent;
}

.marker-title {
    font-size: 0.8rem;
    text-transform: uppercase;
    opacity: 0.8;
}

.marker-value {
    font-size: 1.5rem;
    line-height: 1;
}

/* Labels */
.slider-labels {
    position: absolute;
    bottom: -40px;
    left: 0;
    right: 0;
    display: flex;
    justify-content: space-between;
    font-size: 1.2rem;
    color: rgba(255,255,255,0.6);
    font-weight: 600;
}

.question-text {
    font-weight: 900;
    line-height: 1.25;
    color: #FFFFFF;
    margin: 0;
    text-shadow: 0 2px 4px rgba(0,0,0,0.8);
    word-wrap: break-word;
    white-space: pre-wrap;
    width: 100%;
}

/* RIGHT COLUMN: Stacked Answers - 40% */
.answers-column {
    flex: 0 0 40%;
    display: flex;
    flex-direction: column;
    gap: 15px;
}

.answer-card-stacked {
    display: flex;
    align-items: center;
    padding: 20px 25px;
    border-radius: 12px;
    background: #000;
    border: 3px solid transparent;
    box-shadow: 0 4px 15px rgba(0,0,0,0.5);
    transition: transform 0.2s, box-shadow 0.2s;
    flex: 1;
    min-height: 80px;
}

.answer-card-stacked:hover {
    transform: translateX(-5px);
    box-shadow: 0 6px 20px rgba(0,0,0,0.4);
}

.answer-card-stacked .answer-icon {
    font-size: 2.5rem;
    margin-right: 20px;
    opacity: 0.9;
    flex-shrink: 0;
}

.answer-card-stacked .answer-text {
    font-size: inherit;
    font-weight: 700;
    color: white;
    text-shadow: 1px 1px 2px rgba(0,0,0,0.5);
    word-break: break-word;
    white-space: pre-wrap;
    line-height: 1.2;
}

/* Answer Count Widget */
.answer-counter-widget {
    position: absolute;
    bottom: 80px;
    right: 30px;
    background: #2C2C2E;
    padding: 15px 25px;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    align-items: center;
    box-shadow: 0 0 15px rgba(0,0,0,0.5);
    z-index: 10;
    border: 2px solid #555;
}

.counter-value {
    font-size: 2.5rem;
    font-weight: 900;
    line-height: 1;
}

.counter-label {
    font-size: 0.8rem;
    text-transform: uppercase;
    opacity: 0.7;
    margin-top: 5px;
}

.color-0 { background: #000; border-color: #FF3B30; box-shadow: 0 4px 15px rgba(255,59,48,0.3); }
.color-1 { background: #000; border-color: #0A84FF; box-shadow: 0 4px 15px rgba(10,132,255,0.3); }
.color-2 { background: #000; border-color: #FFD60A; box-shadow: 0 4px 15px rgba(255,214,10,0.3); }
.color-3 { background: #000; border-color: #32D74B; box-shadow: 0 4px 15px rgba(50,215,75,0.3); }

.color-true { background: #000; border-color: #0A84FF; box-shadow: 0 4px 15px rgba(10,132,255,0.3); }
.color-false { background: #000; border-color: #FF3B30; box-shadow: 0 4px 15px rgba(255,59,48,0.3); }

/* CORRECT PICTURE HOST STYLES */
.correct-picture-host-layout {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
}

.cp-host-grid {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;
    padding: 10px;
    align-items: center;
}

.cp-host-card {
    position: relative;
    height: 100%; /* max-height handled by grid/flex */
    display: flex;
    justify-content: center;
    align-items: center;
    background: #1C1C1E;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #333;
}

.cp-host-img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
}

.cp-host-badge {
    position: absolute;
    top: 10px;
    left: 10px;
    width: 30px;
    height: 30px;
    background: rgba(0,0,0,0.7);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    border: 1px solid white;
}

.cp-host-text {
    padding: 20px;
    text-align: center;
}

.cp-host-text h2 {
    font-size: 1.5rem;
    margin: 0;
}

/* CP HOST RESULTS */
/* ===== CORRECT PICTURE RESULTS HOST — HERO LAYOUT ===== */
@keyframes cpHeroReveal {
    from { opacity: 0; transform: scale(0.7); }
    to   { opacity: 1; transform: scale(1); }
}

@keyframes cpCheckPop {
    from { opacity: 0; transform: scale(0); }
    to   { opacity: 1; transform: scale(1); }
}

@keyframes cpThumbFade {
    from { opacity: 0.5; transform: scale(0.85); }
    to   { opacity: 0.2; transform: scale(1); }
}
.correct-picture-results-host {
    width: 100%;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 0 16px 12px;
}

.cp-host-correct-hero {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
}

.cp-host-hero-frame {
    position: relative;
    border-radius: 20px;
    overflow: hidden;
    max-height: 60vh;
    max-width: 65vw;
    width: 100%;
    aspect-ratio: 4/3;
    animation: cpHeroReveal 0.7s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
    border: 3px solid #32D74B;
    box-shadow:
        0 0 0 6px rgba(50, 215, 75, 0.2),
        0 0 40px rgba(50, 215, 75, 0.6),
        0 0 80px rgba(50, 215, 75, 0.25),
        0 20px 60px rgba(0,0,0,0.5);
}

.cp-host-hero-glow {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse at center, rgba(50,215,75,0.12) 0%, transparent 70%);
    z-index: 1;
    pointer-events: none;
}

.cp-host-hero-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
}

.cp-host-hero-check {
    position: absolute;
    top: 14px;
    right: 14px;
    background: #32D74B;
    color: white;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.8rem;
    font-weight: 900;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
    z-index: 5;
    animation: cpCheckPop 0.5s 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.cp-host-hero-label {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 18px 16px 14px;
    background: linear-gradient(to top, rgba(0,0,0,0.85) 0%, transparent 100%);
    color: #fff;
    font-size: 1.4vw;
    font-weight: 700;
    text-align: center;
    z-index: 5;
    text-shadow: 0 2px 6px rgba(0,0,0,0.7);
}

.cp-host-wrong-thumbs {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
}

.cp-host-thumb-card {
    position: relative;
    width: 90px;
    border-radius: 10px;
    overflow: hidden;
    border: 1px solid rgba(255,255,255,0.1);
    opacity: 0.2;
    filter: grayscale(100%) brightness(0.6);
    animation: cpThumbFade 0.5s ease forwards;
}

.cp-host-thumb-img {
    width: 100%;
    aspect-ratio: 4/3;
    object-fit: cover;
    display: block;
}

.cp-host-thumb-count {
    position: absolute;
    bottom: 3px;
    right: 5px;
    background: rgba(0,0,0,0.7);
    color: #fff;
    font-size: 0.75rem;
    font-weight: bold;
    padding: 1px 5px;
    border-radius: 6px;
}

.cp-result-host-card.is-correct .cp-host-bar-fill {
    background: #32D74B;
}

.cp-host-count {
    position: absolute;
    right: 10px;
    top: 50%;
    transform: translateY(-50%);
    font-size: 0.8rem;
    font-weight: bold;
}

/* WRONG PICTURES HOST STYLES */
.wrong-picture-host-layout, .wrong-picture-results-magnified {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    padding: 20px;
    box-sizing: border-box;
    overflow: hidden;
}

.wp-host-header {
    text-align: center;
    margin-bottom: 20px;
    flex-shrink: 0;
}

.wp-host-title {
    font-size: 2.5rem;
    font-weight: 900;
    color: #fff;
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 2px;
}

.wp-host-instruction {
    font-size: 1.2rem;
    color: #8E8E93;
    margin-top: 5px;
}

.wp-host-image-container {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 0;
    position: relative;
    background: #1C1C1E;
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid #333;
}

.wp-host-big-img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
}

/* MAGNIFIED RESULTS LAYOUT */
.wp-magnified-results-grid {
    flex: 1;
    display: flex;
    gap: 30px;
    min-height: 0;
    width: 100%;
    margin: 0 auto;
}

.wp-magnified-column {
    flex: 7; /* 70% of space */
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    grid-auto-rows: min-content;
    gap: 25px;
    overflow-y: auto;
    padding: 20px;
    align-content: center;
}

.wp-labels-column {
    flex: 3; /* 30% of space */
    display: flex;
    flex-direction: column;
    gap: 12px;
    justify-content: center;
    overflow-y: auto;
    padding: 10px;
}

/* Magnified Crop Card */
.wp-magnified-crop-card {
    background: #2C2C2E;
    border-radius: 12px;
    padding: 6px;
    border: 1px solid #444;
    box-shadow: 0 4px 15px rgba(0,0,0,0.4);
    transition: transform 0.2s ease;
}

.wp-magnified-crop-card:hover {
    transform: scale(1.02);
    border-color: #32D74B;
}

.wp-crop-container {
    position: relative;
    width: 100%;
    border-radius: 6px;
    overflow: hidden;
    background: #000;
}

.wp-crop-container.square {
    aspect-ratio: 1 / 1;
}

.wp-crop-image {
    width: 100%;
    height: 100%;
    background-repeat: no-repeat;
}

.wp-crop-badge {
    position: absolute;
    top: 6px;
    left: 6px;
    width: 28px;
    height: 28px;
    background: #32D74B;
    color: #000;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 900;
    font-size: 1rem;
    box-shadow: 0 2px 6px rgba(0,0,0,0.5);
    z-index: 2;
}

/* Result Label Card */
.wp-result-label-card.compact {
    display: flex;
    align-items: center;
    gap: 12px;
    background: #1C1C1E;
    padding: 12px 18px;
    border-radius: 10px;
    border: 1px solid #32D74B;
    box-shadow: 0 2px 10px rgba(50, 215, 75, 0.1);
    animation: wp-label-reveal 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.wp-label-number.small {
    width: 32px;
    height: 32px;
    background: #32D74B;
    color: #000;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.1rem;
    font-weight: 800;
    flex-shrink: 0;
}

.wp-label-text.small {
    font-size: 1.2rem;
    font-weight: 600;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

/* WP Results Targets - Deprecated but keeping style pattern for reference */
.wp-host-target-area {
    position: absolute;
    border: 3px solid #32D74B;
    border-radius: 50%;
    pointer-events: none;
    box-shadow: 0 0 20px rgba(50, 215, 75, 0.6);
    z-index: 10;
}

.wp-host-target-pulse {
    position: absolute;
    inset: -5px;
    border: 2px solid #32D74B;
    border-radius: 50%;
    animation: wp-host-pulse 2s infinite;
}

@keyframes wp-host-pulse {
    0% { transform: scale(1); opacity: 0.8; }
    100% { transform: scale(1.3); opacity: 0; }
}

.wp-host-target-label {
    position: absolute;
    top: -30px;
    left: 50%;
    transform: translateX(-50%);
    background: #32D74B;
    color: #000;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.9rem;
    font-weight: 800;
    white-space: nowrap;
}

/* Responsive Design for Smaller Screens */
@media (max-width: 900px) {
    .question-main-content {
        flex-direction: column;
        gap: 20px;
    }
    
    .question-column {
        flex: none;
        width: 100% !important;
        min-height: 30vh;
    }
    
    .question-card {
        padding: 20px;
    }
    
    .question-image-container {
        min-height: 150px;
    }
    
    .answers-column {
        width: 100% !important;
        flex: 1;
    }
    
    .answer-card-stacked {
        min-height: 60px;
        padding: 15px 20px;
    }
    
    .answer-card-stacked .answer-icon {
        font-size: 1.8rem;
        margin-right: 15px;
    }
    
    .answer-counter-widget {
        bottom: 70px;
        right: 20px;
    }
}

@media (max-width: 600px) {
    .answer-card-stacked .answer-icon {
        font-size: 1.5rem;
    }
}

/* HISTOGRAM */
.histogram-container {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 20px;
}

.histogram-track-host {
    flex: 1;
    display: flex;
    align-items: flex-end;
    gap: 4px;
    border-bottom: 2px solid rgba(255,255,255,0.3);
    padding-bottom: 2px;
}

.histogram-bar-host {
    flex: 1;
    background: #0A84FF;
    min-height: 2px;
    border-radius: 4px 4px 0 0;
    transition: height 0.5s ease-out;
    position: relative;
}

.histogram-bar-host.bar-correct {
    background: #32D74B;
    box-shadow: 0 0 10px rgba(50, 215, 75, 0.5);
}

.bar-count {
    position: absolute;
    top: -20px;
    left: 50%;
    transform: translateX(-50%);
    font-size: 0.8rem;
    font-weight: bold;
}

.histogram-axis {
    display: flex;
    justify-content: space-between;
    margin-top: 10px;
    font-size: 1rem;
    font-weight: bold;
    color: #8E8E93;
    position: relative;
    height: 30px;
}

.correct-marker-label {
    position: absolute;
    color: #32D74B;
    transform: translateX(-50%);
    white-space: nowrap;
}

/* GET READY */
.get-ready-container {
    justify-content: center;
    background-color: #FF3B30;
}
.loading-content {
    text-align: center;
}
.loader {
    width: 60px;
    height: 60px;
    border: 6px solid rgba(255,255,255,0.3);
    border-top: 6px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 30px auto;
}
@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

/* RESULTS */
.results-container {
    padding-top: 20px;
    background: radial-gradient(ellipse at top, #1a1a2e 0%, #0d0d0d 70%);
    display: flex;
    flex-direction: column;
    height: 100vh;
    justify-content: center; /* Vertical centering */
}

.results-content-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    min-height: 0;
}

.host-controls {
    width: 100%;
    display: flex;
    justify-content: center;
    padding: 20px;
    z-index: 10;
}

.results-header {
    margin-bottom: 20px;
}

.results-label {
    font-size: 1rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    font-weight: 600;
}

.correct-label {
    font-size: 0.9rem;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    margin-bottom: 8px;
}

.correct-answer-reveal {
    margin-bottom: 30px;
    text-align: center;
    animation: popIn 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
    transform-origin: center center;
    transform: scale(1.1); /* Slight boost */
}

.correct-answer-display {
    font-size: 2.5rem; /* Larger than before */
    padding: 15px 30px;
    border-radius: 16px;
    margin-top: 12px;
    color: white;
    display: inline-flex;
    align-items: center;
    gap: 16px;
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
    box-shadow: 0 0 40px rgba(255,255,255,0.2);
    border: 2px solid rgba(255,255,255,0.3);
    background-clip: padding-box;
}

.correct-icon { font-size: 1.6rem; }
.correct-text { font-weight: 700; }

.stat-bar-correct {
    box-shadow: 0 0 50px rgba(50, 215, 75, 0.8), 0 0 15px rgba(255,255,255,0.6);
    border: 3px solid rgba(50, 215, 75, 1);
    position: relative;
    z-index: 10;
}

/* LEADERBOARD */
.leaderboard-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-start;
    padding: 30px 40px;
    position: relative;
    overflow: hidden;
    background: radial-gradient(ellipse at 50% 0%, #1a0a3e 0%, #0a0a0a 70%);
}

.lb-bg-glow {
    position: absolute;
    top: -50px;
    left: 50%;
    transform: translateX(-50%);
    width: 80%;
    height: 300px;
    background: radial-gradient(ellipse, rgba(138, 43, 226, 0.12) 0%, transparent 70%);
    pointer-events: none;
}

.lb-header {
    text-align: center;
    margin-bottom: 20px;
    animation: fadeSlideDown 0.6s ease-out;
}

.lb-trophy {
    font-size: 2.5rem;
    filter: drop-shadow(0 0 15px rgba(255, 215, 0, 0.6));
    animation: trophyPulse 2s ease-in-out infinite;
}

@keyframes trophyPulse {
    0%, 100% { transform: scale(1); filter: drop-shadow(0 0 15px rgba(255, 215, 0, 0.6)); }
    50% { transform: scale(1.05); filter: drop-shadow(0 0 25px rgba(255, 215, 0, 0.9)); }
}

.lb-title {
    font-size: 2rem;
    font-weight: 900;
    background: linear-gradient(135deg, #FFD700, #FFA500);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin: 0;
    letter-spacing: 0.05em;
}

.lb-subtitle {
    font-size: 0.85rem;
    color: #888;
    margin-top: 4px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
}

.lb-list {
    width: 100%;
    max-width: 700px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
    overflow: hidden;
}

.lb-row {
    display: flex;
    align-items: center;
    gap: 16px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    padding: 10px 16px;
    animation: slideInFromBottom 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
    backdrop-filter: blur(10px);
}

.lb-rank-1 { border-color: rgba(255, 215, 0, 0.4); background: rgba(255, 215, 0, 0.08); }
.lb-rank-2 { border-color: rgba(192, 192, 192, 0.3); background: rgba(192, 192, 192, 0.05); }
.lb-rank-3 { border-color: rgba(205, 127, 50, 0.3); background: rgba(205, 127, 50, 0.05); }

@keyframes slideInFromBottom {
    from { opacity: 0; transform: translateY(30px) scale(0.95); }
    to { opacity: 1; transform: translateY(0) scale(1); }
}

.lb-rank-badge {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    font-size: 1.2rem;
    font-weight: 900;
    flex-shrink: 0;
    background: rgba(255,255,255,0.1);
    color: #ccc;
}

.lb-badge-1 { background: rgba(255, 215, 0, 0.2); font-size: 1.4rem; }
.lb-badge-2 { background: rgba(192, 192, 192, 0.2); font-size: 1.4rem; }
.lb-badge-3 { background: rgba(205, 127, 50, 0.2); font-size: 1.4rem; }

.lb-player-name {
    flex: 1;
    font-size: 1.2rem;
    font-weight: 700;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.lb-score-area {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 35%;
}

.lb-score-bar-wrap {
    flex: 1;
    height: 6px;
    background: rgba(255,255,255,0.1);
    border-radius: 3px;
    overflow: hidden;
}

.lb-score-bar {
    height: 100%;
    border-radius: 3px;
    transition: width 1s cubic-bezier(0.34, 1.56, 0.64, 1);
    background: linear-gradient(90deg, #8B5CF6, #A78BFA);
}

.lb-bar-1 { background: linear-gradient(90deg, #FFD700, #FFA500); }
.lb-bar-2 { background: linear-gradient(90deg, #C0C0C0, #E8E8E8); }
.lb-bar-3 { background: linear-gradient(90deg, #CD7F32, #E8A87C); }

.lb-score-value {
    font-size: 1.1rem;
    font-weight: 800;
    color: #FFD700;
    min-width: 70px;
    text-align: right;
}

/* FINAL LEADERBOARD */
.final-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-start;
    padding: 20px 30px;
    position: relative;
    overflow: hidden;
    background: radial-gradient(ellipse at 50% 0%, #1a0a3e 0%, #0a0a0a 70%);
}

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
    margin-bottom: 16px;
    animation: fadeSlideDown 0.7s ease-out;
    z-index: 1;
}

.final-trophy {
    font-size: 2.5rem;
    filter: drop-shadow(0 0 20px rgba(255, 215, 0, 0.8));
    animation: trophyPulse 2s ease-in-out infinite;
}

.final-title {
    font-size: 2.2rem;
    font-weight: 900;
    background: linear-gradient(135deg, #FFD700, #FF6B6B, #4ECDC4);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin: 4px 0;
}

.final-subtitle {
    font-size: 0.9rem;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.15em;
    margin: 0;
}

.final-podium {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    gap: 16px;
    margin-bottom: 16px;
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
    from { opacity: 0; transform: translateY(40px) scale(0.8); }
    to { opacity: 1; transform: translateY(0) scale(1); }
}

.fp-crown {
    font-size: 1.8rem;
    animation: crownFloat 2s ease-in-out infinite;
    filter: drop-shadow(0 0 8px rgba(255, 215, 0, 0.8));
}

@keyframes crownFloat {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-4px); }
}

.fp-avatar { font-size: 2rem; margin-bottom: 4px; }

.fp-name {
    font-size: 1rem;
    font-weight: 700;
    color: #fff;
    text-align: center;
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin-bottom: 2px;
}

.fp-score {
    font-size: 0.9rem;
    font-weight: 800;
    margin-bottom: 4px;
}

.fp-first .fp-score { color: #FFD700; }
.fp-second .fp-score { color: #C0C0C0; }
.fp-third .fp-score { color: #CD7F32; }

.fp-block {
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px 6px 0 0;
    font-size: 1.5rem;
    font-weight: 900;
    color: rgba(0,0,0,0.5);
}

.fp-block-1 {
    width: 90px;
    height: 60px;
    background: linear-gradient(180deg, #FFD700, #B8860B);
    box-shadow: 0 0 20px rgba(255, 215, 0, 0.5);
}

.fp-block-2 {
    width: 80px;
    height: 45px;
    background: linear-gradient(180deg, #C0C0C0, #808080);
    box-shadow: 0 0 15px rgba(192, 192, 192, 0.3);
}

.fp-block-3 {
    width: 80px;
    height: 35px;
    background: linear-gradient(180deg, #CD7F32, #8B4513);
    box-shadow: 0 0 15px rgba(205, 127, 50, 0.3);
}

.final-rest {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
    max-width: 500px;
    z-index: 1;
}

.final-rest-row {
    display: flex;
    align-items: center;
    gap: 16px;
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 8px;
    padding: 8px 16px;
    animation: slideInFromBottom 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.final-rest-rank {
    font-size: 1rem;
    font-weight: 900;
    color: #888;
    width: 24px;
    text-align: center;
}

.final-rest-name {
    flex: 1;
    font-size: 1rem;
    font-weight: 600;
    color: #ddd;
}

.final-rest-score {
    font-size: 1rem;
    font-weight: 700;
    color: #aaa;
}

/* Shared animations */
@keyframes fadeSlideDown {
    from { opacity: 0; transform: translateY(-15px); }
    to { opacity: 1; transform: translateY(0); }
}

@keyframes popIn {
    0% { opacity: 0; transform: scale(0.7); }
    70% { transform: scale(1.05); }
    100% { opacity: 1; transform: scale(1); }
}

.control-btn {
    padding: 12px 24px;
    border-radius: 8px;
    border: none;
    font-size: 1rem;
    cursor: pointer;
    background: #3A3A3C; 
    color: white;
}
.control-btn.primary { background: #0A84FF; }

/* ANSWERS STATS (Results Mode) */
.answers-stats {
    height: 260px;
    width: 100%;
}

.stats-grid {
    display: flex;
    height: 100%;
    align-items: flex-end;
    gap: 30px;
    padding: 0 40px;
}

.stat-column {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
    justify-content: flex-end;
}

.stat-col-wrong {
    opacity: 0.15;
    filter: grayscale(100%);
    transform: scale(0.95);
    transition: opacity 0.6s ease, filter 0.6s ease, transform 0.6s ease;
}

.stat-col-correct {
    position: relative;
    z-index: 2;
    animation: popIn 0.5s 0.2s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.stat-bar-container {
    flex: 1;
    width: 100%;
    display: flex;
    align-items: flex-end;
    justify-content: center;
    background: rgba(255,255,255,0.05);
    border-radius: 6px;
    overflow: hidden;
}

.stat-bar {
    width: 60%;
    min-height: 10px;
    transition: height 0.8s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.stat-count {
    font-size: 1.4rem;
    font-weight: bold;
    margin-top: 8px;
}

.stat-icon {
    font-size: 1.4rem;
    margin-top: 4px;
}

/* ===== NEW RESULTS HOST LAYOUT ===== */
.results-container-new {
    display: flex;
    flex-direction: column;
    padding: 2vw;
    box-sizing: border-box;
    height: 100vh;
    background: radial-gradient(ellipse at top, #1a1a2e 0%, #0d0d0d 70%);
}

.results-content-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    width: 100%;
    min-height: 0; 
}

.results-mode-title {
    font-size: 2rem;
    font-weight: bold;
    color: #fff;
    margin-right: auto;
}

/* Dimmed Image */
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
    padding-top: 20px;
    background: rgba(0,0,0,0.3);
}

.dist-chart {
    display: flex;
    gap: 20px;
    width: 80%;
    height: 50%;
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
    min-height: 4px;
    transition: height 1s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 0 10px rgba(0,0,0,0.5);
    opacity: 0.9;
}

.dist-count {
    font-size: 1.5rem;
    font-weight: 900;
    color: white;
    text-shadow: 0 2px 4px rgba(0,0,0,0.8);
    margin-top: 5px;
}

.dist-icon {
    font-size: 1.2rem;
    margin-top: 5px;
    text-shadow: 0 2px 4px rgba(0,0,0,0.8);
}

/* Result Answer Cards */
.result-card {
    transition: all 0.5s ease;
    border: 3px solid transparent; 
}

.result-wrong {
    opacity: 0.3;
    filter: grayscale(100%);
    transform: scale(0.98);
}

.result-correct {
    opacity: 1;
    transform: scale(1.02);
    box-shadow: 0 0 30px rgba(50, 215, 75, 0.7), inset 0 0 0 1px rgba(50, 215, 75, 0.2);
    border-color: #32D74B !important;
    background: #000 !important;
    z-index: 5;
}

.result-check-icon {
    font-size: 1.8rem;
    color: #fff;
    font-weight: 900;
    margin-left: 10px;
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

</style>
