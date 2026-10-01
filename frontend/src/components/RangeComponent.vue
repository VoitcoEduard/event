<template>
  <div class="range-wrapper" ref="wrapperEl">

    <!-- LEFT: Scale + Track -->
    <div class="range-left">

      <!-- Tick labels -->
      <div class="scale-col">
        <div
          v-for="tick in ticks"
          :key="tick.value"
          class="tick-row"
          :style="{ bottom: tick.pct + '%' }"
        >
          <span class="tick-label" :class="{ active: tick.isActive }">{{ tick.label }}</span>
          <div class="tick-mark" :class="{ active: tick.isActive }"></div>
        </div>
      </div>

      <!-- Track -->
      <div class="track-bar" ref="trackEl">
        <div class="track-fill" :style="{ height: fillPercentage + '%' }"></div>

        <!-- Correct answer marker -->
        <div
          v-if="showCorrectMarker && correctValue !== null"
          class="correct-marker"
          :style="{ bottom: correctPercentage + '%' }"
        >
          <div class="correct-tag">{{ formatNumber(correctValue) }}{{ unit }}</div>
        </div>

        <!-- Thumb -->
        <div
          class="thumb"
          :style="{ bottom: fillPercentage + '%' }"
          :class="{ dragging: isScrolling }"
        >
          <div class="thumb-lines">
            <span></span><span></span><span></span>
          </div>
        </div>
      </div>
    </div>

    <!-- CENTER: Value display -->
    <div class="value-display">
      <span class="value-num">{{ formattedValue }}</span>
      <span v-if="unit" class="value-unit">{{ unit }}</span>
    </div>

    <!-- RIGHT: Scroll wheel zone -->
    <div
      class="scroll-zone"
      :class="{ active: isScrolling, hint: !hasInteracted && !isScrolling }"
      ref="scrollZoneEl"
      @touchstart.prevent="onTouchStart"
      @touchmove.prevent="onTouchMove"
      @touchend="onTouchEnd"
      @mousedown="onMouseDown"
      @wheel.prevent="onWheel"
    >
      <!-- Idle hint arrows -->
      <div v-if="!hasInteracted && !isScrolling" class="hint-arrows">
        <AppIcon class="arrow arrow-up" name="chevronUp" :size="24" />
        <AppIcon class="arrow arrow-down" name="chevronDown" :size="24" />
      </div>
      <div class="notches" :class="{ faded: !hasInteracted && !isScrolling }">
        <div v-for="i in 20" :key="i" class="notch" :class="{ center: i === 10 || i === 11 }"></div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  min:               { type: Number,  default: 0 },
  max:               { type: Number,  default: 100 },
  step:              { type: Number,  default: 1 },
  modelValue:        { type: Number,  default: 50 },
  unit:              { type: String,  default: '' },
  decimalPlaces:     { type: Number,  default: 0 },
  orientation:       { type: String,  default: 'vertical' },
  readonly:          { type: Boolean, default: false },
  showCorrectMarker: { type: Boolean, default: false },
  correctValue:      { type: Number,  default: null },
})
const emit = defineEmits(['update:modelValue', 'change'])

const wrapperEl    = ref(null)
const trackEl      = ref(null)
const scrollZoneEl = ref(null)
const isScrolling  = ref(false)
const hasInteracted = ref(false)

// ── Number helpers ─────────────────────────────────────────
const formatNumber = (n) => {
  if (n === null || n === undefined) return ''
  return Number(n).toFixed(props.decimalPlaces)
}
const formattedValue = computed(() => formatNumber(props.modelValue))

// ── Percentages ────────────────────────────────────────────
const range = computed(() => props.max - props.min)

const fillPercentage = computed(() => {
  if (range.value === 0) return 0
  return Math.max(0, Math.min(100, ((props.modelValue - props.min) / range.value) * 100))
})

const correctPercentage = computed(() => {
  if (props.correctValue === null || range.value === 0) return 0
  return Math.max(0, Math.min(100, ((props.correctValue - props.min) / range.value) * 100))
})

// ── Tick marks ─────────────────────────────────────────────
const ticks = computed(() => {
  const result = []
  const r = range.value
  const idealTicks = 6
  let ts = r / idealTicks
  const mag = Math.pow(10, Math.floor(Math.log10(ts)))
  ts = Math.ceil(ts / mag) * mag

  for (let v = props.min; v <= props.max + 1e-9; v += ts) {
    const snapped = Math.round(v / ts) * ts
    const pct = ((snapped - props.min) / r) * 100
    if (pct < -0.5 || pct > 100.5) continue
    result.push({
      value: snapped,
      pct: Math.max(0, Math.min(100, pct)),
      label: formatNumber(snapped),
      isActive: snapped <= props.modelValue,
    })
  }
  return result
})

// ── Haptics ────────────────────────────────────────────────
let lastHapticValue = null
const buzz = () => { if (navigator.vibrate) navigator.vibrate(10) }

// ── Core setter ────────────────────────────────────────────
const applyValue = (raw) => {
  let v = Math.max(props.min, Math.min(props.max, raw))
  if (props.step > 0) {
    v = Math.round((v - props.min) / props.step) * props.step + props.min
  }
  const pow = Math.pow(10, props.decimalPlaces)
  v = Math.round(v * pow) / pow
  if (v !== props.modelValue) {
    if (lastHapticValue === null || Math.abs(v - lastHapticValue) >= props.step) {
      buzz()
      lastHapticValue = v
    }
    emit('update:modelValue', v)
  }
}

// ── Scroll sensitivity ─────────────────────────────────────
// How many pixels of drag = full range. Larger = slower/more precise.
const DRAG_PX_PER_FULL_RANGE = 600   // drag 600px to go 0→max
const WHEEL_STEPS_PER_TICK = 1       // one step per wheel tick

// ── Touch ──────────────────────────────────────────────────
let touchStartY   = null
let touchStartVal = null

const onTouchStart = (e) => {
  if (props.readonly) return
  hasInteracted.value = true
  touchStartY   = e.touches[0].clientY
  touchStartVal = props.modelValue
  isScrolling.value = true
  buzz()
}

const onTouchMove = (e) => {
  if (!isScrolling.value || touchStartY === null) return
  const dy    = touchStartY - e.touches[0].clientY   // up = positive = increase
  const delta = (dy / DRAG_PX_PER_FULL_RANGE) * range.value
  applyValue(touchStartVal + delta)
}

const onTouchEnd = () => {
  isScrolling.value = false
  touchStartY   = null
  touchStartVal = null
  if (navigator.vibrate) navigator.vibrate([15, 10, 15])
}

// ── Mouse drag ─────────────────────────────────────────────
const onMouseDown = (e) => {
  if (props.readonly) return
  hasInteracted.value = true
  const startY   = e.clientY
  const startVal = props.modelValue
  isScrolling.value = true

  const onMove = (ev) => {
    const dy    = startY - ev.clientY
    const delta = (dy / DRAG_PX_PER_FULL_RANGE) * range.value
    applyValue(startVal + delta)
  }
  const onUp = () => {
    isScrolling.value = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// ── Mouse wheel ────────────────────────────────────────────
const onWheel = (e) => {
  if (props.readonly) return
  hasInteracted.value = true
  const dir  = e.deltaY > 0 ? -1 : 1
  const step = props.step > 0 ? props.step : range.value / 100
  applyValue(props.modelValue + dir * step * WHEEL_STEPS_PER_TICK)
}
</script>

<style scoped>
/* ── Root ──────────────────────────────────────────────────────── */
.range-wrapper {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  width: 100%;
  height: 100%;
  min-height: 260px;
  user-select: none;
  touch-action: none;
  font-family: -apple-system, 'Inter', sans-serif;
}

/* ── Left panel ────────────────────────────────────────────────── */
.range-left {
  flex: 1;
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 10px;
  padding: 12px 0;
}

/* ── Tick scale ────────────────────────────────────────────────── */
.scale-col {
  position: relative;
  width: 44px;
}

.tick-row {
  position: absolute;
  right: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  transform: translateY(50%);
}

.tick-label {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255,255,255,0.3);
  text-align: right;
  min-width: 28px;
  transition: color 0.2s;
}
.tick-label.active { color: rgba(255,255,255,0.9); }

.tick-mark {
  width: 8px;
  height: 2px;
  border-radius: 1px;
  background: rgba(255,255,255,0.2);
  transition: background 0.2s;
}
.tick-mark.active { background: rgba(255,255,255,0.75); }

/* ── Track bar ─────────────────────────────────────────────────── */
.track-bar {
  position: relative;
  width: 6px;
  background: rgba(255,255,255,0.12);
  border-radius: 6px;
  flex-shrink: 0;
}

.track-fill {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  border-radius: 6px;
  background: linear-gradient(to top, #0A84FF, #64D2FF);
  transition: height 0.04s linear;
}

/* ── Thumb ─────────────────────────────────────────────────────── */
.thumb {
  position: absolute;
  left: 50%;
  transform: translate(-50%, 50%);
  width: 48px;
  height: 48px;
  background: #ffffff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 3px 12px rgba(0,0,0,0.3);
  transition: box-shadow 0.15s, transform 0.04s linear;
  z-index: 5;
}
.thumb.dragging {
  box-shadow: 0 6px 24px rgba(0,0,0,0.45);
}

.thumb-lines {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.thumb-lines span {
  display: block;
  width: 14px;
  height: 2px;
  border-radius: 1px;
  background: rgba(0,0,0,0.22);
}

/* ── Value display ─────────────────────────────────────────────── */
.value-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 12px;
  gap: 2px;
  pointer-events: none;
}

.value-num {
  font-size: 44px;
  font-weight: 800;
  color: #fff;
  line-height: 1;
  text-shadow: 0 2px 8px rgba(0,0,0,0.2);
}
.value-unit {
  font-size: 16px;
  font-weight: 600;
  color: rgba(255,255,255,0.55);
  letter-spacing: 0.03em;
}

/* ── Correct marker ────────────────────────────────────────────── */
.correct-marker {
  position: absolute;
  left: 50%;
  transform: translate(-50%, 50%);
  width: 20px;
  height: 3px;
  background: #30D158;
  border-radius: 2px;
  z-index: 4;
}
.correct-tag {
  position: absolute;
  left: 26px;
  top: 50%;
  transform: translateY(-50%);
  background: #30D158;
  color: #000;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 5px;
  border-radius: 4px;
  white-space: nowrap;
}

/* ── Scroll zone ───────────────────────────────────────────────── */
.scroll-zone {
  width: 64px;
  flex-shrink: 0;
  margin: 8px 0 8px 10px;
  border-radius: 18px;
  background: rgba(255,255,255,0.06);
  border: 1.5px solid rgba(255,255,255,0.10);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: ns-resize;
  transition: background 0.3s, border-color 0.3s, box-shadow 0.3s;
  overflow: hidden;
  position: relative;
  gap: 6px;
}

/* Idle hint: pulsing glow + bright border */
@keyframes wheel-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(10, 132, 255, 0.0),  0 0 0 0   rgba(10, 132, 255, 0.0); }
  30%  { box-shadow: 0 0 10px 3px rgba(10, 132, 255, 0.55), 0 0 22px 8px rgba(10, 132, 255, 0.2); }
  60%  { box-shadow: 0 0 10px 3px rgba(10, 132, 255, 0.55), 0 0 22px 8px rgba(10, 132, 255, 0.2); }
  100% { box-shadow: 0 0 0 0 rgba(10, 132, 255, 0.0),  0 0 0 0   rgba(10, 132, 255, 0.0); }
}

.scroll-zone.hint {
  background: rgba(10, 132, 255, 0.08);
  border-color: rgba(10, 132, 255, 0.55);
  animation: wheel-pulse 2s ease-in-out infinite;
}

.scroll-zone.active {
  background: rgba(10, 132, 255, 0.15);
  border-color: rgba(10, 132, 255, 0.4);
  animation: none;
  box-shadow: none;
}

/* ── Hint arrows ───────────────────────────────────────────────── */
.hint-arrows {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  justify-content: space-between;
  padding: 10px 0;
  pointer-events: none;
}

@keyframes arrow-bob-up {
  0%, 100% { transform: translateY(0);   opacity: 0.6; }
  50%       { transform: translateY(-5px); opacity: 1;   }
}
@keyframes arrow-bob-down {
  0%, 100% { transform: translateY(0);  opacity: 0.6; }
  50%       { transform: translateY(5px); opacity: 1;   }
}

.arrow {
  width: 22px;
  height: 22px;
  color: #0A84FF;
  filter: drop-shadow(0 0 4px rgba(10,132,255,0.7));
}
.arrow-up   { animation: arrow-bob-up   1.4s ease-in-out infinite; }
.arrow-down { animation: arrow-bob-down 1.4s ease-in-out infinite; }

/* ── Notches ───────────────────────────────────────────────────── */
.notches {
  display: flex;
  flex-direction: column;
  gap: 9px;
  align-items: center;
  transition: opacity 0.3s;
}

.notches.faded {
  opacity: 0.35;
}

.notch {
  width: 34px;
  height: 2px;
  border-radius: 1px;
  background: rgba(255,255,255,0.22);
}

/* The two middle notches are slightly wider/brighter — like a center indent */
.notch.center {
  width: 42px;
  background: rgba(255,255,255,0.45);
}

.scroll-zone.active .notch {
  background: rgba(100, 210, 255, 0.5);
}
.scroll-zone.active .notch.center {
  background: rgba(100, 210, 255, 0.85);
}
</style>
