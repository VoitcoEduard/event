import { readonly, ref } from 'vue'

const activeDialog = ref(null)
const queue = []
let nextDialogId = 0

function normalizeOptions(options) {
  if (typeof options === 'string') {
    return { message: options }
  }
  return options && typeof options === 'object' ? options : {}
}

function enqueueDialog(entry) {
  if (!activeDialog.value) {
    activeDialog.value = entry
  } else {
    queue.push(entry)
  }
}

export function resolveActiveDialog(value) {
  const current = activeDialog.value
  if (!current) return
  activeDialog.value = queue.shift() || null
  current.resolve(value)
}

export function cancelActiveDialog() {
  const current = activeDialog.value
  if (!current) return
  const cancelValue = current.type === 'prompt' ? null : false
  resolveActiveDialog(cancelValue)
}

export function clearDialogQueue() {
  const current = activeDialog.value
  const pending = queue.splice(0, queue.length)
  activeDialog.value = null
  if (current) {
    current.resolve(current.type === 'prompt' ? null : false)
  }
  for (const item of pending) {
    item.resolve(item.type === 'prompt' ? null : false)
  }
}

export function confirm(options = {}) {
  const opts = normalizeOptions(options)
  return new Promise((resolve) => {
    enqueueDialog({
      id: ++nextDialogId,
      type: 'confirm',
      title: opts.title ?? '',
      message: opts.message ?? '',
      confirmLabel: opts.confirmLabel ?? '',
      cancelLabel: opts.cancelLabel ?? '',
      destructive: Boolean(opts.destructive),
      resolve
    })
  })
}

export function prompt(options = {}) {
  const opts = normalizeOptions(options)
  return new Promise((resolve) => {
    enqueueDialog({
      id: ++nextDialogId,
      type: 'prompt',
      title: opts.title ?? '',
      message: opts.message ?? '',
      label: opts.label ?? '',
      defaultValue: opts.defaultValue != null ? String(opts.defaultValue) : '',
      placeholder: opts.placeholder ?? '',
      confirmLabel: opts.confirmLabel ?? '',
      cancelLabel: opts.cancelLabel ?? '',
      resolve
    })
  })
}

export function useDialog() {
  return {
    activeDialog: readonly(activeDialog),
    confirm,
    prompt,
    resolveDialog: resolveActiveDialog,
    cancelDialog: cancelActiveDialog
  }
}
