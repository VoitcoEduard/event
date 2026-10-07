<template>
  <div class="quiz-host-screen">
    <div class="bg-layer"></div>

    <div v-if="stateStatus === 'inactive' || !launched || stateStatus === 'lobby'" class="state-container lobby-container">
      <div class="lobby-header">
        <h1 class="game-title">{{ $t('simple_quiz.host.join_title') }}</h1>
        <div class="pin-display">
          <span class="pin-label">{{ $t('simple_quiz.host.game_pin') }}</span>
          <span class="pin-code">{{ pinCode }}</span>
        </div>
        <div class="lobby-actions">
          <button class="copy-link-btn" @click="copyProjectorLink">{{ $t('simple_quiz.host.copy_projector') }}</button>
          <button v-if="judgeToken" class="copy-link-btn" @click="copyJudgeLink">{{ $t('simple_quiz.host.copy_judge') }}</button>
        </div>
        <p v-if="judgeToken" class="judge-url">{{ judgeUrl }}</p>
      </div>
      <div class="players-area">
        <div class="players-count"><span><AppIcon name="teams" :size="18" /> {{ playersCount }}</span> {{ $t('simple_quiz.host.players') }}</div>
        <div class="players-grid">
          <div v-for="p in playerList" :key="p.playerId" class="player-chip">{{ p.nickname }}</div>
        </div>
      </div>
      <p v-if="error" class="host-error">{{ error }}</p>
      <div class="lobby-footer" style="display:flex;gap:8px;justify-content:center;flex-wrap:wrap;">
        <button v-if="!launched || stateStatus === 'inactive'" class="start-btn" @click="doLaunch">{{ $t('simple_quiz.host.start') }}</button>
        <button v-else class="start-btn" @click="doOpen">{{ $t('simple_quiz.host.open') }} (Q{{ currentIndex + 1 }})</button>
      </div>
    </div>

    <div v-else class="state-container question-container">
      <div class="question-header">
        <div class="timer" v-if="deadlineLeft !== null" :class="{ 'timer-low': deadlineLeft <= 10 }">{{ deadlineLeft }}s</div>
        <div class="timer" v-else>∞</div>
        <div class="question-counter">{{ currentIndex + 1 }} / {{ total }} · {{ stateStatus }}</div>
      </div>

      <div class="question-main-content">
        <div class="question-column">
          <div class="question-card" :class="{ 'has-image': !!currentImage }">
            <div v-if="currentImage" class="question-image-container">
              <img :src="currentImage" class="question-image" alt="" />
              <div class="image-overlay-gradient"></div>
            </div>
            <div class="question-text-overlay">
              <h2 class="question-text">{{ currentQuestion?.text }}</h2>
            </div>
          </div>
          <div v-if="revealedAnswer" class="reveal-card"><AppIcon name="check" :size="16" /> {{ revealedAnswer }}</div>
        </div>
        <div class="answers-column">
          <div class="answer-counter-widget">
            <div class="counter-value">{{ answeredCount }}</div>
            <div class="counter-label">{{ $t('simple_quiz.host.answered') }}</div>
          </div>
          <div class="grading-card">
            <h4>{{ $t('simple_quiz.host.grading_progress') }}</h4>
            <div v-for="p in progress" :key="p.questionIndex" class="prog-row">
              <span>Q{{ p.questionIndex + 1 }}</span>
              <span>{{ p.received }} / {{ p.graded }} <AppIcon name="check" :size="14" /> / {{ p.reviewLater }}…</span>
            </div>
            <p v-if="gradingComplete"><AppIcon name="check" :size="16" /> {{ $t('simple_quiz.host.waiting_grading') }} ready</p>
            <p v-else>{{ $t('simple_quiz.host.waiting_grading') }}</p>
          </div>
          <div v-if="leaderboard.length" class="grading-card">
            <h4>{{ $t('simple_quiz.player.leaderboard') }}</h4>
            <div v-for="(p, i) in leaderboard" :key="p.playerId" class="prog-row">
              <span>{{ i + 1 }}. {{ p.nickname }}</span><span>{{ p.totalScore }}</span>
            </div>
          </div>
        </div>
      </div>

      <p v-if="error" class="host-error">{{ error }}</p>
      <p class="reveal-hint">{{ $t('simple_quiz.host.reveal_hint') }}</p>

      <div class="host-controls">
        <button class="control-btn" @click="doOpen">{{ $t('simple_quiz.host.open') }}</button>
        <button class="control-btn" @click="doLock">{{ $t('simple_quiz.host.lock') }}</button>
        <button class="control-btn" @click="doNext">{{ $t('simple_quiz.host.next') }}</button>
        <button class="control-btn" @click="doReveal">{{ $t('simple_quiz.host.reveal') }}</button>
        <button class="control-btn" @click="doEnd">{{ $t('simple_quiz.host.end_answering') }}</button>
        <button class="control-btn primary" @click="doPublish">{{ $t('simple_quiz.host.publish') }}</button>
        <button class="control-btn" @click="doUpdate">{{ $t('simple_quiz.host.update_published') }}</button>
        <button class="control-btn" @click="doComplete">{{ $t('simple_quiz.host.complete') }}</button>
        <button v-if="stateStatus === 'paused'" class="control-btn primary" @click="doResume">{{ $t('simple_quiz.host.resume') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppIcon from '../components/AppIcon.vue'
import { quizWS } from '../services/websocket'
import { API_BASE_URL } from '../utils'
import { simpleQuizApi } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'

const route = useRoute()
const router = useRouter()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
const { t } = useI18n()
const { success } = useToast()

const launched = ref(false)
const stateStatus = ref('inactive')
const currentIndex = ref(0)
const total = ref(0)
const playersCount = ref(0)
const answeredCount = ref(0)
const playerList = ref([])
const questions = ref([])
const progress = ref([])
const gradingComplete = ref(false)
const leaderboard = ref([])
const revealed = ref({})
const deadlineUnix = ref(0)
const deadlineLeft = ref(null)
const allowReveal = ref(true)
const judgeToken = ref('')
const pinCode = ref(eventId)
const error = ref('')

const currentQuestion = computed(() => questions.value[currentIndex.value] || null)
const currentImage = computed(() => currentQuestion.value?.imageId ? getImageUrl(currentQuestion.value.imageId) : null)
const revealedAnswer = computed(() => {
  if (!allowReveal.value || !revealed.value[currentIndex.value]) return ''
  return currentQuestion.value?.correctAnswer || ''
})

const judgeUrl = computed(() => judgeToken.value ? `${window.location.origin}/simple-quiz/${eventId}/${moduleId}/judge?token=${encodeURIComponent(judgeToken.value)}` : '')

let deadlineTimer = null
const tick = () => {
  if (!deadlineUnix.value) {
    deadlineLeft.value = stateStatus.value === 'question_open' ? null : null
    return
  }
  deadlineLeft.value = Math.max(0, deadlineUnix.value - Math.floor(Date.now() / 1000))
}

const applyState = (s) => {
  if (!s || s.status === 'inactive') {
    launched.value = false
    stateStatus.value = 'inactive'
    return
  }
  launched.value = true
  stateStatus.value = s.status
  currentIndex.value = s.currentIndex || 0
  total.value = s.total || 0
  playersCount.value = s.players || 0
  answeredCount.value = s.answeredCount || 0
  playerList.value = s.playerList || []
  questions.value = (s.questions || []).map((q, i) => ({ ...q, text: q.text, correctAnswer: q.correctAnswer || '' }))
  progress.value = s.progress || []
  gradingComplete.value = !!s.gradingComplete
  leaderboard.value = s.working || []
  revealed.value = s.revealed || {}
  deadlineUnix.value = s.deadlineUnix || 0
  allowReveal.value = s.allowReveal !== false
  tick()
}

const refresh = async () => {
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'host' })
    // Fetch projector view for revealed correct answers.
    try {
      const pub = await simpleQuizApi.state(eventId, moduleId, { view: 'projector' })
      const byIdx = {}
      ;(pub.questions || []).forEach((q, i) => {
        if (q.correctAnswer) byIdx[i] = q.correctAnswer
      })
      s.questions = (s.questions || []).map((q, i) => ({ ...q, correctAnswer: byIdx[i] || '' }))
    } catch { /* ignore */ }
    applyState(s)
    error.value = ''
  } catch (e) {
    error.value = e.message
  }
}

const wrap = (fn, confirmMsg = '') => async () => {
  try {
    if (confirmMsg && !confirm(confirmMsg)) return
    error.value = ''
    const res = await fn()
    if (res?.judgeToken) judgeToken.value = res.judgeToken
    await refresh()
  } catch (e) {
    error.value = e.message
  }
}

const doLaunch = wrap(async () => {
  const res = await simpleQuizApi.launch(eventId, moduleId)
  if (res?.judgeToken) judgeToken.value = res.judgeToken
  return res
})
const doOpen = wrap(() => simpleQuizApi.open(eventId, moduleId))
const doLock = wrap(() => simpleQuizApi.lock(eventId, moduleId))
const doNext = wrap(() => simpleQuizApi.next(eventId, moduleId))
const doReveal = wrap(() => simpleQuizApi.reveal(eventId, moduleId))
const doEnd = wrap(() => simpleQuizApi.endAnswering(eventId, moduleId), t('simple_quiz.host.confirm_end'))
const doPublish = wrap(() => simpleQuizApi.publish(eventId, moduleId), t('simple_quiz.host.confirm_publish'))
const doUpdate = wrap(() => simpleQuizApi.updatePublished(eventId, moduleId))
const doComplete = wrap(() => simpleQuizApi.complete(eventId, moduleId))
const doResume = wrap(() => simpleQuizApi.resume(eventId, moduleId))

const copyJudgeLink = async () => {
  try {
    await navigator.clipboard.writeText(judgeUrl.value)
    success(t('simple_quiz.config.copied'))
  } catch { /* ignore */ }
}
const copyProjectorLink = async () => {
  try {
    await navigator.clipboard.writeText(`${window.location.origin}/projector/${eventId}`)
    success(t('quiz.host.toast.link_copied'))
  } catch { /* ignore */ }
}

let stopped = false
onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}`)
    if (stopped) return
    const data = await res.json()
    if (stopped) return
    pinCode.value = data.pin || eventId
  } catch { /* ignore */ }
  if (stopped) return
  await refresh()
  if (stopped) return
  quizWS.on('simple:state', refresh)
  quizWS.connect(moduleId, '', true)
  quizWS.send('simple:host_join', { moduleId })
  deadlineTimer = setInterval(tick, 1000)
  void router
})

onUnmounted(() => {
  stopped = true
  if (deadlineTimer) clearInterval(deadlineTimer)
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped src="../styles/quiz-host.css"></style>
<style scoped>
.host-error { color: #ff9090; text-align: center; }
.judge-url { font-size: 11px; color: #0A84FF; word-break: break-all; }
.reveal-card { background: rgba(50,215,75,0.15); border: 1px solid #32D74B; border-radius: 8px; padding: 12px; margin-top: 12px; font-weight: 700; }
.grading-card { background: rgba(0,0,0,0.3); border-radius: 8px; padding: 12px; margin-top: 12px; }
.grading-card h4 { margin: 0 0 8px; }
.prog-row { display: flex; justify-content: space-between; font-size: 13px; padding: 4px 0; }
.reveal-hint { text-align: center; color: #8E8E93; font-size: 12px; }
</style>
