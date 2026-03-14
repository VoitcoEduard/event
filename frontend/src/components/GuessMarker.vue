<template>
  <div 
    class="guess-marker"
    :class="{ 
       placed: isPlaced, 
       correct: isCorrect === true, 
       incorrect: isCorrect === false,
       active: isActive
    }"
    :style="markerStyle"
    @touchstart="startDrag"
    @mousedown="startDrag"
  >
    <div class="marker-circle">
      <span class="marker-number" v-if="!isCorrect && isCorrect !== false">{{ markerNumber }}</span>
      <span class="marker-icon" v-if="isCorrect === true">✓</span>
      <span class="marker-icon" v-if="isCorrect === false">✕</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  markerNumber: { type: Number, required: true },
  xPercent: { type: Number, default: 50 },
  yPercent: { type: Number, default: 90 }, // default at bottom
  isPlaced: { type: Boolean, default: false },
  isCorrect: { type: Boolean, default: null }, // true/false/null
  isActive: { type: Boolean, default: false },
  containerRect: { type: Object, default: () => null }
})

const emit = defineEmits(['update:position', 'interaction-start'])

const markerStyle = computed(() => {
  return {
    left: `${props.xPercent}%`,
    top: `${props.yPercent}%`,
  }
})

// Dragging logic handling directly using window listeners to escape boundaries
const isDragging = ref(false)

const startDrag = (e) => {
  if (props.isCorrect !== null) return // Finalized
  
  e.preventDefault()
  e.stopPropagation() // Don't trigger pan
  isDragging.value = true
  emit('interaction-start', props.markerNumber)
  
  if (e.type === 'touchstart') {
     window.addEventListener('touchmove', handleDrag, { passive: false })
     window.addEventListener('touchend', endDrag)
  } else {
     window.addEventListener('mousemove', handleDrag)
     window.addEventListener('mouseup', endDrag)
  }
}

const handleDrag = (e) => {
  if (!isDragging.value || !props.containerRect) return
  e.preventDefault() // prevent scroll on touch
  
  const clientX = e.touches ? e.touches[0].clientX : e.clientX
  const clientY = e.touches ? e.touches[0].clientY : e.clientY
  
  const rect = props.containerRect
  
  // Calculate raw position
  let x = clientX - rect.left
  let y = clientY - rect.top
  
  // Clamp to container
  x = Math.max(0, Math.min(x, rect.width))
  y = Math.max(0, Math.min(y, rect.height))
  
  const xPercent = (x / rect.width) * 100
  const yPercent = (y / rect.height) * 100
  
  emit('update:position', { xPercent, yPercent })
}

const endDrag = (e) => {
  isDragging.value = false
  window.removeEventListener('mousemove', handleDrag)
  window.removeEventListener('mouseup', endDrag)
  window.removeEventListener('touchmove', handleDrag)
  window.removeEventListener('touchend', endDrag)
}
</script>

<style scoped>
.guess-marker {
  position: absolute;
  transform: translate(-50%, -50%);
  z-index: 100;
  touch-action: none;
  cursor: grab;
  /* subtle drop shadow for unplaced state */
  filter: drop-shadow(0 4px 6px rgba(0,0,0,0.4));
  transition: transform 0.1s;
}

.guess-marker:active {
  cursor: grabbing;
}

.marker-circle {
  width: 32px;
  height: 32px;
  background: #007AFF; /* standard blue for player guess */
  border: 3px solid white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0,0,0,0.5);
  transition: all 0.2s ease;
}

.marker-number {
  color: white;
  font-weight: 800;
  font-size: 14px;
}

.marker-icon {
  color: white;
  font-weight: 800;
  font-size: 16px;
}

/* Modifiers */
.guess-marker.active {
   transform: translate(-50%, -50%) scale(1.2);
   z-index: 101;
}

.guess-marker.active .marker-circle {
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.4);
}

.guess-marker.placed .marker-circle {
  background: rgba(0, 122, 255, 0.9);
}

.guess-marker.correct .marker-circle {
  background: #32D74B;
  border-color: #fff;
}

.guess-marker.incorrect .marker-circle {
  background: #FF3B30;
  border-color: #fff;
}
</style>
