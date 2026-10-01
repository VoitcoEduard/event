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

    <!-- Nickname entry -->
    <div v-if="!identity" class="state-container name-entry-container">
      <h1 class="logo">{{ $t('simple_quiz.player.logo') }}</h1>
      <div class="card">
        <input
          v-model="nickname"
          :placeholder="$t('simple_quiz.player.nickname_placeholder')"
          class="nickname-input"
          maxlength="40"
          @keyup.enter="join"
        />
        <button class="join-btn" @click="join" :disabled="!nickname.trim() || joining">
          {{ $t('simple_quiz.player.join_btn') }}
        </button>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- Lobby -->
    <div v-else-if="status === 'lobby'" class="state-container lobby-container">
      <div class="lobby-content">
        <h2 class="waiting-text">{{ $t('simple_quiz.player.you_are_in') }}</h2>
        <div class="nickname-display">{{ identity.nickname }}</div>
        <p class="status-msg">{{ $t('simple_quiz.player.see_name') }}</p>
        <div class="loader"></div>
      </div>
    </div>

    <!-- Question open -->
    <div v-else-if="status === 'question_open'" class="state-container sq-question-container">
      <div class="sq-q-header">
        <div class="q-number">{{ currentIndex + 1 }} / {{ total }}</div>
        <div v-if="deadlineLeft !== null" class="q-timer" :class="{ low: deadlineLeft <= 10 }">{{ deadlineLeft }}s</div>
      </div>
      <h2 v-if="question?.text" class="sq-q-text">{{ question.text }}</h2>
      <img v-if="questionImage" :src="questionImage" class="sq-q-image" alt="" />
      <div class="sq-points">{{ question?.points }} {{ $t('simple_quiz.player.points') }}</div>

      <div v-if="submitted" class="sq-locked">
        <AppIcon class="sq-check" name="check" :size="42" />
        <p>{{ $t('simple_quiz.player.submitted') }}</p>
        <p class="sq-sub">{{ $t('simple_quiz.player.waiting_others') }}</p>
      </div>
      <div v-else class="sq-answer-box">
        <textarea
          v-model="draft"
          :placeholder="$t('simple_quiz.player.type_answer')"
          maxlength="2000"
          rows="5"
          class="sq-textarea"
          @input="onDraftInput"
        ></textarea>
        <div class="sq-counter">{{ draft.length }}/2000</div>
        <button class="sq-submit" :disabled="!draft.trim() || submitting" @click="submit">
          {{ submitting ? '…' : $t('simple_quiz.player.submit') }}
        </button>
        <p v-if="error" class="error-text">{{ error }}</p>
      </div>
    </div>

    <!-- Locked / between -->
    <div v-else-if="status === 'answers_locked' || status === 'between_questions'" class="state-container answered-container">
      <div class="answered-content">
        <div v-if="question?.revealed && question?.correctAnswer" class="sq-reveal">
          <h3>{{ $t('simple_quiz.player.revealed_answer') }}</h3>
          <p class="sq-reveal-text">{{ question.correctAnswer }}</p>
        </div>
        <div v-else class="spinner"></div>
        <h2>{{ submitted ? $t('simple_quiz.player.submitted') : $t('simple_quiz.player.waiting_next') }}</h2>
        <p>{{ $t('simple_quiz.player.waiting_others') }}</p>
      </div>
    </div>

    <!-- Waiting grading -->
    <div v-else-if="status === 'waiting_grading' || status === 'answering_finished'" class="state-container lobby-container">
      <div class="lobby-content">
        <h2 class="waiting-text">{{ $t('simple_quiz.player.waiting_grading') }}</h2>
        <div class="loader"></div>
      </div>
    </div>

    <!-- Published -->
    <div v-else-if="status === 'results_published' || status === 'completed'" class="state-container sq-results">
      <div class="sq-results-card">
        <h2>{{ $t('simple_quiz.player.published') }}</h2>
        <div v-if="me" class="sq-me">
          <div class="sq-me-name">{{ me.nickname }}</div>
          <div class="sq-me-score">{{ me.totalScore }} {{ $t('simple_quiz.player.points') }}</div>
        </div>
        <div class="sq-table">
          <div v-for="(row, qi) in me?.perQuestion || []" :key="qi" class="sq-row">
            <span class="sq-qnum">Q{{ qi + 1 }}</span>
            <span class="sq-verdict" :class="row.verdict">{{ verdictLabel(row.verdict) }}</span>
            <span class="sq-pts">+{{ row.pointsEarned }}</span>
          </div>
        </div>
        <h3>{{ $t('simple_quiz.player.leaderboard') }}</h3>
        <div class="sq-table">
          <div v-for="(p, i) in leaderboard" :key="p.playerId" class="sq-row" :class="{ mine: p.playerId === identity?.playerId }">
            <span class="sq-qnum">{{ i + 1 }}</span>
            <span class="sq-name">{{ p.nickname }}</span>
            <span class="sq-pts">{{ p.totalScore }}</span>
          </div>
        </div>
        <div v-if="revealedAnswers.length" class="sq-revealed-list">
          <div v-for="r in revealedAnswers" :key="r.index" class="sq-rev">
            <strong>Q{{ r.index + 1 }}:</strong> {{ r.text }}
          </div>
        </div>
      </div>
    </div>

    <div v-else class="state-container lobby-container">
      <div class="lobby-content"><div class="loader"></div></div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { quizWS } from '../services/websocket'
import { simpleQuizApi, loadDraft, saveDraft, clearDraft, newAnswerId } from '../services/simpleQuiz'
import { getImageUrl } from '../services/media'
import AppIcon from '../components/AppIcon.vue'

const { locale, t } = useI18n()
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId

const changeLanguage = () => localStorage.setItem('userLanguage', locale.value)

const nickname = ref('')
const joining = ref(false)
const submitting = ref(false)
const error = ref('')
const identity = ref(null) // {playerId, secret, nickname}
const status = ref('lobby')
const currentIndex = ref(0)
const total = ref(0)
const question = ref(null)
const deadlineUnix = ref(0)
const deadlineLeft = ref(null)
const submitted = ref(false)
const draft = ref('')
const me = ref(null)
const leaderboard = ref([])
const revealedAnswers = ref([])

const questionImage = computed(() => question.value?.imageId ? getImageUrl(question.value.imageId) : null)

const storageKey = computed(() => `sq-identity:${eventId}:${moduleId}`)

let pollTimer = null
let deadlineTimer = null

const verdictLabel = (v) => {
  if (v === 'right') return t('simple_quiz.player.right')
  if (v === 'wrong') return t('simple_quiz.player.wrong')
  return t('simple_quiz.player.missing')
}

const tickDeadline = () => {
  if (!deadlineUnix.value) {
    deadlineLeft.value = null
    return
  }
  const left = Math.max(0, deadlineUnix.value - Math.floor(Date.now() / 1000))
  deadlineLeft.value = left
}

const onDraftInput = () => {
  saveDraft(eventId, moduleId, currentIndex.value, draft.value)
}

const join = async () => {
  const nick = nickname.value.trim()
  if (!nick) return
  joining.value = true
  error.value = ''
  try {
    const res = await simpleQuizApi.join(eventId, moduleId, nick)
    identity.value = { playerId: res.playerId, secret: res.secret, nickname: res.nickname }
    try { localStorage.setItem(storageKey.value, JSON.stringify(identity.value)) } catch { /* ignore */ }
    await refresh()
    quizWS.connect(moduleId, nick, false)
    quizWS.send('simple:player_join', { moduleId, playerId: res.playerId })
  } catch (e) {
    error.value = e.message
  } finally {
    joining.value = false
  }
}

const tryReconnect = async () => {
  let saved = null
  try { saved = JSON.parse(localStorage.getItem(storageKey.value) || 'null') } catch { saved = null }
  if (!saved?.playerId || !saved?.secret) return false
  try {
    const res = await simpleQuizApi.join(eventId, moduleId, saved.nickname, saved.secret)
    identity.value = { playerId: res.playerId, secret: res.secret, nickname: res.nickname }
    nickname.value = res.nickname
    return true
  } catch {
    try { localStorage.removeItem(storageKey.value) } catch { /* ignore */ }
    return false
  }
}

const refresh = async () => {
  if (!identity.value) return
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'player', playerId: identity.value.playerId })
    status.value = s.status || 'lobby'
    currentIndex.value = s.currentIndex || 0
    total.value = s.total || 0
    question.value = s.question || null
    deadlineUnix.value = s.deadlineUnix || 0
    submitted.value = !!s.submitted
    tickDeadline()
    if (!submitted.value && status.value === 'question_open') {
      const d = loadDraft(eventId, moduleId, currentIndex.value)
      if (d && !draft.value) draft.value = d
    }
    if (s.status === 'results_published' || s.status === 'completed') {
      await refreshResults()
    }
  } catch (e) {
    error.value = e.message
  }
}

const refreshResults = async () => {
  try {
    const r = await simpleQuizApi.results(eventId, moduleId, { view: 'player', playerId: identity.value.playerId })
    leaderboard.value = r.published?.players || []
    me.value = r.me || leaderboard.value.find(p => p.playerId === identity.value.playerId) || null
    const mine = r.myAnswers || []
    void mine
    // Revealed answers visible in state questions; fetch projector view for texts.
    const pub = await simpleQuizApi.state(eventId, moduleId, { view: 'projector' })
    revealedAnswers.value = (pub.questions || [])
      .map((q, index) => ({ index, text: q.correctAnswer || '' }))
      .filter(r => r.text)
  } catch (e) {
    error.value = e.message
  }
}

const submit = async () => {
  const text = draft.value.trim()
  if (!text || submitting.value || submitted.value) return
  submitting.value = true
  error.value = ''
  const answerId = newAnswerId()
  try {
    await simpleQuizApi.submitAnswer(eventId, moduleId, {
      playerId: identity.value.playerId,
      secret: identity.value.secret,
      questionIndex: currentIndex.value,
      text,
      answerId
    })
    clearDraft(eventId, moduleId, currentIndex.value)
    draft.value = ''
    submitted.value = true
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}

watch(currentIndex, () => {
  draft.value = ''
  submitted.value = false
  if (status.value === 'question_open') {
    draft.value = loadDraft(eventId, moduleId, currentIndex.value)
  }
})

onMounted(async () => {
  const ok = await tryReconnect()
  if (ok) {
    await refresh()
    quizWS.connect(moduleId, identity.value.nickname, false)
    quizWS.send('simple:player_join', { moduleId, playerId: identity.value.playerId })
  }
  quizWS.on('simple:state', refresh)
  pollTimer = setInterval(refresh, 4000)
  deadlineTimer = setInterval(tickDeadline, 1000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (deadlineTimer) clearInterval(deadlineTimer)
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped>
.quiz-player-screen { width: 100vw; height: 100vh; height: 100dvh; background-color: #46178F; color: white; display: flex; flex-direction: column; overflow: hidden; position: relative; font-family: 'Inter', sans-serif; }
.bg-layer { position: absolute; inset: 0; background: radial-gradient(circle at center, rgba(0,0,0,0) 0%, rgba(0,0,0,0.2) 100%); z-index: 0; }
.language-switcher-container { position: absolute; top: 10px; right: 10px; z-index: 100; }
.lang-select { background: rgba(255,255,255,0.1); color: white; border: 1px solid rgba(255,255,255,0.2); padding: 6px 10px; border-radius: 8px; font-size: 12px; }
.state-container { z-index: 1; flex: 1; display: flex; flex-direction: column; align-items: center; width: 100%; overflow-y: auto; padding: 56px 16px 24px; }
.name-entry-container, .lobby-container { justify-content: center; text-align: center; }
.logo { font-size: 3rem; font-weight: 900; margin-bottom: 24px; }
.card { background: white; padding: 20px; border-radius: 8px; width: 100%; max-width: 400px; display: flex; flex-direction: column; gap: 10px; }
.nickname-input { padding: 15px; border: 2px solid #ccc; border-radius: 4px; font-size: 1.2rem; text-align: center; font-weight: bold; color: #111; }
.join-btn { background-color: #333; color: white; padding: 15px; border: none; border-radius: 4px; font-size: 1.2rem; font-weight: bold; cursor: pointer; }
.join-btn:disabled { opacity: 0.5; }
.error-text { color: #ffb3b3; background: rgba(255,255,255,0.15); padding: 8px; border-radius: 6px; font-size: 13px; }
.lobby-content { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.nickname-display { font-size: 2.2rem; font-weight: 900; }
.loader, .spinner { width: 44px; height: 44px; border: 4px solid rgba(255,255,255,0.3); border-top: 4px solid white; border-radius: 50%; animation: spin 1s linear infinite; margin: 16px auto; }
@keyframes spin { to { transform: rotate(360deg); } }
.sq-question-container { align-items: stretch; max-width: 640px; margin: 0 auto; gap: 12px; }
.sq-q-header { display: flex; justify-content: space-between; align-items: center; }
.q-number { background: #fff; color: #111; border-radius: 20px; padding: 6px 14px; font-weight: 800; }
.q-timer { background: #000; border-radius: 20px; padding: 6px 14px; font-weight: 800; }
.q-timer.low { background: #c00; }
.sq-q-text { font-size: 1.5rem; font-weight: 800; text-align: center; }
.sq-q-image { max-width: 100%; max-height: 240px; object-fit: contain; border-radius: 12px; margin: 0 auto; }
.sq-points { text-align: center; opacity: 0.85; font-weight: 700; }
.sq-answer-box { display: flex; flex-direction: column; gap: 8px; }
.sq-textarea { width: 100%; border-radius: 12px; padding: 14px; font-size: 1.05rem; border: 2px solid rgba(255,255,255,0.25); background: rgba(255,255,255,0.12); color: #fff; resize: vertical; min-height: 120px; }
.sq-textarea::placeholder { color: rgba(255,255,255,0.6); }
.sq-counter { text-align: right; font-size: 12px; opacity: 0.7; }
.sq-submit { background: #FFD700; color: #111; font-weight: 900; font-size: 1.2rem; padding: 14px; border-radius: 30px; border: none; cursor: pointer; }
.sq-submit:disabled { opacity: 0.5; }
.sq-locked { text-align: center; background: rgba(0,0,0,0.25); border-radius: 16px; padding: 24px; }
.sq-check { font-size: 3rem; }
.answered-container { justify-content: center; text-align: center; }
.sq-reveal { background: rgba(0,0,0,0.3); padding: 16px; border-radius: 12px; margin-bottom: 12px; }
.sq-reveal-text { font-size: 1.2rem; font-weight: 800; }
.sq-results { align-items: stretch; }
.sq-results-card { background: #1C1C1E; border-radius: 16px; padding: 20px; max-width: 640px; width: 100%; margin: 0 auto; }
.sq-me { display: flex; justify-content: space-between; font-size: 1.2rem; font-weight: 800; margin: 12px 0; }
.sq-table { display: flex; flex-direction: column; gap: 6px; margin: 12px 0; }
.sq-row { display: flex; gap: 10px; align-items: center; background: #2C2C2E; padding: 10px 12px; border-radius: 8px; }
.sq-row.mine { border: 1px solid #FFD700; }
.sq-qnum { font-weight: 800; min-width: 36px; }
.sq-verdict.right { color: #32D74B; font-weight: 800; }
.sq-verdict.wrong { color: #FF6B6B; font-weight: 800; }
.sq-verdict.missing { color: #8E8E93; }
.sq-pts { margin-left: auto; font-weight: 800; }
.sq-name { flex: 1; text-align: left; }
.sq-revealed-list { margin-top: 12px; display: flex; flex-direction: column; gap: 6px; font-size: 14px; }
</style>
