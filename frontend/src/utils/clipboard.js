export async function copyTextToClipboard(text) {
  if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Fall back to legacy execCommand copy below (e.g. insecure HTTP origin or denied permission)
    }
  }

  if (typeof document === 'undefined' || typeof document.execCommand !== 'function') {
    return false
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '-9999px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()

  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  } finally {
    document.body.removeChild(textarea)
  }
  return copied
}

export const SUPPORTED_IMAGE_MIME_TYPES = [
  'image/png',
  'image/jpeg',
  'image/webp',
  'image/gif'
]

function normalizeMimeType(mimeType) {
  return String(mimeType || '').split(';')[0].trim().toLowerCase()
}

export function isSupportedClipboardImageType(mimeType) {
  const normalized = normalizeMimeType(mimeType)
  if (!normalized.startsWith('image/')) return false
  return normalized !== 'image/svg+xml'
}

export function extensionForImageMime(mimeType) {
  switch (normalizeMimeType(mimeType)) {
    case 'image/jpeg':
      return 'jpg'
    case 'image/png':
      return 'png'
    case 'image/gif':
      return 'gif'
    case 'image/webp':
      return 'webp'
    default:
      return 'png'
  }
}

export function blobToImageFile(blob, baseName = 'pasted-image') {
  if (typeof File !== 'undefined' && blob instanceof File && blob.name && blob.name.includes('.')) {
    return blob
  }
  const type = normalizeMimeType(blob?.type) || 'image/png'
  const ext = extensionForImageMime(type)
  return new File([blob], `${baseName}-${Date.now()}.${ext}`, { type })
}

export function extractImageFromClipboardData(clipboardData, { ignoreWhenTextPresent = false } = {}) {
  if (!clipboardData) return null

  if (ignoreWhenTextPresent) {
    const plainText = typeof clipboardData.getData === 'function'
      ? clipboardData.getData('text/plain')
      : ''
    if (String(plainText || '').trim().length > 0) {
      return null
    }
  }

  const items = clipboardData.items ? Array.from(clipboardData.items) : []
  const fileImageItems = items.filter(
    (item) => item && item.kind === 'file' && isSupportedClipboardImageType(item.type)
  )

  for (const preferredType of SUPPORTED_IMAGE_MIME_TYPES) {
    const preferred = fileImageItems.find((item) => normalizeMimeType(item.type) === preferredType)
    if (preferred && typeof preferred.getAsFile === 'function') {
      const file = preferred.getAsFile()
      if (file) return blobToImageFile(file)
    }
  }

  for (const item of fileImageItems) {
    if (typeof item.getAsFile === 'function') {
      const file = item.getAsFile()
      if (file) return blobToImageFile(file)
    }
  }

  const files = clipboardData.files ? Array.from(clipboardData.files) : []
  for (const file of files) {
    if (file && isSupportedClipboardImageType(file.type)) {
      return blobToImageFile(file)
    }
  }

  return null
}

export async function readImageFromClipboard() {
  if (
    typeof navigator === 'undefined' ||
    !navigator.clipboard ||
    typeof navigator.clipboard.read !== 'function'
  ) {
    return { status: 'unsupported' }
  }

  try {
    const items = await navigator.clipboard.read()
    for (const item of items) {
      const types = Array.isArray(item?.types) ? item.types : []
      let matchedType = ''

      for (const preferredType of SUPPORTED_IMAGE_MIME_TYPES) {
        const found = types.find((t) => normalizeMimeType(t) === preferredType)
        if (found) {
          matchedType = found
          break
        }
      }

      if (!matchedType) {
        matchedType = types.find((t) => isSupportedClipboardImageType(t)) || ''
      }

      if (matchedType && typeof item.getType === 'function') {
        const blob = await item.getType(matchedType)
        if (blob) {
          return { status: 'ok', file: blobToImageFile(blob) }
        }
      }
    }

    return { status: 'empty' }
  } catch (error) {
    const name = error?.name || ''
    if (name === 'NotAllowedError' || name === 'SecurityError') {
      return { status: 'denied', error }
    }
    return { status: 'error', error }
  }
}

const NON_TEXT_INPUT_TYPES = new Set([
  'radio',
  'checkbox',
  'button',
  'submit',
  'reset',
  'file',
  'range',
  'color',
  'image',
  'hidden'
])

export function isEditableElement(el) {
  if (!el || typeof el !== 'object') return false
  if (el.isContentEditable) return true
  const tagName = String(el.tagName || '').toUpperCase()
  if (tagName === 'TEXTAREA') return true
  if (tagName === 'INPUT') {
    const inputType = String(el.type || 'text').toLowerCase()
    return !NON_TEXT_INPUT_TYPES.has(inputType)
  }
  return false
}

export function hasActiveBlockingModal(containerEl) {
  if (typeof document === 'undefined') return false
  if (containerEl?.closest?.('[inert]')) return true

  const overlays = document.querySelectorAll(
    'dialog[open], [role="dialog"][aria-modal="true"], [role="alertdialog"][aria-modal="true"], .app-dialog-overlay, .modal-overlay, .dialog-overlay, .library-modal-overlay, .selector-modal'
  )
  for (const el of overlays) {
    if (!containerEl || (!el.contains(containerEl) && !containerEl.contains(el))) {
      return true
    }
  }
  return false
}

export function isPasteTargetInContainer(target, containerEl) {
  if (
    target === window ||
    target === globalThis ||
    (target && target.window === target) ||
    target === document ||
    target === document.body ||
    target === document.documentElement
  ) {
    return true
  }
  return typeof Node !== 'undefined' && target instanceof Node && Boolean(containerEl?.contains?.(target))
}
