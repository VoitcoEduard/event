<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container" :style="modalStyle">
        <div class="header">
          <div>
            <p class="title">{{ $t('simple_quiz.config.title') }}</p>
            <p class="subtitle">{{ module?.name || $t('quiz_config.untitled') }}</p>
          </div>
          <div class="header-buttons">
            <button class="close-button" @click="handleClose" aria-label="Close"><AppIcon name="close" :size="18" /></button>
          </div>
        </div>

        <div class="content">
          <QuizEditor
            :questions="questions"
            :event-id="eventId"
            :module-id="module?.id || ''"
            @update-question="updateQuestion"
            @remove-question="removeQuestion"
            @add-question="() => addQuestion('simple_text')"
            @move-question="moveQuestion"
          />

          <div class="settings-bar">
            <label class="reveal-toggle">
              <input type="checkbox" v-model="allowReveal" />
              <span>{{ $t('simple_quiz.config.allow_reveal') }}</span>
            </label>
            <p class="reveal-hint">{{ $t('simple_quiz.config.allow_reveal_hint') }}</p>
          </div>

          <div v-if="judgeToken" class="judge-link-bar">
            <span class="judge-label">{{ $t('simple_quiz.config.judge_link') }}</span>
            <code class="judge-link">{{ judgeUrl }}</code>
            <button class="copy-btn" @click="copyJudgeLink">{{ $t('simple_quiz.config.copy') }}</button>
          </div>

          <div class="action-bar">
            <div class="status-text">
              {{ $t('quiz_config.status', { count: questions.length }) }}
            </div>
            <div class="action-buttons">
              <button class="publish-btn" @click="handleSave" :disabled="loading">
                {{ loading ? $t('quiz_config.saving') : $t('quiz_config.save_quiz') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import QuizEditor from './QuizEditor.vue'

const { t } = useI18n()
const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})
const emit = defineEmits(['close', 'update'])
const { success, error } = useToast()

const questions = ref([])
const allowReveal = ref(true)
const judgeToken = ref('')
const loading = ref(false)

const judgeUrl = computed(() => {
  if (!judgeToken.value) return ''
  return `${window.location.origin}/simple-quiz/${props.eventId}/${props.module?.id}/judge?token=${encodeURIComponent(judgeToken.value)}`
})

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data) {
    const loaded = Array.isArray(mod.data.questions) ? JSON.parse(JSON.stringify(mod.data.questions)) : []
    questions.value = loaded.map(q => ({
      id: q.id || Date.now().toString() + Math.random().toString(36).slice(2, 6),
      type: 'simple_text',
      text: q.text || '',
      timeLimit: q.timeLimit ?? 0,
      explainTime: 0,
      scoringConfig: q.scoringConfig || { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 },
      imageId: q.imageId || null,
      imageAlt: q.imageAlt || '',
      judgeNotes: q.judgeNotes || '',
      correctAnswerText: q.correctAnswerText || ''
    }))
    allowReveal.value = mod.data.allowAnswerReveal !== false
    judgeToken.value = mod.data.judgeToken || ''
  } else if (visible) {
    questions.value = []
    allowReveal.value = true
    judgeToken.value = ''
  }
}, { immediate: true })

const addQuestion = () => {
  questions.value.push({
    id: Date.now().toString(),
    type: 'simple_text',
    text: '',
    timeLimit: 0,
    explainTime: 0,
    scoringConfig: { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 },
    imageId: null,
    imageAlt: '',
    judgeNotes: '',
    correctAnswerText: ''
  })
}

const removeQuestion = (index) => {
  if (!confirm(t('quiz_config.delete_question'))) return
  questions.value.splice(index, 1)
}
const updateQuestion = (index, updatedQ) => {
  questions.value[index] = { ...updatedQ, type: 'simple_text' }
}
const moveQuestion = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= questions.value.length) return
  const tmp = questions.value[index]
  questions.value[index] = questions.value[newIndex]
  questions.value[newIndex] = tmp
}

const validate = () => {
  if (questions.value.length === 0) return t('quiz_config.validation.min_one')
  for (const q of questions.value) {
    if (!String(q.text || '').trim() && !q.imageId) return t('quiz_config.validation.simple_text_required')
    if ((q.scoringConfig?.maxPoints ?? 0) <= 0) return t('quiz_config.validation.simple_points')
    if ((q.timeLimit ?? 0) < 0) return t('quiz_config.validation.simple_points')
  }
  return null
}

const handleSave = async () => {
  const errMsg = validate()
  if (errMsg) {
    error(errMsg)
    return
  }
  loading.value = true
  try {
    const qRes = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/quiz/questions`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(questions.value)
    })
    if (!qRes.ok) throw new Error('questions save failed')
    const modRes = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ data: { questions: questions.value, allowAnswerReveal: allowReveal.value } })
    })
    if (!modRes.ok) throw new Error('module save failed')
    success(t('quiz_config.saved_success'))
    emit('update')
    handleClose()
  } catch (e) {
    console.error(e)
    error(t('quiz_config.save_error'))
  } finally {
    loading.value = false
  }
}

const copyJudgeLink = async () => {
  try {
    await navigator.clipboard.writeText(judgeUrl.value)
    success(t('simple_quiz.config.copied'))
  } catch { /* ignore */ }
}

const handleClose = () => emit('close')
const modalStyle = computed(() => ({
  transform: `translateY(${props.visible ? '0' : '100%'})`,
  opacity: props.visible ? 1 : 0,
  transition: 'all 0.3s ease'
}))
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: flex-end;
  z-index: 1000;
}
.modal-container {
  background-color: #1C1C1E;
  height: 92%;
  width: 100%;
  max-width: 800px;
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  display: flex;
  flex-direction: column;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #2C2C2E;
}
.title { font-size: 22px; font-weight: bold; color: #FFF; }
.subtitle { font-size: 14px; color: #8E8E93; margin-top: 2px; }
.header-buttons { display: flex; gap: 10px; align-items: center; }
.close-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #3A3A3C;
  color: #FFF;
  border: none;
  cursor: pointer;
}
.content { flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative; }
.settings-bar {
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 14px 20px;
}
.reveal-toggle { display: flex; gap: 10px; align-items: center; color: #fff; font-size: 14px; font-weight: 600; }
.reveal-hint { color: #8E8E93; font-size: 12px; margin: 6px 0 0; }
.judge-link-bar {
  display: flex; gap: 10px; align-items: center;
  padding: 12px 20px; background: #232325; border-top: 1px solid #2C2C2E;
}
.judge-label { color: #8E8E93; font-size: 12px; }
.judge-link { color: #0A84FF; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.copy-btn { background: #2C2C2E; color: #fff; border: 1px solid #3A3A3C; border-radius: 6px; padding: 6px 10px; cursor: pointer; font-size: 12px; }
.action-bar {
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 16px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.status-text { color: #8E8E93; }
.action-buttons { display: flex; gap: 12px; }
.publish-btn {
  background-color: #0A84FF;
  color: white;
  padding: 10px 20px;
  border-radius: 8px;
  font-weight: 600;
  border: none;
  cursor: pointer;
}
.publish-btn:disabled { opacity: 0.5; }
</style>
