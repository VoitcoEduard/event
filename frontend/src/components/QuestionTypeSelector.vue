<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="emit('close')">
      <div class="selector-container">
        <div class="selector-header">
          <h3>{{ $t('quiz_config.question_editor.selector.title') }}</h3>
          <button class="close-btn" @click="emit('close')">✕</button>
        </div>

        <div class="types-grid">
          <div 
            v-for="type in questionTypes" 
            :key="type.id"
            class="type-card"
            :class="{ active: hoveredType === type.id }"
            @mouseenter="hoveredType = type.id"
            @click="emit('select', type.id)"
          >
            <div class="type-icon">{{ type.icon }}</div>
            <div class="type-label">{{ $t(`quiz_config.question_editor.types.${type.id}`) }}</div>
          </div>
        </div>

        <div class="description-area">
          <div v-if="hoveredType" class="description-content">
             <p>{{ $t(`quiz_config.question_editor.type_descriptions.${hoveredType}`) }}</p>
          </div>
          <div v-else class="description-placeholder">
            {{ $t('quiz_config.question_editor.selector.description_hint') }}
          </div>
        </div>

        <div class="selector-footer">
          <button class="cancel-btn" @click="emit('close')">
            {{ $t('quiz_config.question_editor.selector.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false }
})

const emit = defineEmits(['close', 'select'])

const hoveredType = ref(null)

const questionTypes = [
  { id: 'classic', icon: '📝' },
  { id: 'multi_select', icon: '✅' },
  { id: 'true_false', icon: '⚖️' },
  { id: 'range', icon: '↔️' },
  { id: 'number', icon: '🔢' },
  { id: 'correct_picture', icon: '🖼️' },
  { id: 'wrong_pictures', icon: '🎯' }
]
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

.selector-container {
  background-color: #111112;
  width: 95%;
  max-width: 800px;
  border-radius: 24px;
  overflow: hidden;
  border: 2px solid #2C2C2E;
  display: flex;
  flex-direction: column;
  box-shadow: 0 30px 60px rgba(0,0,0,0.6);
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

.type-icon {
  font-size: 3rem;
  margin-bottom: 16px;
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
