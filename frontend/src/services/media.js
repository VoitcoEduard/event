import { API_BASE_URL, getAuthHeaders } from '../utils/index'

/**
 * Convert an image File to WebP format using the Canvas API.
 * Proportionally downscales to maxWidth if the image is wider.
 * Falls back to the original file if WebP encoding is unsupported.
 * @param {File} file - The source image file
 * @param {number} maxWidth - Maximum output width in pixels (default: 1200)
 * @param {number} quality - WebP quality 0–1 (default: 0.85)
 * @returns {Promise<File>} - A new File in WebP format (or original if fallback)
 */
export async function convertToWebP(file, maxWidth = 1200, quality = 0.85) {
  return new Promise((resolve) => {
    const img = new Image()
    const objectUrl = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(objectUrl)

      // Calculate output dimensions
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
            // WebP not supported — return original
            resolve(file)
            return
          }
          const webpFile = new File(
            [blob],
            file.name.replace(/\.[^.]+$/, '') + '.webp',
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
      // If image fails to load, return original
      resolve(file)
    }

    img.src = objectUrl
  })
}

/**
 * Upload an image file to the server
 * @param {File} file - The image file to upload
 * @param {string} eventId - The event ID
 * @param {string} moduleId - The module ID
 * @returns {Promise<Object>} - The media response with id, url, etc.
 */
export async function uploadImage(file, eventId, moduleId) {
  // Convert to WebP before uploading for smaller file sizes
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

/**
 * Get the URL for serving a media file
 * @param {string} imageId - The media ID
 * @returns {string} - The full URL to the image
 */
export function getImageUrl(imageId) {
  if (!imageId) return null
  return `${API_BASE_URL}/media/${imageId}/file`
}

/**
 * Delete a media file
 * @param {string} imageId - The media ID
 * @returns {Promise<Object>} - The response
 */
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

/**
 * Get media metadata
 * @param {string} imageId - The media ID
 * @returns {Promise<Object>} - The media metadata
 */
export async function getMediaInfo(imageId) {
  const response = await fetch(`${API_BASE_URL}/media/${imageId}`)

  if (!response.ok) {
    throw new Error('Failed to get media info')
  }

  return response.json()
}

/**
 * Get all screensavers from the library
 * @returns {Promise<Array>} - Array of screensaver objects
 */
export async function getScreensavers() {
  const response = await fetch(`${API_BASE_URL}/screensavers`)

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to fetch screensavers')
  }

  return response.json()
}

/**
 * Upload a new screensaver to the library
 * @param {File} file - The image file to upload
 * @param {string} name - The name for the screensaver
 * @returns {Promise<Object>} - The created screensaver object
 */
export async function uploadScreensaver(file, name) {
  const formData = new FormData()
  formData.append('file', file)
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

/**
 * Delete a screensaver from the library
 * @param {string} screensaverId - The screensaver ID
 * @returns {Promise<Object>} - The response
 */
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

/**
 * Get all audio tracks from the library
 * @returns {Promise<Array>} - Array of track objects
 */
export async function getTracks() {
  const response = await fetch(`${API_BASE_URL}/tracks`)

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to fetch tracks')
  }

  return response.json()
}

/**
 * Upload a new audio track to the library
 * @param {File} file - The audio file to upload
 * @param {string} name - The name for the track
 * @returns {Promise<Object>} - The created track object
 */
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

/**
 * Delete a track from the library
 * @param {string} trackId - The track ID
 * @returns {Promise<Object>} - The response
 */
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
