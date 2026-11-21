export const safeLocalGet = (key, fallback = null) => {
  try {
    return globalThis.localStorage?.getItem(key) ?? fallback
  } catch {
    return fallback
  }
}

export const safeLocalSet = (key, value) => {
  try {
    globalThis.localStorage?.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export const safeLocalRemove = (key) => {
  try {
    globalThis.localStorage?.removeItem(key)
    return true
  } catch {
    return false
  }
}

export const safeSessionGet = (key, fallback = null) => {
  try {
    return globalThis.sessionStorage?.getItem(key) ?? fallback
  } catch {
    return fallback
  }
}

export const safeSessionSet = (key, value) => {
  try {
    globalThis.sessionStorage?.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export const safeSessionRemove = (key) => {
  try {
    globalThis.sessionStorage?.removeItem(key)
    return true
  } catch {
    return false
  }
}
