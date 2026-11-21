import { computed, getCurrentInstance, onScopeDispose, toValue, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import i18n from '../i18n.js'
import { useDialog } from './useDialog.js'

export function useUnsavedChanges(isDirtySource, options = {}) {
  const { confirm } = useDialog()
  const isDirty = computed(() => Boolean(toValue(isDirtySource)))

  let beforeUnloadAttached = false

  function handleBeforeUnload(event) {
    if (!isDirty.value) return
    event.preventDefault()
    event.returnValue = ''
    return ''
  }

  function attachBeforeUnload() {
    if (beforeUnloadAttached || typeof window === 'undefined') return
    window.addEventListener('beforeunload', handleBeforeUnload)
    beforeUnloadAttached = true
  }

  function detachBeforeUnload() {
    if (!beforeUnloadAttached || typeof window === 'undefined') return
    window.removeEventListener('beforeunload', handleBeforeUnload)
    beforeUnloadAttached = false
  }

  async function confirmLeave() {
    if (!isDirty.value) return true
    const t = i18n.global.t
    const confirmed = await confirm({
      title: toValue(options.title) || t('unsaved_changes.title'),
      message: toValue(options.message) || t('unsaved_changes.message'),
      confirmLabel: toValue(options.confirmLabel) || t('unsaved_changes.leave'),
      cancelLabel: toValue(options.cancelLabel) || t('unsaved_changes.stay'),
      destructive: options.destructive ?? true
    })
    if (confirmed && typeof options.onConfirm === 'function') {
      await options.onConfirm()
    }
    return confirmed
  }

  watch(
    isDirty,
    (dirty) => {
      if (dirty) {
        attachBeforeUnload()
      } else {
        detachBeforeUnload()
      }
    },
    { immediate: true }
  )

  onScopeDispose(detachBeforeUnload)

  if (getCurrentInstance()) {
    onBeforeRouteLeave(async () => confirmLeave())
  }

  return {
    isDirty,
    confirmLeave
  }
}
