<template>
  <div class="quiz-player-screen">
    <div class="bg-layer"></div>

    <div class="language-switcher-container">
      <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

    <!-- NICKNAME ENTRY (Pre-Lobby) -->
    <div v-if="state === States.LOBBY && !joined" class="state-container name-entry-container">
      <h1 class="logo">{{ $t('quiz.player.logo') }}</h1>
      <div class="card">
        <input
          v-model="nickname"
          :placeholder="$t('quiz.player.nickname_placeholder')"
          class="nickname-input"
          @keyup.enter="joinLobby"
        />
        <button class="join-btn" @click="joinLobby" :disabled="!nickname.trim()">
          {{ $t('quiz.player.join_btn') }}
        </button>
      </div>
    </div>

    <!-- LOBBY STATE -->
    <div v-else-if="state === States.LOBBY && joined" class="state-container lobby-container">
      <div class="lobby-content">
        <h2 class="waiting-text">{{ $t('quiz.player.you_are_in') }}</h2>
        <div class="nickname-display">{{ nickname }}</div>
        <p class="status-msg">{{ $t('quiz.player.see_name') }}</p>
        <div class="loader"></div>
      </div>
    </div>

    <!-- GET READY STATE -->
    <div v-else-if="state === States.GET_READY" class="state-container get-ready-container">
      <div class="ready-content">
        <h1>{{ $t('quiz.player.get_ready') }}</h1>
        <div class="loader"></div>
      </div>
    </div>

    <!-- CORRECT PICTURE STATE -->
    <div 
      v-else-if="state === States.QUESTION_ACTIVE && isCorrectPicture" 
      class="state-container correct-picture-container"
    >
      <div class="question-header-area">
        <div class="question-number-badge">{{ currentQuestionIndex + 1 }}</div>
        <h2 class="question-text">{{ currentQuestion?.text }}</h2>
        <div class="timer-bar">
          <div class="timer-progress" :style="{ width: (timeRemaining / (currentQuestion?.timeLimit || 20)) * 100 + '%' }"></div>
        </div>
      </div>
      
      <!-- Interactive Zone: Horizontal Carousel (Middle 60%) -->
      <div class="carousel-zone">
        <HorizontalCarousel
          :images="imageOptionUrls"
          :initial-index="preselectedIndex"
          @snap="onCarouselSnap"
        />
      </div>
      
      <!-- Action Area: Submit Button (Bottom 20%) -->
      <div class="action-area">
        <button 
          class="submit-selection-btn"
          :class="{ 'active': hasPreselection, 'loading': isSubmitting }"
          :disabled="!hasPreselection || hasSubmitted"
          @click="confirmSelection"
        >
          <span v-if="!hasPreselection">{{ $t('quiz.player.swipe_to_view') }}</span>
          <span v-else-if="isSubmitting">
            <span class="spinner-small"></span>
            {{ $t('quiz.player.submitting') }}
          </span>
          <span v-else>{{ $t('quiz.player.select_item') }}</span>
        </button>
      </div>
    </div>

    <!-- SLIDER QUESTION STATE -->
    <div 
      v-else-if="state === States.QUESTION_ACTIVE && isSlider" 
      class="state-container slider-container"
    >
      <div class="question-info">
        <div class="q-number">{{ currentQuestionIndex + 1 }}</div>
      </div>
      
      <div class="slider-question-content">
        <h2 class="question-text">{{ currentQuestion?.text }}</h2>
        
        <div class="slider-wrapper">
          <SliderComponent
            :min="currentQuestion?.sliderConfig?.minValue"
            :max="currentQuestion?.sliderConfig?.maxValue"
            :step="currentQuestion?.sliderConfig?.step"
            :unit="currentQuestion?.sliderConfig?.unit"
            :decimalPlaces="currentQuestion?.sliderConfig?.decimalPlaces"
            :orientation="isMobile ? 'vertical' : 'horizontal'"
            v-model="sliderValue"
            @drag-end="onSliderDragEnd"
          />
        </div>
        
        <p class="instruction-text">{{ $t('quiz.player.slider_instruction') }}</p>
      </div>
    </div>

    <!-- QUESTION/ANSWER STATE (Generic/Multiple Choice) -->
    <div v-else-if="state === States.QUESTION_ACTIVE" class="state-container question-container">
      <div class="question-info">
         <div class="q-number">{{ currentQuestionIndex + 1 }}</div>
      </div>

      <div class="answer-grid" :class="{ 'true-false-grid': isTrueFalse, 'multi-select-grid': isMultiSelect }">
         <button
            v-for="(shape, idx) in shapes"
            :key="idx"
            class="answer-btn"
            :class="[
                isTrueFalse ? (idx === 0 ? 'color-true' : 'color-false') : 'color-' + idx, 
                { 'selected': isSelected(idx), 'dimmed': hasSelected && !isSelected(idx) }
            ]"
            @click="handleAnswerClick(idx)"
            :disabled="hasSubmitted"
         >
            <span class="shape-icon">{{ shape }}</span>
            <span v-if="isTrueFalse" class="tf-label">{{ idx === 0 ? $t('quiz.player.true') : $t('quiz.player.false') }}</span>
            <div v-if="isMultiSelect && isSelected(idx)" class="checkmark-badge">✓</div>
         </button>
      </div>
      
      <div v-if="isMultiSelect && !hasSubmitted" class="action-bar">
          <button class="submit-btn" @click="submitMultiSelect" :disabled="selectedIndices.length === 0">
              {{ $t('quiz.player.submit_btn') }}
          </button>
      </div>
    </div>

    <!-- ANSWER SUBMITTED (Waiting) -->
    <div v-else-if="state === States.ANSWERED" class="state-container answered-container">
       <div class="answered-content">
          <div class="spinner"></div>
          <h2>{{ $t('quiz.player.submitted') }}</h2>
          <p>{{ $t('quiz.player.waiting_others') }}</p>
       </div>
    </div>

    <!-- RESULT STATE -->
    <div v-else-if="state === States.RESULTS" class="state-container result-container" :class="isCorrect ? 'bg-correct' : 'bg-wrong'">
       <div class="result-content">
          <div class="result-icon" :class="isCorrect ? 'icon-correct' : 'icon-wrong'">{{ isCorrect ? '✅' : '❌' }}</div>
          <h1 class="result-title">{{ isCorrect ? $t('quiz.player.correct') : $t('quiz.player.incorrect') }}</h1>

          <div class="score-summary">
             <div class="score-delta" v-if="isCorrect">
               <span class="plus-sign">+</span>{{ currentPoints.toLocaleString() }}
             </div>
             <div class="score-delta wrong-delta" v-else>{{ $t('quiz.player.incorrect') }}</div>
             <div class="streak-badge" v-if="streak > 2">
               🔥 {{ $t('quiz.player.streak', { streak: streak }) }}
             </div>
          </div>

          <div class="rank-chip" v-if="rank">
            🏅 {{ $t('quiz.player.rank_msg', { rank: rank }) }}
          </div>
       </div>
       <div class="footer-score">
          {{ $t('quiz.player.total_score', { score: score.toLocaleString() }) }}
       </div>
    </div>

    <!-- FINISHED STATE -->
    <div v-else-if="state === States.FINISHED" class="state-container finished-container">
       <div class="finished-content">
          <h1>{{ $t('quiz.player.quiz_over') }}</h1>
          <div class="final-rank-badge">
             <span class="rank-num">{{ rank }}</span>
             <span class="rank-label">{{ $t('quiz.player.place') }}</span>
          </div>
          <p class="final-score">{{ $t('quiz.player.final_score', { score: score }) }}</p>
       </div>
       <button class="exit-btn" @click="goHome">{{ $t('quiz.player.exit_btn') }}</button>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { quizWS } from '../services/websocket'
import { useQuizStateMachine } from '../composables/useQuizStateMachine'
import SliderComponent from '../components/SliderComponent.vue'
import HorizontalCarousel from '../components/HorizontalCarousel.vue'
import { questionCache } from '../utils/questionCache'
import { signalQueue } from '../utils/signalQueue'
import { getUserName } from '../utils'
import { getImageUrl } from '../services/media'

const { locale } = useI18n()
const router = useRouter()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

// State machine
const { state, transition, canTransition, questionCache: cache, currentQuestionIndex, score, streak, States } = useQuizStateMachine()

// Local state
const nickname = ref(getUserName())
const joined = ref(false)
const selectedAnswer = ref(null) // For single choice
const selectedIndices = ref([])  // For multi select
const sliderValue = ref(50)      // For slider
const hasSubmitted = ref(false)
const isCorrect = ref(false)
const currentPoints = ref(0)
const rank = ref(0)
const timeRemaining = ref(5)

// Correct Picture specific state
const preselectedIndex = ref(0)
const isSubmitting = ref(false)
const hasPreselection = ref(true) // Default to first item selected effectively

const onCarouselSnap = (idx) => {
    preselectedIndex.value = idx
    hasPreselection.value = true
}

const confirmSelection = () => {
    if (hasSubmitted.value) return
    isSubmitting.value = true
    
    // Simulate short delay for "working" feel or network
    setTimeout(() => {
        submitAnswer(preselectedIndex.value)
        isSubmitting.value = false
    }, 300)
}

const currentQuestion = computed(() => {
    return questionCache.get(currentQuestionIndex.value)
})

const isTrueFalse = computed(() => {
    return currentQuestion.value?.type === 'true_false'
})

const isMultiSelect = computed(() => {
    return currentQuestion.value?.type === 'multi_select'
})

const isSlider = computed(() => {
    return currentQuestion.value?.type === 'slider'
})

const isMobile = computed(() => {
    return window.innerWidth < 600
})

const isCorrectPicture = computed(() => {
    return currentQuestion.value?.type === 'correct_picture'
})

const imageOptionUrls = computed(() => {
    if (!currentQuestion.value?.imageOptions) return []
    return currentQuestion.value.imageOptions.map(opt => ({
        url: getImageUrl(opt.imageId),
        alt: opt.label,
        id: opt.id
    }))
}) 




const hasSelected = computed(() => {
    return selectedAnswer.value !== null || selectedIndices.value.length > 0
})

const isSelected = (idx) => {
    if (isMultiSelect.value) {
        return selectedIndices.value.includes(idx)
    }
    return selectedAnswer.value === idx
}

const shapes = computed(() => {
    if (isTrueFalse.value) {
        return ['✔️', '❌']
    }
    return ['▲', '◆', '●', '■']
})

// Actions
const joinLobby = () => {
    if (!nickname.value.trim()) return
    quizWS.connect(moduleId, nickname.value, false)
    joined.value = true
}

const handleAnswerClick = (idx) => {
    if (hasSubmitted.value) return

    if (isMultiSelect.value) {
        // Toggle selection
        const i = selectedIndices.value.indexOf(idx)
        if (i > -1) {
            selectedIndices.value.splice(i, 1)
        } else {
            selectedIndices.value.push(idx)
        }
    } else {
        // Submit immediately for single choice
        submitAnswer(idx)
    }
}

const submitMultiSelect = () => {
    if (selectedIndices.value.length === 0) return
    submitAnswer(selectedIndices.value)
}

const onSliderDragEnd = () => {
  if (hasSubmitted.value) return 
  
  // Submit slider value
  selectedAnswer.value = sliderValue.value
  submitAnswer(sliderValue.value)
}

const submitAnswer = (answerData) => {
    hasSubmitted.value = true
    clearInterval(timerInterval) // Stop the timer
    transition(States.ANSWERED)

    // Validate and score answer locally
    validateAndScore(answerData)
}

const goHome = () => {
    quizWS.disconnect()
    // Redirect to event page with ID to ensure they go back to the lobby
    router.push({ path: '/event', query: { id: eventId } })
}

let timerInterval

// Answer validation and scoring
const validateAndScore = (answerData) => {
    // Use questionCache directly instead of the ref from state machine
    const question = questionCache.get(currentQuestionIndex.value)
    if (!question) {
        console.error('Question not found in cache at index:', currentQuestionIndex.value)
        return
    }

    let isCorrectAnswer = false
    let answerPayload = {}

    if (question.type === 'multi_select') {
        // Strict equality check for arrays
        const userIndices = Array.isArray(answerData) ? answerData : []
        // Sort for comparison
        const sortedUser = [...userIndices].sort((a, b) => a - b)
        const sortedCorrect = [...(question.correctIndices || [])].sort((a, b) => a - b)
        
        isCorrectAnswer = JSON.stringify(sortedUser) === JSON.stringify(sortedCorrect)
        answerPayload = { answerIndices: userIndices }
    } else if (question.type === 'slider') {
        const val = Number(answerData) || 0
        answerPayload = { answerValue: val }
        
        // Slider Scoring Logic
        const correctValue = Number(question.correctValue !== undefined ? question.correctValue : 0)
        const scoringConfig = question.scoringConfig || {}
        const maxPoints = Number(scoringConfig.maxPoints) || 1000
        const tolerance = Number(scoringConfig.tolerance) || 0
        const mode = scoringConfig.mode || 'symmetric'

        console.log('[Scoring Debug] Slider:', { val, correctValue, maxPoints, tolerance, mode, question })
        
        const distance = Math.abs(val - correctValue)
        let calculatedScore = 0
        
        if (mode === 'asymmetric') {
             // Price is Right style
             if (val <= correctValue) {
                 if (distance === 0) {
                     calculatedScore = maxPoints
                     isCorrectAnswer = true
                 } else if (tolerance > 0 && distance <= tolerance) {
                     // Linear dropoff
                     calculatedScore = Math.round(maxPoints * Math.max(0, 1 - (distance / tolerance)))
                     if (calculatedScore > 0) isCorrectAnswer = true 
                 }
             } else {
                 calculatedScore = 0
                 isCorrectAnswer = false
             }
        } else {
            // Symmetric
             if (distance <= tolerance + 0.0001) { // Float safety
                 if (distance === 0) {
                     calculatedScore = maxPoints
                 } else {
                     calculatedScore = Math.round(maxPoints * Math.max(0, 1 - (distance / tolerance)))
                 }
                 isCorrectAnswer = calculatedScore > 0
             }
        }
        console.log('[Scoring Debug] Result:', { distance, calculatedScore, isCorrectAnswer })
        
        // Override the generic time-based scoring for slider, or combine them?
        // Plan says: "score = maxPoints * max(0, 1 - (distance / tolerance))"
        // This implies distance is the main factor. Time might be secondary or ignored for slider?
        // Let's assume the calculatedScore IS the score.
        // But the generic logic below modifies `currentPoints`.
        // We should signal this is a special case.
        
        // Let's set a flag or just return early with the score?
        // The generic logic below applies time decay to `baseScore`. 
        // Slider usually allows full time to think. 
        // Let's assume Slider Score is PURELY based on accuracy, not time.
        
        currentPoints.value = calculatedScore
        score.value += currentPoints.value
        if (calculatedScore > 0) streak.value++
        else streak.value = 0
        
        isCorrect.value = isCorrectAnswer
        
        // Send signal
        const timeLimit = question.timeLimit || 20
        const timeSpent = timeLimit - timeRemaining.value 
        
        const signal = {
            questionIndex: currentQuestionIndex.value,
            ...answerPayload,
            scoreEarned: currentPoints.value,
            timeSpent: timeSpent * 1000,
            timestamp: Date.now(),
        }
        signalQueue.enqueue(signal)
        return // Exit generic flow
        
    } else {
        const idx = typeof answerData === 'number' ? answerData : -1
        isCorrectAnswer = idx === question.correctIndex
        answerPayload = { answerIndex: idx }
    }
    
    isCorrect.value = isCorrectAnswer

    // Calculate score based on time remaining and streak
    const timeLimit = question.timeLimit || 20
    const timeSpent = timeLimit - timeRemaining.value // Use actual time limit

    if (isCorrectAnswer) {
        // Time-based scoring: 1000 points for instant answer, 500 points for half time, 250 points for almost time up
        const baseScore = Math.max(250, 1000 - (timeSpent * (750 / timeLimit))) // Adjusted formula
        const streakBonus = Math.min(streak.value * 100, 500) // Max 500 bonus for 5+ streak
        currentPoints.value = Math.floor(baseScore + streakBonus)
        score.value += currentPoints.value
        streak.value++
    } else {
        currentPoints.value = 0
        streak.value = 0
    }

    // Send signal to server
    const signal = {
        questionIndex: currentQuestionIndex.value,
        ...answerPayload,
        scoreEarned: currentPoints.value,
        timeSpent: timeSpent * 1000, // Convert to ms
        timestamp: Date.now(),
    }
    signalQueue.enqueue(signal)
}

// Timer logic from countdown - REMOVED as server handles transition now

// Timer logic for question phase
const startLocalTimer = (timeLimit) => {
    clearInterval(timerInterval)
    timeRemaining.value = timeLimit
    timerInterval = setInterval(() => {
        timeRemaining.value--
        if (timeRemaining.value <= 0) {
            clearInterval(timerInterval)
            transition(States.QUESTION_ENDED)
        }
    }, 1000)
}

// WS & Logic
onMounted(async () => {
    // Auto-join if nickname exists
    if (nickname.value) {
        joinLobby()
    }

    // Listen for WS updates
    quizWS.on('cmd:state', (payload) => {
        const newState = payload.state
        console.log('Received state change:', newState, 'Payload:', payload)
        
        if (newState === States.GET_READY) {
            // "Get Ready" phase - Server drives transition now
            
            // Update question index if provided
            if (payload.currentIndex !== undefined) {
                currentQuestionIndex.value = payload.currentIndex
            }

        
            // Reset for new question
            selectedAnswer.value = null
            selectedIndices.value = []
            sliderValue.value = questionCache.get(currentQuestionIndex.value)?.sliderConfig?.minValue || 0
            hasSubmitted.value = false
            isCorrect.value = false
            currentPoints.value = 0
            
            // Correct Picture reset
            preselectedIndex.value = 0
            isSubmitting.value = false
            hasPreselection.value = true
        }
        
        if (canTransition(newState)) {
            transition(newState, payload)
            
            if (newState === States.QUESTION_ACTIVE) {
                 startLocalTimer(payload?.timeLimit || 20)
            }
        } else {
            console.warn('Invalid state transition attempted:', state.value, '->', newState)
        }
    })

    quizWS.on('data:batch', (batch) => {
        questionCache.addBatch(batch)
        console.log('Batch received, cache now has:', questionCache.cache.size, 'questions')
    })

    quizWS.on('sync:minimal', (payload) => {
        // Handle minimal sync updates
        console.log('Minimal sync:', payload)
    })

    quizWS.on('sync:catchup', (payload) => {
        // Handle reconnection catchup
        console.log('Catchup sync:', payload)
        if (payload.missedQuestions.length > 0) {
            console.warn(`You missed ${payload.missedQuestions.length} questions`)
        }
    })

    quizWS.on('state:update', (session) => {
        const playersMap = session.Players || session.players
        if (!playersMap) return

        const players = Object.values(playersMap)
        const myId = quizWS.userId

        // Find current player in the session data
        const me = players.find(p => p.id === myId || p.nickname === nickname.value)

        if (me) {
            // Update authoritative score and rank from server
            score.value = me.score

            const sorted = players.sort((a, b) => b.score - a.score)
            rank.value = sorted.findIndex(p => p.id === me.id) + 1
            console.log(`Updated rank from server: ${rank.value}, score: ${score.value}`)
        }
    })
})

onUnmounted(() => {
    clearInterval(timerInterval)
    quizWS.disconnect()
    quizWS.clearHandlers()
})
</script>

<style scoped>
.quiz-player-screen {
    width: 100vw;
    height: 100vh;
    height: 100dvh; /* dynamic viewport height - accounts for mobile browser bars */
    background-color: #46178F; /* Standard Kahoot-like purple */
    color: white;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    font-family: 'Inter', sans-serif;
    box-sizing: border-box;
}

.bg-layer {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    background: radial-gradient(circle at center, rgba(0,0,0,0) 0%, rgba(0,0,0,0.2) 100%);
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
    overflow: hidden;
    box-sizing: border-box;
    min-height: 0; /* allows flex children to shrink below their content size */
    padding-top: 56px; /* clear the language switcher (top: 10px + ~30px height + buffer) */
}

/* NAME ENTRY */
.name-entry-container {
    justify-content: center;
    padding: 20px;
}

.logo {
    font-size: 3rem;
    font-weight: 900;
    margin-bottom: 40px;
    text-shadow: 2px 2px 0px #000;
}

.card {
    background: white;
    padding: 20px;
    border-radius: 4px;
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.nickname-input {
    padding: 15px;
    border: 2px solid #ccc;
    border-radius: 4px;
    font-size: 1.2rem;
    text-align: center;
    font-weight: bold;
}

.join-btn {
    background-color: #333;
    color: white;
    padding: 15px;
    border: none;
    border-radius: 4px;
    font-size: 1.2rem;
    font-weight: bold;
    cursor: pointer;
}
.join-btn:disabled { opacity: 0.5; }

/* LOBBY */
.lobby-container {
    justify-content: center;
    text-align: center;
}

.nickname-display {
    font-size: 2.5rem;
    font-weight: 900;
    margin: 20px 0;
}

.loader {
    width: 40px;
    height: 40px;
    border: 4px solid rgba(255,255,255,0.3);
    border-top: 4px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 20px auto;
}

@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

.loader {
    width: 60px;
    height: 60px;
    border: 6px solid rgba(255,255,255,0.3);
    border-top: 6px solid white;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 30px auto;
}

/* GET READY */
.get-ready-container {
    justify-content: center;
    background-color: #FF3B30; /* Red alert for ready */
}
.ready-content h1 { font-size: 2rem; margin-bottom: 20px; }
.countdown { font-size: 5rem; font-weight: 900; }

/* QUESTION */
.question-container {
    justify-content: flex-end; /* Buttons at bottom usually */
    padding-bottom: env(safe-area-inset-bottom, 10px);
    padding-bottom: max(env(safe-area-inset-bottom, 0px), 10px);
}

.question-info {
    position: absolute;
    top: 20px;
    left: 20px;
    background: white;
    color: black;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
    z-index: 2;
}

.answer-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    width: 100%;
    padding: 8px;
    box-sizing: border-box;
    /* Let the grid size to content, not a fixed percentage */
    flex: 1;
    min-height: 0;
}

.answer-btn {
    border: none;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 3rem; /* Big icon */
    cursor: pointer;
    transition: transform 0.1s;
    box-shadow: 0 4px 0 rgba(0,0,0,0.2);
    min-height: 0; /* allow shrinking */
}
.answer-btn:active { transform: translateY(4px); box-shadow: none; }

.color-0 { background-color: #FF3B30; }
.color-1 { background-color: #0A84FF; }
.color-2 { background-color: #FFD60A; }
.color-3 { background-color: #32D74B; }


.true-false-grid {
    grid-template-columns: 1fr;
    flex: 1;
}

.color-true { background-color: #0A84FF; } /* Blue for True */
.color-false { background-color: #FF3B30; } /* Red for False */

.tf-label {
    margin-left: 10px;
    font-size: 2rem;
    font-weight: 900;
}

/* ANSWERED */
.answered-container {
    justify-content: center;
    text-align: center;
    background-color: #0A84FF;
}

/* RESULT */
.result-container {
    justify-content: center;
    align-items: center;
    text-align: center;
    padding: 20px;
    gap: 0;
}
.bg-correct { background: linear-gradient(160deg, #1a5c2e 0%, #145c2e 50%, #0e3d1e 100%); }
.bg-wrong { background: linear-gradient(160deg, #5c1a1a 0%, #4a0f0f 50%, #2d0808 100%); }

.result-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    flex: 1;
    justify-content: center;
    animation: resultReveal 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

@keyframes resultReveal {
    from { opacity: 0; transform: scale(0.7) translateY(30px); }
    to   { opacity: 1; transform: scale(1) translateY(0); }
}

.result-icon {
    font-size: 5rem;
    line-height: 1;
    filter: drop-shadow(0 4px 16px rgba(0,0,0,0.4));
    animation: iconBounce 0.6s 0.2s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}
.icon-correct { animation: iconBounce 0.6s 0.2s cubic-bezier(0.34, 1.56, 0.64, 1) both; }
.icon-wrong   { animation: iconShake 0.5s 0.2s ease both; }

@keyframes iconBounce {
    from { transform: scale(0) rotate(-15deg); }
    to   { transform: scale(1) rotate(0deg); }
}
@keyframes iconShake {
    0%,100% { transform: translateX(0); }
    20% { transform: translateX(-12px); }
    40% { transform: translateX(12px); }
    60% { transform: translateX(-8px); }
    80% { transform: translateX(8px); }
}

.result-title {
    font-size: 2.2rem;
    font-weight: 900;
    letter-spacing: -0.5px;
    text-shadow: 0 2px 8px rgba(0,0,0,0.3);
    animation: fadeSlideUp 0.4s 0.3s ease both;
}

@keyframes fadeSlideUp {
    from { opacity: 0; transform: translateY(20px); }
    to   { opacity: 1; transform: translateY(0); }
}

.score-summary {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    animation: fadeSlideUp 0.4s 0.45s ease both;
}

.score-delta {
    font-size: 3.5rem;
    font-weight: 900;
    color: #FFDF00;
    text-shadow: 0 2px 16px rgba(255,220,0,0.4);
    letter-spacing: -1px;
    line-height: 1;
}
.score-delta .plus-sign {
    font-size: 2.5rem;
    margin-right: 2px;
}
.wrong-delta {
    color: rgba(255,255,255,0.3);
    font-size: 1.2rem;
    font-weight: 600;
}

.streak-badge {
    background: rgba(255,150,0,0.25);
    border: 1px solid rgba(255,150,0,0.5);
    color: #FFAA00;
    padding: 6px 16px;
    border-radius: 20px;
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: 0.5px;
}

.rank-chip {
    background: rgba(255,255,255,0.1);
    border: 1px solid rgba(255,255,255,0.15);
    padding: 8px 20px;
    border-radius: 20px;
    font-size: 1rem;
    font-weight: 600;
    animation: fadeSlideUp 0.4s 0.6s ease both;
}

.footer-score {
    padding: 12px 24px;
    background: rgba(0,0,0,0.3);
    border-radius: 20px;
    font-weight: 700;
    font-size: 1.1rem;
    margin-bottom: 20px;
    animation: fadeSlideUp 0.4s 0.75s ease both;
}

/* FINISHED */
.finished-container {
    justify-content: center;
    text-align: center;
}

.final-rank-badge {
    background: #FFD700; /* Gold */
    color: black;
    width: 150px;
    height: 150px;
    border-radius: 50%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    margin: 30px auto;
    box-shadow: 0 0 20px rgba(255, 215, 0, 0.5);
    animation: bounce 2s infinite;
}

@keyframes bounce {
    0%, 20%, 50%, 80%, 100% {transform: translateY(0);}
    40% {transform: translateY(-20px);}
    60% {transform: translateY(-10px);}
}

.rank-num { font-size: 4rem; font-weight: 900; line-height: 1; }
.rank-label { font-size: 1rem; font-weight: 600; text-transform: uppercase; }

.exit-btn {
    margin-top: 40px;
    background: rgba(255,255,255,0.2);
    border: 2px solid white;
    color: white;
    padding: 10px 30px;
    border-radius: 20px;
    cursor: pointer;
    font-weight: bold;
}

.answer-btn.selected {
    border: 8px solid white;
    transform: scale(0.98);
}

.answer-btn.dimmed {
    opacity: 0.5;
    transform: scale(0.95);
}

.checkmark-badge {
    position: absolute;
    top: 10px;
    right: 10px;
    background: white;
    color: #4cd964;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 900;
}

.action-bar {
    width: 100%;
    padding: 20px;
    display: flex;
    justify-content: center;
}

.submit-btn {
    background: #FFD700;
    color: black;
    font-weight: 900;
    font-size: 1.5rem;
    padding: 15px 40px;
    border-radius: 50px;
    border: none;
    box-shadow: 0 4px 0 #b39700;
    cursor: pointer;
    transition: transform 0.1s;
}

.submit-btn:active {
    transform: translateY(4px);
    box-shadow: none;
}
.submit-btn:disabled {
    background: #ccc;
    box-shadow: none;
    cursor: not-allowed;
}

@media (max-width: 600px) {
    .answer-grid {
        grid-template-columns: 1fr;
        gap: 6px;
        padding: 6px;
    }
    .answer-btn {
        font-size: 2.5rem;
    }
    .action-bar {
        padding: 10px;
    }
}

.slider-container {
  justify-content: center;
  padding: 20px;
  padding-top: 64px; /* extra clearance for slider — most collision-prone */
}

/* CORRECT PICTURE STYLES */
.correct-picture-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  justify-content: space-between;
  background: #1C1C1E;
}

.question-header-area {
  flex: 0 0 15%;
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
}

.question-number-badge {
  background: rgba(255,255,255,0.2);
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.8rem;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.question-text {
  font-size: 1.2rem;
  text-align: center;
  margin: 0;
  line-height: 1.3;
}

.timer-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 4px;
  background: rgba(255,255,255,0.1);
}

.timer-progress {
  height: 100%;
  background: #32D74B;
  transition: width 0.1s linear;
}

.carousel-zone {
  flex: 1;
  overflow: hidden;
  position: relative;
  display: flex;
  align-items: center;
}

.action-area {
  flex: 0 0 15%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  z-index: 2;
  background: linear-gradient(to top, rgba(0,0,0,0.5), transparent);
}

.submit-selection-btn {
  background: #3A3A3C; /* Inactive state */
  color: #8E8E93;
  border: none;
  padding: 16px 32px;
  border-radius: 50px;
  font-size: 1.1rem;
  font-weight: 700;
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  width: 100%;
  max-width: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.submit-selection-btn.active {
  background: #32D74B;
  color: white;
  box-shadow: 0 4px 20px rgba(50, 215, 75, 0.4);
  transform: translateY(-2px);
}

.submit-selection-btn:active {
  transform: translateY(1px);
}

.submit-selection-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.spinner-small {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.slider-question-content {
  width: 100%;
  max-width: 600px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  padding-bottom: 20px;
}

.slider-wrapper {
  width: 100%;
  height: 300px; /* Force height for vertical slider */
  display: flex;
  justify-content: center;
  margin: 20px 0;
}

.question-text {
  font-size: 1.5rem;
  margin-bottom: 40px;
}

.slider-wrapper {
  padding: 20px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 16px;
  width: 100%;
  display: flex;
  justify-content: center;
}

.instruction-text {
  margin-top: 20px;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.6);
}
</style>
