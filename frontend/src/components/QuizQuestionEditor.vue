<template>
  <div class="question-editor" :class="{ 'is-active': active, 'has-errors': hasAnyErrors }">
    <div class="question-editor-inner">
    <div class="editor-header">
      <div class="header-left">
        <span class="question-number">{{ $t('quiz_config.question_editor.question_number', { number: index + 1 }) }}</span>
        <span
          v-if="isTypeLocked"
          class="question-type-tag question-type-tag--locked"
        >{{ $t('quiz_config.question_editor.types.' + localQuestion.type) }}</span>
        <button
          v-else
          type="button"
          class="question-type-tag"
          @click="$emit('change-type')"
        >{{ $t('quiz_config.question_editor.types.' + localQuestion.type) }}</button>
      </div>
      <button type="button" class="delete-btn" @click="$emit('delete')" :title="$t('form_editor.delete_question')" :aria-label="$t('form_editor.delete_question')">
        <AppIcon name="trash" :size="16" aria-hidden="true" />
      </button>
    </div>

    <div v-if="fieldError('type')" class="field-error-banner" role="alert">
      <AppIcon name="alert" :size="16" aria-hidden="true" />
      <span>{{ fieldError('type') }}</span>
    </div>

    <div class="editor-main-row">
      <div v-if="localQuestion.type !== 'wrong_pictures'" class="image-section">
        <label>{{ $t('quiz_config.question_editor.image_label') }}</label>
        <ImageUploader
          v-model="localQuestion.imageId"
          v-model:altText="localQuestion.imageAlt"
          :event-id="eventId"
          :module-id="moduleId"
        />
      </div>

      <div class="settings-stack">
        <div class="input-group">
          <label>
            {{ $t('quiz_config.question_editor.time_limit') }}
            <span v-if="localQuestion.timeLimit === 20" class="default-hint">({{ $t('quiz_config.question_editor.default_hint') }})</span>
          </label>
          <select v-model.number="localQuestion.timeLimit" class="select-input" @change="emitUpdate">
            <option v-if="localQuestion.type === 'simple_text' || localQuestion.timeLimit === 0" :value="0">{{ $t('quiz_config.question_editor.no_limit') }}</option>
            <option v-for="sec in timeLimitOptions" :key="sec" :value="sec">{{ sec }}s</option>
          </select>
        </div>
        <div v-if="localQuestion.type !== 'simple_text'" class="input-group">
          <label>
            {{ $t('quiz_config.question_editor.explain_time') }}
            <span v-if="localQuestion.explainTime === 5" class="default-hint">({{ $t('quiz_config.question_editor.default_hint') }})</span>
          </label>
          <select v-model.number="localQuestion.explainTime" class="select-input" @change="emitUpdate">
            <option :value="0">0s</option>
            <option :value="5">5s</option>
            <option :value="10">10s</option>
            <option :value="15">15s</option>
            <option :value="20">20s</option>
            <option :value="30">30s</option>
            <option :value="40">40s</option>
          </select>
        </div>
        <div class="input-group">
          <label>
            {{ $t('quiz_config.question_editor.points') }}
            <span v-if="localMaxPoints === 100 && localQuestion.type === 'range'" class="default-hint">({{ $t('quiz_config.question_editor.default_hint') }})</span>
            <span v-if="localMaxPoints === 50 && localQuestion.type !== 'range'" class="default-hint">({{ $t('quiz_config.question_editor.default_hint') }})</span>
          </label>
          <input
            type="number"
            min="1"
            v-model.number="localMaxPoints"
            class="select-input"
            :class="{ 'has-field-error': !!fieldError('points') }"
            :aria-invalid="!!fieldError('points') || undefined"
            @input="emitUpdate"
          />
          <p v-if="fieldError('points')" class="field-error" role="alert">{{ fieldError('points') }}</p>
        </div>
      </div>
    </div>
    
    <!-- QUESTION TEXT -->
    <div class="input-group text-group">
      <label>{{ $t('quiz_config.question_editor.text_label') }}</label>
      <input 
        type="text" 
        v-model="localQuestion.text" 
        class="main-input"
        :class="{ 'has-field-error': !!fieldError('text') }"
        :aria-invalid="!!fieldError('text') || undefined"
        :placeholder="$t('quiz_config.question_editor.text_placeholder')"
        @input="emitUpdate"
      />
      <p v-if="fieldError('text')" class="field-error" role="alert">{{ fieldError('text') }}</p>
    </div>


    <!-- STANDARD ANSWERS (MC/TF/MS) -->
    <!-- Multi-select warning -->
    <div v-if="localQuestion.type === 'multi_select' && multiSelectWarning && !choiceSectionError" class="multi-select-warning">
      <AppIcon name="alert" :size="16" aria-hidden="true" /> {{ $t('quiz_config.question_editor.multi_select_warning', { count: 2 }) }}
    </div>
    <div v-if="isChoiceType && choiceSectionError" class="field-error-banner" role="alert">
      <AppIcon name="alert" :size="16" aria-hidden="true" />
      <span>{{ choiceSectionError }}</span>
    </div>
    <div
      v-if="isChoiceType"
      class="answers-grid"
      :class="{
        'true-false-grid': localQuestion.type === 'true_false',
        'has-field-error': !!choiceSectionError
      }"
    >
      <div 
        v-for="(answer, idx) in localQuestion.answers" 
        :key="idx"
        class="answer-card"
        :class="[getAnswerColorClass(idx), { 'hidden-answer': !isAnswerVisible(idx), 'is-correct': isCorrect(idx) }]"
        v-show="isAnswerVisible(idx)"
      >
        <div class="answer-header">
          <AppIcon class="shape-icon" :name="getShapeIcon(idx)" :size="20" aria-hidden="true" />
          <label class="correct-toggle" :title="$t('quiz_config.question_editor.mark_correct')">
            <input 
              :type="localQuestion.type === 'multi_select' ? 'checkbox' : 'radio'"
              :name="`correct-${question.id}`" 
              :checked="isCorrect(idx)"
              @change="toggleCorrect(idx)"
            />
            <span class="checkmark"></span>
            <span class="correct-label">{{ $t('quiz_editor.correct_label') }}</span>
          </label>
        </div>
        <textarea 
          v-model="localQuestion.answers[idx]" 
          :placeholder="$t('quiz_config.question_editor.answer_placeholder')"
          class="answer-input"
          :readonly="isAnswerReadOnly(idx)"
          @input="emitUpdate"
        ></textarea>
      </div>
    </div>

    <!-- RANGE CONFIGURATION -->
    <div v-if="localQuestion.type === 'range'" class="range-config-panel">
      
      <div class="config-section">
        <h4 class="section-title">{{ $t('quiz_config.question_editor.range.range_unit') }}</h4>
        <div class="config-grid">
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.min_value') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.rangeConfig.minValue"
                class="config-input"
                :class="{ 'has-field-error': !!fieldError('rangeConfig') }"
                @input="emitUpdate"
              />
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.max_value') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.rangeConfig.maxValue"
                class="config-input"
                :class="{ 'has-field-error': !!fieldError('rangeConfig') }"
                @input="emitUpdate"
              />
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.step') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.rangeConfig.step"
                class="config-input"
                :class="{ 'has-field-error': !!fieldError('step') }"
                min="0"
                step="any"
                @input="emitUpdate"
              />
              <p v-if="fieldError('step')" class="field-error" role="alert">{{ fieldError('step') }}</p>
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.unit') }}</label>
              <input type="text" v-model="localQuestion.rangeConfig.unit" class="config-input" :placeholder="$t('quiz_config.question_editor.range.unit_placeholder')" @input="emitUpdate" />
           </div>
        </div>
        <p v-if="fieldError('rangeConfig')" class="field-error" role="alert">{{ fieldError('rangeConfig') }}</p>
      </div>

      <div class="config-section">
        <h4 class="section-title">{{ $t('quiz_config.question_editor.range.correct_scoring') }}</h4>
        <div class="config-grid">
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.correct_value') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.correctValue"
                class="config-input correct-value-input"
                :class="{ 'has-field-error': !!fieldError('correctValue') }"
                @input="emitUpdate"
              />
              <p v-if="fieldError('correctValue')" class="field-error" role="alert">{{ fieldError('correctValue') }}</p>
           </div>
           
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.decimals') }}</label>
              <select v-model.number="localQuestion.rangeConfig.decimalPlaces" class="select-input small-select" @change="emitUpdate">
                <option :value="0">0</option>
                <option :value="1">1</option>
                <option :value="2">2</option>
              </select>
           </div>
        </div>
        
        <div class="config-grid mt-2">
            <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.scoring_mode') }}</label>
              <select v-model="localQuestion.scoringConfig.mode" class="select-input small-select" @change="emitUpdate">
                <option value="symmetric">{{ $t('quiz_config.question_editor.range.symmetric') }}</option>
                <option value="asymmetric">{{ $t('quiz_config.question_editor.range.asymmetric') }}</option>
              </select>
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.range.tolerance') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.scoringConfig.tolerance"
                class="config-input"
                :class="{ 'has-field-error': !!fieldError('tolerance') }"
                @input="emitUpdate"
              />
              <p v-if="fieldError('tolerance')" class="field-error" role="alert">{{ fieldError('tolerance') }}</p>
           </div>
        </div>
      </div>

      <!-- Preview -->
      <div class="range-preview-container">
         <label>{{ $t('quiz_config.question_editor.range.preview') }}</label>
         <div class="preview-box">
            <RangeComponent
                :min="localQuestion.rangeConfig.minValue"
                :max="localQuestion.rangeConfig.maxValue"
                :step="localQuestion.rangeConfig.step"
                :unit="localQuestion.rangeConfig.unit"
                :decimalPlaces="localQuestion.rangeConfig.decimalPlaces"
                :modelValue="localQuestion.correctValue" 
                :readonly="true"
                :showCorrectMarker="true"
                :correctValue="localQuestion.correctValue"
            />
         </div>
         <p class="preview-note">{{ $t('quiz_config.question_editor.range.preview_note') }}</p>
      </div>

    </div>
    
    <!-- NUMBER CONFIGURATION -->
    <div v-if="localQuestion.type === 'number'" class="number-config-panel">
      <div class="config-section">
        <h4 class="section-title">{{ $t('quiz_config.question_editor.number.configuration') }}</h4>
        <div class="config-grid">
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.number.correct_answer') }}</label>
              <input
                type="number"
                v-model.number="localQuestion.correctNumber"
                class="config-input correct-value-input"
                :class="{ 'has-field-error': !!fieldError('correctNumber') }"
                @input="emitUpdate"
              />
              <p v-if="fieldError('correctNumber')" class="field-error" role="alert">{{ fieldError('correctNumber') }}</p>
           </div>
        </div>
        <p v-if="fieldError('tolerance')" class="field-error" role="alert">{{ fieldError('tolerance') }}</p>
        <p class="config-note">{{ $t('quiz_config.question_editor.number.configuration_hint') }}</p>
      </div>

      <div class="number-input-preview">
        <div class="input-placeholder">
          {{ $t('quiz.player.enter_number') }}
        </div>
      </div>
    </div>

    <!-- CORRECT PICTURE EDITOR -->
    <div
      v-if="localQuestion.type === 'correct_picture'"
      class="image-options-section"
      :class="{ 'has-field-error': !!(fieldError('imageOptions') || fieldError('correctIndex')) }"
    >
      <div v-if="fieldError('imageOptions') || fieldError('correctIndex')" class="field-error-banner" role="alert">
        <AppIcon name="alert" :size="16" aria-hidden="true" />
        <span>{{ fieldError('imageOptions') || fieldError('correctIndex') }}</span>
      </div>
      <ImageOptionEditor
        v-model="localQuestion.imageOptions"
        v-model:correct-index="localQuestion.correctIndex"
        :event-id="eventId"
        :module-id="moduleId"
        @update:modelValue="emitUpdate"
        @update:correctIndex="emitUpdate"
      />
    </div>

    <!-- WRONG PICTURES CONFIGURATION -->
    <div v-if="localQuestion.type === 'wrong_pictures'" class="wrong-pictures-config">
      <div class="config-compact-row">
        <div class="input-group mini">
          <label>{{ $t('quiz_config.question_editor.wrong_pictures.tolerance_short') }}</label>
          <input 
            type="number" 
            min="0"
            v-model.number="localQuestion.wrongPictureConfig.tolerancePercent"
            class="config-input"
            :class="{ 'has-field-error': !!fieldError('tolerancePercent') }"
            @input="emitUpdate"
          />
          <p v-if="fieldError('tolerancePercent')" class="field-error" role="alert">{{ fieldError('tolerancePercent') }}</p>
        </div>

        <div class="input-group mini center">
          <label>{{ $t('quiz_config.question_editor.wrong_pictures.show_count') }}</label>
          <label class="switch-small">
            <input 
              type="checkbox" 
              v-model="localQuestion.wrongPictureConfig.showTargetCount"
              @change="emitUpdate"
            >
            <span class="slider-round"></span>
          </label>
        </div>
      </div>
      
      <div class="image-management-section">
        <ImageUploader
          v-model="localQuestion.imageId"
          v-model:altText="localQuestion.imageAlt"
          :event-id="eventId"
          :module-id="moduleId"
          :hidePreview="true"
        />

        <div
          v-if="localQuestion.imageId"
          class="coordinate-picker-wrapper"
          :class="{ 'has-field-error': !!fieldError('targets') }"
        >
          <p v-if="fieldError('targets')" class="field-error" role="alert">{{ fieldError('targets') }}</p>
          <CoordinatePicker
            :image-id="localQuestion.imageId"
            :targets="localQuestion.wrongPictureConfig.targets"
            @update:targets="updateTargets"
          />
        </div>
        <div v-else class="image-required-warning" :class="{ 'has-field-error': !!fieldError('imageId') }">
          <AppIcon name="alert" :size="16" aria-hidden="true" /> {{ fieldError('imageId') || $t('quiz_config.question_editor.wrong_pictures.image_required') }}
        </div>
      </div>
    </div>

    <!-- SIMPLE TEXT (Simple Quiz) CONFIGURATION -->
    <div v-if="localQuestion.type === 'simple_text'" class="simple-text-config">
      <div class="input-group">
        <label>{{ $t('quiz_config.question_editor.simple_text.judge_notes') }}</label>
        <textarea
          v-model="localQuestion.judgeNotes"
          class="main-input judge-notes-input"
          :placeholder="$t('quiz_config.question_editor.simple_text.judge_notes_placeholder')"
          @input="emitUpdate"
        ></textarea>
        <p class="config-note">{{ $t('quiz_config.question_editor.simple_text.judge_notes_hint') }}</p>
      </div>
      <div class="input-group">
        <label>{{ $t('quiz_config.question_editor.simple_text.correct_answer') }}</label>
        <textarea
          v-model="localQuestion.correctAnswerText"
          class="main-input"
          :placeholder="$t('quiz_config.question_editor.simple_text.correct_answer_placeholder')"
          @input="emitUpdate"
        ></textarea>
        <p class="config-note">{{ $t('quiz_config.question_editor.simple_text.correct_answer_hint') }}</p>
      </div>
    </div>

    </div>
  </div>
</template>

<script setup>
import { reactive, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import RangeComponent from './RangeComponent.vue'
import ImageUploader from './ImageUploader.vue'
import ImageOptionEditor from './ImageOptionEditor.vue'
import CoordinatePicker from './CoordinatePicker.vue'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
const props = defineProps({
  question: {
    type: Object,
    required: true
  },
  index: {
    type: Number,
    required: true
  },
  eventId: {
    type: String,
    default: ''
  },
  moduleId: {
    type: String,
    default: ''
  },
  lockedType: {
    type: [String, Boolean],
    default: ''
  },
  errors: {
    type: Object,
    default: null
  },
  active: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update', 'delete', 'change-type'])

const isTypeLocked = computed(() => Boolean(props.lockedType))
const hasAnyErrors = computed(() => Boolean(props.errors && Object.keys(props.errors).length > 0))
const fieldError = (field) => {
  const err = props.errors?.[field]
  return err ? t(err.messageKey, err.params || {}) : ''
}
const isChoiceType = computed(() => (
  !['range', 'number', 'correct_picture', 'wrong_pictures', 'simple_text'].includes(localQuestion.type)
))
const choiceSectionError = computed(() => (
  fieldError('answers') || fieldError('correctIndex') || fieldError('correctIndices')
))

// Initialize local state from props
const localQuestion = reactive({
  ...props.question,
  type: props.question.type || 'classic',
  answers: props.question.answers && props.question.answers.length >= 2 
    ? [...props.question.answers] 
    : ['', '', '', ''],
  explainTime: props.question.explainTime !== undefined ? props.question.explainTime : 5,
  timeLimit: props.question.timeLimit !== undefined ? props.question.timeLimit : 20,
  correctIndices: props.question.correctIndices || [],
  // Initialize range config if missing
  rangeConfig: props.question.rangeConfig || {
      minValue: 0,
      maxValue: 100,
      step: 1,
      unit: '',
      decimalPlaces: 0
  },
  // A7: default maxPoints is 100 for range, 50 for every other
  // scoringConfig-based type, so the Points field matches game scoring.
  scoringConfig: props.question.scoringConfig || {
      mode: 'symmetric',
      maxPoints: props.question.type === 'range' ? 100 : 50,
      tolerance: 10,
      partialPercent: 0
  },
  correctValue: props.question.correctValue !== undefined ? props.question.correctValue : 0,
  correctNumber: props.question.correctNumber !== undefined ? props.question.correctNumber : 0,
  imageId: props.question.imageId || null,
  imageAlt: props.question.imageAlt || '',
  imageOptions: props.question.imageOptions || [
    { id: '1', imageId: '', label: '' },
    { id: '2', imageId: '', label: '' }
  ],
  wrongPictureConfig: props.question.wrongPictureConfig || {
      targets: [],
      tolerancePercent: 10,
      maxPoints: 50
  },
  judgeNotes: props.question.judgeNotes || '',
  correctAnswerText: props.question.correctAnswerText || ''
})

// Unified Points Accessor
const localMaxPoints = computed({
  get() {
    if (localQuestion.type === 'wrong_pictures') {
      return localQuestion.wrongPictureConfig.maxPoints
    }
    return localQuestion.scoringConfig.maxPoints
  },
  set(val) {
    if (localQuestion.type === 'wrong_pictures') {
      localQuestion.wrongPictureConfig.maxPoints = val
    } else {
      localQuestion.scoringConfig.maxPoints = val
    }
  }
})

const BASE_TIME_LIMITS = [5, 10, 20, 30, 45, 60, 90, 120]
const timeLimitOptions = computed(() => {
  const options = [...BASE_TIME_LIMITS]
  const current = Number(localQuestion.timeLimit)
  if (Number.isFinite(current) && current > 0 && !options.includes(current)) {
    options.push(current)
    options.sort((a, b) => a - b)
  }
  return options
})

// Watch for image changes and emit update
watch(() => localQuestion.imageId, () => {
  emitUpdate()
})

watch(() => localQuestion.imageAlt, () => {
  emitUpdate()
})

watch(() => props.question, (newVal) => {
  // Deep tracking can be tricky, but we want to update if parent logic changes something 
  // without overwriting local typing immediately if we are the ones who triggered it.
  // For simplicity, we sync back basic fields if ID changes or strict updates needed.
  // Ideally, valid Vue form pattern is :modelValue + @update:modelValue.
  // Here we just ensure we don't lose the range/scoring config objects structure.
  
  if (newVal.id !== localQuestion.id || newVal.type !== localQuestion.type) {
       const sameQuestion = newVal.id === localQuestion.id
       const previousType = localQuestion.type
       const typeChanged = newVal.type !== localQuestion.type
       Object.assign(localQuestion, {
          ...newVal,
          type: newVal.type || 'classic',
          answers: newVal.answers && newVal.answers.length >= 2 ? [...newVal.answers] : ['', '', '', ''],
          correctIndices: Array.isArray(newVal.correctIndices) ? [...newVal.correctIndices] : [],
          rangeConfig: newVal.rangeConfig || { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 },
          scoringConfig: newVal.scoringConfig || { mode: 'symmetric', maxPoints: newVal.type === 'range' ? 100 : 50, tolerance: 10, partialPercent: 0 },
          explainTime: newVal.explainTime !== undefined ? newVal.explainTime : 5,
          correctValue: newVal.correctValue !== undefined ? newVal.correctValue : 0,
          correctNumber: newVal.correctNumber !== undefined ? newVal.correctNumber : 0,
          imageId: newVal.imageId || null,
          imageAlt: newVal.imageAlt || '',
          imageOptions: newVal.imageOptions || [{ id: '1', imageId: '', label: '' }, { id: '2', imageId: '', label: '' }],
          wrongPictureConfig: newVal.wrongPictureConfig || { targets: [], tolerancePercent: 10, maxPoints: 50 },
          judgeNotes: newVal.judgeNotes || '',
          correctAnswerText: newVal.correctAnswerText || ''
      })
      if (typeChanged) {
          handleTypeChange(sameQuestion ? previousType : null)
      }
  }
}, { deep: true })

const emitUpdate = () => {
  emit('update', { ...localQuestion })
}

const handleTypeChange = (previousType = null) => {
    if (localQuestion.type === 'true_false') {
        localQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
        if (localQuestion.correctIndex > 1) localQuestion.correctIndex = 0
        // A7: choice scoring reads scoringConfig.maxPoints.
        if (!localQuestion.scoringConfig) {
            localQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
        } else if (!localQuestion.scoringConfig.maxPoints) {
            localQuestion.scoringConfig.maxPoints = 50
        }
    } else if (localQuestion.type === 'range') {
        // Ensure defaults are set
        if (!localQuestion.rangeConfig.maxValue) localQuestion.rangeConfig.maxValue = 100
        if (!localQuestion.scoringConfig.maxPoints) localQuestion.scoringConfig.maxPoints = 100
    } else if (localQuestion.type === 'number') {
        if (!localQuestion.scoringConfig.maxPoints) localQuestion.scoringConfig.maxPoints = 50
    } else if (localQuestion.type === 'correct_picture') {
        // Ensure image options exist
        if (!localQuestion.imageOptions || localQuestion.imageOptions.length === 0) {
            localQuestion.imageOptions = [
                { id: Date.now().toString(), imageId: '', label: '' },
                { id: (Date.now() + 1).toString(), imageId: '', label: '' }
            ]
        }
        // A7: choice scoring reads scoringConfig.maxPoints, so it must exist.
        if (!localQuestion.scoringConfig) {
            localQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
        } else if (!localQuestion.scoringConfig.maxPoints) {
            localQuestion.scoringConfig.maxPoints = 50
        }
    } else if (localQuestion.type === 'wrong_pictures') {
        if (!localQuestion.wrongPictureConfig) {
             localQuestion.wrongPictureConfig = { targets: [], tolerancePercent: 10, maxPoints: 50 }
        }
    } else if (localQuestion.type === 'simple_text') {
        if (!localQuestion.scoringConfig) {
            localQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 }
        } else if (!localQuestion.scoringConfig.maxPoints) {
            localQuestion.scoringConfig.maxPoints = 100
        }
        if (localQuestion.judgeNotes === undefined) localQuestion.judgeNotes = ''
        if (localQuestion.correctAnswerText === undefined) localQuestion.correctAnswerText = ''
        if (localQuestion.timeLimit === undefined) localQuestion.timeLimit = 0
    } else {
        // Reset localized T/F options when switching away from true_false
        if (previousType === 'true_false') {
             localQuestion.answers = ['', '', '', '']
        }
        // A7: classic / true_false / multi_select score from
        // scoringConfig.maxPoints, so backfill it when missing.
        if (['classic', 'true_false', 'multi_select'].includes(localQuestion.type)) {
            if (!localQuestion.scoringConfig) {
                localQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
            } else if (!localQuestion.scoringConfig.maxPoints) {
                localQuestion.scoringConfig.maxPoints = 50
            }
        }
    }

    if (localQuestion.type === 'multi_select' && localQuestion.correctIndices.length === 0) {
        if (typeof localQuestion.correctIndex === 'number') {
            localQuestion.correctIndices = [localQuestion.correctIndex]
        }
    }
    emitUpdate()
}

const updateTargets = (newTargets) => {
    localQuestion.wrongPictureConfig.targets = [...newTargets]
    emitUpdate()
}

// Warning shown when multi_select has fewer than 2 correct answers selected
const multiSelectWarning = computed(() => {
  if (localQuestion.type !== 'multi_select') return false
  return (localQuestion.correctIndices || []).length < 2
})

const isCorrect = (idx) => {
    if (localQuestion.type === 'multi_select') {
        return localQuestion.correctIndices.includes(idx)
    }
    return localQuestion.correctIndex === idx
}

const toggleCorrect = (idx) => {
    if (localQuestion.type === 'multi_select') {
        const i = localQuestion.correctIndices.indexOf(idx)
        if (i > -1) {
            localQuestion.correctIndices.splice(i, 1)
        } else {
            localQuestion.correctIndices.push(idx)
            localQuestion.correctIndices.sort()
        }
    } else {
        localQuestion.correctIndex = idx
    }
    emitUpdate()
}

const getAnswerColorClass = (idx) => {
  if (localQuestion.type === 'true_false') {
      return idx === 0 ? 'answer-blue' : 'answer-red'
  }
  const colors = ['red', 'blue', 'yellow', 'green']
  return `answer-${colors[idx]}`
}

const getShapeIcon = (idx) => {
  if (localQuestion.type === 'true_false') {
      return idx === 0 ? 'check' : 'close'
  }
  const shapes = ['triangle', 'diamond', 'circle', 'square']
  return shapes[idx]
}

const isAnswerVisible = (idx) => {
    if (localQuestion.type === 'true_false') return idx < 2
    return true
}

const isAnswerReadOnly = (idx) => {
    return localQuestion.type === 'true_false'
}
</script>

<style scoped>
.question-editor {
  container: question-editor / inline-size;
  background-color: #2C2C2E;
  border-radius: 12px;
  padding: 0;
  margin-bottom: 24px;
  border: 1px solid #3A3A3C;
  width: 100%;
  min-inline-size: 0;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.question-editor.is-active {
  border-color: var(--admin-primary, #0A84FF);
  box-shadow: 0 0 0 2px rgba(10, 132, 255, 0.22);
}

.question-editor.has-errors {
  border-color: var(--admin-danger, #FF453A);
}

.question-editor,
.question-editor *,
.question-editor *::before,
.question-editor *::after {
  box-sizing: border-box;
}

.question-editor-inner {
  padding: 16px;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #3A3A3C;
}

.header-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  min-inline-size: 0;
}

.question-type-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background-color: #3A3A3C;
  border: 1px solid transparent;
  color: #E5E5EA;
  font-family: inherit;
  font-size: 11px;
  font-weight: 700;
  line-height: 1.2;
  padding: 4px 10px;
  border-radius: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.question-type-tag:hover {
  background-color: #48484A;
}

.question-type-tag.question-type-tag--locked {
  cursor: default;
}

.question-type-tag.question-type-tag--locked:hover {
  background-color: #3A3A3C;
}

.question-number {
  font-weight: bold;
  color: #8E8E93;
  text-transform: uppercase;
  font-size: 12px;
}

.delete-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: none;
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 6px;
  color: #8E8E93;
  cursor: pointer;
  font-size: 16px;
  opacity: 0.7;
}

.delete-btn:hover {
  opacity: 1;
}

.input-group {
  margin-bottom: 16px;
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.input-group label {
  font-size: 12px;
  color: #8E8E93;
  margin-bottom: 6px;
  font-weight: 600;
}

.main-input, .select-input, .config-input {
  width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  padding: 10px;
  color: white;
  font-size: 16px;
}

.editor-main-row {
  display: flex;
  gap: 20px;
  margin-bottom: 16px;
  align-items: flex-start;
}

.image-section {
  flex: 60;
  min-inline-size: 0;
  display: flex;
  flex-direction: column;
}

.image-section label {
  font-size: 12px;
  color: #8E8E93;
  margin-bottom: 6px;
  font-weight: 600;
}

.settings-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 40;
  flex-shrink: 0;
  min-inline-size: 0;
}

.settings-stack .input-group {
  margin-bottom: 0;
}

.text-group {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #3A3A3C;
}

.default-hint {
  color: #8E8E93;
  font-weight: normal;
  font-style: italic;
  font-size: 10px;
  opacity: 0.7;
  margin-left: 4px;
}

.answers-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.true-false-grid {
  grid-template-columns: 1fr;
}

.answer-card {
  border-radius: 8px;
  padding: 10px;
  position: relative;
  min-inline-size: 0;
}

.answer-card.answer-red { background-color: rgba(255, 59, 48, 0.15); border: 1px solid rgba(255, 59, 48, 0.3); }
.answer-card.answer-blue { background-color: rgba(10, 132, 255, 0.15); border: 1px solid rgba(10, 132, 255, 0.3); }
.answer-card.answer-yellow { background-color: rgba(255, 214, 10, 0.15); border: 1px solid rgba(255, 214, 10, 0.3); }
.answer-card.answer-green { background-color: rgba(50, 215, 75, 0.15); border: 1px solid rgba(50, 215, 75, 0.3); }

.answer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.shape-icon {
  font-size: 14px;
  opacity: 0.8;
}

.answer-input {
  width: 100%;
  background: transparent;
  border: none;
  color: white;
  font-size: 14px;
  resize: vertical; 
  min-height: 40px;
}

.answer-input:focus {
  outline: none;
  background-color: rgba(0,0,0,0.1);
  border-radius: 4px;
}

.correct-toggle {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  min-width: 24px;
  min-height: 24px;
  cursor: pointer;
  flex-shrink: 0;
}

.correct-label {
  font-size: 12px;
  font-weight: 600;
  color: #E5E5EA;
  user-select: none;
}

.answer-card.is-correct .correct-label {
  color: #32D74B;
}

.field-error {
  margin: 6px 0 0;
  color: var(--admin-danger, #FF453A);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
}

.field-error-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: var(--admin-danger-soft, rgba(255, 69, 58, 0.14));
  border: 1px solid var(--admin-danger, #FF453A);
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 12px;
  color: var(--admin-danger, #FF6961);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
}

.has-field-error {
  border-color: var(--admin-danger, #FF453A) !important;
  box-shadow: 0 0 0 2px rgba(255, 69, 58, 0.24) !important;
}

.answers-grid.has-field-error,
.image-options-section.has-field-error,
.coordinate-picker-wrapper.has-field-error {
  border: 1px solid var(--admin-danger, #FF453A);
  border-radius: 10px;
  padding: 8px;
}

:global(html[data-admin-appearance="true"] .question-editor.is-active) {
  border-color: var(--admin-primary) !important;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--admin-primary) 24%, transparent) !important;
}

:global(html[data-admin-appearance="true"] .question-editor.has-errors) {
  border-color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .question-editor .has-field-error) {
  border-color: var(--admin-danger) !important;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--admin-danger) 24%, transparent) !important;
}

:global(html[data-admin-appearance="true"] .question-editor .correct-label) {
  color: var(--admin-text);
}

:global(html[data-admin-appearance="true"] .question-editor .answer-card.is-correct .correct-label) {
  color: var(--admin-success);
}

:global(html[data-admin-appearance="true"] .question-editor .question-type-tag--locked:hover) {
  background: var(--admin-primary-soft) !important;
}

.correct-toggle input {
  position: absolute;
  inset: 0;
  opacity: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  cursor: pointer;
}

.checkmark {
  position: relative;
  display: block;
  height: 24px;
  width: 24px;
  background-color: rgba(255,255,255,0.1);
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 50%;
  cursor: pointer;
}

.correct-toggle input:focus-visible ~ .checkmark {
  outline: 3px solid var(--admin-focus, #0A84FF);
  outline-offset: 2px;
}

.correct-toggle input:checked ~ .checkmark {
  background-color: #32D74B; 
  border-color: #32D74B;
}

.checkmark:after {
  content: "";
  position: absolute;
  display: none;
}

.correct-toggle input:checked ~ .checkmark:after {
  display: block;
}

.correct-toggle .checkmark:after {
  left: 7px;
  top: 3px;
  width: 5px;
  height: 10px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

/* RANGE CONFIG STYLES */
.range-config-panel {
    background: #232325;
    padding: 16px;
    border-radius: 8px;
    border: 1px dashed #3A3A3C;
}

.section-title {
    font-size: 0.9rem;
    color: #8E8E93;
    margin-bottom: 12px;
    text-transform: uppercase;
    letter-spacing: 1px;
}

.config-section {
    margin-bottom: 24px;
}

.config-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
}

.mt-2 { margin-top: 12px; }

.small-select {
    padding: 10px;
}

.correct-value-input {
    border-color: #32D74B;
    background: rgba(50, 215, 75, 0.1);
}

:global(html[data-admin-appearance="true"] .question-editor .input-group input.config-input.correct-value-input) {
    background: var(--admin-success-soft, rgba(50, 215, 75, 0.1)) !important;
    border-color: var(--admin-success, #32D74B) !important;
    color: var(--admin-text, #ffffff) !important;
}

.preview-box {
    background: #1C1C1E;
    border: 1px solid #3A3A3C;
    border-radius: 8px;
    padding: 0 20px;
    display: flex;
    justify-content: center;
}

.preview-note {
    font-size: 11px;
    color: #8E8E93;
    text-align: center;
    margin-top: 6px;
    font-style: italic;
}

.multi-select-warning,
.image-required-warning {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: rgba(255, 159, 10, 0.12);
  border: 1px solid rgba(255, 159, 10, 0.45);
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 12px;
  color: #FFB340;
  font-size: 13px;
  line-height: 1.4;
}

.multi-select-warning strong {
  color: #FFCC00;
}

.wrong-pictures-config {
  background: #232325;
  padding: 16px;
  border-radius: 8px;
  border: 1px dashed #3A3A3C;
}

.config-compact-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  align-items: flex-end;
}

.mini {
  width: 80px;
}

.center {
  align-items: center;
}

.image-management-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.coordinate-picker-wrapper {
  border-top: 1px solid #3A3A3C;
  padding-top: 16px;
}

.coordinate-picker-wrapper > label {
  font-size: 13px;
  color: #E5E5EA;
  font-weight: 600;
  display: block;
  margin-bottom: 8px;
}

.number-config-panel {
  background: #232325;
  padding: 16px;
  border-radius: 8px;
  border: 1px dashed #3A3A3C;
}

.simple-text-config {
  background: #232325;
  padding: 16px;
  border-radius: 8px;
  border: 1px dashed #3A3A3C;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.judge-notes-input {
  min-height: 64px;
  resize: vertical;
}

.config-note {
  font-size: 11px;
  color: #8E8E93;
  margin-top: 12px;
  font-style: italic;
}

.number-input-preview {
  margin-top: 20px;
  background: #1C1C1E;
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.input-placeholder {
  color: #48484A;
  font-size: 18px;
  font-weight: 500;
}

/* Small Switch Styles */
.switch-small {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
}

.switch-small input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider-round {
  position: absolute;
  cursor: pointer;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: #3A3A3C;
  transition: .4s;
  border-radius: 20px;
}

.slider-round:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: .4s;
  border-radius: 50%;
}

.switch-small input:checked + .slider-round {
  background-color: #32D74B;
}

.switch-small input:checked + .slider-round:before {
  transform: translateX(16px);
}

.switch-small input:focus-visible + .slider-round {
  outline: 3px solid var(--admin-focus, #0A84FF);
  outline-offset: 2px;
}

@container question-editor (inline-size <= 580px) {
  .question-editor-inner {
    padding: 12px;
  }

  .editor-main-row {
    flex-direction: column;
    gap: 14px;
  }

  .image-section,
  .settings-stack {
    flex: none;
    width: 100%;
  }

  .answers-grid {
    grid-template-columns: 1fr;
  }

  .config-compact-row {
    flex-wrap: wrap;
    align-items: stretch;
  }

  .config-compact-row .mini {
    flex: 1 1 135px;
    width: auto;
    min-inline-size: 0;
  }

  .config-compact-row .mini.center {
    align-items: flex-start;
  }

  .range-config-panel,
  .number-config-panel,
  .wrong-pictures-config,
  .simple-text-config {
    padding: 12px;
  }

  .preview-box {
    padding: 0 12px;
  }
}

@container question-editor (inline-size <= 460px) {
  .config-grid {
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 135px), 1fr));
  }
}

@media (pointer: coarse) {
  .delete-btn,
  .question-type-tag {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .question-type-tag {
    padding: 8px 12px;
  }

  .correct-toggle {
    width: auto;
    height: auto;
    padding: 0 6px;
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .select-input,
  .main-input,
  .config-input {
    min-block-size: 44px;
  }

  .answer-input {
    min-block-size: 44px;
    font-size: 16px;
  }

  .switch-small::after {
    content: "";
    position: absolute;
    top: 50%;
    left: 50%;
    width: 44px;
    height: 44px;
    transform: translate(-50%, -50%);
  }
}
</style>
