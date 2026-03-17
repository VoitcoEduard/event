<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container">
        <div class="header">
          <div>
            <p class="title">{{ $t('admin.details.teams_config.title') }}</p>
            <p class="subtitle">{{ module?.name || $t('form_module.config.untitled') }}</p>
          </div>
          <div class="header-buttons">
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content">
          <!-- Split Method -->
          <div class="config-section">
            <label class="config-label">{{ $t('admin.details.teams_config.split_method') }}</label>
            <select v-model="config.splitMethod" class="config-select">
              <option value="random">{{ $t('admin.details.teams_config.random') }}</option>
              <option value="self_assign">{{ $t('admin.details.teams_config.self_assign') }}</option>
              <option value="manual">{{ $t('admin.details.teams_config.manual') }}</option>
            </select>
          </div>

          <!-- Team Names -->
          <div class="config-section">
            <label class="config-label">{{ $t('admin.details.teams_config.team_names') }}</label>
            <div class="team-names-list">
              <div v-for="(name, index) in teamNamesArray" :key="index" class="team-name-row">
                <input 
                  type="text" 
                  v-model="teamNamesArray[index]" 
                  :placeholder="$t('admin.details.teams_config.team_name_placeholder', { n: index + 1 })"
                  class="config-input"
                />
                <button @click="removeTeamName(index)" class="remove-btn" v-if="teamNamesArray.length > 2">×</button>
              </div>
              <button @click="addTeamName" class="add-btn">+ {{ $t('admin.details.teams_config.add_team') }}</button>
            </div>
          </div>

          <!-- Team Colors -->
          <div class="config-section">
            <label class="config-label">{{ $t('admin.details.teams_config.team_colors') }}</label>
            <div class="team-colors-list">
              <div v-for="(color, index) in teamColorsArray" :key="index" class="team-color-row">
                <input 
                  type="color" 
                  v-model="teamColorsArray[index]" 
                  class="color-picker"
                />
                <span class="color-label">{{ teamNamesArray[index] || 'Team ' + (index + 1) }}</span>
              </div>
            </div>
          </div>

          <!-- Self Assign Option -->
          <div v-if="config.splitMethod === 'self_assign'" class="config-section checkbox-section">
            <label class="checkbox-label">
              <input type="checkbox" v-model="config.allowSelfAssign" />
              {{ $t('admin.details.teams_config.allow_self_assign') }}
            </label>
          </div>

          <!-- Persist After Module -->
          <div class="config-section checkbox-section">
            <label class="checkbox-label">
              <input type="checkbox" v-model="config.persistAfterModule" />
              {{ $t('admin.details.teams_config.persist_after_module') }}
            </label>
          </div>
        </div>

        <div class="action-bar">
          <button @click="saveConfig" class="save-button">
            {{ $t('admin.details.module_save') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders } from '../utils'

const { t } = useI18n()

const props = defineProps({
  visible: { type: Boolean, default: false },
  module: {
    type: Object,
    required: true
  },
  eventId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['close', 'update'])

const config = ref({
  splitMethod: 'random',
  teamNames: [],
  teamColors: [],
  persistAfterModule: true,
  allowSelfAssign: true
})

// Initialize from module data
if (props.module.data) {
  config.value = {
    splitMethod: props.module.data.splitMethod || 'random',
    teamNames: props.module.data.teamNames || [],
    teamColors: props.module.data.teamColors || [],
    persistAfterModule: props.module.data.persistAfterModule !== false,
    allowSelfAssign: props.module.data.allowSelfAssign !== false
  }
}

const teamNamesArray = ref([...config.value.teamNames])
const teamColorsArray = ref([...config.value.teamColors])

// Ensure we have enough team names and colors
const updateArrays = () => {
  const count = teamNamesArray.value.length
  
  if (count === 0) {
    // Add 2 default teams if none configured
    teamNamesArray.value.push('Team 1')
    teamNamesArray.value.push('Team 2')
    teamColorsArray.value.push('#FF5733')
    teamColorsArray.value.push('#33FF57')
  }
  
  const defaultColors = ['#FF5733', '#33FF57', '#3357FF', '#FF33F5', '#F5FF33', '#33FFF5']
  while (teamColorsArray.value.length < teamNamesArray.value.length) {
    teamColorsArray.value.push(defaultColors[teamColorsArray.value.length % defaultColors.length])
  }
}

updateArrays()

const addTeamName = () => {
  teamNamesArray.value.push(`Team ${teamNamesArray.value.length + 1}`)
  const defaultColors = ['#FF5733', '#33FF57', '#3357FF', '#FF33F5', '#F5FF33', '#33FFF5']
  teamColorsArray.value.push(defaultColors[teamColorsArray.value.length % defaultColors.length])
}

const removeTeamName = (index) => {
  teamNamesArray.value.splice(index, 1)
  teamColorsArray.value.splice(index, 1)
}

const handleClose = () => {
  emit('close')
}

const saveConfig = async () => {
  const updatedData = {
    ...props.module.data,
    splitMethod: config.value.splitMethod,
    teamNames: teamNamesArray.value,
    teamColors: teamColorsArray.value,
    persistAfterModule: config.value.persistAfterModule,
    allowSelfAssign: config.value.allowSelfAssign
  }

  try {
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({
        name: props.module.name,
        type: 'teams',
        data: updatedData
      })
    })

    if (response.ok) {
      emit('update')
      handleClose()
    }
  } catch (error) {
    console.error('Error saving teams config:', error)
  }
}

watch(() => teamNamesArray.value.length, () => {
  updateArrays()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-container {
  background-color: #1C1C1E;
  border-radius: 12px;
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #2C2C2E;
}

.title {
  font-size: 22px;
  font-weight: bold;
  color: #FFF;
  margin: 0;
}

.subtitle {
  font-size: 14px;
  color: #8E8E93;
  margin: 4px 0 0;
}

.header-buttons {
  display: flex;
  gap: 8px;
}

.close-button {
  width: 36px;
  height: 36px;
  border: none;
  background-color: #3A3A3C;
  border-radius: 8px;
  cursor: pointer;
  font-size: 16px;
  color: #FFF;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-button:hover {
  background-color: #444;
}

.content {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.config-section {
  margin-bottom: 20px;
}

.config-label {
  display: block;
  font-weight: 600;
  margin-bottom: 8px;
  color: #FFF;
}

.config-select,
.config-input {
  width: 100%;
  padding: 12px 16px;
  background-color: #2C2C2E;
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  font-size: 16px;
  color: #FFF;
}

.config-select:focus,
.config-input:focus {
  outline: none;
  border-color: #0A84FF;
}

.config-input::placeholder {
  color: #666;
}

.config-input[type="number"] {
  max-width: 100px;
}

.team-names-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.team-name-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.team-name-row .config-input {
  flex: 1;
}

.remove-btn {
  width: 36px;
  height: 36px;
  border: none;
  background-color: #ff4444;
  color: white;
  border-radius: 8px;
  cursor: pointer;
  font-size: 18px;
  font-weight: bold;
}

.remove-btn:hover {
  background-color: #ff6666;
}

.add-btn {
  padding: 10px 16px;
  border: 1px dashed #3A3A3C;
  background: none;
  cursor: pointer;
  border-radius: 8px;
  color: #8E8E93;
  font-size: 14px;
}

.add-btn:hover {
  border-color: #666;
  color: #FFF;
}

.team-colors-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.team-color-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.color-picker {
  width: 40px;
  height: 40px;
  border: none;
  cursor: pointer;
  border-radius: 8px;
  background: none;
}

.color-label {
  color: #aaa;
  font-size: 14px;
}

.checkbox-section {
  margin-top: 16px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  color: #FFF;
  font-size: 14px;
}

.checkbox-label input {
  width: 18px;
  height: 18px;
  accent-color: #0A84FF;
}

.action-bar {
  padding: 16px 20px;
  border-top: 1px solid #2C2C2E;
  display: flex;
  justify-content: flex-end;
  background-color: #1C1C1E;
}

.save-button {
  padding: 12px 24px;
  background-color: #0A84FF;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
}

.save-button:hover {
  background-color: #409cff;
}
</style>
