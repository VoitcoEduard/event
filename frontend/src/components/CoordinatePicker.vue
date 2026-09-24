<template>
  <div class="coordinate-picker">
    <div 
      class="image-container" 
      @click="handleImageClick"
      ref="imageContainer"
    >
      <img v-if="imageUrl" :src="imageUrl" @load="onImageLoad" crossorigin="anonymous" decoding="async" />
      <div v-else class="loading-placeholder">Loading...</div>
      
      <!-- Existing targets shown as markers -->
      <div 
        v-for="(target, index) in targets" 
        :key="target.id"
        class="target-marker"
        :class="{ active: selectedTargetId === target.id }"
        :style="getTargetStyle(target)"
        @click.stop="selectTarget(target.id)"
      >
        <div class="marker-point">
           <span class="target-number">{{ index + 1 }}</span>
        </div>
      </div>

      <!-- Hit areas as separate layer for correct percentage sizing -->
      <div
        v-for="target in targets"
        :key="'hit-' + target.id"
        class="hit-area"
        :class="{ active: selectedTargetId === target.id }"
        :style="[getTargetStyle(target), getHitAreaStyle(target)]"
      ></div>
    </div>
    
    <!-- Target list -->
    <div class="targets-list-container">
      <div class="targets-header">
         <h4>{{ $t('quiz_config.question_editor.coordinate_picker.targets_title') }} ({{ targets.length }})</h4>
         <button class="add-btn" @click.prevent="startPlacing" :disabled="isPlacing">
            {{ isPlacing ? $t('quiz_config.question_editor.coordinate_picker.placing_hint') : $t('quiz_config.question_editor.coordinate_picker.add_target') }}
         </button>
      </div>
      
      <div v-if="targets.length === 0" class="empty-targets">
        {{ $t('quiz_config.question_editor.coordinate_picker.no_targets') }}
      </div>
      
      <div class="targets-grid">
        <div v-for="(target, index) in targets" :key="target.id" class="target-row" :class="{ selected: selectedTargetId === target.id }" @click="selectTarget(target.id)">
          <div class="target-index">{{ index + 1 }}</div>
          <div class="target-inputs">
             <input 
                type="text" 
                v-model="target.label" 
                class="target-input label-input" 
                :placeholder="$t('quiz_config.question_editor.coordinate_picker.label_placeholder')" 
                @input="emitUpdate" 
             />
             <div class="target-params">
               <span>{{ $t('quiz_config.question_editor.coordinate_picker.radius_label') }}</span>
               <input type="number" v-model.number="target.radius" class="target-input prop-input" min="1" max="50" step="0.5" @input="emitUpdate" />
               <span>%</span>
             </div>
          </div>
          <button class="remove-btn-small" @click.stop="removeTarget(target.id)">✕</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  imageId: {
    type: String,
    required: true
  },
  targets: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:targets'])

const apiUrl = import.meta.env.VITE_API_URL ?? ''
const imageUrl = computed(() => {
  if (!props.imageId) return ''
  return `${apiUrl}/media/${props.imageId}/file`
})

const imageContainer = ref(null)
const isPlacing = ref(false)
const selectedTargetId = ref(null)

const localTargets = ref([...props.targets])

watch(() => props.targets, (newVal) => {
  localTargets.value = [...newVal]
}, { deep: true })

const emitUpdate = () => {
  emit('update:targets', localTargets.value)
}

const handleImageClick = (e) => {
  if (!isPlacing.value) return
  
  const rect = imageContainer.value.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  
  const width = rect.width
  const height = rect.height
  
  const xPercent = (x / width) * 100
  const yPercent = (y / height) * 100
  
  const newTarget = {
    id: Date.now().toString(),
    label: '',
    xPercent,
    yPercent,
    radius: 5 // Default 5% radius hit area
  }
  
  localTargets.value.push(newTarget)
  isPlacing.value = false
  selectedTargetId.value = newTarget.id
  emitUpdate()
}

const startPlacing = () => {
  isPlacing.value = true
  selectedTargetId.value = null
}

const selectTarget = (id) => {
  selectedTargetId.value = id
  isPlacing.value = false
}

const removeTarget = (id) => {
  localTargets.value = localTargets.value.filter(t => t.id !== id)
  if (selectedTargetId.value === id) selectedTargetId.value = null
  emitUpdate()
}

const getTargetStyle = (target) => {
  return {
    left: `${target.xPercent}%`,
    top: `${target.yPercent}%`
  }
}

const getHitAreaStyle = (target) => {
  return {
    width: `${target.radius * 2}%`,
    aspectRatio: '1',
    transform: 'translate(-50%, -50%)',
  }
}

const onImageLoad = () => {
    // optional image load handler
}
</script>

<style scoped>
.coordinate-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: #1C1C1E;
  border-radius: 8px;
}

.image-container {
  position: relative;
  width: 100%;
  max-width: 100%;
  border-radius: 8px;
  overflow: hidden;
  cursor: crosshair;
  background: #000;
  border: 1px solid #3A3A3C;
}

.image-container img {
  width: 100%;
  height: auto;
  display: block;
}

.targets-list-container {
  background: #232325;
  border-radius: 8px;
  border: 1px solid #3A3A3C;
  padding: 12px;
}

.targets-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.targets-header h4 {
  margin: 0;
  color: #E5E5EA;
  font-size: 13px;
  font-weight: 600;
}

.add-btn {
  background: #0A84FF;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  font-size: 12px;
  transition: all 0.2s;
}

.add-btn:hover:not(:disabled) {
  background: #007AFF;
}

.add-btn:disabled {
  background: #3A3A3C;
  color: #8E8E93;
}

.empty-targets {
  text-align: center;
  padding: 16px;
  color: #8E8E93;
  font-size: 13px;
}

.targets-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.target-row {
  display: flex;
  align-items: center;
  background: #1C1C1E;
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid #3A3A3C;
  gap: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.target-row:hover {
  border-color: #48484A;
}

.target-row.selected {
  border-color: #0A84FF;
  background: rgba(10, 132, 255, 0.05);
}

.target-index {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #3A3A3C;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: bold;
  color: #fff;
  flex-shrink: 0;
}

.target-row.selected .target-index {
  background: #0A84FF;
}

.target-inputs {
  flex: 1;
  display: flex;
  gap: 8px;
  align-items: center;
}

.target-input {
  background: #2C2C2E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.label-input {
  flex: 1;
  min-width: 0;
}

.target-params {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #8E8E93;
  flex-shrink: 0;
}

.prop-input {
  width: 50px;
  text-align: center;
}

.remove-btn-small {
  background: none;
  border: none;
  color: #8E8E93;
  cursor: pointer;
  font-size: 14px;
  padding: 4px;
  transition: color 0.2s;
}

.remove-btn-small:hover {
  color: #FF453A;
}

.target-marker {
  position: absolute;
  pointer-events: auto;
  cursor: pointer;
  z-index: 10;
}

.marker-point {
  position: absolute;
  transform: translate(-50%, -50%);
  width: 20px;
  height: 20px;
  background: rgba(255, 59, 48, 0.9);
  border: 1.5px solid white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  box-shadow: 0 2px 4px rgba(0,0,0,0.5);
  transition: transform 0.2s, background 0.2s;
}

.target-marker.active .marker-point {
  background: #0A84FF;
  transform: translate(-50%, -50%) scale(1.1);
}

.target-number {
  color: white;
  font-size: 10px;
  font-weight: bold;
}

.hit-area {
  position: absolute;
  border: 1.5px dashed rgba(255, 255, 255, 0.4);
  border-radius: 50%;
  background: rgba(255, 59, 48, 0.15);
  z-index: 1;
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.hit-area.active {
  border-color: rgba(10, 132, 255, 0.6);
  background: rgba(10, 132, 255, 0.15);
  z-index: 2;
}

.loading-placeholder {
  width: 100%;
  height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #8E8E93;
  font-size: 13px;
}
</style>
