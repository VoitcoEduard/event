<template>
  <div 
    class="zoom-pan-container"
    ref="container"
    @touchstart="handleTouchStart"
    @touchmove="handleTouchMove"
    @touchend="handleTouchEnd"
    @mousedown="handleMouseDown"
    @mousemove="handleMouseMove"
    @mouseup="handleMouseUp"
    @mouseleave="handleMouseUp"
    @wheel="handleWheel"
  >
    <div 
      class="transform-wrapper"
      ref="wrapper"
      :style="transformStyle"
    >
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const container = ref(null)
const wrapper = ref(null)

const scale = ref(1)
const ZOOM_STEPS = [1, 5]
const translateX = ref(0)
const translateY = ref(0)

// Touch state
const initialDistance = ref(0)
const initialScale = ref(1)

// Pan state
const isDragging = ref(false)
const lastClientX = ref(0)
const lastClientY = ref(0)

const setZoomStep = (step) => {
  const newStep = ZOOM_STEPS.includes(step) ? step : 1
  scale.value = newStep
  if (newStep === 1) {
    translateX.value = 0
    translateY.value = 0
  }
}

const clampTranslation = (tx, ty) => {
  if (!container.value || !wrapper.value) return { tx, ty }
  
  const C = container.value.clientWidth
  const H = container.value.clientHeight
  const w = wrapper.value.clientWidth
  const h = wrapper.value.clientHeight
  
  const V = w * scale.value
  const Vh = h * scale.value
  
  // Math.max ensures that if the scaled dimension is smaller than the container,
  // it doesn't allow panning in that direction (keeps it centered).
  const boundX = Math.max(0, (V - C) / 2)
  const boundY = Math.max(0, (Vh - H) / 2)
  
  return {
    tx: Math.min(Math.max(tx, -boundX), boundX),
    ty: Math.min(Math.max(ty, -boundY), boundY)
  }
}

const transformStyle = computed(() => {
  return {
    transform: `translate(${translateX.value}px, ${translateY.value}px) scale(${scale.value})`,
    transformOrigin: 'center center',
    transition: isDragging.value ? 'none' : 'transform 0.1s ease-out'
  }
})

const getDistance = (touches) => {
  const dx = touches[0].clientX - touches[1].clientX
  const dy = touches[0].clientY - touches[1].clientY
  return Math.sqrt(dx * dx + dy * dy)
}

const handleTouchStart = (e) => {
  if (e.touches.length === 2) {
    initialDistance.value = getDistance(e.touches)
    initialScale.value = scale.value
    isDragging.value = false
  } else if (e.touches.length === 1 && scale.value > 1) {
    // Start panning if zoomed in
    isDragging.value = true
    lastClientX.value = e.touches[0].clientX
    lastClientY.value = e.touches[0].clientY
  }
}

const handleTouchMove = (e) => {
  if (e.touches.length === 2) {
    e.preventDefault() // prevent standard pinch zoom
    const newDistance = getDistance(e.touches)
    const ratio = newDistance / initialDistance.value
    // Pinch out to 5x, pinch in to 1x
    if (ratio > 1.5 && scale.value === 1) {
      setZoomStep(5)
    } else if (ratio < 0.7 && scale.value === 5) {
      setZoomStep(1)
    }
  } else if (e.touches.length === 1 && isDragging.value && scale.value > 1) {
    e.preventDefault() // prevent scrolling
    const deltaX = e.touches[0].clientX - lastClientX.value
    const deltaY = e.touches[0].clientY - lastClientY.value
    
    const { tx, ty } = clampTranslation(translateX.value + deltaX, translateY.value + deltaY)
    
    translateX.value = tx
    translateY.value = ty
    
    lastClientX.value = e.touches[0].clientX
    lastClientY.value = e.touches[0].clientY
  }
}

const handleTouchEnd = (e) => {
  isDragging.value = false
}

// Mouse wheel zoom - toggles between 1x and 5x
const handleWheel = (e) => {
  e.preventDefault()
  // Toggle between 1x and 5x
  if (scale.value === 1) {
    setZoomStep(5)
  } else {
    setZoomStep(1)
  }
}

// Mouse panning
const handleMouseDown = (e) => {
  if (scale.value > 1) {
    e.preventDefault()
    isDragging.value = true
    lastClientX.value = e.clientX
    lastClientY.value = e.clientY
  }
}

const handleMouseMove = (e) => {
  if (isDragging.value && scale.value > 1) {
    e.preventDefault()
    const deltaX = e.clientX - lastClientX.value
    const deltaY = e.clientY - lastClientY.value
    
    const { tx, ty } = clampTranslation(translateX.value + deltaX, translateY.value + deltaY)
    
    translateX.value = tx
    translateY.value = ty
    
    lastClientX.value = e.clientX
    lastClientY.value = e.clientY
  }
}

const handleMouseUp = () => {
  isDragging.value = false
}
</script>

<style scoped>
.zoom-pan-container {
  width: 100%;
  height: 100%;
  overflow: hidden;
  position: relative;
  touch-action: none; /* disables browser panning/zooming */
  display: flex;
  align-items: center;
  justify-content: center;
}

.transform-wrapper {
  transition: transform 0.1s cubic-bezier(0.2, 0, 0.2, 1);
  will-change: transform;
  display: inline-block; /* tight wrap around content */
}
</style>
