<template>
  <div class="judge-screen">
    <div class="judge-utility-strip">
      <AdminThemeToggle variant="inline" />
    </div>
    <div class="judge-header">
      <h1>{{ $t('simple_quiz.judge.title') }}</h1>
      <div class="judge-filters">
        <select v-model="questionFilter" class="judge-select">
          <option :value="-1">{{ $t('simple_quiz.judge.question_all', { total }) }}</option>
          <option v-for="i in total" :key="i" :value="i - 1">Q{{ i }}</option>
        </select>
        <select v-model="pendingFilter" class="judge-select">
          <option value="pending">{{ $t('simple_quiz.judge.pending') }}</option>
          <option value="all">{{ $t('simple_quiz.judge.all') }}</option>
        </select>
        <button type="button" class="judge-btn" @click="undo" :disabled="gradingBusy">{{ $t('simple_quiz.judge.undo') }}</button>
      </div>
    </div>

    <p v-if="error" class="judge-error" role="alert">{{ error }}</p>
    <p v-if="!token" class="judge-error" role="alert">{{ $t('simple_quiz.judge.missing_token') }}</p>

    <div class="judge-progress">
      <div v-for="p in progress" :key="p.questionIndex" class="prog-card" :class="{ active: p.questionIndex === currentIdx }">
        <strong>Q{{ p.questionIndex + 1 }}</strong>
        <span>{{ p.received }} {{ $t('simple_quiz.judge.received') }} · {{ p.graded }} {{ $t('simple_quiz.judge.graded') }} · {{ p.ungraded }} {{ $t('simple_quiz.judge.ungraded') }} · {{ p.reviewLater }} {{ $t('simple_quiz.judge.review') }}</span>
        <button type="button" class="link-btn" @click="questionFilter = p.questionIndex">{{ $t('common.open') }}</button>
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
        <strong>{{ $t('simple_quiz.judge.notes') }}:</strong> {{ questionNotes(current.questionIndex) }}
      </div>
      <div class="judge-notes" v-if="questionCorrectAnswer(current.questionIndex)">
        <strong>{{ $t('simple_quiz.player.revealed_answer') }}:</strong> {{ questionCorrectAnswer(current.questionIndex) }}
      </div>
      <div class="judge-points">{{ $t('simple_quiz.judge.points_short', { points: questionPoints(current.questionIndex) }) }}</div>
      <p class="judge-answer">{{ current.text }}</p>
      <div class="judge-actions">
        <button type="button" class="grade-right" @click="grade('right')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.right') }}</button>
        <button type="button" class="grade-wrong" @click="grade('wrong')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.wrong') }}</button>
        <button type="button" class="grade-later" @click="grade('review_later')" :disabled="gradingBusy">{{ $t('simple_quiz.judge.review_later') }}</button>
      </div>
      <div class="judge-nav">
        <button type="button" @click="step(-1)" :aria-label="$t('common.previous')"><AppIcon name="arrowLeft" :size="18" aria-hidden="true" /></button>
        <span>{{ queueIndex + 1 }} / {{ queue.length }}</span>
        <button type="button" @click="step(1)" :aria-label="$t('common.next')"><AppIcon name="arrowRight" :size="18" aria-hidden="true" /></button>
      </div>
      <p class="judge-hint">{{ $t('simple_quiz.judge.next_pending') }}</p>
    </div>

    <p class="publish-hint">{{ $t('simple_quiz.judge.publish_hint') }} <AppIcon v-if="gradingComplete" name="check" :size="16" aria-hidden="true" /></p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { simpleQuizApi } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import { quizWS } from '../services/websocket'
import { newUUID } from '../utils'
import AppIcon from '../components/AppIcon.vue'
import AdminThemeToggle from '../components/AdminThemeToggle.vue'

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
let judgeKey = ''

const current = computed(() => queue.value[queueIndex.value] || null)
const currentIdx = computed(() => current.value?.questionIndex ?? questionFilter.value)

const questionText = (qi) => questions.value[qi]?.text || ''
const questionNotes = (qi) => questions.value[qi]?.judgeNotes || ''
const questionCorrectAnswer = (qi) => questions.value[qi]?.correctAnswerText || ''
const questionPoints = (qi) => questions.value[qi]?.points || 0
const questionImage = (qi) => {
  const id = questions.value[qi]?.imageId
  return id ? getImageUrl(id) : ''
}

let queueReqSeq = 0

const loadState = async () => {
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'projector' })
    total.value = s.total || 0
    questions.value = s.questions || []
  } catch (e) {
    error.value = e.message
  }
}

const loadQueue = async ({ preserveAnswerId = true } = {}) => {
  if (!token.value) return
  const reqSeq = ++queueReqSeq
  const activeAnswerId = preserveAnswerId ? current.value?.answerId || '' : ''
  try {
    error.value = ''
    const allAnswers = []
    let cursor = ''
    let latestRes = null
    for (let page = 0; page < 20; page++) {
      const res = await simpleQuizApi.queue(eventId, moduleId, token.value, {
        questionIndex: questionFilter.value,
        filter: pendingFilter.value,
        limit: 100,
        cursor
      })
      if (stopped || reqSeq !== queueReqSeq) return
      latestRes = res
      const batch = res.answers || []
      allAnswers.push(...batch)
      if (!res.nextCursor || batch.length === 0) break
      cursor = res.nextCursor
    }
    if (!latestRes || stopped || reqSeq !== queueReqSeq) return
    queue.value = allAnswers
    progress.value = latestRes.progress || []
    gradingComplete.value = !!latestRes.gradingComplete
    if (Array.isArray(latestRes.questions) && latestRes.questions.length) {
      questions.value = latestRes.questions
      total.value = latestRes.total || latestRes.questions.length
    }
    if (activeAnswerId) {
      const keptIdx = allAnswers.findIndex((item) => item.answerId === activeAnswerId)
      if (keptIdx >= 0) {
        queueIndex.value = keptIdx
        return
      }
    }
    if (queueIndex.value >= allAnswers.length) {
      queueIndex.value = Math.max(0, allAnswers.length - 1)
    }
  } catch (e) {
    if (!stopped && reqSeq === queueReqSeq) error.value = e.message
  }
}

const grade = async (verdict) => {
  if (!current.value || gradingBusy.value) return
  gradingBusy.value = true
  const answerId = current.value.answerId
  try {
    if (!judgeKey) judgeKey = newUUID()
    await simpleQuizApi.grade(eventId, moduleId, { answerId, verdict, judgeToken: token.value, judgeKey })
    await loadQueue({ preserveAnswerId: false })
  } catch (e) {
    error.value = e.message
  } finally {
    gradingBusy.value = false
  }
}

const undo = async () => {
  if (gradingBusy.value) return
  gradingBusy.value = true
  try {
    if (!judgeKey) judgeKey = newUUID()
    await simpleQuizApi.undo(eventId, moduleId, { judgeToken: token.value, judgeKey })
    await loadQueue({ preserveAnswerId: false })
  } catch (e) {
    if (e.status === 409) await loadQueue({ preserveAnswerId: true })
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
  loadQueue({ preserveAnswerId: false })
})

let poll = null
let stopped = false
onMounted(async () => {
  if (token.value) {
    await loadQueue({ preserveAnswerId: false })
  } else {
    await loadState()
  }
  if (stopped) return
  quizWS.on('simple:state', () => {
    if (token.value) {
      loadQueue({ preserveAnswerId: true })
    } else {
      loadState()
    }
  })
  if (token.value) quizWS.connect(moduleId, '', false, false, eventId, { type: 'simple:judge_join', payload: { moduleId, judgeToken: token.value } })
  poll = setInterval(() => loadQueue({ preserveAnswerId: true }), 5000)
})
onUnmounted(() => {
  stopped = true
  if (poll) clearInterval(poll)
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped>
.judge-screen {
  container: judge-screen / inline-size;
  background: #1C1C1E;
  color: #fff;
  min-height: 100vh;
  min-height: 100dvh;
  box-sizing: border-box;
  padding: max(16px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  max-width: 760px;
  margin: 0 auto;
}
.judge-utility-strip {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
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
.q-badge { background: #0969da; color: #fff; border-radius: 16px; padding: 4px 10px; font-weight: 800; }
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

:global(html[data-admin-appearance="true"] .judge-screen .prog-card) {
  background: var(--admin-surface) !important;
  border: 1px solid var(--admin-border) !important;
  color: var(--admin-text) !important;
  box-shadow: var(--admin-shadow) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .prog-card.active) {
  border-color: var(--admin-primary-solid) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .judge-notes),
:global(html[data-admin-appearance="true"] .judge-screen .judge-answer) {
  background: var(--admin-surface-subtle) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .judge-points),
:global(html[data-admin-appearance="true"] .judge-screen .judge-hint),
:global(html[data-admin-appearance="true"] .judge-screen .publish-hint),
:global(html[data-admin-appearance="true"] .judge-screen .judge-empty),
:global(html[data-admin-appearance="true"] .judge-screen .prog-card span) {
  color: var(--admin-text-muted) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .link-btn) {
  background: var(--admin-primary-soft) !important;
  border-color: color-mix(in srgb, var(--admin-primary) 35%, var(--admin-border)) !important;
  color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .judge-nav button) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .judge-error) {
  background: var(--admin-danger-soft) !important;
  color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .judge-screen .q-badge) {
  background: var(--admin-primary-solid) !important;
  color: #fff !important;
}

@container judge-screen (inline-size <= 540px) {
  .judge-header {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .judge-filters {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .judge-select {
    width: 100%;
    min-block-size: 44px;
    font-size: 16px;
    box-sizing: border-box;
  }

  .judge-btn {
    grid-column: 1 / -1;
    min-block-size: 44px;
    font-size: 16px;
  }

  .prog-card {
    flex-wrap: wrap;
    gap: 8px;
  }

  .prog-card span {
    order: 3;
    flex: 1 0 100%;
  }

  .judge-actions {
    flex-wrap: wrap;
    gap: 8px;
  }

  .grade-right,
  .grade-wrong,
  .grade-later {
    flex: 1 1 110px;
    min-block-size: 48px;
    white-space: normal;
    line-height: 1.2;
  }

  .judge-nav button {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}

@media (max-width: 768px), (pointer: coarse) {
  .judge-select,
  .judge-btn,
  .link-btn,
  .judge-actions button,
  .judge-nav button,
  .judge-utility-strip :deep(.admin-theme-option) {
    min-block-size: 44px;
  }

  .judge-nav button {
    min-inline-size: 44px;
  }
}
</style>
