<template>
  <div 
    class="slider-component"
    :class="{ 'vertical': orientation === 'vertical', 'horizontal': orientation === 'horizontal' }"
    ref="sliderContainer"
  >
    <!-- Fixed Value Display -->
    <div class="fixed-value-display" :class="orientation">
      <span class="value-text">{{ formattedValue }}</span>
      <span v-if="unit" class="unit-text">{{ unit }}</span>
    </div>

    <!-- Track -->
    <div class="slider-track" ref="track">
      <!-- Filled portion (from min to current value) -->
      <div 
        class="slider-fill"
        :style="{ [fillProperty]: '0%', [sizeProperty]: fillPercentage + '%' }"
      ></div>
      
      <!-- Correct answer marker (shown in results) -->
      <div 
        v-if="showCorrectMarker && correctValue !== null"
        class="correct-marker"
        :style="{ [fillProperty]: correctPercentage + '%' }"
      >
        <div class="marker-label-container">
           <span class="marker-label">{{ formatNumber(correctValue) }}{{ unit }}</span>
        </div>
      </div>
    </div>
    
    <!-- Thumb -->
    <div 
      class="slider-thumb"
      :style="{ [fillProperty]: fillPercentage + '%' }"
      :class="{ 'dragging': isDragging }"
    >
      <div class="thumb-grip" :class="orientation">
        <div class="grip-line"></div>
      </div>
    </div>
    
    <!-- Min/Max Labels -->
    <div class="slider-labels">
      <span class="min-label">{{ formatNumber(min) }}{{ unit }}</span>
      <span class="max-label">{{ formatNumber(max) }}{{ unit }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps({
  min: { type: Number, default: 0 },
  max: { type: Number, default: 100 },
  step: { type: Number, default: 1 },
  modelValue: { type: Number, default: 50 },
  unit: { type: String, default: '' },
  decimalPlaces: { type: Number, default: 0 },
  orientation: { type: String, default: 'horizontal' }, // 'horizontal' | 'vertical'
  readonly: { type: Boolean, default: false },
  showCorrectMarker: { type: Boolean, default: false },
  correctValue: { type: Number, default: null }
})

const emit = defineEmits(['update:modelValue', 'change', 'drag-start', 'drag-end'])

const sliderContainer = ref(null)
const track = ref(null)
const isDragging = ref(false)

const formatNumber = (num) => {
    if (num === null || num === undefined) return ''
    return Number(num).toFixed(props.decimalPlaces).replace(/\.0+$/, '')
}

const fillPercentage = computed(() => {
  const range = props.max - props.min
  if (range === 0) return 0
  return ((props.modelValue - props.min) / range) * 100
})

const correctPercentage = computed(() => {
  if (props.correctValue === null) return 0
  const range = props.max - props.min
  if (range === 0) return 0
  return ((props.correctValue - props.min) / range) * 100
})

const fillProperty = computed(() => {
  return props.orientation === 'vertical' ? 'bottom' : 'left'
})

const sizeProperty = computed(() => {
  return props.orientation === 'vertical' ? 'height' : 'width'
})

const formattedValue = computed(() => {
  return props.modelValue.toFixed(props.decimalPlaces)
})

// Touch/Mouse handling
const handleStart = (event) => {
  if (props.readonly) return
  isDragging.value = true
  emit('drag-start')
  updateValue(event)
  document.addEventListener('mousemove', handleMove)
  document.addEventListener('mouseup', handleEnd)
  document.addEventListener('touchmove', handleMove, { passive: false })
  document.addEventListener('touchend', handleEnd)
}

const handleMove = (event) => {
  if (!isDragging.value) return
  event.preventDefault() // Prevent scrolling on touch
  updateValue(event)
}

const handleEnd = () => {
  if (!isDragging.value) return
  isDragging.value = false
  emit('drag-end')
  emit('change', props.modelValue)
  document.removeEventListener('mousemove', handleMove)
  document.removeEventListener('mouseup', handleEnd)
  document.removeEventListener('touchmove', handleMove)
  document.removeEventListener('touchend', handleEnd)
}

const updateValue = (event) => {
  if (!track.value) return
  
  const rect = track.value.getBoundingClientRect()
  const clientX = event.touches ? event.touches[0].clientX : event.clientX
  const clientY = event.touches ? event.touches[0].clientY : event.clientY
  
  let position
  if (props.orientation === 'vertical') {
     // For vertical, 0 is at bottom, 1 is at top
     // clientY increases downwards. 
     // rect.bottom is the y-coordinate of the bottom edge
     position = (rect.bottom - clientY) / rect.height
  } else {
    position = (clientX - rect.left) / rect.width
  }
  
  // Clamp position 0..1
  position = Math.max(0, Math.min(1, position))
  
  let value = props.min + (position * (props.max - props.min))
  
  // Apply step if specified
  if (props.step > 0) {
    const steps = Math.round(value / props.step)
    value = steps * props.step
  }
  
  // Clamp to range (fix float precision issues potentially)
  value = Math.max(props.min, Math.min(props.max, value))

  // Round to decimal places
  const pow = Math.pow(10, props.decimalPlaces)
  value = Math.round(value * pow) / pow
  
  emit('update:modelValue', value)
}

onMounted(() => {
  if (track.value) {
    track.value.addEventListener('mousedown', handleStart)
    track.value.addEventListener('touchstart', handleStart, { passive: false })
  }
})

onUnmounted(() => {
  if (track.value) {
      track.value.removeEventListener('mousedown', handleStart)
      track.value.removeEventListener('touchstart', handleStart)
  }
  document.removeEventListener('mousemove', handleMove)
  document.removeEventListener('mouseup', handleEnd)
  document.removeEventListener('touchmove', handleMove)
  document.removeEventListener('touchend', handleEnd)
})
</script>

<style scoped>
.slider-component {
  position: relative;
  width: 100%;
  padding: 20px 0;
  user-select: none;
  touch-action: none;
}

.slider-component.horizontal {
  height: 120px;
}

.slider-component.vertical {
  width: 120px;
  height: 100%;
  padding: 0 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.slider-track {
  position: relative;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  cursor: pointer;
}

/* Expand touch/click area for the track */
.slider-track::before {
  content: '';
  position: absolute;
  top: -30px;
  bottom: -30px;
  left: -20px;
  right: -20px;
  background: transparent;
  z-index: -1;
}

.horizontal .slider-track {
  height: 12px;
  width: 100%;
}

.vertical .slider-track {
  width: 12px;
  height: 100%;
  margin: 0 auto;
}

.slider-fill {
  position: absolute;
  background: #0A84FF;
  border-radius: 8px;
}

.horizontal .slider-fill {
  height: 100%;
  left: 0;
}

.vertical .slider-fill {
  width: 100%;
  bottom: 0;
}

.slider-thumb {
  position: absolute;
  width: 64px;
  height: 64px;
  background: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  transform: translate(-50%, 50%); /* Center on the point */
  transition: transform 0.1s;
  z-index: 5;
}

/* Expand touch area around thumb */
.slider-thumb::after {
  content: '';
  position: absolute;
  top: -30px;
  bottom: -30px;
  left: -30px;
  right: -30px;
  border-radius: 50%;
  background: transparent;
}

.horizontal .slider-thumb {
  top: 50%;
  left: 0; 
  transform: translate(-50%, -50%);
}

.vertical .slider-thumb {
  left: 50%;
  bottom: 0;
  transform: translate(-50%, 50%);
}

.slider-thumb.dragging {
  cursor: grabbing;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  transform: translate(-50%, -50%) scale(1.1);
}

.vertical .slider-thumb.dragging {
  transform: translate(-50%, 50%) scale(1.1);
}

/* Grip handle lines inside thumb */
.thumb-grip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  pointer-events: none;
}
.thumb-grip.horizontal {
  flex-direction: row;
}
.thumb-grip.horizontal::before, .thumb-grip.horizontal::after, .thumb-grip.horizontal .grip-line {
  content: '';
  width: 3px;
  height: 18px;
  background-color: #c7c7cc;
  border-radius: 2px;
  display: block;
}
.thumb-grip.vertical {
  flex-direction: column;
}
.thumb-grip.vertical::before, .thumb-grip.vertical::after, .thumb-grip.vertical .grip-line {
  content: '';
  width: 18px;
  height: 3px;
  background-color: #c7c7cc;
  border-radius: 2px;
  display: block;
}

/* Fixed Value Display */
.fixed-value-display {
  display: flex;
  justify-content: center;
  align-items: baseline;
  margin-bottom: 24px;
}

.fixed-value-display.vertical {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  margin-bottom: 0;
  white-space: nowrap;
}

.value-text {
  font-size: 42px;
  font-weight: 800;
  color: white;
  text-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.unit-text {
  font-size: 20px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  margin-left: 4px;
}

.correct-marker {
  position: absolute;
  width: 4px;
  height: 24px;
  background: #32D74B;
  z-index: 1;
}

.horizontal .correct-marker {
  height: 24px;
  top: 50%;
  transform: translate(-50%, -50%);
}

.vertical .correct-marker {
  width: 24px;
  height: 4px;
  left: 50%;
  transform: translate(-50%, 50%);
}

.marker-label-container {
    position: absolute;
    background: #32D74B;
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
}

.horizontal .marker-label-container {
    top: -30px;
    left: 50%;
    transform: translateX(-50%);
}

.vertical .marker-label-container {
    right: -40px;
    top: 50%;
    transform: translateY(-50%);
}

.marker-label {
  font-size: 12px;
  font-weight: bold;
  color: #000;
}

.slider-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.7);
  width: 100%;
}

.vertical .slider-labels {
    flex-direction: column-reverse; /* Min at bottom */
    height: 100%;
    position: absolute;
    left: 0;
    top: 0;
    margin: 0;
    padding: 20px 0;
    align-items: center;
    pointer-events: none;
}

.vertical .min-label {
    position: absolute;
    bottom: -10px;
    left: 50%;
    transform: translateX(-50%);
}

.vertical .max-label {
    position: absolute;
    top: -10px;
    left: 50%;
    transform: translateX(-50%);
}
</style>
