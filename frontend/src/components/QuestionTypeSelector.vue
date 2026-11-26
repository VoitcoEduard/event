<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div
        ref="modalContainerRef"
        class="modal-container"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('quiz_config.question_editor.selector.title')"
        tabindex="-1"
        @keydown.escape.stop.prevent="handleClose"
      >
        <div class="selector-header">
          <h3>{{ $t('quiz_config.question_editor.selector.title') }}</h3>
          <button type="button" class="close-btn" @click="handleClose" :aria-label="$t('common.close')">
            <AppIcon name="close" :size="18" aria-hidden="true" />
          </button>
        </div>

        <div class="types-grid">
          <button
            v-for="type in questionTypes"
            :key="type.id"
            type="button"
            class="type-card"
            @click="selectType(type.id)"
          >
            <AppIcon class="type-icon-svg" :name="type.icon" :size="36" aria-hidden="true" />
            <span class="type-label">{{ type.label }}</span>
            <span class="type-desc">{{ type.hint }}</span>
          </button>
        </div>

        <div class="selector-footer">
          <button type="button" class="cancel-btn" @click="handleClose">
            {{ $t('quiz_config.question_editor.selector.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'

const props = defineProps({ visible: { type: Boolean, default: false } })
const emit = defineEmits(['close', 'select'])
const { t } = useI18n()
const modalContainerRef = ref(null)
let previouslyFocusedEl = null

const questionTypes = computed(() => [
  { id: 'classic', icon: 'quiz', label: t('quiz_config.question_editor.types.classic'), hint: t('quiz_editor.type_hints.classic') },
  { id: 'multi_select', icon: 'listCheck', label: t('quiz_config.question_editor.types.multi_select'), hint: t('quiz_editor.type_hints.multi_select') },
  { id: 'true_false', icon: 'check', label: t('quiz_config.question_editor.types.true_false'), hint: t('quiz_editor.type_hints.true_false') },
  { id: 'range', icon: 'sliders', label: t('quiz_config.question_editor.types.range'), hint: t('quiz_editor.type_hints.range') },
  { id: 'number', icon: 'hash', label: t('quiz_config.question_editor.types.number'), hint: t('quiz_editor.type_hints.number') },
  { id: 'correct_picture', icon: 'image', label: t('quiz_config.question_editor.types.correct_picture'), hint: t('quiz_editor.type_hints.correct_picture') },
  { id: 'wrong_pictures', icon: 'search', label: t('quiz_config.question_editor.types.wrong_pictures'), hint: t('quiz_editor.type_hints.wrong_pictures') },
  { id: 'simple_text', icon: 'form', label: t('quiz_config.question_editor.types.simple_text'), hint: t('quiz_editor.type_hints.simple_text') }
])

const handleClose = () => emit('close')
const selectType = (typeId) => {
  emit('select', typeId)
  handleClose()
}

const handleWindowKeydown = (event) => {
  if (event.key === 'Escape' && props.visible) {
    event.preventDefault()
    event.stopPropagation()
    event.stopImmediatePropagation()
    handleClose()
  }
}

watch(() => props.visible, (isOpen) => {
  if (typeof window !== 'undefined') {
    if (isOpen) {
      window.addEventListener('keydown', handleWindowKeydown, true)
    } else {
      window.removeEventListener('keydown', handleWindowKeydown, true)
    }
  }
  if (typeof document === 'undefined') return
  if (isOpen) {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      modalContainerRef.value?.focus()
    })
  } else {
    if (previouslyFocusedEl && typeof previouslyFocusedEl.focus === 'function') {
      const target = previouslyFocusedEl
      previouslyFocusedEl = null
      nextTick(() => {
        if (target.isConnected) target.focus()
      })
    }
  }
}, { immediate: true })

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleWindowKeydown, true)
  }
})
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, .74); display: flex; justify-content: center; align-items: center; z-index: 3000; backdrop-filter: blur(5px); overscroll-behavior: contain; }
.modal-container { background: var(--ops-surface); width: 95%; max-width: 800px; border-radius: var(--ops-radius-lg); overflow: hidden; border: 1px solid var(--ops-border); display: flex; flex-direction: column; box-shadow: var(--ops-shadow); overscroll-behavior: contain; }
.modal-container:focus { outline: none; }
.selector-header { padding: 24px 28px; display: flex; justify-content: space-between; align-items: center; gap: 12px; border-bottom: 1px solid var(--ops-border); }
.selector-header h3 { margin: 0; color: var(--ops-text); font-size: 1.35rem; font-weight: 800; letter-spacing: -.02em; }
.close-btn { background: var(--ops-surface-raised); border: 1px solid var(--ops-border); color: var(--ops-text); width: 36px; height: 36px; border-radius: var(--ops-radius-sm); display: flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0; }
.close-btn:hover { background: var(--ops-surface-hover); }
.types-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; padding: 24px 28px 8px; }
.type-card { width: 100%; min-inline-size: 0; font: inherit; text-align: center; background: var(--ops-surface-raised); border-radius: var(--ops-radius-md); padding: 18px 12px; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; gap: 8px; cursor: pointer; transition: .2s ease; border: 1px solid var(--ops-border); color: var(--ops-text-muted); }
.type-card:hover { border-color: var(--ops-blue); color: var(--ops-blue); transform: translateY(-2px); box-shadow: 0 8px 20px rgba(20, 127, 245, .15); }
.type-card:focus-visible { outline: 3px solid var(--ops-blue); outline-offset: 2px; }
.type-label { color: var(--ops-text); font-size: .92rem; font-weight: 700; text-align: center; overflow-wrap: break-word; }
.type-desc { color: var(--ops-text-muted); font-size: .78rem; line-height: 1.35; text-align: center; overflow-wrap: break-word; }
.selector-footer { padding: 16px 28px 24px; display: flex; justify-content: flex-end; }
.cancel-btn { background: transparent; border: 1px solid var(--ops-border); color: var(--ops-text-muted); border-radius: var(--ops-radius-sm); padding: 10px 16px; font-weight: 700; cursor: pointer; }
.cancel-btn:hover { color: var(--ops-text); border-color: var(--ops-text-muted); }

@media (max-width: 640px) {
  .modal-container {
    width: calc(100% - 24px);
    max-height: 92vh;
    max-height: 92dvh;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .selector-header {
    padding: 14px 18px;
  }

  .selector-header h3 {
    font-size: 1.15rem;
  }

  .types-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    padding: 16px 18px 8px;
  }

  .type-card {
    padding: 14px 10px;
    gap: 6px;
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .close-btn,
  .cancel-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .selector-footer {
    padding: 12px 18px max(16px, env(safe-area-inset-bottom, 0px));
  }
}

@media (pointer: coarse) {
  .close-btn,
  .cancel-btn,
  .type-card {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}
</style>
