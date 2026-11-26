<template>
  <div ref="editorRootRef" class="wrong-pictures-editor" :class="{ 'wrong-pictures-editor--page-flow': pageFlow }">
    <div class="scroll-content">
      <div v-if="localRounds.length === 0" class="empty-state">
        <AppIcon class="empty-icon" name="search" :size="42" aria-hidden="true" />
        <p class="empty-text">{{ $t('wrong_pictures.empty_text') }}</p>
      </div>

      <div v-else class="rounds-container">
        <div
          v-for="(round, index) in localRounds"
          :key="round.id || index"
          :ref="(el) => setRoundEl(index, el)"
          :data-round-index="index"
          class="round-card"
          :class="{ 'is-active': activeIndex === index, 'has-errors': hasRoundErrors(index) }"
          @click="handleRoundInteract(index)"
          @focusin="handleRoundInteract(index)"
          @paste="(event) => handleRoundPaste(index, event)"
        >
          <div class="round-header">
            <h3>{{ $t('wrong_pictures.round_title', { number: index + 1 }) }}</h3>
            <div class="round-actions">
              <button
                type="button"
                class="move-btn"
                :disabled="index === 0"
                :title="$t('wrong_pictures.move_up')"
                :aria-label="$t('wrong_pictures.move_up')"
                @click.stop="moveRound(index, -1)"
              >
                <AppIcon name="chevronUp" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="move-btn"
                :disabled="index === localRounds.length - 1"
                :title="$t('wrong_pictures.move_down')"
                :aria-label="$t('wrong_pictures.move_down')"
                @click.stop="moveRound(index, 1)"
              >
                <AppIcon name="chevronDown" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="remove-btn"
                :title="$t('wrong_pictures.remove_round')"
                :aria-label="$t('wrong_pictures.remove_round')"
                @click.stop="removeRound(index)"
              >
                <AppIcon name="trash" :size="16" aria-hidden="true" />
              </button>
            </div>
          </div>

          <div class="config-compact-row">
            <div class="input-group flex-grow">
              <label>{{ $t('wrong_pictures.instructions_label') }}</label>
              <input
                type="text"
                :value="round.text"
                class="setting-input"
                :class="{ 'has-field-error': !!fieldError(index, 'text') }"
                :aria-invalid="!!fieldError(index, 'text') || undefined"
                :placeholder="$t('wrong_pictures.instructions_placeholder')"
                @input="updateRoundField(index, 'text', $event.target.value)"
              />
              <p v-if="fieldError(index, 'text')" class="field-error" role="alert">{{ fieldError(index, 'text') }}</p>
            </div>

            <div class="input-group mini">
              <label>{{ $t('wrong_pictures.points') }}</label>
              <input
                type="number"
                min="1"
                :value="round.wrongPictureConfig?.maxPoints ?? ''"
                class="setting-input"
                :class="{ 'has-field-error': !!fieldError(index, 'points') }"
                :aria-invalid="!!fieldError(index, 'points') || undefined"
                @input="updateConfigField(index, 'maxPoints', parseNumber($event.target.value))"
              />
              <p v-if="fieldError(index, 'points')" class="field-error" role="alert">{{ fieldError(index, 'points') }}</p>
            </div>

            <div class="input-group mini">
              <label>{{ $t('wrong_pictures.tolerance') }}</label>
              <input
                type="number"
                min="0"
                :value="round.wrongPictureConfig?.tolerancePercent ?? ''"
                class="setting-input"
                :class="{ 'has-field-error': !!fieldError(index, 'tolerancePercent') }"
                :aria-invalid="!!fieldError(index, 'tolerancePercent') || undefined"
                @input="updateConfigField(index, 'tolerancePercent', parseNumber($event.target.value))"
              />
              <p v-if="fieldError(index, 'tolerancePercent')" class="field-error" role="alert">{{ fieldError(index, 'tolerancePercent') }}</p>
            </div>

            <div class="input-group mini">
              <label>{{ $t('wrong_pictures.time_limit') }}</label>
              <input
                type="number"
                min="0"
                :value="round.timeLimit ?? ''"
                class="setting-input"
                :class="{ 'has-field-error': !!fieldError(index, 'timeLimit') }"
                :aria-invalid="!!fieldError(index, 'timeLimit') || undefined"
                @input="updateRoundField(index, 'timeLimit', parseNumber($event.target.value))"
              />
              <p v-if="fieldError(index, 'timeLimit')" class="field-error" role="alert">{{ fieldError(index, 'timeLimit') }}</p>
            </div>

            <div class="input-group mini center">
              <label>{{ $t('wrong_pictures.show_count') }}</label>
              <label class="switch small">
                <input
                  type="checkbox"
                  :checked="Boolean(round.wrongPictureConfig?.showTargetCount)"
                  @change="updateConfigField(index, 'showTargetCount', $event.target.checked)"
                />
                <span class="slider round"></span>
              </label>
            </div>
          </div>

          <div class="image-management-section">
            <ImageUploader
              :ref="(el) => setRoundUploaderRef(index, el)"
              :model-value="round.imageId"
              :alt-text="round.imageAlt"
              :event-id="eventId"
              :module-id="moduleId"
              :hide-preview="true"
              @update:modelValue="val => updateRoundField(index, 'imageId', val)"
              @update:altText="val => updateRoundField(index, 'imageAlt', val)"
            />

            <div
              v-if="round.imageId"
              class="coordinate-picker-container"
              :class="{ 'has-field-error': !!fieldError(index, 'targets') }"
            >
              <p v-if="fieldError(index, 'targets')" class="field-error" role="alert">{{ fieldError(index, 'targets') }}</p>
              <CoordinatePicker
                :image-id="round.imageId"
                :targets="round.wrongPictureConfig?.targets || []"
                @update:targets="val => updateConfigField(index, 'targets', val)"
              />
            </div>
            <div
              v-else
              class="image-required-warning"
              :class="{ 'has-field-error': !!fieldError(index, 'imageId') }"
            >
              <AppIcon name="alert" :size="16" aria-hidden="true" />
              <span>{{ fieldError(index, 'imageId') || $t('wrong_pictures.image_required') }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="add-round-container">
        <button type="button" class="add-round-btn" @click="addRound">
          {{ $t('wrong_pictures.add_round') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialog } from '../composables/useDialog'
import ImageUploader from './ImageUploader.vue'
import CoordinatePicker from './CoordinatePicker.vue'
import AppIcon from './AppIcon.vue'
import { hasActiveBlockingModal, isPasteTargetInContainer } from '../utils/clipboard'

const { t } = useI18n()
const { confirm: confirmDialog } = useDialog()

const props = defineProps({
  rounds: { type: Array, default: null },
  modelValue: { type: Array, default: null },
  eventId: { type: String, default: '' },
  moduleId: { type: String, default: '' },
  activeIndex: { type: Number, default: -1 },
  errorsByIndex: { type: Object, default: () => ({}) },
  pageFlow: { type: Boolean, default: false }
})

const emit = defineEmits(['update:rounds', 'update:modelValue', 'update:activeIndex', 'select-round'])

const editorRootRef = ref(null)
const localRounds = ref([])
const lastInteractedIndex = ref(-1)
const roundEls = new Map()
const roundUploaderRefs = new Map()
let lastEmittedRef = null
let selfSelectedIndex = -1

const setRoundUploaderRef = (index, el) => {
  if (el) {
    roundUploaderRefs.set(index, el)
  } else {
    roundUploaderRefs.delete(index)
  }
}

const effectiveActiveIndex = computed(() => {
  if (props.activeIndex >= 0 && props.activeIndex < localRounds.value.length) {
    return props.activeIndex
  }
  if (lastInteractedIndex.value >= 0 && lastInteractedIndex.value < localRounds.value.length) {
    return lastInteractedIndex.value
  }
  return localRounds.value.length === 1 ? 0 : -1
})

const handleRoundPaste = (index, event) => {
  if (event?.defaultPrevented) return false
  return roundUploaderRefs.get(index)?.handleClipboardPasteEvent?.(event) ?? false
}

const handleWindowPaste = (event) => {
  if (event?.defaultPrevented) return
  const targetIndex = effectiveActiveIndex.value
  if (targetIndex < 0) return
  if (hasActiveBlockingModal(editorRootRef.value)) return
  if (!isPasteTargetInContainer(event?.target, editorRootRef.value)) return

  handleRoundPaste(targetIndex, event)
}

onMounted(() => {
  if (typeof window !== 'undefined') {
    window.addEventListener('paste', handleWindowPaste)
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('paste', handleWindowPaste)
  }
})

const normalizeRound = (round = {}) => ({
  id: round.id || `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
  type: 'wrong_pictures',
  text: round.text || '',
  timeLimit: round.timeLimit !== undefined ? round.timeLimit : 60,
  explainTime: round.explainTime !== undefined ? round.explainTime : 5,
  imageId: round.imageId || null,
  imageAlt: round.imageAlt || '',
  wrongPictureConfig: {
    targets: Array.isArray(round.wrongPictureConfig?.targets)
      ? round.wrongPictureConfig.targets.map(target => ({ ...target }))
      : [],
    maxPoints: round.wrongPictureConfig?.maxPoints !== undefined ? round.wrongPictureConfig.maxPoints : 50,
    tolerancePercent: round.wrongPictureConfig?.tolerancePercent !== undefined ? round.wrongPictureConfig.tolerancePercent : 5,
    showTargetCount: Boolean(round.wrongPictureConfig?.showTargetCount)
  }
})

watch(
  () => props.rounds ?? props.modelValue,
  (source) => {
    if (source && source === lastEmittedRef) return
    const list = Array.isArray(source) ? source : []
    localRounds.value = list.map(normalizeRound)
  },
  { immediate: true }
)

const emitRounds = (nextRounds) => {
  localRounds.value = nextRounds
  lastEmittedRef = nextRounds
  emit('update:rounds', nextRounds)
  emit('update:modelValue', nextRounds)
}

const parseNumber = (val) => {
  if (val === '' || val === null || val === undefined) return null
  const num = Number(val)
  return Number.isFinite(num) ? num : null
}

const setRoundEl = (index, el) => {
  if (el) {
    roundEls.set(index, el)
  } else {
    roundEls.delete(index)
  }
}

const hasRoundErrors = (index) => {
  const errMap = props.errorsByIndex?.[index]
  return Boolean(errMap && Object.keys(errMap).length > 0)
}

const fieldError = (index, field) => {
  const err = props.errorsByIndex?.[index]?.[field]
  return err ? t(err.messageKey, err.params || {}) : ''
}

const scrollToRound = (index) => {
  if (typeof index !== 'number' || index < 0) return
  nextTick(() => {
    const el = roundEls.get(index)
    if (el && typeof el.scrollIntoView === 'function') {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  })
}

const handleRoundInteract = (index) => {
  lastInteractedIndex.value = index
  if (props.activeIndex === index) return
  selfSelectedIndex = index
  emit('update:activeIndex', index)
  emit('select-round', index)
}

watch(() => props.activeIndex, (newIndex) => {
  if (newIndex === selfSelectedIndex) {
    selfSelectedIndex = -1
    return
  }
  selfSelectedIndex = -1
  if (typeof newIndex === 'number' && newIndex >= 0 && newIndex < localRounds.value.length) {
    scrollToRound(newIndex)
  }
})

const addRound = () => {
  const next = [...localRounds.value, normalizeRound()]
  emitRounds(next)
  const newIndex = next.length - 1
  emit('update:activeIndex', newIndex)
  scrollToRound(newIndex)
}

const removeRound = async (index) => {
  const confirmed = await confirmDialog({
    title: t('wrong_pictures.remove_round_title'),
    message: t('wrong_pictures.remove_round_confirm'),
    confirmLabel: t('wrong_pictures.remove_round'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return
  const next = localRounds.value.filter((_, i) => i !== index)
  emitRounds(next)
}

const moveRound = (index, direction) => {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= localRounds.value.length) return
  const next = [...localRounds.value]
  const temp = next[index]
  next[index] = next[targetIndex]
  next[targetIndex] = temp
  emitRounds(next)
  emit('update:activeIndex', targetIndex)
}

const updateRoundField = (index, key, value) => {
  const next = localRounds.value.map((round, i) => (
    i === index ? { ...round, [key]: value } : round
  ))
  emitRounds(next)
}

const updateConfigField = (index, key, value) => {
  const next = localRounds.value.map((round, i) => {
    if (i !== index) return round
    return {
      ...round,
      wrongPictureConfig: {
        ...round.wrongPictureConfig,
        [key]: value
      }
    }
  })
  emitRounds(next)
}

defineExpose({
  scrollToRound,
  addRound
})
</script>

<style scoped>
.wrong-pictures-editor {
  container: wrong-pictures-editor / inline-size;
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  min-inline-size: 0;
  width: 100%;
  box-sizing: border-box;
}

.wrong-pictures-editor *,
.wrong-pictures-editor *::before,
.wrong-pictures-editor *::after {
  box-sizing: border-box;
}

.scroll-content {
  flex: 1;
  min-height: 0;
  overflow-y: visible;
  padding: 0 0 24px;
}

.wrong-pictures-editor:not(.wrong-pictures-editor--page-flow) .scroll-content {
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 20px;
}

.empty-state {
  text-align: center;
  padding: 56px 20px;
  color: var(--admin-text-muted, #8E8E93);
}

.empty-icon {
  margin-bottom: 12px;
}

.empty-text {
  font-size: 14px;
  margin: 0;
}

.rounds-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.round-card {
  background: #2C2C2E;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #3A3A3C;
  min-inline-size: 0;
  scroll-margin-top: 84px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.round-card.is-active {
  border-color: var(--admin-primary, #0A84FF);
  box-shadow: 0 0 0 2px rgba(10, 132, 255, 0.2);
}

.round-card.has-errors {
  border-color: var(--admin-danger, #FF453A);
}

.round-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.round-header h3 {
  margin: 0;
  color: #FFF;
  font-size: 18px;
}

.round-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.move-btn,
.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: 1px solid #3A3A3C;
  background: #1C1C1E;
  color: #E5E5EA;
  cursor: pointer;
}

.move-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.remove-btn {
  color: #FF453A;
  border-color: transparent;
  background: rgba(255, 69, 58, 0.12);
}

.remove-btn:hover {
  background: rgba(255, 69, 58, 0.22);
}

.config-compact-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  align-items: flex-start;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-inline-size: 0;
}

.flex-grow { flex: 1; }
.mini { width: 88px; }
.center { align-items: center; }

.input-group label {
  font-size: 11px;
  color: #8E8E93;
  font-weight: 500;
  white-space: nowrap;
}

.setting-input {
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 14px;
  width: 100%;
}

.field-error {
  margin: 4px 0 0;
  color: var(--admin-danger, #FF453A);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
}

.has-field-error {
  border-color: var(--admin-danger, #FF453A) !important;
  box-shadow: 0 0 0 2px rgba(255, 69, 58, 0.22) !important;
}

.image-management-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.coordinate-picker-container {
  border-top: 1px solid #3A3A3C;
  padding-top: 12px;
}

.coordinate-picker-container.has-field-error {
  border: 1px solid var(--admin-danger, #FF453A);
  border-radius: 10px;
  padding: 12px;
}

.image-required-warning {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #FF9F0A;
  background: rgba(255, 159, 10, 0.1);
  padding: 12px;
  border-radius: 8px;
  font-size: 14px;
  border: 1px solid rgba(255, 159, 10, 0.3);
}

.add-round-container {
  padding: 20px 0 0;
  display: flex;
  justify-content: center;
}

.add-round-btn {
  background: #3A3A3C;
  color: white;
  border: 1px dashed #666;
  padding: 12px 24px;
  border-radius: 8px;
  cursor: pointer;
  width: 100%;
  font-weight: 600;
  min-block-size: 44px;
}

.add-round-btn:hover {
  background: #4A4A4C;
}

.switch.small {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  margin-top: 6px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background-color: #3A3A3C;
  transition: .3s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: .3s;
}

input:checked + .slider {
  background-color: #0A84FF;
}

input:focus-visible + .slider {
  outline: 3px solid var(--admin-focus, #0A84FF);
  outline-offset: 2px;
}

input:checked + .slider:before {
  transform: translateX(16px);
}

.slider.round {
  border-radius: 24px;
}

.slider.round:before {
  border-radius: 50%;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .round-card) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .round-card.is-active) {
  border-color: var(--admin-primary) !important;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--admin-primary) 24%, transparent) !important;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .round-card.has-errors) {
  border-color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .has-field-error) {
  border-color: var(--admin-danger) !important;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--admin-danger) 24%, transparent) !important;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .add-round-btn) {
  background: var(--admin-surface) !important;
  border: 1px dashed var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .wrong-pictures-editor .add-round-btn:hover) {
  background: var(--admin-surface-hover) !important;
  border-color: var(--admin-primary) !important;
}

@container wrong-pictures-editor (inline-size <= 640px) {
  .round-card {
    padding: 14px;
  }

  .config-compact-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    align-items: start;
  }

  .config-compact-row .flex-grow {
    grid-column: 1 / -1;
    min-inline-size: 0;
  }

  .config-compact-row .mini {
    width: auto;
    min-inline-size: 0;
  }

  .config-compact-row .mini.center {
    align-items: flex-start;
  }

  .input-group label {
    white-space: normal;
    overflow-wrap: break-word;
  }

  .setting-input {
    font-size: 16px;
  }
}

@media (pointer: coarse) {
  .move-btn,
  .remove-btn,
  .add-round-btn,
  .setting-input {
    min-block-size: var(--target-size, 44px);
  }

  .move-btn,
  .remove-btn {
    min-inline-size: var(--target-size, 44px);
  }

  .switch.small::after {
    content: '';
    position: absolute;
    inset: -12px -4px;
  }
}
</style>
