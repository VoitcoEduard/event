import { API_BASE_URL } from '../utils/index'

/**
 * Upload an image file to the server
 * @param {File} file - The image file to upload
 * @param {string} eventId - The event ID
 * @param {string} moduleId - The module ID
 * @returns {Promise<Object>} - The media response with id, url, etc.
 */
export async function uploadImage(file, eventId, moduleId) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('eventId', eventId || '')
  formData.append('moduleId', moduleId || '')

  const response = await fetch(`${API_BASE_URL}/media/upload`, {
    method: 'POST',
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
    method: 'DELETE'
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
    method: 'DELETE'
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || 'Failed to delete screensaver')
  }

  return response.json()
}
