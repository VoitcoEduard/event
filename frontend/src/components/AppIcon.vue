<template>
  <svg
    class="app-icon"
    :class="`app-icon--${name}`"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    :fill="isAnswerShape ? 'currentColor' : 'none'"
    stroke="currentColor"
    stroke-width="1.9"
    stroke-linecap="round"
    stroke-linejoin="round"
    :aria-hidden="label ? undefined : 'true'"
    :aria-label="label"
    :role="label ? 'img' : undefined"
  >
    <path v-for="(path, index) in iconPaths" :key="index" :d="path" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  size: { type: [Number, String], default: 20 },
  label: { type: String, default: '' }
})

const icons = {
  alert: ['M12 3 2.8 20h18.4L12 3Z', 'M12 9v4', 'M12 17h.01'],
  arrowLeft: ['M19 12H5', 'm11 6-6 6 6 6'],
  arrowRight: ['M5 12h14', 'm13 6 6 6-6 6'],
  camera: ['M4 7h3l1.5-2h7L17 7h3v12H4V7Z', 'M12 16a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z'],
  chart: ['M5 19V9', 'M12 19V5', 'M19 19v-7'],
  check: ['m5 12 4 4L19 6'],
  chevronDown: ['m6 9 6 6 6-6'],
  chevronLeft: ['m15 18-6-6 6-6'],
  chevronRight: ['m9 18 6-6-6-6'],
  chevronUp: ['m18 15-6-6-6 6'],
  circle: ['M12 20a8 8 0 1 0 0-16 8 8 0 0 0 0 16Z'],
  close: ['M6 6l12 12', 'M18 6 6 18'],
  copy: ['M8 8h11v11H8z', 'M5 16H4V5h11v1'],
  crown: ['m3 8 4 4 5-7 5 7 4-4-2 11H5L3 8Z', 'M5 16h14'],
  download: ['M12 4v11', 'm7 11 5 5 5-5', 'M5 20h14'],
  edit: ['M4 16.5V20h3.5L19 8.5 15.5 5 4 16.5Z', 'm14 6 4 4'],
  eye: ['M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z', 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z'],
  eyeOff: ['m3 3 18 18', 'M10.6 6.2A10.8 10.8 0 0 1 12 6c6 0 9.5 6 9.5 6a17 17 0 0 1-3.2 3.7', 'M6.7 6.7C4 8.3 2.5 12 2.5 12s3.5 6 9.5 6c1.3 0 2.5-.3 3.6-.8'],
  external: ['M14 5h5v5', 'm19 5-8 8', 'M19 13v6H5V5h6'],
  fire: ['M12 21c4.1 0 7-2.7 7-6.5 0-2.6-1.4-5.1-4.1-7.5.2 2.2-.5 3.6-1.8 4.3.1-3.3-1.5-6-4.8-8.3.3 3.6-2.3 5.2-3.4 7.8C3.4 15.7 6.5 21 12 21Z'],
  form: ['M5 3h10l4 4v14H5V3Z', 'M15 3v5h4', 'M8 12h8', 'M8 16h6'],
  hash: ['M9 3 7 21', 'M17 3l-2 18', 'M4 9h16', 'M3 15h16'],
  image: ['M4 5h16v14H4V5Z', 'm5 16 4-4 3 3 2-2 5 5', 'M8 9h.01'],
  info: ['M12 11v5', 'M12 8h.01', 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z'],
  link: ['M10 13a5 5 0 0 0 7.1.1l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1', 'M14 11a5 5 0 0 0-7.1-.1l-2 2A5 5 0 0 0 12 20l1.1-1.1'],
  listCheck: ['m3 6 1.5 1.5L7 4.5', 'm3 12 1.5 1.5L7 10.5', 'm3 18 1.5 1.5L7 16.5', 'M11 6h10', 'M11 12h10', 'M11 18h10'],
  medal: ['m8 12-3-9h5l2 6 2-6h5l-3 9', 'M12 21a6 6 0 1 0 0-12 6 6 0 0 0 0 12Z'],
  message: ['M4 5h16v11H8l-4 4V5Z', 'M8 9h8', 'M8 12h5'],
  minus: ['M5 12h14'],
  music: ['M9 18V5l10-2v13', 'M9 18a3 3 0 1 1-3-3 3 3 0 0 1 3 3Z', 'M19 16a3 3 0 1 1-3-3 3 3 0 0 1 3 3Z'],
  palette: ['M12 3a9 9 0 0 0 0 18h1.5a2 2 0 0 0 0-4H12a2 2 0 0 1 0-4h2.5A6.5 6.5 0 0 0 12 3Z', 'M7.5 10h.01', 'M9 6.5h.01', 'M14 6.5h.01', 'M17 10h.01'],
  pin: ['M12 21s6-5.1 6-11a6 6 0 1 0-12 0c0 5.9 6 11 6 11Z', 'M12 12.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z'],
  play: ['m8 5 11 7-11 7V5Z'],
  plus: ['M12 5v14', 'M5 12h14'],
  question: ['M9.5 9a2.6 2.6 0 1 1 4.4 1.9c-1.3 1.1-1.9 1.5-1.9 3.1', 'M12 17h.01', 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z'],
  quiz: ['M6 8h12a4 4 0 0 1 3.8 5l-1 3.5a2.5 2.5 0 0 1-4.6.5L15 15H9l-1.2 2a2.5 2.5 0 0 1-4.6-.5l-1-3.5A4 4 0 0 1 6 8Z', 'M7 11v4', 'M5 13h4', 'M17 12h.01', 'M19 14h.01'],
  search: ['M10.8 18a7.2 7.2 0 1 0 0-14.4 7.2 7.2 0 0 0 0 14.4Z', 'm16 16 5 5'],
  settings: ['M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z', 'M10.24 3.17 13.76 3.17 14.03 5.30 15.30 5.83 17.00 4.52 19.48 7.00 18.17 8.70 18.70 9.97 20.83 10.24 20.83 13.76 18.70 14.03 18.17 15.30 19.48 17.00 17.00 19.48 15.30 18.17 14.03 18.70 13.76 20.83 10.24 20.83 9.97 18.70 8.70 18.17 7.00 19.48 4.52 17.00 5.83 15.30 5.30 14.03 3.17 13.76 3.17 10.24 5.30 9.97 5.83 8.70 4.52 7.00 7.00 4.52 8.70 5.83 9.97 5.30Z'],
  sliders: ['M4 7h16', 'M4 17h16', 'M8 5v4', 'M16 15v4'],
  square: ['M5 5h14v14H5V5Z'],
  stop: ['M6 6h12v12H6V6Z'],
  teams: ['M8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z', 'M16.5 10a2.5 2.5 0 1 0 0-5', 'M3 19a5 5 0 0 1 10 0', 'M14 14a4 4 0 0 1 7 3'],
  triangle: ['m12 4 8 15H4L12 4Z'],
  diamond: ['m12 3 8 9-8 9-8-9 8-9Z'],
  trash: ['M4 7h16', 'M10 11v6', 'M14 11v6', 'M6 7l1 14h10l1-14', 'M9 7V4h6v3'],
  trophy: ['M7 3h10v6a5 5 0 0 1-10 0V3Z', 'M7 5H3v2a4 4 0 0 0 4 4', 'M17 5h4v2a4 4 0 0 1-4 4', 'M12 14v4', 'M8 21v-3h8v3H8Z'],
  upload: ['M12 20V9', 'm7 13 5-5 5 5', 'M5 4h14'],
  volume: ['M4 10v4h4l5 4V6l-5 4H4Z', 'M17 9a5 5 0 0 1 0 6', 'M19.5 6.5a9 9 0 0 1 0 11'],
  x: ['M6 6l12 12', 'M18 6 6 18']
}

const isAnswerShape = computed(() => ['triangle', 'diamond', 'circle', 'square'].includes(props.name))
const iconPaths = computed(() => icons[props.name] || icons.question)
</script>

<style scoped>
.app-icon {
  display: inline-block;
  flex: 0 0 auto;
  vertical-align: middle;
}
</style>
