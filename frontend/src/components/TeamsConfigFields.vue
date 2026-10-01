<template>
  <div class="teams-config-fields">
    <label>{{ $t('teams_module.source') }}
      <select :value="modelValue.sourceModuleId" :disabled="sourceLocked" @change="update('sourceModuleId', $event.target.value)">
        <option value="">{{ $t('teams_module.new_teams') }}</option>
        <option v-for="module in sources" :key="module.id" :value="module.id">{{ module.name }}</option>
      </select>
    </label>
    <label>{{ $t('teams_module.formation') }}
      <select :value="modelValue.method" @change="update('method', $event.target.value)">
        <option value="random">{{ $t('teams_module.random') }}</option>
        <option value="choice">{{ $t('teams_module.choice') }}</option>
        <option value="manual">{{ $t('teams_module.manual') }}</option>
      </select>
    </label>
    <div v-if="modelValue.method === 'random'" class="config-grid">
      <label>{{ $t('teams_module.team_count') }}<input type="number" min="0" max="200" :value="modelValue.teamCount" @input="update('teamCount', Number($event.target.value))" /></label>
      <label>{{ $t('teams_module.target_size') }}<input type="number" min="0" :value="modelValue.targetSize" @input="update('targetSize', Number($event.target.value))" /></label>
    </div>
    <p v-if="modelValue.method === 'random'" class="hint">{{ $t('teams_module.random_help') }}</p>
    <label>{{ $t('teams_module.capacity') }}<input type="number" min="0" :value="modelValue.capacity" @input="update('capacity', Number($event.target.value))" /></label>
    <label>{{ $t('teams_module.presets') }}<input :value="modelValue.presets.join(', ')" @change="update('presets', $event.target.value.split(',').map(value => Number(value.trim())))" /></label>
    <p class="hint">{{ $t('teams_module.full_points_help') }}</p>
  </div>
</template>
<script setup>
const props = defineProps({ modelValue: { type: Object, required: true }, sources: { type: Array, default: () => [] }, sourceLocked: Boolean })
const emit = defineEmits(['update:modelValue'])
function update(key, value) {
  const next = { ...props.modelValue, [key]: value }
  if (key === 'teamCount' && value > 0) next.targetSize = 0
  if (key === 'targetSize' && value > 0) next.teamCount = 0
  emit('update:modelValue', next)
}
</script>
<style scoped>
.teams-config-fields{display:grid;gap:14px}label{display:grid;gap:6px;font-weight:600}select,input{width:100%;box-sizing:border-box;background:#101725;border:1px solid #43516a;border-radius:8px;padding:10px;color:#fff;font:inherit}.config-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}.hint{font-size:13px;line-height:1.5;color:#aab9ce;margin:0}
</style>
