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
      :style="transformStyle"
    >
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const container = ref(null)

const scale = ref(1)
const translateX = ref(0)
const translateY = ref(0)

// Touch state
const initialDistance = ref(0)
const initialScale = ref(1)

// Pan state
const isDragging = ref(false)
const lastClientX = ref(0)
const lastClientY = ref(0)

const transformStyle = computed(() => {
  return {
    transform: `translate(${translateX.value}px, ${translateY.value}px) scale(${scale.value})`,
    transformOrigin: 'center center' // or dynamically based on pinch center
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
    const newScale = (newDistance / initialDistance.value) * initialScale.value
    scale.value = Math.min(Math.max(newScale, 1), 5) // clamp 1x to 5x
    
    // Auto-reset translation if returning to 1x
    if (scale.value === 1) {
      translateX.value = 0
      translateY.value = 0
    }
  } else if (e.touches.length === 1 && isDragging.value && scale.value > 1) {
    e.preventDefault() // prevent scrolling
    const deltaX = e.touches[0].clientX - lastClientX.value
    const deltaY = e.touches[0].clientY - lastClientY.value
    
    translateX.value += deltaX
    translateY.value += deltaY
    
    lastClientX.value = e.touches[0].clientX
    lastClientY.value = e.touches[0].clientY
  }
}

const handleTouchEnd = (e) => {
  isDragging.value = false
}

// Mouse wheel zoom
const handleWheel = (e) => {
  // Only allow zooming over the image
  e.preventDefault()
  
  const delta = e.deltaY > 0 ? 0.9 : 1.1
  const newScale = Math.min(Math.max(scale.value * delta, 1), 5)
  
  scale.value = newScale
  if (scale.value === 1) {
    translateX.value = 0
    translateY.value = 0
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
    
    translateX.value += deltaX
    translateY.value += deltaY
    
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
