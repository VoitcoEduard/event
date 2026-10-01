<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container">
        <div class="selector-header">
          <h3>{{ $t('quiz_config.question_editor.selector.title') }}</h3>
          <button class="close-btn" @click="handleClose" aria-label="Close">
            <AppIcon name="close" :size="18" />
          </button>
        </div>

        <div class="types-grid">
          <div
            v-for="type in questionTypes"
            :key="type.id"
            class="type-card"
            :class="{ active: hoveredType === type.id }"
            @mouseenter="hoveredType = type.id"
            @mouseleave="hoveredType = null"
            @click="selectType(type.id)"
          >
            <AppIcon class="type-icon-svg" :name="type.icon" :size="48" />
            <span class="type-label">{{ type.label }}</span>
          </div>
        </div>

        <div class="description-area">
          <div v-if="hoveredType" class="description-content">
            <p>{{ getTypeDescription(hoveredType) }}</p>
          </div>
          <div v-else class="description-placeholder">
            {{ $t('quiz_config.question_editor.selector.description_hint') }}
          </div>
        </div>

        <div class="selector-footer">
          <button class="cancel-btn" @click="handleClose">
            {{ $t('quiz_config.question_editor.selector.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'

defineProps({ visible: { type: Boolean, default: false } })
const emit = defineEmits(['close', 'select'])
const { t } = useI18n()
const hoveredType = ref(null)

const questionTypes = computed(() => [
  { id: 'classic', icon: 'quiz', label: t('quiz_config.question_editor.types.classic') },
  { id: 'multi_select', icon: 'listCheck', label: t('quiz_config.question_editor.types.multi_select') },
  { id: 'true_false', icon: 'check', label: t('quiz_config.question_editor.types.true_false') },
  { id: 'range', icon: 'sliders', label: t('quiz_config.question_editor.types.range') },
  { id: 'number', icon: 'hash', label: t('quiz_config.question_editor.types.number') },
  { id: 'correct_picture', icon: 'image', label: t('quiz_config.question_editor.types.correct_picture') },
  { id: 'wrong_pictures', icon: 'search', label: t('quiz_config.question_editor.types.wrong_pictures') },
  { id: 'simple_text', icon: 'form', label: t('quiz_config.question_editor.types.simple_text') }
])

const handleClose = () => emit('close')
const selectType = (typeId) => {
  emit('select', typeId)
  handleClose()
}
const getTypeDescription = (typeId) => t(`quiz_config.question_editor.type_descriptions.${typeId}`)
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, .74); display: flex; justify-content: center; align-items: center; z-index: 3000; backdrop-filter: blur(5px); }
.modal-container { background: var(--ops-surface); width: 95%; max-width: 800px; border-radius: var(--ops-radius-lg); overflow: hidden; border: 1px solid var(--ops-border); display: flex; flex-direction: column; box-shadow: var(--ops-shadow); }
.selector-header { padding: 24px 28px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--ops-border); }
.selector-header h3 { margin: 0; color: var(--ops-text); font-size: 1.35rem; font-weight: 800; letter-spacing: -.02em; }
.close-btn { background: var(--ops-surface-raised); border: 1px solid var(--ops-border); color: var(--ops-text); width: 36px; height: 36px; border-radius: var(--ops-radius-sm); display: flex; align-items: center; justify-content: center; cursor: pointer; }
.close-btn:hover { background: var(--ops-surface-hover); }
.types-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; padding: 28px; }
.type-card { background: var(--ops-surface-raised); border-radius: var(--ops-radius-md); padding: 24px 14px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; cursor: pointer; transition: .2s ease; border: 1px solid var(--ops-border); color: var(--ops-text-muted); }
.type-card:hover, .type-card.active { border-color: var(--ops-blue); color: var(--ops-blue); transform: translateY(-2px); box-shadow: 0 8px 20px rgba(20, 127, 245, .15); }
.type-label { color: var(--ops-text); font-size: .9rem; font-weight: 700; text-align: center; }
.description-area { padding: 0 28px 24px; min-height: 112px; }
.description-content, .description-placeholder { background: var(--ops-surface-raised); border: 1px solid var(--ops-border); border-radius: var(--ops-radius-md); padding: 18px; color: var(--ops-text-muted); font-size: .95rem; line-height: 1.5; text-align: center; }
.description-content p { margin: 0; }
.selector-footer { padding: 16px 28px 24px; display: flex; justify-content: flex-end; }
.cancel-btn { background: transparent; border: 1px solid var(--ops-border); color: var(--ops-text-muted); border-radius: var(--ops-radius-sm); padding: 10px 16px; font-weight: 700; cursor: pointer; }
.cancel-btn:hover { color: var(--ops-text); border-color: var(--ops-text-muted); }
@media (max-width: 640px) { .types-grid { grid-template-columns: repeat(2, 1fr); padding: 18px; } .description-area, .selector-footer { padding-left: 18px; padding-right: 18px; } }
</style>
