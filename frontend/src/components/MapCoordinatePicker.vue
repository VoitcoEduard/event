<template>
  <div class="map-picker-container" ref="container">
    <!-- The Image Layer (Pannable/Zoomable) -->
    <div 
      class="map-overflow-wrapper"
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
        class="image-transform-layer"
        :style="transformStyle"
      >
        <div class="image-wrapper" ref="imageElement">
          <img :src="imageUrl" @load="onImageLoad" draggable="false" decoding="async" />
          
          <!-- Dropped Markers -->
          <div 
            v-for="(marker, index) in localMarkers" 
            :key="index"
            class="dropped-marker"
            :style="{ left: marker.xPercent + '%', top: marker.yPercent + '%' }"
            @click.stop="removeMarker(index)"
          >
            <div class="marker-pulse"></div>
            <div class="marker-pin">
              <span class="marker-index">{{ index + 1 }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Fixed Center Selector (Google Maps style) - Only in placement mode -->
    <Transition name="scale">
      <div class="fixed-selector-overlay" v-if="isPlacementMode && !isMaxTargets">
        <div class="center-crosshair">
          <div class="crosshair-v"></div>
          <div class="crosshair-h"></div>
        </div>
        <div class="center-marker-floating">
          <div class="floating-pin-head">
            <div class="pin-inner"></div>
          </div>
          <div class="floating-pin-bottom"></div>
          <div class="floating-shadow"></div>
        </div>
      </div>
    </Transition>

    <!-- Placement Mode Indicator / Layer -->
    <Transition name="fade">
      <div class="placement-layer-overlay" v-if="isPlacementMode">
        <button class="exit-placement-btn" @click="isPlacementMode = false">
          <span>{{ $t('map_coordinate_picker.close_selection') }}</span>
          <AppIcon name="close" :size="16" />
        </button>
      </div>
    </Transition>

    <!-- UI Controls -->
    <div class="picker-controls">
      <div class="top-controls">
          <div class="target-counter" v-if="showTargetCount">
            <span class="count">{{ localMarkers.length }}</span> / {{ maxTargets }} 
            <span class="label">{{ $t('map_coordinate_picker.answers') }}</span>
          </div>
      </div>
      
      <div class="bottom-controls">
          <!-- Main Toggle / Add Action -->
          <button 
            v-if="!isPlacementMode"
            class="enter-placement-btn" 
            @click="isPlacementMode = true"
            :disabled="isMaxTargets"
          >
            <AppIcon class="btn-icon" name="pin" :size="18" />
            <span class="btn-text">{{ $t('map_coordinate_picker.place_answer') }}</span>
          </button>

          <button 
            v-else
            class="confirm-placement-btn" 
            @click="addCurrentPosition" 
            :disabled="isMaxTargets"
          >
            <AppIcon class="btn-icon" name="check" :size="18" />
            <span class="btn-text">{{ $t('map_coordinate_picker.drop_pin') }}</span>
          </button>
      </div>

      <div class="side-controls">
          <div class="zoom-controls">
            <button 
              v-for="step in ZOOM_STEPS" 
              :key="step"
              :class="{ active: scale === step }"
              @click="setZoomStep(step)"
            >
              {{ step }}x
            </button>
            <button @click="toggleZoom" class="toggle-btn">
              <AppIcon name="search" :size="18" />
            </button>
          </div>
      </div>
    </div>

    <!-- Mini Feedback Toast -->
    <Transition name="fade">
      <div v-if="showHint" class="interaction-hint">
        {{ $t('map_coordinate_picker.interaction_hint') }}
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import AppIcon from './AppIcon.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  imageUrl: { type: String, required: true },
  maxTargets: { type: Number, default: 3 },
  initialMarkers: { type: Array, default: () => [] },
  showTargetCount: { type: Boolean, default: false }
})

const emit = defineEmits(['update:markers'])

const container = ref(null)
const imageElement = ref(null)
const scale = ref(1)
const ZOOM_STEPS = [1, 5]
const translateX = ref(0)
const translateY = ref(0)
const localMarkers = ref([...props.initialMarkers])
const showHint = ref(true)
const isPlacementMode = ref(false)

// Stay in sync with the parent: it owns the marker list (resets it on new
// questions, replaces it with scored guesses on submit). Without this, pins
// from the previous round would carry over visually.
watch(() => props.initialMarkers, (newVal) => {
  if (newVal === localMarkers.value) return
  localMarkers.value = [...(Array.isArray(newVal) ? newVal : [])]
})

// Navigation state
const isDragging = ref(false)
const lastClientX = ref(0)
const lastClientY = ref(0)
const initialDistance = ref(0)
const initialScale = ref(1)

const isMaxTargets = computed(() => localMarkers.value.length >= props.maxTargets)

const transformStyle = computed(() => ({
  transform: `translate(-50%, -50%) translate(${translateX.value}px, ${translateY.value}px) scale(${scale.value})`,
  transition: isDragging.value ? 'none' : 'transform 0.1s ease-out'
}))

const onImageLoad = () => {
  setTimeout(() => {
    showHint.value = false
  }, 4000)
}

const addCurrentPosition = () => {
  if (isMaxTargets.value || !container.value || !imageElement.value) return
  
  // Calculate center of container
  const rect = container.value.getBoundingClientRect()
  const centerX = rect.width / 2
  const centerY = rect.height / 2

  // Get image dimensions (unscaled)
  const img = imageElement.value.querySelector('img')
  if (!img) return
  
  const imgNaturalWidth = img.naturalWidth
  const imgNaturalHeight = img.naturalHeight
  
  // Get image display dimensions (current scaled size)
  const imgDisplayRect = img.getBoundingClientRect()
  
  // Find where the container's center falls relative to the image's top-left in the viewport
  const centerRelToImgX = centerX + rect.left - imgDisplayRect.left
  const centerRelToImgY = centerY + rect.top - imgDisplayRect.top
  
  // Calculate percentage purely based on current display size
  const xPercent = (centerRelToImgX / imgDisplayRect.width) * 100
  const yPercent = (centerRelToImgY / imgDisplayRect.height) * 100

  if (xPercent >= 0 && xPercent <= 100 && yPercent >= 0 && yPercent <= 100) {
    localMarkers.value.push({
      xPercent,
      yPercent,
      isPlaced: true
    })

    // Optional: auto-exit placement after drop
    isPlacementMode.value = false

    emit('update:markers', localMarkers.value)
  }
}

const removeMarker = (index) => {
  localMarkers.value.splice(index, 1)
  emit('update:markers', localMarkers.value)
}

const setZoomStep = (step) => {
  const newStep = ZOOM_STEPS.includes(step) ? step : 1
  scale.value = newStep
  if (newStep === 1) {
    translateX.value = 0
    translateY.value = 0
  }
}

const toggleZoom = () => {
  setZoomStep(scale.value === 1 ? 5 : 1)
}

const adjustZoom = (direction) => {
  // Toggle between 1x and 5x
  if (scale.value === 1) {
    setZoomStep(5)
  } else {
    setZoomStep(1)
  }
}

// Interaction Handlers
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
  } else if (e.touches.length === 1) {
    isDragging.value = true
    lastClientX.value = e.touches[0].clientX
    lastClientY.value = e.touches[0].clientY
  }
}

const handleTouchMove = (e) => {
  if (e.touches.length === 2) {
    e.preventDefault()
    // Pinch toggles between 1x and 5x based on distance
    const newDistance = getDistance(e.touches)
    const ratio = newDistance / initialDistance.value
    // If pinching out (increasing), go to 5x; pinching in, go to 1x
    if (ratio > 1.5 && scale.value === 1) {
      setZoomStep(5)
    } else if (ratio < 0.7 && scale.value === 5) {
      setZoomStep(1)
    }
  } else if (e.touches.length === 1 && isDragging.value) {
    e.preventDefault()
    const deltaX = e.touches[0].clientX - lastClientX.value
    const deltaY = e.touches[0].clientY - lastClientY.value
    translateX.value += deltaX
    translateY.value += deltaY
    lastClientX.value = e.touches[0].clientX
    lastClientY.value = e.touches[0].clientY
  }
}

const handleTouchEnd = () => isDragging.value = false

const handleWheel = (e) => {
  e.preventDefault()
  // Toggle between 1x and 5x on wheel scroll
  adjustZoom()
}

const handleMouseDown = (e) => {
  isDragging.value = true
  lastClientX.value = e.clientX
  lastClientY.value = e.clientY
}

const handleMouseMove = (e) => {
  if (isDragging.value) {
    e.preventDefault()
    const deltaX = e.clientX - lastClientX.value
    const deltaY = e.clientY - lastClientY.value
    translateX.value += deltaX
    translateY.value += deltaY
    lastClientX.value = e.clientX
    lastClientY.value = e.clientY
  }
}

const handleMouseUp = () => isDragging.value = false

</script>

<style scoped>
.map-picker-container {
  position: relative;
  width: 100%;
  height: 100%; /* Take full height of parent */
  background: #000;
  overflow: hidden;
  touch-action: none;
}

.map-overflow-wrapper {
  width: 100%;
  height: 100%;
  cursor: grab;
  position: absolute;
  top: 0;
  left: 0;
}

.map-overflow-wrapper:active {
  cursor: grabbing;
}

.image-transform-layer {
  will-change: transform;
  position: absolute;
  top: 50%;
  left: 50%;
  /* We use the transform from state, plus the centering offset */
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-wrapper {
  position: relative;
  display: inline-block;
}

.image-wrapper img {
  display: block;
  max-width: none;
  width: 100vw; /* Default to screen width */
  height: auto;
  pointer-events: none;
  -webkit-user-drag: none;
}

/* Fixed Selector Styles */
.fixed-selector-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  z-index: 10;
}

.center-crosshair {
  position: absolute;
  width: 40px;
  height: 40px;
  opacity: 0.3;
}

.crosshair-v, .crosshair-h {
  position: absolute;
  background: white;
}

.crosshair-v { width: 1px; height: 100%; left: 50%; }
.crosshair-h { height: 1px; width: 100%; top: 50%; }

.center-marker-floating {
  position: relative;
  width: 44px;
  height: 44px;
  margin-top: -44px; /* offset to point at center */
  animation: float-pin 2s ease-in-out infinite;
}

@keyframes float-pin {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.floating-pin-head {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border: 3px solid white;
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3);
}

.pin-inner {
  width: 14px;
  height: 14px;
  background: white;
  border-radius: 50%;
}

.floating-shadow {
  position: absolute;
  bottom: -15px;
  left: 50%;
  width: 20px;
  height: 6px;
  background: rgba(0,0,0,0.4);
  filter: blur(4px);
  border-radius: 50%;
  transform: translateX(-50%) scale(var(--shadow-scale, 1));
  animation: float-shadow 2s ease-in-out infinite;
}

@keyframes float-shadow {
  0%, 100% { transform: translateX(-50%) scale(1); opacity: 0.4; }
  50% { transform: translateX(-50%) scale(0.6); opacity: 0.2; }
}

/* Dropped Markers */
.dropped-marker {
  position: absolute;
  transform: translate(-50%, -100%);
  z-index: 5;
  cursor: pointer;
}

.marker-pin {
  width: 32px;
  height: 32px;
  background: #ef4444;
  border: 2px solid white;
  border-radius: 50% 50% 50% 0;
  transform: rotate(-45deg);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 6px -1px rgba(0,0,0,0.2);
}

.marker-index {
  color: white;
  font-weight: bold;
  font-size: 14px;
  transform: rotate(45deg);
}

.marker-pulse {
  position: absolute;
  bottom: 0;
  left: 50%;
  width: 4px;
  height: 4px;
  background: #ef4444;
  border-radius: 50%;
  transform: translateX(-50%);
}

.marker-pulse::after {
  content: '';
  position: absolute;
  width: 40px;
  height: 40px;
  border: 2px solid #ef4444;
  border-radius: 50%;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  animation: marker-ripple 2s infinite;
}

@keyframes marker-ripple {
  0% { transform: translate(-50%, -50%) scale(0.2); opacity: 0.8; }
  100% { transform: translate(-50%, -50%) scale(1.5); opacity: 0; }
}

/* Controls Layer */
.picker-controls {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 200;
  display: flex;
  flex-direction: column;
}

.top-controls {
    padding: 1rem;
    pointer-events: none;
}

.bottom-controls {
    margin-top: auto;
    padding: 1.5rem;
    padding-bottom: 90px; /* Space for parent submit btn */
    display: flex;
    justify-content: center;
    pointer-events: none;
}

.side-controls {
    position: absolute;
    right: 1rem;
    top: 50%;
    transform: translateY(-50%);
    pointer-events: none;
}

.placement-layer-overlay {
    position: absolute;
    inset: 0;
    background: radial-gradient(circle at center, transparent 10%, rgba(0,0,0,0.3) 100%);
    pointer-events: none;
    z-index: 50;
    border: 2px solid rgba(59, 130, 246, 0.4);
    pointer-events: none;
}

.exit-placement-btn {
    position: absolute;
    top: 1.5rem;
    right: 1.5rem;
    pointer-events: auto;
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: white;
    padding: 0.6rem 1rem;
    border-radius: 9999px;
    font-size: 0.9rem;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
}

.enter-placement-btn, .confirm-placement-btn {
  pointer-events: auto;
  background: linear-gradient(to right, #2563eb, #3b82f6);
  color: white;
  border: none;
  padding: 1rem 2rem;
  border-radius: 9999px;
  font-weight: 800;
  font-size: 1.1rem;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-shadow: 0 10px 25px rgba(37, 99, 235, 0.4);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.confirm-placement-btn {
    background: linear-gradient(to right, #10b981, #059669);
    box-shadow: 0 10px 25px rgba(16, 185, 129, 0.4);
}

.enter-placement-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 20px 35px rgba(37, 99, 235, 0.5);
}

.confirm-placement-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 20px 35px rgba(16, 185, 129, 0.5);
}

.enter-placement-btn:active, .confirm-placement-btn:active {
  transform: translateY(0);
}

.enter-placement-btn:disabled {
  background: #3f3f46;
  opacity: 0.5;
  box-shadow: none;
}

.target-counter {
  pointer-events: auto;
  background: rgba(15, 15, 18, 0.8);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0.6rem 1rem;
  border-radius: 1rem;
  color: #a1a1aa;
  font-size: 0.9rem;
  font-weight: 600;
  display: inline-block;
}

.zoom-controls {
  pointer-events: auto;
  display: flex;
  flex-direction: column;
  background: rgba(15, 15, 18, 0.8);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 1rem;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.5);
  gap: 2px;
  padding: 4px;
}

.zoom-controls button {
  background: rgba(255, 255, 255, 0.08);
  border: none;
  color: rgba(255, 255, 255, 0.7);
  width: 40px;
  height: 26px;
  font-size: 0.7rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.1s ease-out;
  border-radius: 6px;
}

.zoom-controls button:hover {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.zoom-controls button.active {
  background: #0A84FF;
  color: white;
  transform: scale(1.05);
}

.interaction-hint {
  position: absolute;
  top: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 9999px;
  font-size: 0.85rem;
  white-space: nowrap;
  backdrop-filter: blur(4px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  z-index: 30;
}

/* Transitions */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.5s ease, transform 0.5s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

.scale-enter-active, .scale-leave-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.scale-enter-from, .scale-leave-to {
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.5);
}
</style>
