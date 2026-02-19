<template>
  <div class="image-options-editor">
    <div class="options-header">
      <label>Select the correct image (2-4 options)</label>
      <button 
        v-if="imageOptions.length < 4" 
        @click="addOption"
        class="add-option-btn"
      >
        + Add Image
      </button>
    </div>
    
    <div class="image-options-grid">
      <div 
        v-for="(option, idx) in imageOptions" 
        :key="option.id"
        class="image-option-card"
        :class="{ 'is-correct': correctIndex === idx }"
      >
        <div class="option-header">
          <span class="option-number">{{ idx + 1 }}</span>
          <label class="correct-radio">
            <input 
              type="radio" 
              :value="idx" 
              :checked="correctIndex === idx"
              @change="updateCorrectIndex(idx)"
            />
            <span class="radio-label">Correct</span>
          </label>
          <button 
            v-if="imageOptions.length > 2" 
            @click="removeOption(idx)"
            class="remove-btn"
            title="Remove option"
          >
            ✕
          </button>
        </div>
        
        <ImageUploader
          v-model="option.imageId"
          :event-id="eventId"
          :module-id="moduleId"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits, computed } from 'vue'
import ImageUploader from './ImageUploader.vue'

const props = defineProps({
  modelValue: {
    type: Array,
    required: true
  },
  correctIndex: {
    type: Number,
    required: true
  },
  eventId: {
    type: String,
    required: true
  },
  moduleId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:modelValue', 'update:correctIndex'])

const imageOptions = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const updateCorrectIndex = (idx) => {
  emit('update:correctIndex', idx)
}

const addOption = () => {
  const newOption = {
    id: Date.now().toString(), // Simple unique ID
    imageId: '',
    label: ''
  }
  // Create a new array to ensure reactivity
  const newOptions = [...props.modelValue, newOption]
  emit('update:modelValue', newOptions)
}

const removeOption = (idx) => {
  const newOptions = [...props.modelValue]
  newOptions.splice(idx, 1)
  emit('update:modelValue', newOptions)
  
  // Adjust correct index if needed
  if (props.correctIndex === idx) {
    emit('update:correctIndex', 0)
  } else if (props.correctIndex > idx) {
    emit('update:correctIndex', props.correctIndex - 1)
  }
}
</script>

<style scoped>
.image-options-editor {
  margin-top: 1rem;
  border-top: 1px solid #3A3A3C;
  padding-top: 1rem;
}

.options-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.options-header label {
  font-weight: 600;
  color: #8E8E93;
}

.add-option-btn {
  background: #32D74B;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
}

.add-option-btn:hover {
  background: #28c03d;
}

.image-options-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.image-option-card {
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  padding: 1rem;
  background: #1C1C1E;
  transition: all 0.2s;
}

.image-option-card.is-correct {
  border-color: #32D74B;
  background: rgba(50, 215, 75, 0.1);
  box-shadow: 0 0 5px rgba(50, 215, 75, 0.3);
}

.option-header {
  display: flex;
  align-items: center;
  margin-bottom: 0.5rem;
  gap: 0.5rem;
}

.option-number {
  font-weight: bold;
  background: #2C2C2E;
  color: #fff;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 0.8rem;
}

.correct-radio {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  cursor: pointer;
  flex: 1;
}

.radio-label {
  font-size: 0.9rem;
  color: #fff;
}

.remove-btn {
  background: none;
  border: none;
  color: #8E8E93;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0;
  line-height: 1;
}

.remove-btn:hover {
  color: #FF453A;
}
</style>
