import { ref, onMounted, onUnmounted } from 'vue'
import { teamsActivity, teamsBase } from '../services/teamsActivity'

export function useTeamsActivity(eventId, moduleId, role = 'player') {
  const state = ref(null)
  const loading = ref(true)
  const disconnected = ref(false)
  const error = ref('')
  let stream, interval, running, disposed = false
  const load = () => {
    if (running) return running
    running = (role === 'admin' ? teamsActivity.admin(eventId, moduleId) : role === 'projector' ? teamsActivity.projector(eventId, moduleId) : teamsActivity.state(eventId, moduleId))
      .then(value => { if (!disposed) { state.value = value; disconnected.value = false; error.value = '' } return value })
      .catch(err => { if (!disposed) { disconnected.value = true; error.value = err.message } })
      .finally(() => { if (!disposed) loading.value = false; running = null })
    return running
  }
  onMounted(() => {
    load()
    stream = new EventSource(`${teamsBase(eventId, moduleId)}/stream`)
    stream.addEventListener('ready', load)
    stream.onmessage = load
    stream.onerror = () => { if (!disposed) disconnected.value = true }
    interval = setInterval(load, 3000)
  })
  onUnmounted(() => { disposed = true; stream?.close(); clearInterval(interval) })
  return { state, loading, disconnected, error, load }
}
