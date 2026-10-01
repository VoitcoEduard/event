<template>
  <div class="quiz-editor">
    <div class="scroll-content">
      <div v-if="questions.length === 0" class="empty-state">
        <AppIcon class="empty-icon" name="quiz" :size="42" />
        <p class="empty-text">{{ $t('quiz_editor.empty_text') }}</p>
      </div>
      
      <div v-else class="questions-list">
        <div v-for="(question, index) in questions" :key="question.id" class="question-wrapper-outer">
           <div class="question-sort-controls">
              <button @click="emit('move-question', index, -1)" :disabled="index === 0" class="move-btn">↑</button>
              <button @click="emit('move-question', index, 1)" :disabled="index === questions.length - 1" class="move-btn">↓</button>
           </div>
           <div class="question-component-wrapper">
              <QuizQuestionEditor 
                 :question="question" 
                 :index="index"
                 :event-id="eventId"
                 :module-id="moduleId"
                 @update="(updatedQ) => emit('update-question', index, updatedQ)"
                 @delete="emit('remove-question', index)"
                 @change-type="handleChangeType(index)"
              />
            </div>
        </div>
      </div>
      
      <div class="add-section">
        <button class="add-btn" @click="handleAddClick">
          {{ $t('quiz_editor.add_question') }}
        </button>
      </div>

      <QuestionTypeSelector 
        :visible="showTypeSelector"
        @close="showTypeSelector = false"
        @select="handleTypeSelect"
      />

      <div style="height: 100px;"></div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import QuizQuestionEditor from './QuizQuestionEditor.vue'
import QuestionTypeSelector from './QuestionTypeSelector.vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  questions: { type: Array, default: () => [] },
  eventId: { type: String, default: '' },
  moduleId: { type: String, default: '' }
})

const emit = defineEmits(['update-question', 'remove-question', 'add-question', 'move-question'])

const showTypeSelector = ref(false)
const editingQuestionIndex = ref(null)

const handleAddClick = () => {
  editingQuestionIndex.value = null
  showTypeSelector.value = true
}

const handleChangeType = (index) => {
  editingQuestionIndex.value = index
  showTypeSelector.value = true
}

const handleTypeSelect = (type) => {
  showTypeSelector.value = false
  if (editingQuestionIndex.value !== null) {
      // Change existing question type
      const question = { ...props.questions[editingQuestionIndex.value], type }
      emit('update-question', editingQuestionIndex.value, question)
  } else {
      // Add new question
      emit('add-question', type)
  }
}
</script>

<style scoped>
.quiz-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #8E8E93;
}
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-text { font-size: 14px; }

.questions-list {
  display: flex;
  flex-direction: column;
}

.question-wrapper-outer {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
}

.question-sort-controls {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  padding-top: 10px;
  gap: 8px;
}

.move-btn {
  background-color: #3A3A3C;
  color: white;
  border: none;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.move-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.move-btn:hover:not(:disabled) { background-color: #48484A; }

.question-component-wrapper {
  flex: 1;
}

.add-section {
  text-align: center;
  margin-top: 20px;
}

.add-btn {
  background-color: rgba(10, 132, 255, 0.1);
  color: #0A84FF;
  padding: 12px 24px;
  border-radius: 8px;
  font-weight: 600;
  border: none;
  cursor: pointer;
}

.quiz-editor::-webkit-scrollbar {
  display: none;
}
</style>
