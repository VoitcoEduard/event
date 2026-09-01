<template>
  <div class="quiz-player-screen brutalist" :style="shellStyle">
    <div class="language-switcher-container">
      <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select b-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

    <!-- NICKNAME ENTRY (Pre-Lobby) -->
    <div v-if="state === States.LOBBY && !joined" class="state-container name-entry-container">
      <h1 class="logo b-mono">{{ $t('quiz.player.logo') }}</h1>
      <div class="card b-card b-card-accent">
        <div class="input-row">
          <input
            v-model="nickname"
            :placeholder="$t('quiz.player.nickname_placeholder')"
            class="nickname-input b-input"
            @keyup.enter="joinLobby"
          />
          <button v-if="nickname.trim()" class="confirm-btn b-btn b-btn-accent" @click="joinLobby">
            <CornerDownLeft :size="20" :stroke-width="3" />
          </button>
        </div>
        <p class="enter-hint b-label">{{ $t('quiz.player.press_enter') }}</p>
      </div>
    </div>

    <!-- LOBBY STATE -->
    <div v-else-if="state === States.LOBBY && joined" class="state-container lobby-container">
      <div class="lobby-content b-card b-card-accent">
        <h2 class="waiting-text">{{ $t('quiz.player.you_are_in') }}</h2>
        <div class="nickname-badge b-badge b-badge-cyan b-mono">{{ nickname }}</div>
        <p class="status-msg">{{ $t('quiz.player.see_name') }}</p>
        <div class="loader-blocks b-loader"><span></span><span></span><span></span><span></span><span></span></div>
      </div>
    </div>

    <!-- GET READY / PRESENTING STATE -->
    <div v-else-if="state === States.GET_READY || state === States.QUESTION_PRESENTING" class="state-container get-ready-container">
      <div class="ready-content">
        <h1 class="ready-title">{{ $t('quiz.player.get_ready') }}</h1>
        <div v-if="explainTimeRemaining > 0" class="explain-timer b-mono">{{ explainTimeRemaining }}</div>
        <div v-else class="loader-blocks b-loader"><span></span><span></span><span></span><span></span><span></span></div>
      </div>
    </div>

    <!-- CORRECT PICTURE STATE -->
    <div
      v-else-if="state === States.QUESTION_ACTIVE && isCorrectPicture"
      class="state-container correct-picture-container"
    >
      <div class="question-header-area">
        <div class="progress-row">
          <div class="q-number b-mono">{{ currentQuestionIndex + 1 }}</div>
          <div v-if="totalQuestions > 1" class="progress-blocks">
            <div
              v-for="i in totalQuestions"
              :key="i"
              class="b-progress-block"
              :class="{ done: i - 1 < currentQuestionIndex, current: i - 1 === currentQuestionIndex }"
            ></div>
          </div>
        </div>
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
            <span class="mini-blocks"><span></span><span></span><span></span></span>
            {{ $t('quiz.player.submitting') }}
          </span>
          <span v-else>{{ $t('quiz.player.select_item') }}</span>
        </button>
      </div>
    </div>

    <!-- WRONG PICTURES STATE -->
    <div
      v-else-if="state === States.QUESTION_ACTIVE && isWrongPictures"
      class="state-container wrong-pictures-container"
    >
      <div class="question-header-area">
        <div class="progress-row">
          <div class="q-number b-mono">{{ currentQuestionIndex + 1 }}</div>
          <div v-if="totalQuestions > 1" class="progress-blocks">
            <div
              v-for="i in totalQuestions"
              :key="i"
              class="b-progress-block"
              :class="{ done: i - 1 < currentQuestionIndex, current: i - 1 === currentQuestionIndex }"
            ></div>
          </div>
        </div>
        <h2 class="question-text">{{ currentQuestion?.text }}</h2>
        <div class="timer-bar">
          <div class="timer-progress" :style="{ width: (timeRemaining / (currentQuestion?.timeLimit || 20)) * 100 + '%' }"></div>
        </div>
      </div>

      <div class="wp-interactive-zone-modern" ref="wpOuterContainer">
        <MapCoordinatePicker
          v-if="currentQuestion?.imageId"
          :image-url="getImageUrl(currentQuestion.imageId)"
          :max-targets="currentQuestion?.wrongPictureConfig?.showTargetCount
            ? (currentQuestion?.wrongPictureConfig?.targets?.length || 1)
            : (currentQuestion?.wrongPictureConfig?.targets?.length || 1) + 3"
          :show-target-count="currentQuestion?.wrongPictureConfig?.showTargetCount"
          :initial-markers="wpGuesses"
          @update:markers="wpGuesses = $event"
        />
      </div>

      <div class="action-area">
        <button
          v-if="!hasSubmitted"
          class="submit-selection-btn"
          :class="{ 'active': wpGuesses.some(g => g.isPlaced) }"
          @click="submitWrongPictures"
        >
          {{ $t('quiz.player.submit_btn') }}
        </button>
        <button v-else class="submit-selection-btn active">
          {{ isCorrect ? Math.round(currentPoints) + ' ' + $t('quiz.player.points') : '0 ' + $t('quiz.player.points') }}
        </button>
      </div>
    </div>

    <!-- SLIDER QUESTION STATE -->
    <div
      v-else-if="state === States.QUESTION_ACTIVE && isRange"
      class="state-container range-container"
    >
      <div class="progress-row">
        <div class="q-number b-mono">{{ currentQuestionIndex + 1 }}</div>
        <div v-if="totalQuestions > 1" class="progress-blocks">
          <div
            v-for="i in totalQuestions"
            :key="i"
            class="b-progress-block"
            :class="{ done: i - 1 < currentQuestionIndex, current: i - 1 === currentQuestionIndex }"
          ></div>
        </div>
      </div>

      <div class="range-question-content">
        <div class="range-holder">
          <RangeComponent
            :min="currentQuestion?.rangeConfig?.minValue"
            :max="currentQuestion?.rangeConfig?.maxValue"
            :step="currentQuestion?.rangeConfig?.step"
            :unit="currentQuestion?.rangeConfig?.unit"
            :decimalPlaces="currentQuestion?.rangeConfig?.decimalPlaces"
            v-model="rangeValue"
          />
        </div>

        <button
          class="range-submit-btn b-btn"
          :disabled="hasSubmitted"
          @click="submitRangeAnswer"
        >
          <Check v-if="hasSubmitted" :size="20" :stroke-width="3" />
          <span v-else>{{ $t('quiz.player.submit_btn') }}</span>
        </button>
      </div>
    </div>

    <!-- NUMBER QUESTION STATE -->
    <div
      v-else-if="state === States.QUESTION_ACTIVE && isNumber"
      class="state-container number-container"
    >
      <div class="progress-row">
        <div class="q-number b-mono">{{ currentQuestionIndex + 1 }}</div>
        <div v-if="totalQuestions > 1" class="progress-blocks">
          <div
            v-for="i in totalQuestions"
            :key="i"
            class="b-progress-block"
            :class="{ done: i - 1 < currentQuestionIndex, current: i - 1 === currentQuestionIndex }"
          ></div>
        </div>
      </div>

      <div class="number-question-content">
        <div class="number-input-wrapper">
          <input
            type="number"
            inputmode="numeric"
            pattern="[0-9]*"
            v-model.number="numberValue"
            class="number-answer-input b-input"
            :placeholder="$t('quiz.player.enter_number')"
            :disabled="hasSubmitted"
            @keyup.enter="submitNumberAnswer"
          />
        </div>

        <button
          class="number-submit-btn b-btn"
          :class="{ 'active': numberValue !== null && numberValue !== '' }"
          :disabled="hasSubmitted || numberValue === null || numberValue === ''"
          @click="submitNumberAnswer"
        >
          <Check v-if="hasSubmitted" :size="20" :stroke-width="3" />
          <span v-else>{{ $t('quiz.player.submit_btn') }}</span>
        </button>
      </div>
    </div>

    <!-- QUESTION/ANSWER STATE (Generic/Multiple Choice) -->
    <div v-else-if="state === States.QUESTION_ACTIVE" class="state-container question-container">
      <div class="progress-row">
        <div class="q-number b-mono">{{ currentQuestionIndex + 1 }}</div>
        <div v-if="totalQuestions > 1" class="progress-blocks">
          <div
            v-for="i in totalQuestions"
            :key="i"
            class="b-progress-block"
            :class="{ done: i - 1 < currentQuestionIndex, current: i - 1 === currentQuestionIndex }"
          ></div>
        </div>
      </div>

      <!-- Multi-select mode hint -->
      <div v-if="isMultiSelect" class="multi-hint b-badge b-badge-amber">
        <ListChecks :size="14" :stroke-width="2.5" />
        <span>{{ $t('quiz.player.select_multiple') }}</span>
      </div>

      <div class="answer-grid" :class="{ 'true-false-grid': isTrueFalse }">
         <button
            v-for="(shape, idx) in shapes"
            :key="idx"
            class="answer-btn"
            :class="[
                isTrueFalse ? (idx === 0 ? 'tf-true' : 'tf-false') : '',
                { 'selected': isSelected(idx), 'dimmed': hasSelected && !isSelected(idx) && !isMultiSelect, 'has-text': currentQuestion?.answers?.[idx] }
            ]"
            @click="handleAnswerClick(idx)"
            :disabled="hasSubmitted"
         >
            <span class="letter-badge b-mono">{{ letterLabels[idx] }}</span>
            <span class="answer-main">
               <Check v-if="isTrueFalse && idx === 0" class="tf-icon" :size="26" :stroke-width="3" />
               <X v-else-if="isTrueFalse" class="tf-icon" :size="26" :stroke-width="3" />
               <span v-else-if="!currentQuestion?.answers?.[idx]" class="shape-glyph">{{ shape }}</span>
               <span v-if="isTrueFalse" class="tf-label b-mono">{{ idx === 0 ? $t('quiz.player.true') : $t('quiz.player.false') }}</span>
               <span v-else-if="currentQuestion?.answers?.[idx]" class="answer-text">{{ currentQuestion.answers[idx] }}</span>
            </span>
            <span class="opt-checkbox" :class="{ 'is-checked': isSelected(idx) }">
              <Check v-if="isSelected(idx)" :size="14" :stroke-width="3.5" />
            </span>
         </button>
      </div>

      <div v-if="isMultiSelect && !hasSubmitted" class="action-bar">
          <button class="submit-btn b-btn" @click="submitMultiSelect" :disabled="selectedIndices.length === 0">
              {{ $t('quiz.player.submit_btn') }}
          </button>
      </div>
    </div>

    <!-- ANSWER SUBMITTED (Waiting) -->
    <div v-else-if="state === States.ANSWERED" class="state-container answered-container">
       <div class="answered-content b-card">
          <div class="loader-blocks b-loader"><span></span><span></span><span></span><span></span><span></span></div>
          <h2 class="answered-title">{{ $t('quiz.player.submitted') }}</h2>
          <p class="answered-sub">{{ $t('quiz.player.waiting_others') }}</p>
       </div>
    </div>

    <!-- RESULT STATE -->
    <div v-else-if="state === States.RESULTS" class="state-container result-container" :class="isCorrect ? 'result-correct' : 'result-wrong'">
       <div class="result-content b-card">
          <div class="result-icon-box">
            <Check v-if="isCorrect" :size="40" :stroke-width="3.5" />
            <X v-else :size="40" :stroke-width="3.5" />
          </div>
          <span class="verdict-badge b-badge" :class="isCorrect ? 'b-badge-lime' : 'b-badge-rose'">
            {{ isCorrect ? $t('quiz.player.correct') : $t('quiz.player.incorrect') }}
          </span>

          <div class="score-summary">
             <div class="score-delta b-mono" v-if="isCorrect">
               <span class="plus-sign">+</span>{{ currentPoints.toLocaleString() }}
             </div>
             <div class="score-delta wrong-delta b-label" v-else>{{ $t('quiz.player.incorrect') }}</div>
             <div class="streak-badge b-badge b-badge-amber" v-if="streak > 2">
               <Flame :size="14" :stroke-width="2.5" />
               {{ $t('quiz.player.streak', { streak: streak }) }}
             </div>
          </div>

       </div>
       <div class="footer-score b-badge b-mono">
          {{ $t('quiz.player.total_score', { score: score.toLocaleString() }) }}
       </div>
    </div>

    <!-- LEADERBOARD STATE -->
    <div v-else-if="state === States.LEADERBOARD" class="state-container player-leaderboard-container">
       <div class="player-leaderboard-content b-card b-shadow-lime">
          <div class="leaderboard-icon-box b-icon-box">
            <Trophy :size="32" :stroke-width="2.5" />
          </div>
          <h2 class="leaderboard-title b-label">{{ $t('quiz.player.leaderboard') }}</h2>
          <div class="rank-square b-mono">
             <span class="rank-num">{{ rank }}</span>
          </div>
          <span class="rank-label b-label">{{ $t('quiz.player.place') }}</span>
          <div class="score-display b-badge b-badge-lime b-mono">
             {{ score.toLocaleString() }} {{ $t('quiz.player.points') }}
          </div>
       </div>
    </div>

    <!-- FINISHED STATE -->
    <div v-else-if="state === States.FINISHED" class="state-container finished-container">
       <div class="finished-content b-card b-shadow-lime">
          <div class="done-icon-box b-icon-box">
            <Check :size="32" :stroke-width="3" />
          </div>
          <h1 class="finished-title">{{ $t('quiz.player.quiz_over') }}</h1>
          <div class="final-rank-square b-mono">
             <span class="rank-num">{{ rank }}</span>
          </div>
          <span class="rank-label b-label">{{ $t('quiz.player.place') }}</span>
          <p class="final-score b-badge b-badge-lime b-mono">{{ $t('quiz.player.final_score', { score: score }) }}</p>
       </div>
       <div class="finished-actions">
          <button class="restart-btn b-btn" @click="restartQuiz">
            <RotateCcw :size="18" :stroke-width="2.5" />
            {{ $t('quiz.player.play_again') }}
          </button>
          <button class="exit-btn b-btn" @click="goHome">
            <LogOut :size="18" :stroke-width="2.5" />
            {{ $t('quiz.player.exit_btn') }}
          </button>
       </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { quizWS } from '../services/websocket'
import { useQuizStateMachine } from '../composables/useQuizStateMachine'
import RangeComponent from '../components/RangeComponent.vue'
import HorizontalCarousel from '../components/HorizontalCarousel.vue'
import MapCoordinatePicker from '../components/MapCoordinatePicker.vue'
import { Check, X, CornerDownLeft, ListChecks, Flame, Trophy, RotateCcw, LogOut } from 'lucide-vue-next'
import { useVisualViewportHeight } from '../composables/useVisualViewportHeight'
import { questionCache } from '../utils/questionCache'
import { signalQueue } from '../utils/signalQueue'
import { getUserName } from '../utils'
import { getImageUrl } from '../services/media'
import { calculateTotalScore } from '../utils/wrongPicturesScoring'

const { locale } = useI18n()
const router = useRouter()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

// Visual viewport tracking — keeps submit/confirm buttons above the virtual keyboard
const { viewportHeight, isMobileViewport } = useVisualViewportHeight()

// State machine
const requestedBatchIndexes = new Set()
const questionCacheVersion = ref(0)

const requestBatch = (batchIndex) => {
    if (!Number.isInteger(batchIndex) || batchIndex < 0 || requestedBatchIndexes.has(batchIndex)) return

    requestedBatchIndexes.add(batchIndex)
    quizWS.send('player:batch_request', { currentBatchIndex: batchIndex })
}

const { state, transition, canTransition, currentQuestionIndex, score, streak, States } = useQuizStateMachine({ requestBatch })

// Local state
const nickname = ref(getUserName())
const joined = ref(false)
const selectedAnswer = ref(null) // For single choice
const selectedIndices = ref([])  // For multi select
const rangeValue = ref(50)      // For slider
const hasSubmitted = ref(false)
const isCorrect = ref(false)
const currentPoints = ref(0)
const rank = ref(0)
const timeRemaining = ref(5)
const explainTimeRemaining = ref(0)
const numberValue = ref(null)
const totalQuestions = ref(0)

// Correct Picture specific state
const preselectedIndex = ref(0)
const isSubmitting = ref(false)
const hasPreselection = ref(true) // Default to first item selected effectively

// Wrong Pictures specific state
const wpGuesses = ref([])
const activeMarkerIndex = ref(null)
const wpOuterContainer = ref(null)
const wpImageContainer = ref(null)
const wpContainerRect = ref(null)

const updateWpContainerRect = () => {
    // Legacy - kept for safety if needed by other components, but MapCoordinatePicker handles its own rect
}

const updateGuessPosition = (index, pos) => {
    // Legacy - MapCoordinatePicker emits whole array
}

const submitWrongPictures = () => {
    if (hasSubmitted.value) return
    const placedGuesses = wpGuesses.value.filter(g => g.isPlaced)
    submitAnswer(placedGuesses)
}

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
    // QuestionCache is a plain class, so make cache mutations an explicit
    // dependency for Vue's computed value.
    questionCacheVersion.value
    return questionCache.get(currentQuestionIndex.value)
})

const isTrueFalse = computed(() => {
    return currentQuestion.value?.type === 'true_false'
})

const isMultiSelect = computed(() => {
    return currentQuestion.value?.type === 'multi_select'
})

const isRange = computed(() => {
    return currentQuestion.value?.type === 'range'
})

const isMobile = computed(() => {
    return window.innerWidth < 600
})

const isCorrectPicture = computed(() => {
    return currentQuestion.value?.type === 'correct_picture'
})

const isWrongPictures = computed(() => {
    return currentQuestion.value?.type === 'wrong_pictures'
})

const isNumber = computed(() => {
    return currentQuestion.value?.type === 'number'
})

const imageOptionUrls = computed(() => {
    if (!currentQuestion.value?.imageOptions) return []
    return currentQuestion.value.imageOptions.map(opt => ({
        url: getImageUrl(opt.imageId),
        alt: opt.label,
        id: opt.id
    }))
})

// Preload images for correct_picture questions during presenting phase
const preloadImages = (question) => {
    if (question?.type !== 'correct_picture') return
    const imageUrls = question.imageOptions?.map(opt => getImageUrl(opt.imageId)) || []
    console.log('[DEBUG] Preloading images for correct_picture question:', imageUrls.length, 'images')
    imageUrls.forEach(url => {
        if (url) {
            const img = new Image()
            img.src = url  // Browser will cache this
        }
    })
}

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

const letterLabels = ['А', 'Б', 'В', 'Г']

// Per-context accent color: default=fuchsia, correct-picture=amber,
// wrong-pictures=rose, range/number=indigo, lobby/name=cyan, results/finish=lime
const accentColor = computed(() => {
    if (state.value === States.LOBBY) return 'var(--b-accent-cyan)'
    if (state.value === States.QUESTION_ACTIVE) {
        if (isCorrectPicture.value) return 'var(--b-accent-amber)'
        if (isWrongPictures.value) return 'var(--b-accent-rose)'
        if (isRange.value || isNumber.value) return 'var(--b-accent-indigo)'
        return 'var(--b-accent-fuchsia)'
    }
    if (state.value === States.RESULTS || state.value === States.LEADERBOARD || state.value === States.FINISHED) return 'var(--b-accent-lime)'
    return 'var(--b-accent-fuchsia)'
})

const shellStyle = computed(() => {
    const style = { '--q-accent': accentColor.value }
    if (isMobileViewport.value && viewportHeight.value) {
        style.height = viewportHeight.value
    }
    return style
})

watch(currentQuestion, (question) => {
    if (!question) return

    if (question.type === 'range' && selectedAnswer.value === null) {
        rangeValue.value = question.rangeConfig?.minValue ?? 0
    }
    preloadImages(question)
})

// Actions
const joinLobby = () => {
    if (!nickname.value.trim()) return
    quizWS.connect(moduleId, nickname.value, false)
    joined.value = true
}

const restartQuiz = () => {
    window.location.reload()
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

const submitRangeAnswer = () => {
  if (hasSubmitted.value) return
  selectedAnswer.value = rangeValue.value
  submitAnswer(rangeValue.value)
}

const submitNumberAnswer = () => {
  if (hasSubmitted.value || numberValue.value === null || numberValue.value === '') return
  submitAnswer(numberValue.value)
}

const submitAnswer = (answerData) => {
    console.log('[DEBUG] submitAnswer called:', {
        answerData,
        currentQuestionIndex: currentQuestionIndex.value,
        hasSubmitted: hasSubmitted.value,
        cacheSize: questionCache.cache.size,
        cacheKeys: [...questionCache.cache.keys()]
    })
    hasSubmitted.value = true
    clearInterval(timerInterval) // Stop the timer
    transition(States.ANSWERED)

    // Validate and score answer locally
    validateAndScore(answerData)
}

const goHome = () => {
    quizWS.disconnect()
    router.push({ path: '/event', query: { id: eventId } })
}

let timerInterval
let explainTimerInterval

// Answer validation and scoring
const validateAndScore = (answerData) => {
    // Use questionCache directly instead of the ref from state machine
    console.log('[DEBUG] validateAndScore called:', {
        currentQuestionIndex: currentQuestionIndex.value,
        answerData,
        cacheSize: questionCache.cache.size,
        cacheKeys: [...questionCache.cache.keys()],
        hasQuestion: questionCache.has(currentQuestionIndex.value)
    })

    const question = questionCache.get(currentQuestionIndex.value)
    if (!question) {
        console.error('[DEBUG] Question NOT FOUND in cache at index:', currentQuestionIndex.value, 'Cache contents:', JSON.stringify([...questionCache.cache.entries()]))
        return
    }
    console.log('[DEBUG] Question found:', question.text?.substring(0, 50), 'type:', question.type)

    let isCorrectAnswer = false
    let answerPayload = {}

    if (question.type === 'wrong_pictures') {
        const guessesData = Array.isArray(answerData) ? answerData : []
        const config = question.wrongPictureConfig || { targets: [], tolerancePercent: 10, maxPoints: 1000 }

        const result = calculateTotalScore(guessesData, config.targets, config)

        // Update local guesses with feedback
        wpGuesses.value = result.scoredGuesses

        answerPayload = { guesses: result.scoredGuesses }
        currentPoints.value = result.totalScore
        isCorrectAnswer = result.totalScore > 0
        score.value += currentPoints.value

        if (isCorrectAnswer) streak.value++
        else streak.value = 0

        isCorrect.value = isCorrectAnswer

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
        return
    } else if (question.type === 'multi_select') {
        // Strict equality check for arrays
        const userIndices = Array.isArray(answerData) ? answerData : []
        // Sort for comparison
        const sortedUser = [...userIndices].sort((a, b) => a - b)
        const sortedCorrect = [...(question.correctIndices || [])].sort((a, b) => a - b)

        isCorrectAnswer = JSON.stringify(sortedUser) === JSON.stringify(sortedCorrect)
        answerPayload = { answerIndices: userIndices }
    } else if (question.type === 'range') {
        const val = Number(answerData) || 0
        answerPayload = { answerValue: val }

        // Slider Scoring Logic
        const correctValue = Number(question.correctValue !== undefined ? question.correctValue : 0)
        const scoringConfig = question.scoringConfig || {}
        const maxPoints = Number(scoringConfig.maxPoints) || 100
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

    } else if (question.type === 'number') {
        const val = Number(answerData)
        answerPayload = { answerValue: val }

        // Binary scoring for number type
        const correctNumber = Number(question.correctNumber || 0)
        const maxPoints = Number(question.scoringConfig?.maxPoints || 1000)

        isCorrectAnswer = val === correctNumber
        currentPoints.value = isCorrectAnswer ? maxPoints : 0
        score.value += currentPoints.value

        if (isCorrectAnswer) streak.value++
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
    console.log('[DEBUG] Enqueuing signal:', JSON.stringify(signal))
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

const startExplainTimer = (timeLimit) => {
    clearInterval(explainTimerInterval)
    explainTimeRemaining.value = timeLimit
    if (timeLimit <= 0) return
    explainTimerInterval = setInterval(() => {
        explainTimeRemaining.value--
        if (explainTimeRemaining.value <= 0) {
            clearInterval(explainTimerInterval)
        }
    }, 1000)
}

// WS & Logic
onMounted(async () => {
    let hasOpenedOnce = false

    // Auto-join if nickname exists
    if (nickname.value) {
        joinLobby()
    }

    // Listen for WS updates
    quizWS.on('cmd:state', (payload) => {
        const newState = payload.state
        console.log('[DEBUG] cmd:state received:', newState, 'Payload:', JSON.stringify(payload))
        console.log('[DEBUG] Current state before transition:', state.value, 'currentIndex:', currentQuestionIndex.value)

        if (newState === States.GET_READY || newState === States.QUESTION_PRESENTING) {
            // "Get Ready" / "Presenting" phase - Server drives transition now

            // Update question index if provided
            if (payload.currentIndex !== undefined) {
                console.log('[DEBUG] Setting currentQuestionIndex to:', payload.currentIndex)
                currentQuestionIndex.value = payload.currentIndex
            }


            // Reset for new question
            selectedAnswer.value = null
            selectedIndices.value = []
            const cachedQuestion = questionCache.get(currentQuestionIndex.value)
            if (cachedQuestion?.type === 'range') {
                rangeValue.value = cachedQuestion.rangeConfig?.minValue ?? 0
            }
            hasSubmitted.value = false
            isCorrect.value = false
            currentPoints.value = 0

            // Correct Picture reset
            preselectedIndex.value = 0
            isSubmitting.value = false
            hasPreselection.value = true

            // Wrong Pictures reset
            wpGuesses.value = []
            activeMarkerIndex.value = null

            const question = questionCache.get(currentQuestionIndex.value)

            if (question && question.type === 'wrong_pictures' && question.wrongPictureConfig) {
                 // Start with empty guesses for new map-style picker
                 wpGuesses.value = []
                 setTimeout(updateWpContainerRect, 500)
            }
            if (question && question.type === 'number') {
                numberValue.value = null
            }
            if (question) {
                preloadImages(question)
            }
        }

        if (canTransition(newState)) {
            transition(newState, payload)

            if (newState === States.QUESTION_ACTIVE) {
                 startLocalTimer(payload?.timeLimit || 20)
            } else if (newState === States.QUESTION_PRESENTING) {
                 startExplainTimer(payload?.timeLimit || 5)
            }
        } else {
            console.warn('Invalid state transition attempted:', state.value, '->', newState)
        }
    })

    quizWS.on('data:batch', (batch) => {
        console.log('[DEBUG] data:batch received:', JSON.stringify({
            startIndex: batch.startIndex,
            endIndex: batch.endIndex,
            questionCount: batch.questions?.length,
            firstQuestion: batch.questions?.[0]?.text?.substring(0, 50)
        }))
        // Clear cache at the start of a new quiz (startIndex === 0 = first batch)
        if (batch.startIndex === 0) {
            console.log('[DEBUG] New quiz batch (startIndex=0) - clearing question cache before loading')
            questionCache.clear()
            requestedBatchIndexes.clear()
            totalQuestions.value = 0
        }
        questionCache.addBatch(batch)
        requestedBatchIndexes.add(batch.batchIndex)
        questionCacheVersion.value++
        totalQuestions.value = Math.max(totalQuestions.value, batch.startIndex + (batch.questions?.length || 0))
        console.log('[DEBUG] After addBatch, cache size:', questionCache.cache.size, 'keys:', [...questionCache.cache.keys()])
    })

    quizWS.on('sync:minimal', (payload) => {
        // Handle minimal sync updates
        console.log('Minimal sync:', payload)
    })

    quizWS.on('sync:catchup', (payload) => {
        // Handle reconnection catchup
        console.log('Catchup sync:', payload)
        if (Number.isInteger(payload.currentQuestionIndex)) {
            currentQuestionIndex.value = payload.currentQuestionIndex
        }
        if (typeof payload.score === 'number') {
            score.value = payload.score
        }
        const missedQuestions = payload.missedQuestions || []
        if (missedQuestions.length > 0) {
            console.warn(`You missed ${missedQuestions.length} questions`)
        }
        const batchesNeeded = payload.batchesNeeded || []
        batchesNeeded.forEach(requestBatch)
    })

    quizWS.on('open', () => {
        if (hasOpenedOnce) {
            quizWS.send('player:reconnect', {
                moduleId,
                nickname: nickname.value,
                lastKnownQuestionIndex: currentQuestionIndex.value,
                lastKnownState: state.value,
            })
        }
        hasOpenedOnce = true
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
    clearInterval(explainTimerInterval)
    quizWS.disconnect()
    quizWS.clearHandlers()
})
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;700&display=swap');

.quiz-player-screen {
    width: 100vw;
    height: 100vh;
    height: 100dvh;
    background-color: var(--b-bg);
    color: var(--b-text);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    --b-progress-accent: var(--q-accent, var(--b-accent-fuchsia));
    --b-loader-accent: var(--q-accent, var(--b-accent-cyan));
}

.language-switcher-container {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 100;
}

.state-container {
    z-index: 1;
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    overflow: hidden;
    min-height: 0;
    padding-top: 56px; /* clear the language switcher */
}

/* ── Shared question header pieces ─────────────────────────── */
.progress-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 0 16px;
    flex-shrink: 0;
}

.q-number {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--b-bg);
    border: var(--b-border-w) solid var(--q-accent, var(--b-accent-fuchsia));
    color: var(--q-accent, var(--b-accent-fuchsia));
    font-size: 14px;
    font-weight: 700;
    flex-shrink: 0;
}

.progress-blocks {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    flex: 1;
    min-width: 0;
}

/* ── NAME ENTRY ────────────────────────────────────────────── */
.name-entry-container {
    justify-content: center;
    padding: 20px;
}

.logo {
    font-size: 2.6rem;
    font-weight: 700;
    margin-bottom: 40px;
    text-transform: uppercase;
    letter-spacing: -1px;
    color: var(--b-text);
    text-shadow: 3px 3px 0 var(--b-accent-cyan);
}

.card {
    width: 100%;
    max-width: 420px;
    display: flex;
    flex-direction: column;
    gap: 14px;
}

.input-row {
    display: flex;
    gap: 10px;
    align-items: stretch;
}

.nickname-input {
    flex: 1;
    min-width: 0;
}

.confirm-btn {
    padding: 0 18px;
    flex-shrink: 0;
}

.enter-hint {
    margin: 0;
    text-align: center;
}

/* ── LOBBY ─────────────────────────────────────────────────── */
.lobby-container {
    justify-content: center;
    padding: 20px;
}

.lobby-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    text-align: center;
    width: 100%;
    max-width: 400px;
}

.waiting-text {
    font-size: 1.4rem;
    font-weight: 900;
    text-transform: uppercase;
    letter-spacing: -0.3px;
    margin: 0;
}

.nickname-badge {
    font-size: 18px;
    padding: 10px 18px;
}

.status-msg {
    color: var(--b-text-dim);
    font-size: 13px;
    margin: 0;
}

.loader-blocks {
    margin-top: 8px;
}

/* ── GET READY ─────────────────────────────────────────────── */
.get-ready-container {
    justify-content: center;
}

.ready-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 24px;
}

.ready-title {
    font-size: 2.2rem;
    font-weight: 900;
    text-transform: uppercase;
    letter-spacing: -1px;
    margin: 0;
    text-align: center;
    text-shadow: 3px 3px 0 var(--q-accent, var(--b-accent-fuchsia));
}

.explain-timer {
    font-size: 4.5rem;
    font-weight: 700;
    color: var(--q-accent, var(--b-accent-fuchsia));
    border: var(--b-border-w) solid var(--q-accent, var(--b-accent-fuchsia));
    padding: 8px 28px;
    line-height: 1;
}

/* ── QUESTION (generic / multi-choice) ─────────────────────── */
.question-container {
    padding-bottom: max(env(safe-area-inset-bottom, 0px), 10px);
    gap: 12px;
}

.multi-hint {
    flex-shrink: 0;
}

.answer-grid {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    padding: 0 12px 12px;
    flex: 1;
    min-height: 0;
    overflow-y: auto;
}

.answer-btn {
    display: flex;
    align-items: center;
    gap: 14px;
    width: 100%;
    text-align: left;
    background: var(--b-surface);
    border: var(--b-border-w) solid var(--b-border);
    padding: 14px 16px;
    cursor: pointer;
    font-family: inherit;
    color: var(--b-text);
    transition: transform 0.1s ease, box-shadow 0.1s ease, border-color 0.1s ease, background 0.1s ease;
    flex-shrink: 0;
}

.answer-btn:active:not(:disabled) {
    transform: translate(2px, 2px);
}

.answer-btn:disabled {
    cursor: default;
}

.answer-btn.selected {
    border-color: var(--b-accent-fuchsia);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-fuchsia);
    background: rgba(232, 121, 249, 0.1);
}

.answer-btn.dimmed {
    opacity: 0.4;
}

.answer-btn.tf-true.selected {
    border-color: var(--b-accent-cyan);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-cyan);
    background: rgba(34, 211, 238, 0.1);
}

.answer-btn.tf-false.selected {
    border-color: var(--b-accent-rose);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-rose);
    background: rgba(251, 113, 133, 0.1);
}

.letter-badge {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: var(--b-border-w) solid var(--b-border);
    background: var(--b-bg);
    font-size: 14px;
    font-weight: 700;
    color: var(--b-text-dim);
    flex-shrink: 0;
}

.answer-btn.selected .letter-badge {
    border-color: var(--b-accent-fuchsia);
    color: var(--b-accent-fuchsia);
}

.answer-btn.tf-true.selected .letter-badge { border-color: var(--b-accent-cyan); color: var(--b-accent-cyan); }
.answer-btn.tf-false.selected .letter-badge { border-color: var(--b-accent-rose); color: var(--b-accent-rose); }

.answer-main {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;
}

.tf-icon {
    color: var(--b-accent-cyan);
    flex-shrink: 0;
}

.tf-false .tf-icon {
    color: var(--b-accent-rose);
}

.tf-label {
    font-size: 1.3rem;
    font-weight: 700;
    text-transform: uppercase;
}

.shape-glyph {
    font-size: 1.8rem;
    line-height: 1;
}

.answer-text {
    font-size: 1rem;
    font-weight: 700;
    line-height: 1.3;
    word-break: break-word;
}

.opt-checkbox {
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: var(--b-border-w) solid var(--b-border);
    background: var(--b-bg);
    color: #000;
    flex-shrink: 0;
    transition: background 0.1s ease, border-color 0.1s ease;
}

.opt-checkbox.is-checked {
    background: var(--b-accent-fuchsia);
    border-color: var(--b-accent-fuchsia);
}

.answer-btn.tf-true .opt-checkbox.is-checked { background: var(--b-accent-cyan); border-color: var(--b-accent-cyan); }
.answer-btn.tf-false .opt-checkbox.is-checked { background: var(--b-accent-rose); border-color: var(--b-accent-rose); }

.true-false-grid {
    justify-content: center;
}

.action-bar {
    width: 100%;
    padding: 8px 16px 16px;
    display: flex;
    justify-content: center;
    flex-shrink: 0;
}

.submit-btn {
    background: var(--b-accent-fuchsia);
    color: #000;
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-fuchsia);
    min-width: 200px;
}

.submit-btn:hover:not(:disabled) { box-shadow: 6px 6px 0 0 var(--b-accent-fuchsia); }
.submit-btn:active:not(:disabled) { box-shadow: 1px 1px 0 0 var(--b-accent-fuchsia); }

/* ── ANSWERED ──────────────────────────────────────────────── */
.answered-container {
    justify-content: center;
    padding: 20px;
}

.answered-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    text-align: center;
    width: 100%;
    max-width: 400px;
}

.answered-title {
    font-size: 1.3rem;
    font-weight: 900;
    text-transform: uppercase;
    margin: 0;
}

.answered-sub {
    color: var(--b-text-dim);
    font-size: 13px;
    margin: 0;
}

/* ── RESULT ────────────────────────────────────────────────── */
.result-container {
    justify-content: center;
    align-items: center;
    padding: 20px;
    gap: 20px;
}

.result-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
    width: 100%;
    max-width: 400px;
    text-align: center;
    animation: b-reveal 0.25s ease-out both;
}

@keyframes b-reveal {
    from { opacity: 0; transform: translateY(16px); }
    to   { opacity: 1; transform: translateY(0); }
}

.result-container.result-correct .result-content {
    border-color: var(--b-accent-lime);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-lime);
}

.result-container.result-wrong .result-content {
    border-color: var(--b-accent-rose);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-rose);
}

.result-icon-box {
    width: 72px;
    height: 72px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--b-bg);
    border: var(--b-border-w) solid var(--b-text);
}

.result-container.result-correct .result-icon-box {
    color: var(--b-accent-lime);
    border-color: var(--b-accent-lime);
}

.result-container.result-wrong .result-icon-box {
    color: var(--b-accent-rose);
    border-color: var(--b-accent-rose);
}

.verdict-badge {
    font-size: 13px;
    padding: 8px 16px;
}

.score-summary {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
}

.score-delta {
    font-size: 2.8rem;
    font-weight: 700;
    color: var(--b-accent-amber);
    line-height: 1;
}

.score-delta .plus-sign {
    font-size: 2rem;
}

.wrong-delta {
    color: var(--b-text-dim);
    font-size: 12px;
}

.streak-badge {
    font-size: 12px;
}

.footer-score {
    font-size: 13px;
    padding: 10px 18px;
}

/* ── LEADERBOARD ───────────────────────────────────────────── */
.player-leaderboard-container {
    justify-content: center;
    padding: 20px;
}

.player-leaderboard-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    width: 100%;
    max-width: 400px;
    text-align: center;
    animation: b-reveal 0.25s ease-out both;
}

.leaderboard-icon-box {
    width: 56px;
    height: 56px;
    color: var(--b-accent-lime);
    border-color: var(--b-accent-lime);
}

.leaderboard-title {
    margin: 0;
    font-size: 12px;
}

.rank-square {
    width: 120px;
    height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--b-bg);
    border: var(--b-border-w) solid var(--b-text);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-lime);
}

.rank-num {
    font-size: 3.5rem;
    font-weight: 700;
    line-height: 1;
}

.rank-label {
    margin-top: -6px;
}

.score-display {
    font-size: 14px;
    padding: 10px 18px;
}

/* ── FINISHED ──────────────────────────────────────────────── */
.finished-container {
    justify-content: center;
    padding: 20px;
    gap: 24px;
}

.finished-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    width: 100%;
    max-width: 400px;
    text-align: center;
    animation: b-reveal 0.25s ease-out both;
}

.done-icon-box {
    width: 56px;
    height: 56px;
    color: var(--b-accent-lime);
    border-color: var(--b-accent-lime);
}

.finished-title {
    font-size: 1.6rem;
    font-weight: 900;
    text-transform: uppercase;
    letter-spacing: -0.5px;
    margin: 0;
}

.final-rank-square {
    width: 100px;
    height: 100px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--b-bg);
    border: var(--b-border-w) solid var(--b-text);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-lime);
}

.final-score {
    margin: 0;
    font-size: 13px;
}

.finished-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    justify-content: center;
}

.restart-btn {
    background: var(--b-accent-lime);
    color: #000;
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-lime);
}

.restart-btn:hover:not(:disabled) { box-shadow: 6px 6px 0 0 var(--b-accent-lime); }
.restart-btn:active:not(:disabled) { box-shadow: 1px 1px 0 0 var(--b-accent-lime); }

.exit-btn {
    color: var(--b-accent-rose);
    border-color: var(--b-accent-rose);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-rose);
}

.exit-btn:hover:not(:disabled) { box-shadow: 6px 6px 0 0 var(--b-accent-rose); }
.exit-btn:active:not(:disabled) { box-shadow: 1px 1px 0 0 var(--b-accent-rose); }

/* ── CORRECT PICTURE ───────────────────────────────────────── */
.correct-picture-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    justify-content: space-between;
    background: var(--b-bg);
}

.question-header-area {
    flex: 0 0 auto;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 100;
    background: var(--b-bg);
    border-bottom: var(--b-border-w) solid var(--b-border);
    padding: 56px 0 18px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
}

.question-text {
    font-size: 1rem;
    font-weight: 800;
    text-align: center;
    margin: 0;
    line-height: 1.3;
    padding: 0 16px;
    max-width: 560px;
}

.timer-bar {
    position: absolute;
    bottom: 0;
    left: 0;
    width: 100%;
    height: 6px;
    background: var(--b-border);
}

.timer-progress {
    height: 100%;
    background: var(--q-accent, var(--b-accent-fuchsia));
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
    flex: 0 0 auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 20px;
    z-index: 100;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    background: var(--b-bg);
    border-top: var(--b-border-w) solid var(--b-border);
}

.submit-selection-btn {
    background: var(--b-surface);
    color: var(--b-text-dim);
    border: var(--b-border-w) solid var(--b-border);
    padding: 14px 28px;
    font-size: 1rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    font-family: inherit;
    width: auto;
    min-width: 220px;
    max-width: 320px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    cursor: pointer;
    transition: box-shadow 0.1s ease, border-color 0.1s ease, background 0.1s ease, color 0.1s ease;
}

.submit-selection-btn.active {
    background: var(--q-accent, var(--b-accent-amber));
    color: #000;
    border-color: var(--b-text);
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--q-accent, var(--b-accent-amber));
}

.submit-selection-btn:active:not(:disabled) {
    transform: translate(2px, 2px);
    box-shadow: 1px 1px 0 0 var(--q-accent, var(--b-accent-amber));
}

.submit-selection-btn:disabled:not(.active) {
    opacity: 0.5;
    cursor: not-allowed;
}

.mini-blocks {
    display: inline-flex;
    gap: 3px;
}

.mini-blocks span {
    width: 6px;
    height: 6px;
    background: currentColor;
    animation: mini-pulse 1s ease-in-out infinite;
}

.mini-blocks span:nth-child(2) { animation-delay: 0.15s; }
.mini-blocks span:nth-child(3) { animation-delay: 0.3s; }

@keyframes mini-pulse {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.2; }
}

/* ── RANGE ─────────────────────────────────────────────────── */
.range-container {
    justify-content: flex-start;
    padding: 0 0 20px;
    gap: 12px;
}

.range-container .progress-row {
    padding-top: 8px;
}

.range-question-content {
    width: 100%;
    max-width: 480px;
    display: flex;
    flex-direction: column;
    align-items: center;
    flex: 1;
    min-height: 0;
    padding: 8px 16px 16px;
    gap: 12px;
}

.range-wrapper {
    width: 100%;
    flex: 1;
    min-height: 0;
    display: flex;
    justify-content: center;
}

/* Hard-edged overrides for RangeComponent internals (component file untouched) */
.range-container :deep(.track-bar) {
    width: 10px;
    background: var(--b-border);
    border-radius: 0;
    border: var(--b-border-w) solid var(--b-border);
}

.range-container :deep(.track-fill) {
    background: var(--q-accent, var(--b-accent-indigo));
    border-radius: 0;
}

.range-container :deep(.thumb) {
    border-radius: 0;
    background: var(--q-accent, var(--b-accent-indigo));
    border: var(--b-border-w) solid var(--b-text);
    box-shadow: 3px 3px 0 0 var(--b-text);
}

.range-container :deep(.thumb-lines span) {
    background: rgba(0, 0, 0, 0.55);
    border-radius: 0;
}

.range-container :deep(.value-num) {
    font-family: 'JetBrains Mono', ui-monospace, monospace;
    text-shadow: none;
}

.range-container :deep(.scroll-zone) {
    border-radius: 0;
    background: var(--b-surface);
    border: var(--b-border-w) solid var(--b-border);
}

.range-container :deep(.scroll-zone.hint) {
    border-color: var(--q-accent, var(--b-accent-indigo));
    animation: none;
}

.range-container :deep(.scroll-zone.active) {
    border-color: var(--q-accent, var(--b-accent-indigo));
    animation: none;
}

.range-container :deep(.arrow) {
    color: var(--q-accent, var(--b-accent-indigo));
    filter: none;
}

.range-container :deep(.notch),
.range-container :deep(.tick-mark) {
    border-radius: 0;
}

.range-container :deep(.correct-tag) {
    border-radius: 0;
}

.range-submit-btn {
    width: 100%;
    max-width: 320px;
    background: var(--b-accent-indigo);
    color: #000;
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-indigo);
    flex-shrink: 0;
}

.range-submit-btn:hover:not(:disabled) { box-shadow: 6px 6px 0 0 var(--b-accent-indigo); }
.range-submit-btn:active:not(:disabled) { box-shadow: 1px 1px 0 0 var(--b-accent-indigo); }
.range-submit-btn:disabled {
    opacity: 0.4;
}

/* ── WRONG PICTURES ────────────────────────────────────────── */
.wrong-pictures-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    height: 100dvh;
    width: 100%;
    justify-content: flex-start;
    background: var(--b-bg);
    overflow: hidden;
}

.wp-interactive-zone-modern {
    flex: 1;
    width: 100%;
    position: relative;
    overflow: hidden;
    background: var(--b-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
}

/* ── NUMBER ────────────────────────────────────────────────── */
.number-container {
    justify-content: center;
    padding: 20px;
    gap: 20px;
}

.number-question-content {
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    align-items: center;
}

.number-input-wrapper {
    width: 100%;
}

.number-answer-input {
    text-align: center;
    font-size: 2.4rem;
    padding: 20px;
}

.number-answer-input:focus {
    border-color: var(--q-accent, var(--b-accent-indigo));
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--q-accent, var(--b-accent-indigo));
}

.number-answer-input::-webkit-outer-spin-button,
.number-answer-input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
}

.number-submit-btn {
    width: 100%;
    background: var(--b-accent-indigo);
    color: #000;
    box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-indigo);
}

.number-submit-btn:hover:not(:disabled) { box-shadow: 6px 6px 0 0 var(--b-accent-indigo); }
.number-submit-btn:active:not(:disabled) { box-shadow: 1px 1px 0 0 var(--b-accent-indigo); }
.number-submit-btn:disabled {
    opacity: 0.4;
}
</style>
