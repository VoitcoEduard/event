<template>
  <div class="quiz-host-screen">
    <div class="bg-layer"></div>

    <div class="host-topbar">
      <div class="host-topbar-left">
        <a :href="backToEventUrl" class="host-back-link" @click="goBackToEvent">
          <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
          <span>{{ $t('host.back_to_event') }}</span>
        </a>
        <span v-if="hostContextLabel" class="host-topbar-context">{{ hostContextLabel }}</span>
      </div>
      <AdminThemeToggle variant="inline" />
    </div>
    
    <div v-if="gameState === 'lobby'" class="state-container lobby-container">
      <div class="lobby-header">
        <h1 class="game-title">{{ $t('quiz.host.join_title') }}</h1>
        <div class="pin-display">
          <span class="pin-label">{{ $t('quiz.host.game_pin') }}</span>
          <span class="pin-code">{{ pinCode }}</span>
        </div>
        <div class="qr-container">
          <QRCodeVue
            :value="qrCodeUrl"
            :size="150"
            level="H"
            render-as="svg"
            role="img"
            :aria-label="$t('host.qr_alt', { pin: pinCode })"
          />
        </div>
        <div class="lobby-actions">
            <button type="button" class="copy-link-btn" @click="copyProjectorLink">
                {{ $t('quiz.host.copy_link') }}
            </button>
        </div>
      </div>
      
      <div class="players-area">
        <div class="players-count">
          <span><AppIcon name="teams" :size="18" aria-hidden="true" /> {{ players.length }}</span> {{ $t('quiz.host.players', players.length) }}
        </div>
        <div class="players-grid">
          <div v-for="player in players" :key="player.id" class="player-chip">
            {{ player.nickname }}
          </div>
        </div>
      </div>

      <div class="lobby-footer">
        <button type="button" class="start-btn" @click="startQuiz" :disabled="players.length === 0">
          {{ $t('quiz.host.start_btn') }}
        </button>
        <p v-if="players.length === 0" class="lobby-start-hint">
          {{ $t('host.waiting_for_players') }}
        </p>
      </div>
    </div>

    <div v-else-if="gameState === 'get-ready'" class="state-container get-ready-container">
      <div class="loading-content">
        <h1>{{ $t('quiz.player.get_ready') }}</h1>
        <div class="loader"></div>
      </div>
    </div>

    <div v-else-if="gameState === 'question-active' || gameState === 'question-presenting'" class="state-container question-container">
      <div class="question-header">
        <div class="timer" :class="{ 'timer-low': timeRemaining <= 5 }" v-if="gameState === 'question-active'">
          {{ timeRemaining }}
        </div>
        <div class="timer" v-else-if="gameState === 'question-presenting'" :style="explainTimeRemaining > 0 ? '' : 'font-size: 1.5rem'">
          {{ explainTimeRemaining > 0 ? explainTimeRemaining : $t('quiz.player.get_ready') }}
        </div>
        <div class="question-header-right">
          <div class="answer-counter-widget">
            <div class="counter-value">{{ totalAnswered }}</div>
            <div class="counter-label">{{ $t('quiz.host.answers_label') }}</div>
          </div>
          <div class="question-counter">{{ currentQuestionIndex + 1 }} / {{ totalQuestions }}</div>
        </div>
      </div>

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
        <div class="question-column">
          <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
            
            <div v-if="questionImageUrl" class="question-image-container">
               <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || $t('quiz.host.question_image_alt')" class="question-image" decoding="async" loading="eager" />
               <div class="image-overlay-gradient"></div>
            </div>

            <div class="question-text-overlay">
               <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
            </div>

          </div>
        </div>

        <div class="answers-column">
          <div v-if="isRange" class="range-projector-view">
            <div class="sbs-label sbs-label-max">
              {{ formatWithUnit(currentQuestion?.rangeConfig?.maxValue, currentQuestion?.rangeConfig?.unit) }}
            </div>
            <div class="sbs-track">
              <div class="sbs-track-fill sbs-fill-anim"></div>
              <div class="sbs-thumb sbs-thumb-pulse">
                <div class="sbs-thumb-lines"><span></span><span></span><span></span></div>
              </div>
            </div>
            <div class="sbs-label sbs-label-min">
              {{ formatWithUnit(currentQuestion?.rangeConfig?.minValue, currentQuestion?.rangeConfig?.unit) }}
            </div>
            <div class="sbs-hint">↕ {{ $t('quiz.host.range_to_answer') }}</div>
          </div>
          <template v-else-if="isNumber">
            <div class="number-host-view">
              <div class="nh-input-icon"><AppIcon name="hash" :size="32" aria-hidden="true" /></div>
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
                <AppIcon class="answer-icon" :name="getShapeIcon(idx)" :size="28" aria-hidden="true" />
                <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ isTrueFalse ? (idx === 0 ? $t('quiz.player.true') : $t('quiz.player.false')) : answer }}</div>
            </div>
          </template>
        </div>
      </div>
      
      <div class="host-controls">
         <button v-if="timeRemaining > 0" type="button" class="control-btn" @click="endQuestionEarly">{{ $t('quiz.host.skip_timer') }}</button>
         <button v-else type="button" class="control-btn primary" @click="goToResults">{{ $t('quiz.host.go_to_results') }}</button>
      </div>
    </div>

    <div v-else-if="gameState === 'results'" class="state-container results-container-new">
       <div class="results-content-wrapper">
         <div class="question-header" style="margin-bottom: 1rem;">
             <div class="results-mode-title">{{ $t('quiz.host.results_title', { num: currentQuestionIndex + 1 }) }}</div>
         </div>

         <div v-if="isCorrectPicture" class="correct-picture-results-host">
           <div class="cp-host-correct-hero">
             <div v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
               <div v-if="isAnswerCorrect(idx)" class="cp-host-hero-frame">
                 <div class="cp-host-hero-glow"></div>
                 <img :src="getImageUrl(option.imageId)" class="cp-host-hero-img" decoding="async" loading="eager" />
                 <div class="cp-host-hero-check"><AppIcon name="check" :size="24" aria-hidden="true" /></div>
                 <div class="cp-host-hero-label">{{ currentQuestion.text }}</div>
               </div>
             </div>
           </div>
           <div class="cp-host-wrong-thumbs">
             <template v-for="(option, idx) in currentQuestion.imageOptions" :key="idx">
               <div v-if="!isAnswerCorrect(idx)" class="cp-host-thumb-card">
                 <img :src="getImageUrl(option.imageId)" class="cp-host-thumb-img" decoding="async" loading="eager" />
                 <div class="cp-host-thumb-count">{{ answerCounts[idx] || 0 }}</div>
               </div>
             </template>
           </div>
         </div>

          <div v-else-if="isWrongPictures" class="wrong-picture-results-magnified">
             <div class="wp-magnified-results-grid">
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

          <div v-else-if="isRange" class="range-results-new">
            <div class="question-column">
              <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
                <div v-if="questionImageUrl" class="question-image-container">
                  <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || $t('quiz.host.question_image_alt')" class="question-image" decoding="async" loading="eager" />
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
                  <AppIcon class="srt-check" name="check" :size="18" aria-hidden="true" />
                  <span class="srt-value">{{ formatWithUnit(currentQuestion?.correctValue, currentQuestion?.rangeConfig?.unit) }}</span>
                </div>
                <div class="srt-label srt-label-max">{{ formatWithUnit(currentQuestion?.rangeConfig?.maxValue, currentQuestion?.rangeConfig?.unit) }}</div>
                <div class="srt-track">
                  <div class="srt-fill" :style="{ height: revealCorrect ? correctMarkerPercentage + '%' : '0%' }"></div>
                  <div class="srt-marker" :class="{ 'srt-marker-reveal': revealCorrect }" :style="{ bottom: correctMarkerPercentage + '%' }">
                    <div class="srt-marker-line"></div>
                    <div class="srt-marker-bubble">{{ formatWithUnit(currentQuestion?.correctValue, currentQuestion?.rangeConfig?.unit) }}</div>
                  </div>
                </div>
                <div class="srt-label srt-label-min">{{ formatWithUnit(currentQuestion?.rangeConfig?.minValue, currentQuestion?.rangeConfig?.unit) }}</div>
              </div>
            </div>
          </div>

          <div v-else-if="isNumber" class="number-results-host">
            <div class="question-column">
              <div class="question-card" :class="{ 'has-image': !!questionImageUrl }">
                <div v-if="questionImageUrl" class="question-image-container">
                  <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || $t('quiz.host.question_image_alt')" class="question-image" decoding="async" loading="eager" />
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
                  <div class="nr-check"><AppIcon name="check" :size="20" aria-hidden="true" /></div>
               </div>
            </div>
          </div>

          <div v-else class="question-main-content">
            <div class="question-column">
              <div class="question-card has-distribution" :class="{ 'has-image': !!questionImageUrl }">
                <div v-if="questionImageUrl" class="question-image-container">
                  <img :src="questionImageUrl" :alt="currentQuestion?.imageAlt || $t('quiz.host.question_image_alt')" class="question-image dimmed-image" decoding="async" loading="eager" />
                  <div class="image-overlay-gradient"></div>
                </div>

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
                            <AppIcon class="dist-icon" :name="getShapeIcon(idx)" :size="20" aria-hidden="true" />
                        </div>
                    </div>
                </div>
                
                <div class="question-text-overlay">
                  <h2 class="question-text" :style="{ fontSize: questionFontSize }">{{ currentQuestion?.text }}</h2>
                </div>
              </div>
            </div>

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
                  <AppIcon class="answer-icon" :name="getShapeIcon(idx)" :size="28" aria-hidden="true" />
                  <div class="answer-text" :style="{ fontSize: getAnswerFontSize(answer) }">{{ isTrueFalse ? (idx === 0 ? $t('quiz.player.true') : $t('quiz.player.false')) : answer }}</div>
                  <AppIcon v-if="isAnswerCorrect(idx)" class="result-check-icon" name="check" :size="18" aria-hidden="true" />
                </div>
            </div>
          </div>
       </div>

       <div class="host-controls">
         <button type="button" class="control-btn primary" @click="nextState">
           {{ $t('quiz.host.show_leaderboard') }}
         </button>
       </div>
    </div>

    <div v-else-if="gameState === 'leaderboard'" class="state-container leaderboard-container" :key="leaderboardAnimKey">
      <div class="lb-bg-glow"></div>
      <div class="lb-header">
        <AppIcon class="lb-trophy" name="trophy" :size="54" aria-hidden="true" />
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
            <AppIcon v-if="idx < 3" name="medal" :size="22" aria-hidden="true" />
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
        <button type="button" class="control-btn primary" @click="nextState">
          {{ isLastQuestion ? $t('quiz.host.end_quiz') : $t('quiz.host.next_question') }}
        </button>
      </div>
    </div>
    
    <div v-else-if="gameState === 'finished'" class="state-container final-container" :key="finalAnimKey">
       <div class="confetti-wrap">
         <div v-for="i in 20" :key="i" class="confetti-particle" :style="getConfettiStyle(i)"></div>
       </div>

       <div class="final-header">
         <AppIcon class="final-trophy" name="trophy" :size="64" aria-hidden="true" />
         <h1 class="final-title">{{ $t('quiz.host.quiz_complete') }}</h1>
         <p class="final-subtitle">{{ $t('quiz.host.final_standings') }}</p>
       </div>

       <div class="final-podium">
          <div class="final-podium-place fp-second" v-if="topPlayers[1]">
            <AppIcon class="fp-avatar" name="medal" :size="32" aria-hidden="true" />
            <div class="fp-name">{{ topPlayers[1].nickname }}</div>
            <div class="fp-score">{{ topPlayers[1].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-2"><span>2</span></div>
          </div>
          <div class="final-podium-place fp-first" v-if="topPlayers[0]">
            <AppIcon class="fp-crown" name="crown" :size="28" aria-hidden="true" />
            <AppIcon class="fp-avatar" name="medal" :size="32" aria-hidden="true" />
            <div class="fp-name">{{ topPlayers[0].nickname }}</div>
            <div class="fp-score">{{ topPlayers[0].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-1"><span>1</span></div>
          </div>
          <div class="final-podium-place fp-third" v-if="topPlayers[2]">
            <AppIcon class="fp-avatar" name="medal" :size="32" aria-hidden="true" />
            <div class="fp-name">{{ topPlayers[2].nickname }}</div>
            <div class="fp-score">{{ topPlayers[2].score.toLocaleString() }}</div>
            <div class="fp-block fp-block-3"><span>3</span></div>
          </div>
       </div>

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

       <button type="button" class="control-btn" @click="exitQuiz" style="margin-top: 20px; z-index: 1;">{{ $t('quiz.host.end_quiz') }}</button>
    </div>

    <audio ref="bgMusicPlayer" :src="fullMusicUrl" loop id="bg-music-player"></audio>
    
    <div v-if="showUnmuteButton" class="unmute-overlay">
        <button type="button" @click="enableAudio" class="unmute-btn">
            {{ $t('quiz.host.enable_sound') }}
        </button>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import QRCodeVue from 'qrcode.vue'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'
import { useI18n } from 'vue-i18n'
import { preloader } from '../utils/imagePreload'
import { copyTextToClipboard } from '../utils/clipboard'
import WrongPictureTargetCrop from '../components/WrongPictureTargetCrop.vue'
import AppIcon from '../components/AppIcon.vue'
import AdminThemeToggle from '../components/AdminThemeToggle.vue'

const props = defineProps({
  eventName: { type: String, default: '' },
  moduleName: { type: String, default: '' }
})

const route = useRoute()
const router = useRouter()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
const backToEventUrl = `/admin/events/${eventId}`
const hostContextLabel = computed(() => [props.eventName, props.moduleName].filter(Boolean).join(' / '))
const goBackToEvent = (event) => {
  event?.preventDefault?.()
  router.push(backToEventUrl)
}

const gameState = ref('lobby') 
const pinCode = ref(eventId)
const qrCodeUrl = computed(() => {
    const origin = typeof window !== 'undefined' ? window.location.origin : ''
    return `${origin}/?pin=${pinCode.value}`
})
const players = ref([])
const questions = ref([])
const currentQuestionIndex = ref(0)
const timeRemaining = ref(20)
const explainTimeRemaining = ref(0)
const answerCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 }) 
const playerSignals = ref([])
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
let revealTimer = null
const leaderboardFrames = new Set()
const nextLeaderboardFrame = (callback) => {
    const id = requestAnimationFrame((now) => { leaderboardFrames.delete(id); callback(now) })
    leaderboardFrames.add(id)
}
const stopLeaderboardFrames = () => {
    for (const id of leaderboardFrames) cancelAnimationFrame(id)
    leaderboardFrames.clear()
}

watch(questions, (newQuestions) => {
    if (Array.isArray(newQuestions) && newQuestions.length > 0) {
        preloader.preloadQuestions(newQuestions)
    }
}, { immediate: true })

watch(currentQuestionIndex, (newIdx) => {
    if (questions.value && questions.value[newIdx + 1]) {
        preloader.preloadQuestion(questions.value[newIdx + 1])
    }
})

watch(gameState, (newVal) => {
    clearTimeout(revealTimer)
    revealTimer = null
    stopLeaderboardFrames()
    if (newVal === 'results' && isRange.value) {
        revealCorrect.value = false
        revealTimer = setTimeout(() => {
            revealCorrect.value = true
            revealTimer = null
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
                    nextLeaderboardFrame(animate)
                    return
                }
                const elapsed = currentTime - startTime
                const progress = Math.min(elapsed / duration, 1)
                const easeOut = 1 - Math.pow(1 - progress, 3) 
                
                displayedScores.value[p.id] = Math.floor(startScore + (pointsGained * easeOut))
                
                if (progress < 1) {
                    nextLeaderboardFrame(animate)
                } else {
                    displayedScores.value[p.id] = targetScore
                }
            }
            nextLeaderboardFrame(animate)
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

const questionFontSize = computed(() => {
    const text = currentQuestion.value?.text || ''
    const hasImage = !!questionImageUrl.value

    if (!hasImage) {
        if (text.length < 30) {
            return 'clamp(1.5rem, 0.9rem + 4.5vw, 3.5rem)'
        } else if (text.length < 60) {
            return 'clamp(1.25rem, 0.75rem + 3.5vw, 2.8rem)'
        } else if (text.length < 100) {
            return 'clamp(1.1rem, 0.7rem + 2.5vw, 2.4rem)'
        } else {
            return 'clamp(1rem, 0.65rem + 2.0vw, 2.1rem)'
        }
    } else {
        if (text.length < 30) {
            return 'clamp(1.25rem, 0.75rem + 3.5vw, 2.8rem)'
        } else if (text.length < 60) {
            return 'clamp(1.1rem, 0.7rem + 2.5vw, 2.4rem)'
        } else if (text.length < 100) {
            return 'clamp(1rem, 0.65rem + 2.2vw, 2.2rem)'
        } else {
            return 'clamp(0.95rem, 0.65rem + 1.6vw, 1.85rem)'
        }
    }
})

const getAnswerFontSize = (answerText) => {
    const text = answerText || ''
    const length = text.length

    if (length < 15) {
        return 'clamp(1rem, 0.8rem + 1.4vw, 2.1rem)'
    } else if (length < 25) {
        return 'clamp(0.95rem, 0.75rem + 1.2vw, 1.85rem)'
    } else if (length < 40) {
        return 'clamp(0.95rem, 0.75rem + 1.1vw, 1.75rem)'
    } else if (length < 60) {
        return 'clamp(0.9rem, 0.72rem + 0.9vw, 1.5rem)'
    } else if (length < 100) {
        return 'clamp(0.875rem, 0.7rem + 0.7vw, 1.3rem)'
    } else {
        return 'clamp(0.85rem, 0.7rem + 0.5vw, 1.15rem)'
    }
}

const formatNumber = (num, decimals = 0) => {
    if (num === undefined || num === null) return ''
    return Number(num).toFixed(decimals).replace(/\.0+$/, '')
}

const formatWithUnit = (num, unit, decimals = 0) => {
    const formatted = formatNumber(num, decimals)
    if (!formatted) return ''
    const cleanUnit = typeof unit === 'string' ? unit.trim() : ''
    if (!cleanUnit) return formatted
    return /^[%°'"′″]/.test(cleanUnit) ? `${formatted}${cleanUnit}` : `${formatted} ${cleanUnit}`
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
    const min = q?.rangeConfig?.minValue ?? 0
    const max = q?.rangeConfig?.maxValue ?? 100
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

let timerInterval
let explainTimerInterval

const startLocalTimer = (duration) => {
    clearInterval(timerInterval)
    timeRemaining.value = duration
    const deadline = Date.now() + duration * 1000
    if (gameState.value !== 'question-active') return
    
    timerInterval = setInterval(() => {
        timeRemaining.value = Math.max(0, Math.ceil((deadline - Date.now()) / 1000))
        if (timeRemaining.value === 0) {
            clearInterval(timerInterval)
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
    const deadline = Date.now() + duration * 1000
    explainTimerInterval = setInterval(() => {
        explainTimeRemaining.value = Math.max(0, Math.ceil((deadline - Date.now()) / 1000))
        if (explainTimeRemaining.value === 0) {
            clearInterval(explainTimerInterval)
        }
    }, 1000)
}

const startQuiz = () => {
    quizWS.send('host:start_quiz', { moduleId })
}

const endQuestionEarly = () => {
    quizWS.send('host:show_results', { moduleId })
}

const goToResults = () => {
    quizWS.send('host:show_results', { moduleId })
}

const nextState = () => {
    if (gameState.value === 'results') {
        quizWS.send('host:show_leaderboard', { moduleId })
    } else if (gameState.value === 'leaderboard') {
        if (isLastQuestion.value) {
            quizWS.send('host:end_quiz', { moduleId })
        } else {
            quizWS.send('host:next_question', { moduleId })
        }
    }
}

const exitQuiz = async () => {
    quizWS.send('host:reset_quiz', { moduleId })
    
    await new Promise(resolve => setTimeout(resolve, 300))
    router.push(backToEventUrl)
}

const bgMusicPlayer = ref(null)
const backgroundMusicUrl = ref('')
const audioOutput = ref('projector')
const showUnmuteButton = ref(false)
const isMusicPlaying = ref(false)

const fullMusicUrl = computed(() => {
    if (!backgroundMusicUrl.value) return ''
    return `${API_BASE_URL}${backgroundMusicUrl.value}`
})

const manageAudioPlayback = async () => {
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

const { success, error: showError } = useToast()
const { t } = useI18n()

const copyProjectorLink = async () => {
    const url = `${window.location.origin}/projector/${eventId}`
    const copied = await copyTextToClipboard(url)
    if (copied) {
        success(t('quiz.host.toast.link_copied'))
    } else {
        showError(t('host.copy_link_failed'))
    }
}

watch(gameState, () => {
    manageAudioPlayback()
})

let stopped = false
onMounted(async () => {
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
            backgroundMusicUrl.value = mod.data.backgroundMusicUrl || ''
            audioOutput.value = mod.data.audioOutput || 'projector'
            manageAudioPlayback()
        }
    } catch (e) {
        console.error('Failed to load questions', e)
    }

    if (stopped) return

    quizWS.on('cmd:error', (payload) => {
        if (['save_failed', 'invalid_image_metadata', 'quiz_finishing', 'start_failed'].includes(payload?.code) && payload?.message) {
            showError(payload.message)
        }
    })

    quizWS.on('cmd:state', (payload) => {
        const prevState = gameState.value
        const prevIndex = currentQuestionIndex.value
        gameState.value = payload.state
        
        if (payload.currentIndex !== undefined) {
            currentQuestionIndex.value = payload.currentIndex
        }
        
        if (payload.state === 'question-active') {
            if (prevState !== 'question-active' || (payload.currentIndex !== undefined && payload.currentIndex !== prevIndex)) {
                startLocalTimer(currentQuestion.value?.timeLimit || 20)
            }
        } else if (payload.state === 'question-presenting') {
            clearInterval(timerInterval)
            if (currentQuestionIndex.value === 0) {
                displayedScores.value = {}
            }
            timeRemaining.value = currentQuestion.value?.timeLimit || 20
            startExplainTimer(payload.timeLimit || 5)
            realTimeAnsweredCount.value = 0
            answerCounts.value = { 0: 0, 1: 0, 2: 0, 3: 0 }
            playerSignals.value = []
        } else {
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

    quizWS.on('sync:minimal', (payload) => {
        if (payload.questionIndex !== undefined && payload.questionIndex !== currentQuestionIndex.value) return;
        
        if (payload.answerCounts) {
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
             realTimeAnsweredCount.value = playerSignals.value.length
        }

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

    quizWS.connect(moduleId, '', true)
 })

onUnmounted(() => {
    clearTimeout(revealTimer)
    stopLeaderboardFrames()
    stopped = true
    clearInterval(timerInterval)
    clearInterval(explainTimerInterval)
    preloader.clear()
    quizWS.disconnect()
    quizWS.clearHandlers()
})
</script>

<style scoped src="../styles/quiz-host.css"></style>
