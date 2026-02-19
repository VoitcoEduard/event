<template>
  <div class="carousel-container" ref="containerRef">
    <div class="carousel-track" ref="trackRef" @scroll="onScroll">
      <div 
        v-for="(image, idx) in images" 
        :key="idx"
        class="carousel-item"
        :class="{ 
          'is-center': idx === centerIndex,
          'is-side': idx !== centerIndex 
        }"
        :style="getItemStyle(idx)"
        @click="snapToIndex(idx)"
      >
        <div class="image-wrapper" :style="{ transform: idx === centerIndex && zoomScale > 1 ? `scale(${zoomScale})` : '' }">
          <img 
            :src="image.url" 
            :alt="image.alt || 'Option ' + (idx + 1)"
            class="option-image"
            @load="onImageLoad"
            draggable="false"
          />
        </div>
      </div>
    </div>
    
    <!-- Navigation dots -->
    <div class="carousel-dots">
      <span 
        v-for="(_, idx) in images" 
        :key="idx"
        class="dot"
        :class="{ 'active': idx === centerIndex }"
        @click="snapToIndex(idx)"
      ></span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps({
  images: { type: Array, required: true },
  initialIndex: { type: Number, default: 0 }
})

const emit = defineEmits(['snap', 'zoom-change'])

const centerIndex = ref(props.initialIndex)
const zoomScale = ref(1)
const containerRef = ref(null)
const trackRef = ref(null)

// CSS for snap scrolling and visual effects
const getItemStyle = (idx) => {
  const isCenter = idx === centerIndex.value
  return {
    transform: isCenter ? 'scale(1)' : 'scale(0.85)',
    opacity: isCenter ? 1 : 0.6,
    filter: isCenter ? 'none' : 'blur(1px)',
    zIndex: isCenter ? 10 : 1
  }
}

const onScroll = () => {
  if (!trackRef.value) return
  
  const track = trackRef.value
  const itemWidth = track.scrollWidth / props.images.length
  const scrollLeft = track.scrollLeft
  
  // Calculate center index based on scroll position
  const rawIndex = Math.round(scrollLeft / itemWidth)
  const newIndex = Math.max(0, Math.min(props.images.length - 1, rawIndex))
  
  if (newIndex !== centerIndex.value) {
    centerIndex.value = newIndex
    emit('snap', newIndex)
  }
}

const snapToIndex = (idx) => {
  if (!trackRef.value) return
  
  const track = trackRef.value
  const itemWidth = track.scrollWidth / props.images.length
  
  track.scrollTo({
    left: idx * itemWidth,
    behavior: 'smooth'
  })
  
  centerIndex.value = idx
  emit('snap', idx)
}

// Initial scroll position
onMounted(() => {
    // Wait for next tick/render to ensure dimensions are correct
    setTimeout(() => {
        snapToIndex(props.initialIndex)
    }, 100)
    
    // Add pinch zoom listeners manually if needed, or use a library
    // For simplicity, we just basic tap-zoom toggling for now
})

const onImageLoad = () => {
    // Optional: Recalculate layout if needed
}
</script>

<style scoped>
.carousel-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
  position: relative;
}

.carousel-track {
  display: flex;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none; /* Firefox */
  height: 80%;
  align-items: center;
  padding: 0 50%; /* Center the first item */
  box-sizing: content-box; /* Important for padding to work with scroll width */
}

/* Adjust padding dynamically via JS or use a cleaner CSS approach */
.carousel-track {
    /* Better approach for centering: */
    padding: 0; 
    scroll-padding: 0;
}

.carousel-track::-webkit-scrollbar {
  display: none;
}

.carousel-item {
  flex: 0 0 80%; /* Show part of next items */
  width: 80%;
  height: 90%;
  scroll-snap-align: center;
  display: flex;
  justify-content: center;
  align-items: center;
  transition: all 0.3s ease;
  position: relative;
  margin: 0 10%; /* Spacing */
}

/* Override margin/width for simple centering */
.carousel-item {
    flex: 0 0 100%; /* Take full width container, but scale down content */
    width: 100%;
    margin: 0;
}

.image-wrapper {
    width: 85%;
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
    transition: transform 0.2s ease;
}

.option-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 12px;
  filter: drop-shadow(0 10px 20px rgba(0,0,0,0.3));
  
  /* The glow effect */
  outline: 3px solid rgba(255,255,255,0.0);
  transition: outline 0.3s ease;
}

.is-center .option-image {
    outline-color: rgba(255,255,255,0.8);
    box-shadow: 0 0 25px rgba(255, 255, 255, 0.2);
}

.carousel-dots {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 20px;
  height: 20px;
}

.dot {
  width: 8px;
  height: 8px;
  background-color: rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  cursor: pointer;
  transition: background-color 0.3s;
}

.dot.active {
  background-color: white;
  transform: scale(1.2);
}
</style>
