<template>
  <div class="question-editor">
    <div class="editor-header">
      <span class="question-number">Question {{ index + 1 }}</span>
      <button class="delete-btn" @click="$emit('delete')">🗑</button>
    </div>

    <div class="input-group">
      <label>Question Type</label>
      <select v-model="localQuestion.type" class="select-input" @change="handleTypeChange">
        <option value="multiple_choice">Single Choice</option>
        <option value="true_false">True / False</option>
      </select>
    </div>

    <div class="input-group">
      <label>Question Text</label>
      <input 
        type="text" 
        v-model="localQuestion.text" 
        class="main-input"
        placeholder="e.g., What is the capital of France?"
        @input="emitUpdate"
      />
    </div>

    <div class="settings-row">
      <div class="input-group">
        <label>Time Limit (seconds)</label>
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

    <div class="answers-grid" :class="{ 'true-false-grid': localQuestion.type === 'true_false' }">
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
              type="radio" 
              :name="`correct-${question.id}`" 
              :checked="localQuestion.correctIndex === idx"
              @change="setCorrect(idx)"
            />
            <span class="checkmark" title="Mark as correct"></span>
          </label>
        </div>
        <textarea 
          v-model="localQuestion.answers[idx]" 
          placeholder="Type answer option"
          class="answer-input"
          :readonly="isAnswerReadOnly(idx)"
          @input="emitUpdate"
        ></textarea>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'

const props = defineProps({
  question: {
    type: Object,
    required: true
  },
  index: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['update', 'delete'])

// Initialize local state from props
const localQuestion = reactive({
  ...props.question,
  type: props.question.type || 'multiple_choice',
  // Ensure we have 4 answers for MC, but for T/F we might only have 2 effectively used
  answers: props.question.answers && props.question.answers.length >= 2 
    ? [...props.question.answers] 
    : ['', '', '', '']
})

watch(() => props.question, (newVal) => {
  Object.assign(localQuestion, {
      ...newVal,
      type: newVal.type || 'multiple_choice'
  })
}, { deep: true })

const emitUpdate = () => {
  emit('update', { ...localQuestion })
}

const handleTypeChange = () => {
    if (localQuestion.type === 'true_false') {
        localQuestion.answers = ['True', 'False', '', '']
        if (localQuestion.correctIndex > 1) localQuestion.correctIndex = 0
    } else {
        // Reset to empty if switching back, or keep if existing
        if (localQuestion.answers[0] === 'True' && localQuestion.answers[1] === 'False') {
             localQuestion.answers = ['', '', '', '']
        }
    }
    emitUpdate()
}

const setCorrect = (idx) => {
  localQuestion.correctIndex = idx
  emitUpdate()
}

const getAnswerColorClass = (idx) => {
  if (localQuestion.type === 'true_false') {
      return idx === 0 ? 'answer-blue' : 'answer-red' // True=Blue/Green, False=Red commonly
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

.main-input, .select-input {
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
  resize: vertical; /* Allow resizing if needed */
  min-height: 40px;
}

.answer-input:focus {
  outline: none;
  background-color: rgba(0,0,0,0.1);
  border-radius: 4px;
}

/* Custom Radio Button for Correct Answer */
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
  background-color: #32D74B; /* Green */
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

@media (max-width: 600px) {
  .answers-grid {
    grid-template-columns: 1fr;
  }
}
</style>
