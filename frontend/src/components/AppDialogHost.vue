<template>
  <Teleport to="body">
    <div
      v-if="activeDialog"
      class="app-dialog-overlay"
      @click.self="handleCancel"
    >
      <div
        ref="dialogRef"
        class="app-dialog"
        :role="activeDialog.destructive ? 'alertdialog' : 'dialog'"
        aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="activeDialog.message ? messageId : undefined"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <div class="app-dialog-header">
          <h2 :id="titleId" class="app-dialog-title">
            {{ resolvedTitle }}
          </h2>
          <button
            type="button"
            class="app-dialog-close"
            :aria-label="t('dialog.close')"
            @click="handleCancel"
          >
            <AppIcon name="close" :size="18" aria-hidden="true" />
          </button>
        </div>

        <form v-if="activeDialog.type === 'prompt'" class="app-dialog-body" @submit.prevent="handleConfirm">
          <p v-if="activeDialog.message" :id="messageId" class="app-dialog-message">
            {{ activeDialog.message }}
          </p>
          <div class="app-dialog-field">
            <label v-if="activeDialog.label" :for="inputId" class="app-dialog-label">
              {{ activeDialog.label }}
            </label>
            <input
              :id="inputId"
              ref="inputRef"
              v-model="promptValue"
              type="text"
              class="app-dialog-input"
              :placeholder="activeDialog.placeholder || ''"
              :aria-label="activeDialog.label ? undefined : resolvedTitle"
            />
          </div>
          <div class="app-dialog-actions">
            <button
              ref="cancelBtnRef"
              type="button"
              class="app-dialog-btn app-dialog-btn--secondary"
              @click="handleCancel"
            >
              {{ resolvedCancelLabel }}
            </button>
            <button
              ref="confirmBtnRef"
              type="submit"
              class="app-dialog-btn app-dialog-btn--primary"
            >
              {{ resolvedConfirmLabel }}
            </button>
          </div>
        </form>

        <div v-else class="app-dialog-body">
          <p v-if="activeDialog.message" :id="messageId" class="app-dialog-message">
            {{ activeDialog.message }}
          </p>
          <div class="app-dialog-actions">
            <button
              ref="cancelBtnRef"
              type="button"
              class="app-dialog-btn app-dialog-btn--secondary"
              @click="handleCancel"
            >
              {{ resolvedCancelLabel }}
            </button>
            <button
              ref="confirmBtnRef"
              type="button"
              class="app-dialog-btn"
              :class="activeDialog.destructive ? 'app-dialog-btn--danger' : 'app-dialog-btn--primary'"
              @click="handleConfirm"
            >
              {{ resolvedConfirmLabel }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import i18n from '../i18n.js'
import AppIcon from './AppIcon.vue'
import { clearDialogQueue, useDialog } from '../composables/useDialog'

const t = (...args) => i18n.global.t(...args)
const { activeDialog, resolveDialog, cancelDialog } = useDialog()

// Never leave pending dialog promises dangling when the host unmounts.
onUnmounted(() => { clearDialogQueue() })

const dialogRef = ref(null)
const inputRef = ref(null)
const cancelBtnRef = ref(null)
const confirmBtnRef = ref(null)
const promptValue = ref('')

let previousFocus = null

const titleId = computed(() => (activeDialog.value ? `app-dialog-title-${activeDialog.value.id}` : undefined))
const messageId = computed(() => (activeDialog.value ? `app-dialog-msg-${activeDialog.value.id}` : undefined))
const inputId = computed(() => (activeDialog.value ? `app-dialog-input-${activeDialog.value.id}` : undefined))

const resolvedTitle = computed(() => {
  if (!activeDialog.value) return ''
  if (activeDialog.value.title) return activeDialog.value.title
  return activeDialog.value.type === 'prompt' ? t('dialog.prompt_title') : t('dialog.confirm_title')
})

const resolvedConfirmLabel = computed(() => activeDialog.value?.confirmLabel || t('dialog.confirm'))
const resolvedCancelLabel = computed(() => activeDialog.value?.cancelLabel || t('dialog.cancel'))

function handleConfirm() {
  if (!activeDialog.value) return
  if (activeDialog.value.type === 'prompt') {
    resolveDialog(promptValue.value)
  } else {
    resolveDialog(true)
  }
}

function handleCancel() {
  cancelDialog()
}

function getFocusableElements() {
  if (!dialogRef.value) return []
  const nodes = dialogRef.value.querySelectorAll(
    'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [href], [tabindex]:not([tabindex="-1"])'
  )
  return Array.from(nodes)
}

function handleKeydown(event) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    handleCancel()
    return
  }

  if (event.key === 'Tab') {
    const focusables = getFocusableElements()
    if (!focusables.length) {
      event.preventDefault()
      return
    }
    const first = focusables[0]
    const last = focusables[focusables.length - 1]
    if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.value)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
    return
  }

  if (event.key === 'Enter' && activeDialog.value?.type === 'confirm') {
    const tag = event.target?.tagName?.toLowerCase()
    if (tag !== 'button' && tag !== 'a' && tag !== 'textarea') {
      event.preventDefault()
      handleConfirm()
    }
  }
}

watch(
  activeDialog,
  async (next, prev) => {
    if (next && !prev && typeof document !== 'undefined') {
      previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    }

    if (next) {
      promptValue.value = next.type === 'prompt' ? next.defaultValue : ''
      await nextTick()
      if (next.type === 'prompt') {
        inputRef.value?.focus()
      } else if (next.destructive) {
        dialogRef.value?.focus()
      } else {
        confirmBtnRef.value?.focus()
      }
      return
    }

    if (!next && prev) {
      await nextTick()
      if (previousFocus && typeof document !== 'undefined' && document.contains(previousFocus)) {
        previousFocus.focus()
      }
      previousFocus = null
    }
  },
  { flush: 'post' }
)
</script>

<style scoped>
.app-dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 9500;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: max(16px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) max(16px, env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  background: rgba(16, 24, 40, 0.52);
  overscroll-behavior: contain;
}

.app-dialog {
  width: min(100%, 440px);
  max-height: calc(100dvh - 32px);
  overflow-y: auto;
  overscroll-behavior: contain;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: var(--admin-radius-md, 12px);
  box-shadow: 0 20px 44px rgba(16, 24, 40, 0.24);
  padding: 20px;
  outline: none;
}

.app-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.app-dialog-title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--admin-text, #101828);
}

.app-dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--admin-text-muted, #475467);
  cursor: pointer;
}

.app-dialog-close:hover {
  background: var(--admin-surface-hover, #eef2f7);
  color: var(--admin-text, #101828);
}

.app-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.app-dialog-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--admin-text-muted, #475467);
  overflow-wrap: anywhere;
}

.app-dialog-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.app-dialog-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text, #101828);
}

.app-dialog-input {
  width: 100%;
  min-height: 42px;
  padding: 9px 12px;
  box-sizing: border-box;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: var(--admin-radius-sm, 8px);
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 14px;
}

.app-dialog-input::placeholder {
  color: var(--admin-text-dim, #667085);
}

.app-dialog-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
}

.app-dialog-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
  padding: 8px 16px;
  border-radius: var(--admin-radius-sm, 8px);
  font: inherit;
  font-size: 14px;
  font-weight: 650;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background-color 0.15s, border-color 0.15s;
}

.app-dialog-btn--secondary {
  background: var(--admin-surface-subtle, #f8f9fb);
  border-color: var(--admin-border-strong, #c8ced8);
  color: var(--admin-text, #101828);
}

.app-dialog-btn--secondary:hover {
  background: var(--admin-surface-hover, #eef2f7);
}

.app-dialog-btn--primary {
  background: var(--admin-primary-solid, #0969da);
  border-color: var(--admin-primary-solid, #0969da);
  color: #ffffff;
}

.app-dialog-btn--primary:hover {
  background: var(--admin-primary-hover, #0758b8);
}

.app-dialog-btn--danger {
  background: var(--admin-danger-solid, #b42318);
  border-color: var(--admin-danger-solid, #b42318);
  color: #ffffff;
}

.app-dialog-btn--danger:hover {
  filter: brightness(0.92);
}

@media (pointer: coarse) {
  .app-dialog-close {
    min-width: 44px;
    min-height: 44px;
  }
  .app-dialog-btn {
    min-height: 44px;
  }
}

@media (max-width: 480px) {
  .app-dialog {
    padding: 16px;
  }
  .app-dialog-actions {
    flex-direction: column-reverse;
  }
  .app-dialog-btn {
    width: 100%;
  }
}
</style>
