<template>
  <div class="wrong-picture-target-crop">
    <div
      class="wrong-picture-target-crop__image-layer"
      :style="imageLayerStyle"
    >
      <img
        :src="imageUrl"
        class="wrong-picture-target-crop__image"
        alt=""
        decoding="async"
        loading="eager"
        draggable="false"
      />

      <div
        v-if="showMarker"
        class="wrong-picture-target-crop__marker"
        :style="markerStyle"
        aria-hidden="true"
      >
        <span class="wrong-picture-target-crop__marker-center"></span>
      </div>
    </div>

    <div v-if="targetNumber != null" class="wrong-picture-target-crop__badge">
      {{ targetNumber }}
    </div>

    <div v-if="label" class="wrong-picture-target-crop__label">
      {{ label }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  imageUrl: {
    type: String,
    required: true
  },
  target: {
    type: Object,
    required: true
  },
  targetNumber: {
    type: Number,
    default: null
  },
  label: {
    type: String,
    default: ''
  },
  showMarker: {
    type: Boolean,
    default: true
  }
})

const clampPercent = (value, fallback) => {
  const number = Number(value)
  if (!Number.isFinite(number)) return fallback
  return Math.min(100, Math.max(0, number))
}

const xPercent = computed(() => clampPercent(props.target?.xPercent, 50))
const yPercent = computed(() => clampPercent(props.target?.yPercent, 50))
const radius = computed(() => {
  const number = Number(props.target?.radius)
  if (!Number.isFinite(number) || number <= 0) return 5
  return Math.min(50, number)
})

// Keep the configured target at a consistent size in the crop. Smaller targets
// receive a larger zoom while very large targets are only mildly magnified.
const zoom = computed(() => Math.min(12, Math.max(1.5, 0.7 / (2 * radius.value))))

const imageLayerStyle = computed(() => ({
  width: `${zoom.value * 100}%`,
  transform: `translate(-${xPercent.value}%, -${yPercent.value}%)`
}))

const markerStyle = computed(() => ({
  left: `${xPercent.value}%`,
  top: `${yPercent.value}%`,
  width: `${radius.value * 2}%`,
  height: `${radius.value * 2}%`
}))
</script>

<style scoped>
.wrong-picture-target-crop {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: 8px;
  background: #09090b;
  isolation: isolate;
}

.wrong-picture-target-crop__image-layer {
  position: absolute;
  top: 50%;
  left: 50%;
  transform-origin: center;
  transition:
    width 600ms cubic-bezier(0.22, 1, 0.36, 1),
    transform 600ms cubic-bezier(0.22, 1, 0.36, 1);
  will-change: transform, width;
}

.wrong-picture-target-crop__image {
  display: block;
  width: 100%;
  height: auto;
  user-select: none;
  pointer-events: none;
}

.wrong-picture-target-crop__marker {
  position: absolute;
  border: clamp(2px, 0.35vw, 5px) solid #32d74b;
  border-radius: 50%;
  background: rgb(50 215 75 / 16%);
  box-shadow:
    0 0 0 clamp(1px, 0.12vw, 2px) rgb(255 255 255 / 72%),
    0 0 clamp(12px, 2vw, 30px) rgb(50 215 75 / 80%);
  transform: translate(-50%, -50%);
  pointer-events: none;
  animation: wrong-picture-marker-appear 450ms ease-out both;
}

.wrong-picture-target-crop__marker-center {
  position: absolute;
  top: 50%;
  left: 50%;
  width: clamp(5px, 0.7vw, 11px);
  aspect-ratio: 1;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 0 0 2px rgb(0 0 0 / 45%);
  transform: translate(-50%, -50%);
}

.wrong-picture-target-crop__badge {
  position: absolute;
  z-index: 3;
  top: 10px;
  left: 10px;
  display: flex;
  width: 38px;
  aspect-ratio: 1;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #32d74b;
  color: #09090b;
  font-size: 1.25rem;
  font-weight: 900;
  box-shadow: 0 4px 12px rgb(0 0 0 / 50%);
}

.wrong-picture-target-crop__label {
  position: absolute;
  z-index: 3;
  right: 0;
  bottom: 0;
  left: 0;
  padding: 14px 16px 10px;
  background: linear-gradient(transparent, rgb(0 0 0 / 88%));
  color: #fff;
  font-size: 1.2rem;
  font-weight: 750;
  text-align: center;
  text-shadow: 0 2px 6px #000;
}

@keyframes wrong-picture-marker-appear {
  from {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.65);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .wrong-picture-target-crop__image-layer,
  .wrong-picture-target-crop__marker {
    transition: none;
    animation: none;
  }
}
</style>
