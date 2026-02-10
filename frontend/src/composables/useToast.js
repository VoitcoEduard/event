import { ref } from 'vue'

const toasts = ref([])
let id = 0

export function useToast() {
    const show = (message, type = 'info', duration = 3000) => {
        const toastId = ++id
        toasts.value.push({ id: toastId, message, type })

        setTimeout(() => {
            remove(toastId)
        }, duration)
    }

    const success = (message, duration) => show(message, 'success', duration)
    const error = (message, duration) => show(message, 'error', duration)
    const warning = (message, duration) => show(message, 'warning', duration)
    const info = (message, duration) => show(message, 'info', duration)

    const remove = (toastId) => {
        const index = toasts.value.findIndex(t => t.id === toastId)
        if (index > -1) {
            toasts.value.splice(index, 1)
        }
    }

    return {
        toasts,
        show,
        success,
        error,
        warning,
        info,
        remove
    }
}
