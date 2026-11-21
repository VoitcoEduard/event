<template>
  <Teleport to="body">
    <div class="toast-container" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="toast"
          :class="toast.type"
          :role="toast.type === 'error' ? 'alert' : 'status'"
        >
          <span class="toast-icon">
            <AppIcon :name="getIcon(toast.type)" :size="15" aria-hidden="true" />
          </span>
          <span class="toast-message">{{ toast.message }}</span>
          <button class="toast-close" @click="remove(toast.id)" aria-label="Close notification">
            <AppIcon name="close" :size="16" aria-hidden="true" />
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
  top: 76px;
  right: 20px;
  bottom: auto;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 400px;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  min-width: 0;
  max-width: 100%;
  pointer-events: auto;
  background-color: var(--admin-surface, #1E1E1E);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  border: 1px solid var(--admin-border, #333);
}

.toast.success {
  border-left: 4px solid var(--admin-success, #30D158);
}

.toast.error {
  border-left: 4px solid var(--admin-danger, #FF453A);
}

.toast.warning {
  border-left: 4px solid var(--admin-warning, #FF9F0A);
}

.toast.info {
  border-left: 4px solid var(--admin-primary, #007AFF);
}

.toast-icon {
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
}

.toast.success .toast-icon {
  background-color: var(--admin-success-soft, rgba(48, 209, 88, 0.2));
  color: var(--admin-success, #30D158);
}

.toast.error .toast-icon {
  background-color: var(--admin-danger-soft, rgba(255, 69, 58, 0.2));
  color: var(--admin-danger, #FF453A);
}

.toast.warning .toast-icon {
  background-color: var(--admin-warning-soft, rgba(255, 159, 10, 0.2));
  color: var(--admin-warning, #FF9F0A);
}

.toast.info .toast-icon {
  background-color: var(--admin-primary-soft, rgba(0, 122, 255, 0.2));
  color: var(--admin-primary, #007AFF);
}

.toast-message {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 14px;
  color: var(--admin-text, #FFF);
  font-weight: 500;
}

.toast-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--admin-text-muted, #A1A1AA);
  font-size: 14px;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: color 0.2s, background-color 0.2s;
}

.toast-close:hover {
  color: var(--admin-text, #FFF);
  background-color: var(--admin-surface-hover, #333);
}

@media (max-width: 640px) {
  .toast-container {
    top: auto;
    bottom: max(16px, env(safe-area-inset-bottom, 0px));
    left: max(12px, env(safe-area-inset-left, 0px));
    right: max(12px, env(safe-area-inset-right, 0px));
    max-width: none;
  }
}

@media (pointer: coarse) {
  .toast-close {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
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
