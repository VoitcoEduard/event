<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container">
        <div class="selector-header">
          <h3>{{ $t('quiz_config.question_editor.selector.title') }}</h3>
          <button class="close-btn" @click="handleClose">✕</button>
        </div>

        <div class="types-grid">
          <div 
            v-for="type in questionTypes" 
            :key="type.id"
            class="type-card"
            :class="{ active: hoveredType === type.id }"
            @mouseenter="hoveredType = type.id"
            @mouseleave="hoveredType = null"
            @click="selectType(type.id)"
          >
            <div class="type-icon-svg" v-html="type.icon"></div>
            <span class="type-label">{{ type.label }}</span>
          </div>
        </div>

        <div class="description-area">
          <div v-if="hoveredType" class="description-content slide-up">
            <p>{{ getTypeDescription(hoveredType) }}</p>
          </div>
          <div v-else class="description-placeholder">
            {{ $t('quiz_config.question_editor.selector.description_hint') }}
          </div>
        </div>

        <div class="selector-footer">
          <button class="cancel-btn" @click="handleClose">{{ $t('quiz_config.question_editor.selector.cancel') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  visible: { type: Boolean, default: false }
})

const emit = defineEmits(['close', 'select'])
const { t } = useI18n()

const hoveredType = ref(null)

const questionTypes = computed(() => [
  { 
    id: 'classic', 
    label: t('quiz_config.question_editor.types.classic'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5" fill="currentColor" stroke="none"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/><path d="M14.5 5.5l1.5 1.5 3-3" stroke="#1C1C1E" stroke-width="1.5"/></svg>`
  },
  { 
    id: 'multi_select', 
    label: t('quiz_config.question_editor.types.multi_select'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="3" width="4" height="4" rx="1"/><line x1="11" y1="5" x2="20" y2="5"/><rect x="4" y="10" width="4" height="4" rx="1" fill="currentColor"/><path d="M4.5 12l1 1 2-2" stroke="#1C1C1E"/><line x1="11" y1="12" x2="20" y2="12"/><rect x="4" y="17" width="4" height="4" rx="1" fill="currentColor"/><path d="M4.5 19l1 1 2-2" stroke="#1C1C1E"/><line x1="11" y1="19" x2="20" y2="19"/></svg>`
  },
  { 
    id: 'true_false', 
    label: t('quiz_config.question_editor.types.true_false'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="5" width="5" height="5" rx="1" fill="currentColor"/><path d="M5 7.5L6 8.5L8 6.5" stroke="#1C1C1E"/><line x1="12" y1="7.5" x2="20" y2="7.5"/><rect x="4" y="14" width="5" height="5" rx="1" fill="currentColor"/><path d="M5 15l3 3 M8 15l-3 3" stroke="#1C1C1E"/><line x1="12" y1="16.5" x2="20" y2="16.5"/></svg>`
  },
  { 
    id: 'range', 
    label: t('quiz_config.question_editor.types.range'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="2" y1="20" x2="22" y2="20" stroke-width="2"/><line x1="2" y1="18" x2="2" y2="22"/><line x1="7" y1="19" x2="7" y2="21"/><line x1="12" y1="18" x2="12" y2="22"/><line x1="17" y1="19" x2="17" y2="21"/><line x1="22" y1="18" x2="22" y2="22"/><path d="M12 3c-2.5 0-4.5 2-4.5 4.5 0 3.5 4.5 8.5 4.5 8.5s4.5-5 4.5-8.5C16.5 5 14.5 3 12 3z" fill="currentColor" stroke="none"/><circle cx="12" cy="7.5" r="2" fill="#1C1C1E"/></svg>`
  },
  { 
    id: 'number', 
    label: t('quiz_config.question_editor.types.number'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9.5 9.5a3 3 0 0 1 5.8 1.1c0 2.2-3.3 3.3-3.3 3.3" stroke-width="2.5"/><path d="M12 18h.01" stroke-width="3.5"/><text x="3" y="8" font-family="sans-serif" font-weight="bold" font-size="8" stroke="none" transform="rotate(-15 5 8)" fill="currentColor">4</text><text x="18" y="6" font-family="sans-serif" font-weight="bold" font-size="6" stroke="none" transform="rotate(20 18 6)" fill="currentColor">9</text><text x="5" y="21" font-family="sans-serif" font-weight="bold" font-size="5" stroke="none" transform="rotate(10 5 21)" fill="currentColor">2</text><text x="19" y="21" font-family="sans-serif" font-weight="bold" font-size="7" stroke="none" transform="rotate(-25 19 21)" fill="currentColor">7</text></svg>`
  },
  { 
    id: 'correct_picture', 
    label: t('quiz_config.question_editor.types.correct_picture'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="5" width="5.5" height="7.5" rx="1"/><circle cx="3.75" cy="8" r="1" fill="currentColor" stroke="none"/><path d="M1 12.5l2-1.5 2.5 2"/><path d="M2.5 16.5l2.5 2.5m0-2.5l-2.5 2.5" stroke-width="2"/><rect x="8.5" y="4" width="7" height="9" rx="1" stroke-width="2"/><circle cx="12" cy="7.5" r="1.5" fill="currentColor" stroke="none"/><path d="M8.5 13l2.5-2 4 3" stroke-width="1.5"/><rect x="10" y="15.5" width="4" height="4" rx="1" fill="currentColor" stroke="none"/><path d="M10.5 17.5l1 1 2-2" stroke="#1C1C1E" stroke-width="1.5"/><rect x="17.5" y="5" width="5.5" height="7.5" rx="1"/><circle cx="20.25" cy="8" r="1" fill="currentColor" stroke="none"/><path d="M17.5 12.5l2-1.5 2.5 2"/><path d="M19 16.5l2.5 2.5m0-2.5l-2.5 2.5" stroke-width="2"/></svg>`
  },
  { 
    id: 'wrong_pictures', 
    label: t('quiz_config.question_editor.types.wrong_pictures'), 
    icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="16" height="12" rx="1.5"/><circle cx="8" cy="8" r="1.5" fill="currentColor" stroke="none"/><path d="M3 16l4-4 3 3 3-3 6 6"/><circle cx="14" cy="14" r="5" fill="#1C1C1E" stroke="currentColor" stroke-width="2"/><line x1="17.5" y1="17.5" x2="22" y2="22" stroke-width="3"/><line x1="14" y1="12" x2="14" y2="16" stroke-width="1.5" stroke="currentColor"/><line x1="12" y1="14" x2="16" y2="14" stroke-width="1.5" stroke="currentColor"/><circle cx="14" cy="14" r="2" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`
  }
])

const handleClose = () => {
  emit('close')
}

const selectType = (typeId) => {
  emit('select', typeId)
  handleClose()
}

const getTypeDescription = (typeId) => {
  return t(`quiz_config.question_editor.type_descriptions.${typeId}`)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 3000;
  backdrop-filter: blur(4px);
}

.modal-container {
  background-color: #111112;
  width: 95%;
  max-width: 800px;
  border-radius: 24px;
  overflow: hidden;
  border: 3px solid #3A3A3C;
  display: flex;
  flex-direction: column;
  box-shadow: 0 0 0 1px #000, 0 30px 60px rgba(0,0,0,0.8);
}

.selector-header {
  padding: 32px 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 2px solid #1C1C1E;
}

.selector-header h3 {
  margin: 0;
  color: white;
  font-size: 1.8rem;
  font-weight: 800;
  letter-spacing: -0.5px;
}

.close-btn {
  background: #2C2C2E;
  border: none;
  color: white;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.2rem;
  cursor: pointer;
  transition: background 0.2s;
}

.close-btn:hover {
  background: #3A3A3C;
}

.types-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  padding: 32px 40px;
}

.type-card {
  background-color: #1C1C1E;
  border-radius: 20px;
  padding: 32px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  border: 2px solid #2C2C2E;
}

.type-card:hover, .type-card.active {
  background-color: #1C1C1E;
  border-color: #0A84FF;
  transform: scale(1.05);
  box-shadow: 0 10px 20px rgba(10, 132, 255, 0.15);
}

.type-icon-svg {
  width: 72px;
  height: 72px;
  margin-bottom: 24px;
  color: white;
}

.type-card:hover .type-icon-svg, .type-card.active .type-icon-svg {
  color: #0A84FF;
}

.type-label {
  color: white;
  font-size: 1rem;
  font-weight: 700;
  text-align: center;
}

.description-area {
  padding: 0 40px 32px;
  min-height: 180px;
}

.description-content, .description-placeholder {
  background-color: #1C1C1E;
  border: 2px solid #2C2C2E;
  border-radius: 16px;
  padding: 24px;
  color: #8E8E93;
  font-size: 1.1rem;
  line-height: 1.5;
  text-align: center;
}

.description-content p {
  margin: 0;
  color: #E5E5EA;
  font-weight: 500;
}

.description-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-style: italic;
  font-size: 1rem;
}

.selector-footer {
  padding: 24px 40px;
  background-color: #1C1C1E;
  border-top: 2px solid #2C2C2E;
  display: flex;
  justify-content: flex-end;
}

.cancel-btn {
  background: #2C2C2E;
  border: none;
  color: white;
  padding: 12px 24px;
  border-radius: 12px;
  cursor: pointer;
  font-weight: 700;
  font-size: 1rem;
  transition: background 0.2s;
}

.cancel-btn:hover {
  background: #3A3A3C;
}
</style>
