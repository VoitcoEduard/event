<template>
  <div class="data-view-container">
    <h2>Data for Module: {{ moduleId }}</h2>
    <div v-if="loading" class="loading-indicator">Loading data...</div>
    <div v-else-if="error" class="error-message">Error: {{ error }}</div>
    <div v-else-if="responses.length === 0" class="no-data-message">No data available for this module.</div>
    <div v-else class="handsontable-container">
      <HotTable :settings="hotSettings" />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/dist/handsontable.full.css';

// Register all Handsontable modules
registerAllModules();

const props = defineProps({
  eventId: {
    type: String,
    required: true
  },
  moduleId: {
    type: String,
    required: true
  }
});

const responses = ref([]);
const loading = ref(false);
const error = ref(null);

const hotSettings = reactive({
  data: [],
  colHeaders: true,
  rowHeaders: true,
  readOnly: true,
  licenseKey: 'non-commercial-and-evaluation', // Handsontable Community Edition license key
  // Further customization can be added here
});

const fetchData = async () => {
  loading.value = true;
  error.value = null;
  try {
    const response = await fetch(`/events/${props.eventId}/modules/${props.moduleId}/responses`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const result = await response.json();
    responses.value = result.responses || [];
    transformDataForHandsontable(responses.value);
  } catch (e) {
    error.value = e.message;
    console.error("Failed to fetch module responses:", e);
  } finally {
    loading.value = false;
  }
};

const transformDataForHandsontable = (rawData) => {
  if (rawData.length === 0) {
    hotSettings.data = [];
    return;
  }

  // Collect all unique keys from 'data' objects and other top-level fields
  const allKeys = new Set();
  rawData.forEach(item => {
    for (const key in item) {
      if (key === 'data' && typeof item[key] === 'object' && item[key] !== null) {
        for (const dataKey in item[key]) {
          allKeys.add(dataKey);
        }
      } else if (key !== '_id' && key !== 'eventId' && key !== 'moduleId' && key !== '__v') { // Exclude internal fields
        allKeys.add(key);
      }
    }
  });

  const headers = ['_id', 'userId', 'createdAt', ...Array.from(allKeys)].filter(Boolean); // Ensure _id, userId, createdAt are first

  const tableData = [headers]; // First row is headers

  rawData.forEach(item => {
    const row = [];
    headers.forEach(header => {
      if (header === 'createdAt' && item[header]) {
        row.push(new Date(item[header]).toLocaleString()); // Format date for readability
      } else if (header === 'data' && item[header]) { // This case might not be hit if we flatten data keys
        row.push(JSON.stringify(item[header])); // Fallback for 'data' if not flattened
      } else if (item.data && item.data[header] !== undefined) {
        row.push(item.data[header]); // Extract from nested 'data'
      } else {
        row.push(item[header] !== undefined ? item[header] : ''); // Top-level or empty
      }
    });
    tableData.push(row);
  });

  hotSettings.data = tableData;
  hotSettings.colHeaders = headers; // Update colHeaders for Handsontable
};


onMounted(fetchData);

watch([() => props.eventId, () => props.moduleId], fetchData);
</script>

<style scoped>
.data-view-container {
  padding: 20px;
  background-color: #f9f9f9;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

h2 {
  color: #333;
  margin-bottom: 15px;
}

.loading-indicator, .error-message, .no-data-message {
  padding: 15px;
  border-radius: 4px;
  margin-top: 15px;
}

.loading-indicator {
  background-color: #e7f3ff;
  color: #0056b3;
}

.error-message {
  background-color: #ffe7e7;
  color: #cc0000;
}

.no-data-message {
  background-color: #ffffed;
  color: #666;
}

.handsontable-container {
  margin-top: 20px;
  /* Add specific styling for Handsontable if needed */
}
</style>
