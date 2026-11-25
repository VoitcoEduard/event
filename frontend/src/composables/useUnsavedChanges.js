import { computed, getCurrentInstance, onScopeDispose, toValue, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import i18n from '../i18n.js'
import { useDialog } from './useDialog.js'

// All active guards, so flows outside the router (e.g. signing out) can ask
// every screen-mounted editor whether it is safe to leave.
const activeGuards = []
let bypassLeaveDepth = 0

/**
 * Confirm pending unsaved changes across all screens mounted by this
 * application. Returns false (and shows a single confirm dialog) when any
 * guard is dirty and the user chooses to stay.
 */
export async function confirmActiveUnsavedChanges() {
  for (const guard of [...activeGuards]) {
    if (!guard.isDirty.value) continue
    const allowed = await guard.confirmLeave()
    if (!allowed) return false
  }
  return true
}

/**
 * Run fn (e.g. a navigation to /login) without triggering the unsaved-changes
 * guards again.
 */
export async function runBypassingUnsavedGuard(fn) {
  bypassLeaveDepth += 1
  try {
    return await fn()
  } finally {
    bypassLeaveDepth -= 1
  }
}

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

  const guard = { isDirty, confirmLeave }
  activeGuards.push(guard)
  const removeGuard = () => {
    const idx = activeGuards.indexOf(guard)
    if (idx !== -1) activeGuards.splice(idx, 1)
  }
  onScopeDispose(() => {
    detachBeforeUnload()
    removeGuard()
  })

  if (getCurrentInstance()) {
    onBeforeRouteLeave(async () => {
      if (bypassLeaveDepth > 0) return true
      return confirmLeave()
    })
  }

  return {
    isDirty,
    confirmLeave
  }
}
