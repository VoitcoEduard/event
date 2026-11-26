<template>
  <div ref="quizEditorRef" class="quiz-editor" :class="{ 'quiz-editor--page-flow': pageFlow, 'quiz-editor--with-outline': showOutline && questions.length > 0 }">
    <aside
      v-if="showOutline && questions.length > 0"
      class="quiz-outline-sidebar"
      :aria-label="$t('quiz_editor.outline_title')"
    >
      <div class="outline-header">
        <span class="outline-title">{{ $t('quiz_editor.outline_title') }}</span>
        <span class="outline-count">{{ questions.length }}</span>
      </div>
      <div class="outline-list" role="list">
        <div
          v-for="(question, index) in questions"
          :key="question.id || index"
          class="outline-item"
          :class="{
            'is-active': activeIndex === index,
            'has-error': hasQuestionErrors(index)
          }"
          role="listitem"
        >
          <button
            type="button"
            class="outline-select-btn"
            :aria-current="activeIndex === index ? 'true' : undefined"
            @click="selectAndScrollToQuestion(index)"
          >
            <span class="outline-num">{{ $t('quiz_editor.question_short', { number: index + 1 }) }}</span>
            <AppIcon class="outline-type-icon" :name="getTypeIcon(question.type)" :size="15" aria-hidden="true" />
            <span class="outline-text">{{ getOutlineTitle(question) }}</span>
            <span
              v-if="hasQuestionErrors(index)"
              class="outline-error-dot"
              :title="$t('quiz_editor.has_errors')"
              :aria-label="$t('quiz_editor.has_errors')"
            ></span>
          </button>
          <div class="outline-reorder">
            <button
              type="button"
              class="outline-move-btn"
              :disabled="index === 0"
              :title="$t('quiz_editor.move_up')"
              :aria-label="$t('quiz_editor.move_up')"
              @click.stop="emit('move-question', index, -1)"
            >
              <AppIcon name="chevronUp" :size="13" aria-hidden="true" />
            </button>
            <button
              type="button"
              class="outline-move-btn"
              :disabled="index === questions.length - 1"
              :title="$t('quiz_editor.move_down')"
              :aria-label="$t('quiz_editor.move_down')"
              @click.stop="emit('move-question', index, 1)"
            >
              <AppIcon name="chevronDown" :size="13" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </aside>

    <div class="scroll-content">
      <div v-if="questions.length === 0" class="empty-state">
        <AppIcon class="empty-icon" name="quiz" :size="42" aria-hidden="true" />
        <p class="empty-text">{{ $t('quiz_editor.empty_text') }}</p>
      </div>
      
      <div v-else class="questions-list">
        <div
          v-for="(question, index) in questions"
          :key="question.id"
          :ref="(el) => setQuestionEl(index, el)"
          :data-question-index="index"
          class="question-wrapper-outer"
          :class="{ 'is-active': activeIndex === index, 'has-error': hasQuestionErrors(index) }"
          @click="handleQuestionInteract(index)"
          @focusin="handleQuestionInteract(index)"
        >
           <div class="question-sort-controls">
              <button
                type="button"
                @click.stop="emit('move-question', index, -1)"
                :disabled="index === 0"
                class="move-btn"
                :title="$t('quiz_editor.move_up')"
                :aria-label="$t('quiz_editor.move_up')"
              >↑</button>
              <button
                type="button"
                @click.stop="emit('move-question', index, 1)"
                :disabled="index === questions.length - 1"
                class="move-btn"
                :title="$t('quiz_editor.move_down')"
                :aria-label="$t('quiz_editor.move_down')"
              >↓</button>
           </div>
           <div class="question-component-wrapper">
              <QuizQuestionEditor 
                 :ref="(el) => setQuestionEditorRef(index, el)"
                 :question="question" 
                 :index="index"
                 :event-id="eventId"
                 :module-id="moduleId"
                 :locked-type="lockedType"
                 :errors="errorsByIndex?.[index] || null"
                 :active="activeIndex === index"
                 @update="(updatedQ) => emit('update-question', index, updatedQ)"
                 @delete="emit('remove-question', index)"
                 @change-type="handleChangeType(index)"
              />
            </div>
        </div>
      </div>
      
      <div class="add-section">
        <button
          v-if="lockedType"
          type="button"
          class="add-btn add-classic-btn"
          @click="handleAddLocked"
        >
          {{ $t('quiz_editor.add_question') }}
        </button>
        <div v-else class="quick-add-toolbar">
          <button
            type="button"
            class="add-classic-btn"
            @click="handleAddClassic"
          >
            {{ $t('quiz_editor.add_classic') }}
          </button>
          <button
            type="button"
            class="add-btn add-other-btn"
            @click="handleAddOtherClick"
          >
            {{ $t('quiz_editor.add_other_type') }}
          </button>
        </div>
      </div>

      <QuestionTypeSelector 
        v-if="!lockedType"
        :visible="showTypeSelector"
        @close="showTypeSelector = false"
        @select="handleTypeSelect"
      />

      <div v-if="!pageFlow" class="modal-bottom-spacer"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import QuizQuestionEditor from './QuizQuestionEditor.vue'
import QuestionTypeSelector from './QuestionTypeSelector.vue'
import AppIcon from './AppIcon.vue'
import { hasActiveBlockingModal, isPasteTargetInContainer } from '../utils/clipboard'

const TYPE_ICONS = {
  classic: 'quiz',
  multiple_choice: 'quiz',
  single_select: 'quiz',
  multi_select: 'listCheck',
  true_false: 'check',
  range: 'sliders',
  number: 'hash',
  correct_picture: 'image',
  wrong_pictures: 'search',
  simple_text: 'form'
}

const { t } = useI18n()

const props = defineProps({
  questions: { type: Array, default: () => [] },
  eventId: { type: String, default: '' },
  moduleId: { type: String, default: '' },
  lockedType: { type: String, default: '' },
  activeIndex: { type: Number, default: -1 },
  errorsByIndex: { type: Object, default: () => ({}) },
  showOutline: { type: Boolean, default: false },
  pageFlow: { type: Boolean, default: false }
})

const emit = defineEmits([
  'update-question',
  'remove-question',
  'add-question',
  'move-question',
  'update:activeIndex',
  'select-question'
])

const quizEditorRef = ref(null)
const showTypeSelector = ref(false)
const editingQuestionIndex = ref(null)
const lastInteractedIndex = ref(-1)
const questionEls = new Map()
const questionEditorRefs = new Map()
let selfSelectedIndex = -1

const setQuestionEditorRef = (index, el) => {
  if (el) {
    questionEditorRefs.set(index, el)
  } else {
    questionEditorRefs.delete(index)
  }
}

const effectiveActiveIndex = computed(() => {
  if (props.activeIndex >= 0 && props.activeIndex < props.questions.length) {
    return props.activeIndex
  }
  if (lastInteractedIndex.value >= 0 && lastInteractedIndex.value < props.questions.length) {
    return lastInteractedIndex.value
  }
  return props.questions.length === 1 ? 0 : -1
})

const handleWindowPaste = (event) => {
  if (event?.defaultPrevented || showTypeSelector.value) return
  const targetIndex = effectiveActiveIndex.value
  if (targetIndex < 0) return
  if (hasActiveBlockingModal(quizEditorRef.value)) return
  if (!isPasteTargetInContainer(event?.target, quizEditorRef.value)) return

  questionEditorRefs.get(targetIndex)?.handleClipboardPasteEvent?.(event)
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    window.addEventListener('paste', handleWindowPaste)
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('paste', handleWindowPaste)
  }
})

const setQuestionEl = (index, el) => {
  if (el) {
    questionEls.set(index, el)
  } else {
    questionEls.delete(index)
  }
}

const getTypeIcon = (type) => TYPE_ICONS[type] || 'quiz'

const getOutlineTitle = (question) => {
  const text = String(question?.text || '').trim()
  return text || t('quiz_editor.untitled_question')
}

const hasQuestionErrors = (index) => {
  const errMap = props.errorsByIndex?.[index]
  return Boolean(errMap && Object.keys(errMap).length > 0)
}

const scrollToQuestion = (index) => {
  if (typeof index !== 'number' || index < 0) return
  nextTick(() => {
    const el = questionEls.get(index)
    if (el && typeof el.scrollIntoView === 'function') {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  })
}

const selectAndScrollToQuestion = (index) => {
  selfSelectedIndex = -1
  lastInteractedIndex.value = index
  emit('update:activeIndex', index)
  emit('select-question', index)
  scrollToQuestion(index)
}

const handleQuestionInteract = (index) => {
  lastInteractedIndex.value = index
  if (props.activeIndex === index) return
  selfSelectedIndex = index
  emit('update:activeIndex', index)
  emit('select-question', index)
}

watch(() => props.activeIndex, (newIndex) => {
  if (newIndex === selfSelectedIndex) {
    selfSelectedIndex = -1
    return
  }
  selfSelectedIndex = -1
  if (typeof newIndex === 'number' && newIndex >= 0 && newIndex < props.questions.length) {
    scrollToQuestion(newIndex)
  }
})

const handleAddLocked = () => {
  if (!props.lockedType) return
  emit('add-question', props.lockedType)
}

const handleAddClassic = () => {
  emit('add-question', 'classic')
}

const handleAddOtherClick = () => {
  if (props.lockedType) {
    emit('add-question', props.lockedType)
    return
  }
  editingQuestionIndex.value = null
  showTypeSelector.value = true
}

const handleChangeType = (index) => {
  if (props.lockedType) return
  editingQuestionIndex.value = index
  showTypeSelector.value = true
}

const handleTypeSelect = (type) => {
  showTypeSelector.value = false
  if (editingQuestionIndex.value !== null) {
    const question = { ...props.questions[editingQuestionIndex.value], type }
    emit('update-question', editingQuestionIndex.value, question)
  } else {
    emit('add-question', type)
  }
}

defineExpose({
  scrollToQuestion
})
</script>

<style scoped>
.quiz-editor {
  container: quiz-editor / inline-size;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  min-inline-size: 0;
}

.quiz-editor--with-outline {
  flex-direction: row;
  align-items: stretch;
  gap: 20px;
}

.quiz-editor--page-flow {
  height: auto;
  overflow: visible;
}

.quiz-outline-sidebar {
  width: 260px;
  max-width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  flex-shrink: 0;
  background-color: var(--admin-surface, #1C1C1E);
  border: 1px solid var(--admin-border, #3A3A3C);
  border-radius: 12px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-self: flex-start;
  max-height: calc(100vh - 140px);
  overflow-y: auto;
  position: sticky;
  top: 76px;
}

.outline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 4px;
}

.outline-title {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--admin-text-muted, #8E8E93);
}

.outline-count {
  font-size: 12px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--admin-surface-subtle, #2C2C2E);
  color: var(--admin-text-muted, #E5E5EA);
}

.outline-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-inline-size: 0;
  max-width: 100%;
}

.outline-item {
  display: flex;
  align-items: center;
  gap: 4px;
  box-sizing: border-box;
  border: 1px solid var(--admin-border, #3A3A3C);
  border-radius: 8px;
  background: var(--admin-surface-subtle, #2C2C2E);
  padding: 4px;
  transition: border-color 0.15s, background-color 0.15s;
}

.outline-item.is-active {
  border-color: var(--admin-primary, #0A84FF);
  background: var(--admin-primary-soft, rgba(10, 132, 255, 0.14));
}

.outline-item.has-error {
  border-color: var(--admin-danger, #FF453A);
}

.outline-select-btn {
  flex: 1;
  min-inline-size: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  padding: 6px;
  color: var(--admin-text, #FFFFFF);
  font: inherit;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}

.outline-num {
  font-weight: 700;
  font-size: 12px;
  color: var(--admin-text-muted, #8E8E93);
  flex-shrink: 0;
}

.outline-item.is-active .outline-num {
  color: var(--admin-primary, #0A84FF);
}

.outline-type-icon {
  color: var(--admin-text-muted, #8E8E93);
  flex-shrink: 0;
}

.outline-text {
  flex: 1;
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.outline-error-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--admin-danger, #FF453A);
  flex-shrink: 0;
}

.outline-reorder {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex-shrink: 0;
}

.outline-move-btn {
  width: 22px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--admin-border, #3A3A3C);
  border-radius: 4px;
  background: var(--admin-surface, #1C1C1E);
  color: var(--admin-text-muted, #E5E5EA);
  cursor: pointer;
  padding: 0;
}

.outline-move-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.scroll-content {
  flex: 1;
  min-inline-size: 0;
  overflow-y: auto;
  padding: 20px;
  box-sizing: border-box;
}

.quiz-editor--page-flow .scroll-content {
  overflow-y: visible;
  padding: 0 0 24px;
}

.modal-bottom-spacer {
  height: 100px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--admin-text-muted, #8E8E93);
}
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-text { font-size: 14px; }

.questions-list {
  display: flex;
  flex-direction: column;
}

.question-wrapper-outer {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
  scroll-margin-top: 84px;
}

.question-sort-controls {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  padding-top: 10px;
  gap: 8px;
}

.move-btn {
  background-color: var(--admin-surface-subtle, #3A3A3C);
  color: var(--admin-text, white);
  border: none;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.move-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.move-btn:hover:not(:disabled) { background-color: var(--admin-surface-hover, #48484A); }

.question-component-wrapper {
  flex: 1;
  min-inline-size: 0;
}

.add-section {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.quick-add-toolbar {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}

.add-classic-btn {
  background-color: var(--admin-primary-solid, #0A84FF);
  color: #FFFFFF;
  padding: 12px 24px;
  border-radius: 8px;
  font-weight: 600;
  border: 1px solid var(--admin-primary-solid, #0A84FF);
  cursor: pointer;
  min-block-size: 44px;
}

.add-classic-btn:hover {
  background-color: var(--admin-primary-hover, #0070E0);
}

.add-other-btn {
  background-color: var(--admin-surface-subtle, #2C2C2E);
  color: var(--admin-text, #FFFFFF);
  padding: 12px 20px;
  border-radius: 8px;
  font-weight: 600;
  border: 1px solid var(--admin-border-strong, #48484A);
  cursor: pointer;
  min-block-size: 44px;
}

.add-other-btn:hover {
  background-color: var(--admin-surface-hover, #3A3A3C);
}

:global(html[data-admin-appearance="true"] .quiz-editor .add-classic-btn) {
  background: var(--admin-primary-solid) !important;
  border-color: var(--admin-primary-solid) !important;
  color: #fff !important;
}

:global(html[data-admin-appearance="true"] .quiz-editor .add-classic-btn:hover) {
  background: var(--admin-primary-hover) !important;
}

:global(html[data-admin-appearance="true"] .quiz-editor .add-other-btn) {
  background: var(--admin-surface) !important;
  border: 1px solid var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .quiz-editor .add-other-btn:hover) {
  background: var(--admin-surface-hover) !important;
}

:global(html[data-admin-appearance="true"] .quiz-editor .outline-select-btn) {
  background: transparent !important;
  border: none !important;
  color: var(--admin-text) !important;
}

.quiz-editor::-webkit-scrollbar {
  display: none;
}

@media (max-width: 768px) {
  .quiz-editor--with-outline {
    flex-direction: column;
    gap: 12px;
  }
}

@container quiz-editor (inline-size <= 760px) {
  .quiz-outline-sidebar {
    width: 100%;
    max-width: 100%;
    min-inline-size: 0;
    box-sizing: border-box;
    position: static;
    max-height: none;
    padding: 10px;
  }

  .outline-list {
    flex-direction: row;
    overflow-x: auto;
    max-width: 100%;
    min-inline-size: 0;
    padding-bottom: 4px;
  }

  .outline-item {
    flex: 0 0 auto;
    max-width: min(210px, 100%);
  }

  .outline-reorder {
    flex-direction: row;
  }
}

@container quiz-editor (inline-size <= 580px) {
  .scroll-content {
    padding: 12px;
  }

  .quiz-editor--page-flow .scroll-content {
    padding: 0 0 20px;
  }

  .question-wrapper-outer {
    flex-direction: column;
    gap: 8px;
  }

  .question-sort-controls {
    flex-direction: row;
    justify-content: flex-end;
    padding-top: 0;
  }

  .question-component-wrapper {
    flex: none;
    width: 100%;
    min-inline-size: 0;
  }

  .quick-add-toolbar {
    width: 100%;
  }

  .add-classic-btn,
  .add-other-btn {
    flex: 1 1 140px;
  }
}

@media (pointer: coarse) {
  .move-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .outline-select-btn,
  .add-classic-btn,
  .add-other-btn {
    min-block-size: 44px;
  }

  .outline-move-btn {
    min-block-size: 32px;
    min-inline-size: 32px;
  }
}
</style>
