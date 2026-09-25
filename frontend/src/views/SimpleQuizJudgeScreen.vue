<template>
  <div class="judge-screen">
    <div class="judge-header">
      <h1>{{ $t('simple_quiz.judge.title') }}</h1>
      <div class="judge-filters">
        <select v-model="questionFilter" class="judge-select">
          <option :value="-1">Q: all ({{ total }})</option>
          <option v-for="i in total" :key="i" :value="i - 1">Q{{ i }}</option>
        </select>
        <select v-model="pendingFilter" class="judge-select">
          <option value="pending">{{ $t('simple_quiz.judge.pending') }}</option>
          <option value="all">{{ $t('simple_quiz.judge.all') }}</option>
        </select>
        <button class="judge-btn" @click="undo" :disabled="gradingBusy">{{ $t('simple_quiz.judge.undo') }}</button>
      </div>
    </div>

    <p v-if="error" class="judge-error">{{ error }}</p>
    <p v-if="!token" class="judge-error">Missing ?token= judge token</p>

    <div class="judge-progress">
      <div v-for="p in progress" :key="p.questionIndex" class="prog-card" :class="{ active: p.questionIndex === currentIdx }">
        <strong>Q{{ p.questionIndex + 1 }}</strong>
        <span>{{ p.received }} {{ $t('simple_quiz.judge.received') }} · {{ p.graded }} {{ $t('simple_quiz.judge.graded') }} · {{ p.ungraded }} {{ $t('simple_quiz.judge.ungraded') }} · {{ p.reviewLater }} {{ $t('simple_quiz.judge.review') }}</span>
        <button class="link-btn" @click="questionFilter = p.questionIndex; currentIdx = p.questionIndex">open</button>
      </div>
    </div>

    <div v-if="queue.length === 0" class="judge-empty">{{ $t('simple_quiz.judge.empty') }}</div>

    <div v-else class="judge-card">
      <div class="judge-qmeta">
        <span class="q-badge">Q{{ current.questionIndex + 1 }}</span>
        <span class="q-text">{{ questionText(current.questionIndex) }}</span>
      </div>
      <div v-if="questionImage(current.questionIndex)" class="judge-img-wrap">
        <img :src="questionImage(current.questionIndex)" alt="" class="judge-img" />
      </div>
      <div class="judge-notes" v-if="questionNotes(current.questionIndex)">
        <strong>Judge notes:</strong> {{ questionNotes(current.questionIndex) }}
      </div>
      <div class="judge-points">{{ questionPoints(current.questionIndex) }} pts</div>
      <p class="judge-answer">{{ current.text }}</p>
      <div class="judge-actions">
        <button class="grade-right" @click="grade('right')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.right') }}</button>
        <button class="grade-wrong" @click="grade('wrong')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.wrong') }}</button>
        <button class="grade-later" @click="grade('review_later')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.review_later') }}</button>
      </div>
      <div class="judge-nav">
        <button @click="step(-1)">←</button>
        <span>{{ queueIndex + 1 }} / {{ queue.length }}</span>
        <button @click="step(1)">→</button>
      </div>
      <p class="judge-hint">{{ $t('simple_quiz.judge.next_pending') }}</p>
    </div>

    <p class="publish-hint">{{ $t('simple_quiz.judge.publish_hint') }} <span v-if="gradingComplete">✓</span></p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { simpleQuizApi } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import { quizWS } from '../services/websocket'

const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
const token = ref(route.query.token || '')

const queue = ref([])
const progress = ref([])
const gradingComplete = ref(false)
const questions = ref([])
const total = ref(0)
const questionFilter = ref(-1)
const pendingFilter = ref('pending')
const queueIndex = ref(0)
const error = ref('')
const gradingBusy = ref(false)

const current = computed(() => queue.value[queueIndex.value] || null)
const currentIdx = computed(() => current.value?.questionIndex ?? questionFilter.value)

const questionText = (qi) => questions.value[qi]?.text || ''
const questionNotes = (qi) => questions.value[qi]?.judgeNotes || ''
const questionPoints = (qi) => questions.value[qi]?.points || 0
const questionImage = (qi) => {
  const id = questions.value[qi]?.imageId
  return id ? getImageUrl(id) : ''
}

const loadState = async () => {
  // Host-view questions include no notes; fetch judge via queue progress + separate state?
  // For notes/image/points we use projector state (no notes) + judge queue.
  // Notes need a judge-aware state: reuse queue + fetch module template for notes.
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'projector' })
    total.value = s.total || 0
    questions.value = s.questions || []
  } catch (e) {
    error.value = e.message
  }
}

const loadQueue = async () => {
  if (!token.value) return
  try {
    error.value = ''
    const res = await simpleQuizApi.queue(eventId, moduleId, token.value, {
      questionIndex: questionFilter.value,
      filter: pendingFilter.value,
      limit: 50
    })
    queue.value = res.answers || []
    progress.value = res.progress || []
    gradingComplete.value = !!res.gradingComplete
    if (Array.isArray(res.questions) && res.questions.length) {
      questions.value = res.questions
      total.value = res.total || res.questions.length
    }
    if (queueIndex.value >= queue.value.length) queueIndex.value = 0
  } catch (e) {
    error.value = e.message
  }
}

const grade = async (verdict) => {
  if (!current.value) return
  gradingBusy.value = true
  const answerId = current.value.answerId
  try {
    await simpleQuizApi.grade(eventId, moduleId, { answerId, verdict, judgeToken: token.value })
    await loadQueue()
    // Next pending takes its place: keep index, list shrinks in pending mode.
    if (queueIndex.value >= queue.value.length) queueIndex.value = Math.max(0, queue.value.length - 1)
  } catch (e) {
    error.value = e.message
  } finally {
    gradingBusy.value = false
  }
}

const undo = async () => {
  gradingBusy.value = true
  try {
    await simpleQuizApi.undo(eventId, moduleId, { judgeToken: token.value })
    await loadQueue()
  } catch (e) {
    error.value = e.message
  } finally {
    gradingBusy.value = false
  }
}

const step = (d) => {
  if (!queue.value.length) return
  queueIndex.value = (queueIndex.value + d + queue.value.length) % queue.value.length
}

watch([questionFilter, pendingFilter], () => {
  queueIndex.value = 0
  loadQueue()
})

let poll = null
onMounted(async () => {
  await loadState()
  await loadQueue()
  quizWS.on('simple:state', () => {
    loadState()
    loadQueue()
  })
  quizWS.connect(moduleId, '', false)
  if (token.value) quizWS.send('simple:judge_join', { moduleId, judgeToken: token.value })
  poll = setInterval(loadQueue, 5000)
})
onUnmounted(() => {
  if (poll) clearInterval(poll)
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped>
.judge-screen { background: #1C1C1E; color: #fff; min-height: 100vh; padding: 20px; max-width: 760px; margin: 0 auto; }
.judge-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.judge-filters { display: flex; gap: 8px; align-items: center; }
.judge-select { background: #2C2C2E; color: #fff; border: 1px solid #3A3A3C; border-radius: 8px; padding: 8px; }
.judge-btn { background: #2C2C2E; color: #fff; border: 1px solid #3A3A3C; border-radius: 8px; padding: 8px 12px; cursor: pointer; }
.judge-error { color: #ff9090; background: rgba(255,0,0,0.12); padding: 10px; border-radius: 8px; }
.judge-progress { display: flex; flex-direction: column; gap: 8px; margin: 16px 0; }
.prog-card { background: #2C2C2E; border-radius: 8px; padding: 10px 12px; display: flex; gap: 10px; align-items: center; font-size: 13px; }
.prog-card.active { border: 1px solid #0A84FF; }
.link-btn { margin-left: auto; background: none; border: 1px solid #3A3A3C; color: #0A84FF; border-radius: 6px; padding: 4px 8px; cursor: pointer; }
.judge-empty { color: #8E8E93; text-align: center; padding: 32px; }
.judge-card { background: #2C2C2E; border-radius: 12px; padding: 16px; }
.judge-qmeta { display: flex; gap: 10px; align-items: center; margin-bottom: 8px; }
.q-badge { background: #0A84FF; border-radius: 16px; padding: 4px 10px; font-weight: 800; }
.q-text { font-weight: 700; }
.judge-img-wrap { text-align: center; }
.judge-img { max-width: 100%; max-height: 220px; border-radius: 8px; }
.judge-notes { background: #1C1C1E; border: 1px dashed #48484A; border-radius: 8px; padding: 10px; font-size: 13px; margin: 8px 0; }
.judge-points { font-size: 12px; color: #8E8E93; }
.judge-answer { font-size: 17px; line-height: 1.5; background: #1C1C1E; border-radius: 8px; padding: 14px; white-space: pre-wrap; word-break: break-word; }
.judge-actions { display: flex; gap: 8px; margin-top: 12px; }
.grade-right { flex: 1; background: #32D74B; border: none; border-radius: 8px; padding: 12px; font-weight: 800; cursor: pointer; }
.grade-wrong { flex: 1; background: #FF3B30; border: none; border-radius: 8px; padding: 12px; font-weight: 800; color: #fff; cursor: pointer; }
.grade-later { flex: 1; background: #48484A; border: none; border-radius: 8px; padding: 12px; font-weight: 700; color: #fff; cursor: pointer; }
.judge-nav { display: flex; justify-content: space-between; align-items: center; margin-top: 12px; }
.judge-nav button { background: #1C1C1E; color: #fff; border: 1px solid #3A3A3C; border-radius: 8px; padding: 8px 14px; cursor: pointer; }
.judge-hint { color: #8E8E93; font-size: 12px; }
.publish-hint { color: #8E8E93; font-size: 13px; margin-top: 16px; }
</style>
