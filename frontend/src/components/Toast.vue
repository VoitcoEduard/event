<template>
  <Teleport to="body">
    <div class="toast-container">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="toast"
          :class="toast.type"
        >
          <span class="toast-icon">
            <AppIcon :name="getIcon(toast.type)" :size="15" />
          </span>
          <span class="toast-message">{{ toast.message }}</span>
          <button class="toast-close" @click="remove(toast.id)" aria-label="Close notification">
            <AppIcon name="close" :size="16" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { useToast } from '../composables/useToast'
import AppIcon from './AppIcon.vue'

const { toasts, remove } = useToast()

const getIcon = (type) => {
  switch (type) {
    case 'success': return 'check'
    case 'error': return 'close'
    case 'warning': return 'alert'
    default: return 'info'
  }
}
</script>

<style scoped>
.toast-container {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 400px;
}

.toast {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background-color: #1E1E1E;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  border: 1px solid #333;
}

.toast.success {
  border-left: 4px solid #30D158;
}

.toast.error {
  border-left: 4px solid #FF453A;
}

.toast.warning {
  border-left: 4px solid #FF9F0A;
}

.toast.info {
  border-left: 4px solid #007AFF;
}

.toast-icon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
}

.toast.success .toast-icon {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.toast.error .toast-icon {
  background-color: rgba(255, 69, 58, 0.2);
  color: #FF453A;
}

.toast.warning .toast-icon {
  background-color: rgba(255, 159, 10, 0.2);
  color: #FF9F0A;
}

.toast.info .toast-icon {
  background-color: rgba(0, 122, 255, 0.2);
  color: #007AFF;
}

.toast-message {
  flex: 1;
  font-size: 14px;
  color: #FFF;
  font-weight: 500;
}

.toast-close {
  background: none;
  border: none;
  color: #666;
  font-size: 14px;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: color 0.2s, background-color 0.2s;
}

.toast-close:hover {
  color: #FFF;
  background-color: #333;
}

.toast-enter-active {
  animation: toast-in 0.3s ease-out;
}

.toast-leave-active {
  animation: toast-out 0.3s ease-in;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(100%);
  }
}
</style>
