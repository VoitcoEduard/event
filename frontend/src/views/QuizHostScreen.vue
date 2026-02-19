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
                🔗 Copy Big Screen Link
            </button>
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

    <!-- QUESTION STATE -->
    <div v-else-if="gameState === 'question-active'" class="state-container question-container">
      <div class="question-header">
        <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }">
          {{ timeRemaining }}
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
    <div v-else-if="gameState === 'results'" class="state-container results-container">
       <div class="results-header">
         <span class="results-label">Question {{ currentQuestionIndex + 1 }} Results</span>
       </div>

       <!-- CORRECT PICTURE RESULTS -->
       <div v-if="isCorrectPicture" class="correct-picture-results-host">
           <div class="cp-results-host-grid">
              <div 
                v-for="(option, idx) in currentQuestion.imageOptions" 
                :key="idx"
                class="cp-result-host-card"
                :class="{ 
                    'is-correct': idx === currentQuestion.correctIndex,
                    'is-wrong': idx !== currentQuestion.correctIndex
                }"
              >
                 <div class="cp-result-host-img-wrapper">
                    <img :src="getImageUrl(option.imageId)" class="cp-host-img" />
                    <div v-if="idx === currentQuestion.correctIndex" class="cp-host-correct-badge">✓</div>
                 </div>
                 <div class="cp-host-bar-container">
                    <div class="cp-host-bar-fill" :style="{ width: getBarHeight(idx) + '%' }"></div>
                    <span class="cp-host-count">{{ answerCounts[idx] || 0 }}</span>
                 </div>
              </div>
           </div>
       </div>

       <div v-else-if="!isSlider" class="correct-answer-reveal">
          <p class="correct-label">Correct Answer</p>
          <div class="correct-answer-display" :class="getAnswerColorClass(currentQuestion?.correctIndex)">
             <span class="correct-icon">{{ getShapeIcon(currentQuestion?.correctIndex) }}</span>
             <span class="correct-text">{{ currentQuestion?.answers[currentQuestion?.correctIndex] }}</span>
          </div>
       </div>

       <!-- Show Stats in Results -->
       <div v-if="!isCorrectPicture && !isSlider" class="answers-stats results-mode">
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

        <!-- Slider Histogram Stats -->
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
                :style="{ width: getScoreBarWidth(player.score) + '%' }"
              ></div>
            </div>
            <div class="lb-score-value">{{ player.score.toLocaleString() }}</div>
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

// Timer logic
let timerInterval
const startLocalTimer = (duration) => {
    clearInterval(timerInterval)
    timeRemaining.value = duration
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
        (gameState.value !== 'finished') 

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

const copyBigScreenLink = () => {
    const url = `${window.location.origin}/bigscreen/${eventId}/${moduleId}`
    navigator.clipboard.writeText(url)
    success('Big Screen link copied to clipboard!')
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
        } else if (payload.state === 'results') {
            clearInterval(timerInterval)
        }
        
        manageAudioPlayback()
    })

    // Listen for minimal sync updates (answer counts, player info)
    quizWS.on('sync:minimal', (payload) => {
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
    })

    // Listen for full state updates (legacy support - includes players)
    quizWS.on('state:update', (session) => {
        // Update players
        if (session.Players || session.players) {
            const playersMap = session.Players || session.players
            players.value = Object.values(playersMap)
        }
        
        // Update Player Signals (for histogram)
        if (session.PlayerSignals || session.playerSignals) {
             const signals = session.PlayerSignals || session.playerSignals
             playerSignals.value = signals.filter(s => s.questionIndex === currentQuestionIndex.value)
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
    background: white;
    box-shadow: 0 4px 15px rgba(0,0,0,0.3);
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

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }

.color-true { background-color: #0A84FF; }
.color-false { background-color: #FF3B30; }

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
.correct-picture-results-host {
    width: 100%;
    flex: 1;
    display: flex;
    flex-direction: column;
}

.cp-results-host-grid {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 15px;
    padding: 0 20px;
    margin-bottom: 20px;
}

.cp-result-host-card {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.cp-result-host-img-wrapper {
    flex: 1;
    position: relative;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid #333;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #1C1C1E;
}

.cp-result-host-card.is-wrong .cp-result-host-img-wrapper {
    filter: grayscale(100%);
    opacity: 0.5;
}

.cp-result-host-card.is-correct .cp-result-host-img-wrapper {
    border-color: #32D74B;
    box-shadow: 0 0 15px rgba(50, 215, 75, 0.3);
}

.cp-host-correct-badge {
    position: absolute;
    bottom: 5px;
    right: 5px;
    background: #32D74B;
    color: white;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.2rem;
    font-weight: bold;
}

.cp-host-bar-container {
    height: 20px;
    background: #333;
    margin-top: 8px;
    border-radius: 10px;
    position: relative;
    overflow: hidden;
}

.cp-host-bar-fill {
    height: 100%;
    background: #0A84FF;
    transition: width 0.5s ease;
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
    padding-top: 40px;
    background: radial-gradient(ellipse at top, #1a1a2e 0%, #0d0d0d 70%);
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
}

.correct-answer-display {
    font-size: 1.8rem;
    padding: 10px 20px;
    border-radius: 10px;
    margin-top: 8px;
    color: white;
    display: inline-flex;
    align-items: center;
    gap: 12px;
    text-shadow: 0 1px 2px rgba(0,0,0,0.5);
    box-shadow: 0 6px 24px rgba(0,0,0,0.4);
}

.correct-icon { font-size: 1.6rem; }
.correct-text { font-weight: 700; }

.stat-bar-correct {
    box-shadow: 0 0 15px rgba(255,255,255,0.3);
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
    gap: 20px;
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
</style>
