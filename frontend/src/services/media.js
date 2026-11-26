import { API_BASE_URL, getAuthHeaders } from '../utils/index'

export async function convertToWebP(file, maxWidth = 1200, quality = 0.85) {
  return new Promise((resolve) => {
    const img = new Image()
    const objectUrl = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(objectUrl)

      let { width, height } = img
      if (width > maxWidth) {
        height = Math.round((height * maxWidth) / width)
        width = maxWidth
      }

      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height

      const ctx = canvas.getContext('2d')
      ctx.drawImage(img, 0, 0, width, height)

      canvas.toBlob(
        (blob) => {
          if (!blob) {
            resolve(file)
            return
          }
          const webpFile = new File(
            [blob],
            (file.name || 'image').replace(/\.[^.]+$/, '') + '.webp',
            { type: 'image/webp' }
          )
          resolve(webpFile)
        },
        'image/webp',
        quality
      )
    }

    img.onerror = () => {
      URL.revokeObjectURL(objectUrl)
      resolve(file)
    }

    img.src = objectUrl
  })
}

export async function uploadImage(file, eventId, moduleId) {
  const processedFile = await convertToWebP(file)

  const formData = new FormData()
  formData.append('file', processedFile)
  formData.append('eventId', eventId || '')
  formData.append('moduleId', moduleId || '')

  const response = await fetch(`${API_BASE_URL}/media/upload`, {
    method: 'POST',
    headers: { ...getAuthHeaders() },
    body: formData
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to upload image')
  }

  return response.json()
}

export function getImageUrl(imageId) {
  if (!imageId) return null
  return `${API_BASE_URL}/media/${imageId}/file`
}

export async function deleteImage(imageId) {
  const response = await fetch(`${API_BASE_URL}/media/${imageId}`, {
    method: 'DELETE',
    headers: { ...getAuthHeaders() }
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to delete image')
  }

  return response.json()
}

export async function getMediaInfo(imageId) {
  const response = await fetch(`${API_BASE_URL}/media/${imageId}`, {
    headers: { ...getAuthHeaders() }
  })

  if (!response.ok) {
    throw new Error('Failed to get media info')
  }

  return response.json()
}

export async function getScreensavers() {
  const response = await fetch(`${API_BASE_URL}/screensavers`)

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to fetch screensavers')
  }

  return response.json()
}

export async function uploadScreensaver(file, name) {
  const processedFile = await convertToWebP(file, 1920, 0.85)

  const formData = new FormData()
  formData.append('file', processedFile)
  formData.append('name', name || file.name)

  const response = await fetch(`${API_BASE_URL}/screensavers`, {
    method: 'POST',
    headers: { ...getAuthHeaders() },
    body: formData
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to upload screensaver')
  }

  return response.json()
}

export async function deleteScreensaver(screensaverId) {
  const response = await fetch(`${API_BASE_URL}/screensavers/${screensaverId}`, {
    method: 'DELETE',
    headers: { ...getAuthHeaders() }
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to delete screensaver')
  }

  return response.json()
}

export async function getTracks() {
  const response = await fetch(`${API_BASE_URL}/tracks`)

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to fetch tracks')
  }

  return response.json()
}

export async function uploadTrack(file, name) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('name', name || file.name)

  const response = await fetch(`${API_BASE_URL}/tracks`, {
    method: 'POST',
    headers: { ...getAuthHeaders() },
    body: formData
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to upload track')
  }

  return response.json()
}

export async function deleteTrack(trackId) {
  const response = await fetch(`${API_BASE_URL}/tracks/${trackId}`, {
    method: 'DELETE',
    headers: { ...getAuthHeaders() }
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to delete track')
  }

  return response.json()
}
