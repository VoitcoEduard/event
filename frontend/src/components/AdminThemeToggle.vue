<template>
  <div
    class="admin-theme-control"
    :class="{ 'admin-theme-control--inline': variant === 'inline' }"
    role="group"
    :aria-label="$t('appearance.label')"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="admin-theme-option"
      :class="{ active: theme === option.value }"
      :aria-pressed="theme === option.value"
      @click="setTheme(option.value)"
    >
      <AppIcon :name="option.icon" :size="15" />
      <span>{{ $t(option.label) }}</span>
    </button>
  </div>
</template>

<script setup>
import AppIcon from './AppIcon.vue'
import { useAdminTheme } from '../composables/useAdminTheme'

defineProps({
  variant: {
    type: String,
    default: 'floating',
    validator: (value) => ['floating', 'inline'].includes(value)
  }
})

const { theme, setTheme } = useAdminTheme()
const options = [
  { value: 'light', label: 'appearance.light', icon: 'sun' },
  { value: 'dark', label: 'appearance.dark', icon: 'moon' }
]
</script>
