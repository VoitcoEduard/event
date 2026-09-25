<template>
  <div class="simple-quiz-host">
    <div class="sq-header">
      <h1>Simple Quiz — Host</h1>
      <p>{{ eventId }} / {{ moduleId }}</p>
    </div>
    <div v-if="loading" class="sq-loading">Loading…</div>
    <div v-else-if="error" class="sq-error">{{ error }}</div>
    <div v-else-if="state">
      <p>Status: <strong>{{ state.status }}</strong> · Q {{ state.currentIndex + 1 }}/{{ state.total }} · Players {{ state.players }} · Answered {{ state.answeredCount }}</p>
      <div class="sq-actions">
        <button @click="doLaunch">Launch</button>
        <button @click="doOpen">Open</button>
        <button @click="doLock">Lock</button>
        <button @click="doNext">Next</button>
        <button @click="doReveal">Reveal</button>
        <button @click="doEnd">End answering</button>
        <button @click="doPublish">Publish</button>
        <button @click="doUpdate">Update published</button>
        <button @click="doComplete">Complete</button>
        <button @click="doResume">Resume</button>
      </div>
      <p v-if="judgeToken">Judge: <code>{{ judgeUrl }}</code> <button @click="copyJudge">Copy</button></p>
      <p>Projector: <code>{{ projectorUrl }}</code></p>
      <pre class="sq-pre">{{ JSON.stringify(state, null, 2) }}</pre>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { simpleQuizApi } from '../services/simpleQuiz'
import { quizWS } from '../services/websocket'

const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
const state = ref(null)
const loading = ref(true)
const error = ref('')
const judgeToken = ref('')

const judgeUrl = computed(() => judgeToken.value ? `${window.location.origin}/simple-quiz/${eventId}/${moduleId}/judge?token=${encodeURIComponent(judgeToken.value)}` : '')
const projectorUrl = computed(() => `${window.location.origin}/projector/${eventId}`)

const refresh = async () => {
  try {
    const s = await simpleQuizApi.state(eventId, moduleId, { view: 'host' })
    if (s.status === 'inactive') {
      state.value = s
      return
    }
    state.value = s
    error.value = ''
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const wrap = (fn) => async () => {
  try {
    error.value = ''
    const res = await fn()
    if (res?.judgeToken) judgeToken.value = res.judgeToken
    if (res?.session) state.value = res.session
    else await refresh()
  } catch (e) {
    error.value = e.message
  }
}
const doLaunch = wrap(() => simpleQuizApi.launch(eventId, moduleId))
const doOpen = wrap(() => simpleQuizApi.open(eventId, moduleId))
const doLock = wrap(() => simpleQuizApi.lock(eventId, moduleId))
const doNext = wrap(() => simpleQuizApi.next(eventId, moduleId))
const doReveal = wrap(() => simpleQuizApi.reveal(eventId, moduleId))
const doEnd = wrap(() => {
  if (!confirm('End answering?')) return Promise.resolve()
  return simpleQuizApi.endAnswering(eventId, moduleId)
})
const doPublish = wrap(() => {
  if (!confirm('Publish results?')) return Promise.resolve()
  return simpleQuizApi.publish(eventId, moduleId)
})
const doUpdate = wrap(() => simpleQuizApi.updatePublished(eventId, moduleId))
const doComplete = wrap(() => simpleQuizApi.complete(eventId, moduleId))
const doResume = wrap(() => simpleQuizApi.resume(eventId, moduleId))
const copyJudge = async () => {
  try { await navigator.clipboard.writeText(judgeUrl.value) } catch { /* ignore */ }
}

onMounted(async () => {
  await refresh()
  quizWS.on('simple:state', refresh)
  quizWS.connect(moduleId, '', true)
  quizWS.send('simple:host_join', { moduleId })
})
onUnmounted(() => {
  quizWS.disconnect()
  quizWS.clearHandlers()
})
</script>

<style scoped>
.simple-quiz-host { padding: 24px; color: #fff; background: #1C1C1E; min-height: 100vh; }
.sq-actions { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0; }
.sq-actions button { padding: 8px 12px; border-radius: 8px; border: 1px solid #3A3A3C; background: #2C2C2E; color: #fff; cursor: pointer; }
.sq-pre { background: #000; padding: 12px; border-radius: 8px; font-size: 12px; overflow: auto; }
.sq-error { color: #ff8080; }
</style>
