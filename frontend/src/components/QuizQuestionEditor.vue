<template>
  <div class="question-editor">
    <div class="editor-header">
      <span class="question-number">{{ $t('quiz_config.question_editor.question_number', { number: index + 1 }) }}</span>
      <button class="delete-btn" @click="$emit('delete')">🗑</button>
    </div>

    <div class="input-group">
      <label>{{ $t('quiz_config.question_editor.type_label') }}</label>
      <select v-model="localQuestion.type" class="select-input" @change="handleTypeChange">
        <option value="multiple_choice">{{ $t('quiz_config.question_editor.types.multiple_choice') }}</option>
        <option value="multi_select">{{ $t('quiz_config.question_editor.types.multi_select') }}</option>
        <option value="true_false">{{ $t('quiz_config.question_editor.types.true_false') }}</option>
        <option value="slider">{{ $t('quiz_config.question_editor.types.slider') }}</option>
        <option value="correct_picture">{{ $t('quiz_config.question_editor.types.correct_picture') }}</option>
      </select>
    </div>

    <div class="input-group">
      <label>{{ $t('quiz_config.question_editor.text_label') }}</label>
      <input 
        type="text" 
        v-model="localQuestion.text" 
        class="main-input"
        :placeholder="$t('quiz_config.question_editor.text_placeholder')"
        @input="emitUpdate"
      />
    </div>

    <div class="input-group">
      <label>{{ $t('quiz_config.question_editor.image_label') }}</label>
      <ImageUploader
        v-model="localQuestion.imageId"
        v-model:altText="localQuestion.imageAlt"
        :event-id="eventId"
        :module-id="moduleId"
      />
    </div>

    <div class="settings-row">
      <div class="input-group">
        <label>{{ $t('quiz_config.question_editor.time_limit') }}</label>
        <select v-model.number="localQuestion.timeLimit" class="select-input" @change="emitUpdate">
          <option :value="5">5s</option>
          <option :value="10">10s</option>
          <option :value="20">20s</option>
          <option :value="30">30s</option>
          <option :value="60">60s</option>
          <option :value="90">90s</option>
          <option :value="120">120s</option>
        </select>
      </div>
    </div>

    <!-- STANDARD ANSWERS (MC/TF/MS) -->
    <div v-if="localQuestion.type !== 'slider' && localQuestion.type !== 'correct_picture'" class="answers-grid" :class="{ 'true-false-grid': localQuestion.type === 'true_false' }">
      <div 
        v-for="(answer, idx) in localQuestion.answers" 
        :key="idx"
        class="answer-card"
        :class="[getAnswerColorClass(idx), { 'hidden-answer': !isAnswerVisible(idx) }]"
        v-show="isAnswerVisible(idx)"
      >
        <div class="answer-header">
          <span class="shape-icon">{{ getShapeIcon(idx) }}</span>
          <label class="correct-toggle">
            <input 
              :type="localQuestion.type === 'multi_select' ? 'checkbox' : 'radio'"
              :name="`correct-${question.id}`" 
              :checked="isCorrect(idx)"
              @change="toggleCorrect(idx)"
            />
            <span class="checkmark" title="Mark as correct"></span>
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

    <!-- SLIDER CONFIGURATION -->
    <div v-if="localQuestion.type === 'slider'" class="slider-config-panel">
      
      <div class="config-section">
        <h4 class="section-title">{{ $t('quiz_config.question_editor.slider.range_unit') }}</h4>
        <div class="config-grid">
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.min_value') }}</label>
              <input type="number" v-model.number="localQuestion.sliderConfig.minValue" class="config-input" @input="emitUpdate" />
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.max_value') }}</label>
              <input type="number" v-model.number="localQuestion.sliderConfig.maxValue" class="config-input" @input="emitUpdate" />
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.step') }}</label>
              <input type="number" v-model.number="localQuestion.sliderConfig.step" class="config-input" min="0" step="any" @input="emitUpdate" />
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.unit') }}</label>
              <input type="text" v-model="localQuestion.sliderConfig.unit" class="config-input" :placeholder="$t('quiz_config.question_editor.slider.unit_placeholder')" @input="emitUpdate" />
           </div>
        </div>
      </div>

      <div class="config-section">
        <h4 class="section-title">{{ $t('quiz_config.question_editor.slider.correct_scoring') }}</h4>
        <div class="config-grid">
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.correct_value') }}</label>
              <input type="number" v-model.number="localQuestion.correctValue" class="config-input correct-value-input" @input="emitUpdate" />
           </div>
           
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.decimals') }}</label>
              <select v-model.number="localQuestion.sliderConfig.decimalPlaces" class="select-input small-select" @change="emitUpdate">
                <option :value="0">0</option>
                <option :value="1">1</option>
                <option :value="2">2</option>
              </select>
           </div>
        </div>
        
        <div class="config-grid mt-2">
            <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.scoring_mode') }}</label>
              <select v-model="localQuestion.scoringConfig.mode" class="select-input small-select" @change="emitUpdate">
                <option value="symmetric">{{ $t('quiz_config.question_editor.slider.symmetric') }}</option>
                <option value="asymmetric">{{ $t('quiz_config.question_editor.slider.asymmetric') }}</option>
              </select>
           </div>
           <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.tolerance') }}</label>
              <input type="number" v-model.number="localQuestion.scoringConfig.tolerance" class="config-input" @input="emitUpdate" />
           </div>
            <div class="input-group">
              <label>{{ $t('quiz_config.question_editor.slider.max_points') }}</label>
              <input type="number" v-model.number="localQuestion.scoringConfig.maxPoints" class="config-input" @input="emitUpdate" />
           </div>
        </div>
      </div>

      <!-- Preview -->
      <div class="slider-preview-container">
         <label>{{ $t('quiz_config.question_editor.slider.preview') }}</label>
         <div class="preview-box">
            <SliderComponent
                :min="localQuestion.sliderConfig.minValue"
                :max="localQuestion.sliderConfig.maxValue"
                :step="localQuestion.sliderConfig.step"
                :unit="localQuestion.sliderConfig.unit"
                :decimalPlaces="localQuestion.sliderConfig.decimalPlaces"
                :modelValue="localQuestion.correctValue" 
                :readonly="true"
                :showCorrectMarker="true"
                :correctValue="localQuestion.correctValue"
            />
         </div>
         <p class="preview-note">{{ $t('quiz_config.question_editor.slider.preview_note') }}</p>
      </div>

    </div>

    <!-- CORRECT PICTURE EDITOR -->
    <div v-if="localQuestion.type === 'correct_picture'" class="image-options-section">
      <ImageOptionEditor
        v-model="localQuestion.imageOptions"
        v-model:correct-index="localQuestion.correctIndex"
        :event-id="eventId"
        :module-id="moduleId"
        @update:modelValue="emitUpdate"
        @update:correctIndex="emitUpdate"
      />
    </div>

  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SliderComponent from './SliderComponent.vue'
import ImageUploader from './ImageUploader.vue'
import ImageOptionEditor from './ImageOptionEditor.vue'

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
  }
})

const emit = defineEmits(['update', 'delete'])

// Initialize local state from props
const localQuestion = reactive({
  ...props.question,
  type: props.question.type || 'multiple_choice',
  answers: props.question.answers && props.question.answers.length >= 2 
    ? [...props.question.answers] 
    : ['', '', '', ''],
  correctIndices: props.question.correctIndices || [],
  // Initialize slider config if missing
  sliderConfig: props.question.sliderConfig || {
      minValue: 0,
      maxValue: 100,
      step: 1,
      unit: '',
      decimalPlaces: 0
  },
  scoringConfig: props.question.scoringConfig || {
      mode: 'symmetric',
      maxPoints: 1000,
      tolerance: 10,
      partialPercent: 0
  },
  correctValue: props.question.correctValue !== undefined ? props.question.correctValue : 0,
  imageId: props.question.imageId || null,
  imageAlt: props.question.imageAlt || '',
  imageOptions: props.question.imageOptions || [
    { id: '1', imageId: '', label: '' },
    { id: '2', imageId: '', label: '' }
  ]
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
  // Here we just ensure we don't lose the slider/scoring config objects structure.
  
  if (newVal.id !== localQuestion.id) {
       Object.assign(localQuestion, {
          ...newVal,
          type: newVal.type || 'multiple_choice',
          sliderConfig: newVal.sliderConfig || { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 },
          scoringConfig: newVal.scoringConfig || { mode: 'symmetric', maxPoints: 1000, tolerance: 10, partialPercent: 0 },
          correctValue: newVal.correctValue !== undefined ? newVal.correctValue : 0,
          imageId: newVal.imageId || null,
          imageAlt: newVal.imageAlt || '',
          imageOptions: newVal.imageOptions || [{ id: '1', imageId: '', label: '' }, { id: '2', imageId: '', label: '' }]
      })
  }
}, { deep: true })

const emitUpdate = () => {
  emit('update', { ...localQuestion })
}

const handleTypeChange = () => {
    if (localQuestion.type === 'true_false') {
        localQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
        if (localQuestion.correctIndex > 1) localQuestion.correctIndex = 0
    } else if (localQuestion.type === 'slider') {
        // Ensure defaults are set
        if (!localQuestion.sliderConfig.maxValue) localQuestion.sliderConfig.maxValue = 100
        if (!localQuestion.scoringConfig.maxPoints) localQuestion.scoringConfig.maxPoints = 1000
    } else if (localQuestion.type === 'correct_picture') {
        // Ensure image options exist
        if (!localQuestion.imageOptions || localQuestion.imageOptions.length === 0) {
            localQuestion.imageOptions = [
                { id: Date.now().toString(), imageId: '', label: '' },
                { id: (Date.now() + 1).toString(), imageId: '', label: '' }
            ]
        }
    } else {
        // Reset to empty if switching back from T/F
        if (localQuestion.answers[0] === 'True' && localQuestion.answers[1] === 'False') {
             localQuestion.answers = ['', '', '', '']
        }
    }

    if (localQuestion.type === 'multi_select' && localQuestion.correctIndices.length === 0) {
        if (typeof localQuestion.correctIndex === 'number') {
            localQuestion.correctIndices = [localQuestion.correctIndex]
        }
    }
    emitUpdate()
}

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
      return idx === 0 ? '✔️' : '❌'
  }
  const shapes = ['▲', '◆', '●', '■']
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
  background-color: #2C2C2E;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 24px;
  border: 1px solid #3A3A3C;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.question-number {
  font-weight: bold;
  color: #8E8E93;
  text-transform: uppercase;
  font-size: 12px;
}

.delete-btn {
  background: none;
  border: none;
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
}

.input-group label {
  font-size: 12px;
  color: #8E8E93;
  margin-bottom: 6px;
  font-weight: 600;
}

.main-input, .select-input, .config-input {
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  padding: 10px;
  color: white;
  font-size: 16px;
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
}

.answer-card.answer-red { background-color: rgba(255, 59, 48, 0.15); border: 1px solid rgba(255, 59, 48, 0.3); }
.answer-card.answer-blue { background-color: rgba(10, 132, 255, 0.15); border: 1px solid rgba(10, 132, 255, 0.3); }
.answer-card.answer-yellow { background-color: rgba(255, 214, 10, 0.15); border: 1px solid rgba(255, 214, 10, 0.3); }
.answer-card.answer-green { background-color: rgba(50, 215, 75, 0.15); border: 1px solid rgba(50, 215, 75, 0.3); }

.answer-header {
  display: flex;
  justify-content: space-between;
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
  display: inline-block;
  width: 24px;
  height: 24px;
}

.correct-toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.checkmark {
  position: absolute;
  top: 0;
  left: 0;
  height: 24px;
  width: 24px;
  background-color: rgba(255,255,255,0.1);
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 50%;
  cursor: pointer;
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
  left: 8px;
  top: 4px;
  width: 5px;
  height: 10px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

/* SLIDER CONFIG STYLES */
.slider-config-panel {
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

@media (max-width: 600px) {
  .answers-grid {
    grid-template-columns: 1fr;
  }
}
</style>
