<template>
  <div class="coordinate-picker">
    <div 
      class="image-container" 
      @click="handleImageClick"
      ref="imageContainer"
    >
      <img v-if="imageUrl" :src="imageUrl" @load="onImageLoad" crossorigin="anonymous" />
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
        <div class="hit-area" :style="getHitAreaStyle(target)"></div>
        <div class="marker-point">
           <span class="target-number">{{ index + 1 }}</span>
        </div>
      </div>
    </div>
    
    <!-- Target list -->
    <div class="targets-list">
      <div class="targets-header">
         <h4>Targets ({{ targets.length }})</h4>
         <button class="add-btn" @click.prevent="startPlacing" :disabled="isPlacing">
            {{ isPlacing ? 'Click on image...' : '+ Add Target' }}
         </button>
      </div>
      
      <div v-if="targets.length === 0" class="empty-targets">
        No targets set. Click "+ Add Target" to begin.
      </div>
      
      <div v-for="(target, index) in targets" :key="target.id" class="target-item" :class="{ selected: selectedTargetId === target.id }" @click="selectTarget(target.id)">
        <div class="target-index">{{ index + 1 }}</div>
        <div class="target-inputs">
           <input type="text" v-model="target.label" class="target-input label-input" placeholder="e.g. Modern Sneakers" @input="emitUpdate" />
           <div class="target-params">
             <label>Radius %:</label>
             <input type="number" v-model.number="target.radius" class="target-input prop-input" min="1" max="50" step="0.5" @input="emitUpdate" />
           </div>
        </div>
        <button class="remove-btn" @click.stop="removeTarget(target.id)">✕</button>
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

const apiUrl = import.meta.env.VITE_API_URL || ''
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
  gap: 16px;
  background: #1C1C1E;
  border-radius: 8px;
  padding: 12px;
  border: 1px dashed #3A3A3C;
  margin-top: 12px;
}

.image-container {
  position: relative;
  width: 100%;
  max-width: 600px;
  margin: 0 auto;
  border-radius: 6px;
  overflow: hidden;
  cursor: crosshair;
  background: #000;
}

.image-container img {
  width: 100%;
  height: auto;
  display: block;
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
  width: 24px;
  height: 24px;
  background: rgba(255, 59, 48, 0.9);
  border: 2px solid white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  box-shadow: 0 2px 4px rgba(0,0,0,0.5);
  transition: transform 0.2s, background 0.2s;
}

.target-marker.active .marker-point {
  background: #32D74B;
  transform: translate(-50%, -50%) scale(1.2);
}

.target-number {
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.hit-area {
  position: absolute;
  border: 2px dashed rgba(255, 255, 255, 0.5);
  border-radius: 50%;
  background: rgba(255, 59, 48, 0.2);
  z-index: 1;
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.target-marker.active .hit-area {
  border-color: rgba(50, 215, 75, 0.8);
  background: rgba(50, 215, 75, 0.3);
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
  font-size: 14px;
}

.add-btn {
  background: rgba(50, 215, 75, 0.2);
  color: #32D74B;
  border: 1px solid rgba(50, 215, 75, 0.4);
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
  font-size: 12px;
}
.add-btn:disabled {
  background: #3A3A3C;
  color: #8E8E93;
  border-color: #3A3A3C;
}

.empty-targets {
  text-align: center;
  padding: 20px;
  color: #8E8E93;
  font-size: 13px;
  background: #232325;
  border-radius: 6px;
}

.targets-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.target-item {
  display: flex;
  align-items: center;
  background: #232325;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #3A3A3C;
  gap: 12px;
  cursor: pointer;
}

.target-item.selected {
  border-color: #32D74B;
  background: #1C1C1E;
}

.target-index {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #3A3A3C;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
  color: #fff;
  flex-shrink: 0;
}

.target-item.selected .target-index {
  background: #32D74B;
}

.target-inputs {
  flex: 1;
  display: flex;
  gap: 12px;
  align-items: center;
}

.target-input {
  background: #1C1C1E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 6px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.label-input {
  flex: 2;
  min-width: 0;
}

.target-params {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #8E8E93;
  flex-shrink: 0;
}

.prop-input {
  width: 60px;
}

.remove-btn {
  background: none;
  border: none;
  color: #FF3B30;
  cursor: pointer;
  opacity: 0.7;
  font-size: 16px;
  flex-shrink: 0;
}

.remove-btn:hover {
  opacity: 1;
}

.loading-placeholder {
  width: 100%;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #8E8E93;
  font-style: italic;
}
</style>
