<template>
  <div class="quiz-host-screen simple-quiz-host-screen">
    <div class="bg-layer"></div>

    <div v-if="stateStatus === 'inactive' || !launched || stateStatus === 'lobby'" class="state-container lobby-container">
      <div class="lobby-header">
        <h1 class="game-title">{{ $t('simple_quiz.host.join_title') }}</h1>
        <div class="pin-display">
          <span class="pin-label">{{ $t('simple_quiz.host.game_pin') }}</span>
          <span class="pin-code">{{ pinCode }}</span>
        </div>
        <div class="lobby-actions">
          <button type="button" class="copy-link-btn" @click="copyProjectorLink">{{ $t('simple_quiz.host.copy_projector') }}</button>
          <button v-if="judgeToken" type="button" class="copy-link-btn" @click="copyJudgeLink">{{ $t('simple_quiz.host.copy_judge') }}</button>
        </div>
        <p v-if="judgeToken" class="judge-url">{{ judgeUrl }}</p>
      </div>
      <div class="players-area">
        <div class="players-count"><span><AppIcon name="teams" :size="18" aria-hidden="true" /> {{ playersCount }}</span> {{ $t('simple_quiz.host.players') }}</div>
        <div class="players-grid">
          <div v-for="p in playerList" :key="p.playerId" class="player-chip">{{ p.nickname }}</div>
        </div>
      </div>
      <p v-if="error" class="host-error">{{ error }}</p>
      <div class="lobby-footer" style="display:flex;gap:8px;justify-content:center;flex-wrap:wrap;">
        <button v-if="!launched || stateStatus === 'inactive'" type="button" class="start-btn" :disabled="actionBusy" @click="doLaunch">{{ $t('simple_quiz.host.start') }}</button>
        <button v-else type="button" class="start-btn" :disabled="actionBusy" @click="doOpen">{{ $t('simple_quiz.host.open') }} (Q{{ currentIndex + 1 }})</button>
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
          <div v-if="revealedAnswer" class="reveal-card"><AppIcon name="check" :size="16" aria-hidden="true" /> {{ revealedAnswer }}</div>
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
              <span>{{ p.received }} / {{ p.graded }} <AppIcon name="check" :size="14" aria-hidden="true" /> / {{ p.reviewLater }}…</span>
            </div>
            <p v-if="gradingComplete"><AppIcon name="check" :size="16" aria-hidden="true" /> {{ $t('simple_quiz.host.waiting_grading') }} ready</p>
            <p v-else>{{ $t('simple_quiz.host.waiting_grading') }}</p>
          </div>
          <div v-if="leaderboard.length" class="grading-card">
            <h4>{{ $t('simple_quiz.player.leaderboard') }}</h4>
            <div v-for="(p, i) in leaderboard" :key="p.playerId" class="prog-row">
              <span>{{ i + 1 }}. {{ p.nickname }}</span><span>{{ p.totalScore }}</span>
            </div>
          </div>
		  <div v-if="playerList.length && participantOptions.length" class="grading-card">
			<h4>Event identities</h4>
			<div v-for="p in playerList" :key="p.playerId" class="prog-row">
				<span>{{ p.nickname }}</span>
				<select :value="p.participantId || ''" :aria-label="`Link ${p.nickname} to an event participant`" @change="linkPlayer(p.playerId, $event)">
					<option value="">Separate score row</option>
					<option v-for="person in participantOptions" :key="person.id" :value="person.id">{{ person.displayName }} ({{ person.id }})</option>
				</select>
			</div>
		  </div>
        </div>
      </div>

      <p v-if="error" class="host-error">{{ error }}</p>
      <p class="reveal-hint">{{ $t('simple_quiz.host.reveal_hint') }}</p>

      <div class="host-controls">
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doOpen">{{ $t('simple_quiz.host.open') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doLock">{{ $t('simple_quiz.host.lock') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doNext">{{ $t('simple_quiz.host.next') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doReveal">{{ $t('simple_quiz.host.reveal') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doEnd">{{ $t('simple_quiz.host.end_answering') }}</button>
        <button type="button" class="control-btn primary" :disabled="actionBusy" @click="doPublish">{{ $t('simple_quiz.host.publish') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doUpdate">{{ $t('simple_quiz.host.update_published') }}</button>
        <button type="button" class="control-btn" :disabled="actionBusy" @click="doComplete">{{ $t('simple_quiz.host.complete') }}</button>
        <button v-if="stateStatus === 'paused'" type="button" class="control-btn primary" :disabled="actionBusy" @click="doResume">{{ $t('simple_quiz.host.resume') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppIcon from '../components/AppIcon.vue'
import { quizWS } from '../services/websocket'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { simpleQuizApi } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'

const route = useRoute()
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
const participantOptions = ref([])
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
const actionBusy = ref(false)

const currentQuestion = computed(() => questions.value[currentIndex.value] || null)
const currentImage = computed(() => currentQuestion.value?.imageId ? getImageUrl(currentQuestion.value.imageId) : null)
const revealedAnswer = computed(() => {
  if (!allowReveal.value || !revealed.value[currentIndex.value]) return ''
  return currentQuestion.value?.correctAnswer || ''
})

const judgeUrl = computed(() => judgeToken.value ? `${window.location.origin}/simple-quiz/${eventId}/${moduleId}/judge?token=${encodeURIComponent(judgeToken.value)}` : '')

let deadlineTimer = null
let refreshSeq = 0
const tick = () => {
  if (!deadlineUnix.value) {
    deadlineLeft.value = null
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
  questions.value = (s.questions || []).map(q => ({ ...q, text: q.text, correctAnswer: q.correctAnswer || '' }))
  progress.value = s.progress || []
  gradingComplete.value = !!s.gradingComplete
  leaderboard.value = s.working || []
  revealed.value = s.revealed || {}
  deadlineUnix.value = s.deadlineUnix || 0
  allowReveal.value = s.allowReveal !== false
  tick()
}

const refresh = async () => {
  const reqSeq = ++refreshSeq
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'host' })
    if (stopped || reqSeq !== refreshSeq) return
    applyState(s)
    error.value = ''
  } catch (e) {
    if (!stopped && reqSeq === refreshSeq) error.value = e.message
  }
}

const wrap = (fn, confirmMsg = '') => async () => {
  if (actionBusy.value) return
  if (confirmMsg && !confirm(confirmMsg)) return
  actionBusy.value = true
  try {
    error.value = ''
    const res = await fn()
    if (res?.judgeToken) judgeToken.value = res.judgeToken
    await refresh()
  } catch (e) {
    error.value = e.message
  } finally {
    actionBusy.value = false
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

const linkPlayer = async (playerId, event) => {
	try {
		await simpleQuizApi.linkParticipant(eventId, moduleId, playerId, event.target.value)
		await refresh()
	} catch (e) {
		error.value = e.message
		const player = playerList.value.find(p => p.playerId === playerId)
		event.target.value = player?.participantId || ''
	}
}

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
	try {
		const response = await fetch(`${API_BASE_URL}/events/${eventId}/participants`, { headers: getAuthHeaders() })
		if (response.ok && !stopped) participantOptions.value = await response.json()
	} catch { /* identity linking is optional */ }
	if (stopped) return
  try {
    const modRes = await fetch(`${API_BASE_URL}/events/${eventId}/modules/${moduleId}`, { headers: getAuthHeaders() })
    if (modRes.ok && !stopped) {
      const mod = await modRes.json()
      if (!stopped && mod?.data?.judgeToken && !judgeToken.value) {
        judgeToken.value = mod.data.judgeToken
      }
    }
  } catch { /* judge token load is best-effort */ }
  if (stopped) return
  await refresh()
  if (stopped) return
  quizWS.on('simple:state', refresh)
  quizWS.connect(moduleId, '', true, false, eventId, { type: 'simple:host_join', payload: { moduleId } })
  deadlineTimer = setInterval(tick, 1000)
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
.lobby-actions { flex-wrap: wrap; gap: 10px; }
.judge-url { font-size: 11px; color: #0A84FF; word-break: break-all; overflow-wrap: anywhere; }
.reveal-card { background: rgba(50,215,75,0.15); border: 1px solid #32D74B; border-radius: 8px; padding: 12px; margin-top: 12px; font-weight: 700; }
.grading-card { background: rgba(0,0,0,0.3); border-radius: 8px; padding: 12px; margin-top: 12px; }
.grading-card h4 { margin: 0 0 8px; }
.prog-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-size: 13px; padding: 4px 0; }
.reveal-hint { text-align: center; color: #8E8E93; font-size: 12px; }

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .grading-card) {
  background: var(--admin-surface) !important;
  border: 1px solid var(--admin-border) !important;
  color: var(--admin-text) !important;
  box-shadow: var(--admin-shadow) !important;
}

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .reveal-hint) {
  color: var(--admin-text-muted) !important;
  opacity: 1;
}

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .reveal-card) {
  background: var(--admin-success-soft) !important;
  border: 1px solid var(--admin-success) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .host-error) {
  background: var(--admin-danger-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-danger) 35%, var(--admin-border)) !important;
  color: var(--admin-danger) !important;
  padding: 10px 14px;
  border-radius: 8px;
}

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .judge-url) {
  color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .simple-quiz-host-screen .prog-row select) {
  border: 1px solid var(--admin-border-strong) !important;
  border-radius: 8px;
  padding: 6px 10px;
}

@media (max-width: 900px) {
  .quiz-host-screen,
  .state-container,
  .question-container {
    width: 100%;
    min-height: 100vh;
    min-height: 100dvh;
    height: auto;
    overflow-y: auto;
    overflow-x: clip;
    box-sizing: border-box;
    padding-bottom: calc(88px + env(safe-area-inset-bottom, 0px));
  }

  .answer-counter-widget {
    position: static;
    align-self: stretch;
  }
}

@media (max-width: 640px) {
  .lobby-footer {
    width: 100%;
  }

  .start-btn {
    width: 100%;
    min-block-size: 44px;
  }

  .prog-row:has(select) {
    flex-direction: column;
    align-items: stretch;
    gap: 6px;
  }

  .prog-row select {
    width: 100%;
    min-block-size: 44px;
    font-size: 16px;
    box-sizing: border-box;
  }

  .host-controls {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    width: 100%;
    padding: 12px 0;
    box-sizing: border-box;
  }

  .control-btn {
    min-block-size: 44px;
    width: 100%;
    padding: 10px 12px;
  }

  .control-btn.primary {
    grid-column: 1 / -1;
  }
}

@media (pointer: coarse) {
  .copy-link-btn,
  .start-btn,
  .control-btn,
  .prog-row select {
    min-block-size: 44px;
  }
}
</style>
