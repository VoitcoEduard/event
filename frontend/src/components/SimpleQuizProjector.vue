<template>
  <div v-if="visible" class="sq-projector">
    <div v-if="status === 'lobby' || status === 'inactive'" class="state-container lobby-container">
      <div class="lobby-header">
        <h1 class="game-title">{{ $t('simple_quiz.projector.join_title') }}</h1>
        <div class="pin-display">
          <span class="pin-label">{{ $t('simple_quiz.host.game_pin') }}</span>
          <span class="pin-code">{{ pin }}</span>
        </div>
      </div>
      <div class="players-area">
        <div class="players-count"><span><AppIcon name="teams" :size="18" /> {{ players }}</span></div>
      </div>
    </div>

    <div v-else-if="status === 'question_open' || status === 'answers_locked' || status === 'between_questions'" class="state-container question-container">
      <div class="question-header">
        <div class="question-counter">{{ currentIndex + 1 }} / {{ total }}</div>
      </div>
      <div class="question-main-content">
        <div class="question-column">
          <div class="question-card" :class="{ 'has-image': !!image }">
            <div v-if="image" class="question-image-container">
              <img :src="image" class="question-image" alt="" />
              <div class="image-overlay-gradient"></div>
            </div>
            <div class="question-text-overlay">
              <h2 class="question-text">{{ question?.text }}</h2>
            </div>
          </div>
          <div v-if="revealedAnswer" class="reveal-card"><AppIcon name="check" :size="16" /> {{ revealedAnswer }}</div>
        </div>
        <div class="answers-column">
          <div class="answer-counter-widget">
            <div class="counter-value">{{ answeredCount }}</div>
            <div class="counter-label">{{ $t('simple_quiz.host.answered') }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="status === 'waiting_grading' || status === 'answering_finished'" class="state-container get-ready-container">
      <div class="loading-content">
        <h1>{{ $t('simple_quiz.projector.grading') }}</h1>
        <div class="loader"></div>
      </div>
    </div>

    <div v-else-if="status === 'results_published' || status === 'completed'" class="state-container leaderboard-container">
      <div class="lb-header">
        <AppIcon class="lb-trophy" name="trophy" :size="54" />
        <h2 class="lb-title">{{ $t('simple_quiz.projector.results') }}</h2>
      </div>
      <div class="lb-list">
        <div v-for="(p, idx) in leaderboard" :key="p.playerId" class="lb-row">
          <div class="lb-rank-badge">{{ idx + 1 }}</div>
          <div class="lb-player-name">{{ p.nickname }}</div>
          <div class="lb-score-value">{{ p.totalScore }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { simpleQuizApi } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import { quizWS } from '../services/websocket'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  eventId: { type: String, required: true },
  moduleId: { type: String, required: true },
  pin: { type: String, default: '' },
  visible: { type: Boolean, default: true }
})

const status = ref('lobby')
const currentIndex = ref(0)
const total = ref(0)
const players = ref(0)
const answeredCount = ref(0)
const question = ref(null)
const revealedAnswer = ref('')
const leaderboard = ref([])

const image = computed(() => question.value?.imageId ? getImageUrl(question.value.imageId) : null)

const refresh = async () => {
  if (!props.visible) return
  try {
    const s = await simpleQuizApi.state(props.eventId, props.moduleId, { view: 'projector' })
    status.value = s.status || 'lobby'
    currentIndex.value = s.currentIndex || 0
    total.value = s.total || 0
    players.value = s.players || 0
    answeredCount.value = s.answeredCount || 0
    const qs = s.questions || []
    question.value = qs[currentIndex.value] || null
    revealedAnswer.value = question.value?.correctAnswer || ''
    if (s.status === 'results_published' || s.status === 'completed') {
      const r = await simpleQuizApi.results(props.eventId, props.moduleId, { view: 'projector' })
      leaderboard.value = r.published?.players || []
    }
  } catch { /* ignore polling errors */ }
}

let poll = null
watch(() => [props.moduleId, props.visible], refresh)

onMounted(() => {
  refresh()
  quizWS.on('simple:state', refresh)
  poll = setInterval(refresh, 4000)
})
onUnmounted(() => {
  if (poll) clearInterval(poll)
})
</script>

<style scoped>
.sq-projector { width: 100%; }
.reveal-card { background: rgba(50,215,75,0.15); border: 1px solid #32D74B; border-radius: 8px; padding: 16px; margin-top: 12px; font-weight: 800; font-size: 1.4rem; text-align: center; }
</style>
