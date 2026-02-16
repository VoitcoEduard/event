<template>
  <div class="data-view-container">
    <div class="page-header">
      <div class="header-left">
        <h1>Event Data Tables: {{ moduleId }}</h1>
      </div>
      <div class="header-right">
        <button class="export-btn">Export Current Tab</button>
        <button class="export-btn">Export Selected Modules</button>
        <button class="refresh-btn">Refresh</button>
        <span class="record-count">{{ responses.length }} records</span>
      </div>
    </div>

    <!-- Horizontal Tabs Navigation -->
    <div class="horizontal-tabs">
      <button 
        v-for="tab in tabs" 
        :key="tab.id" 
        class="tab-btn"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tab Content Area -->
    <div class="tab-content">
      <!-- Loading State -->
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>Loading data...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <p>Error: {{ error }}</p>
        <button class="retry-btn" @click="fetchData">Retry</button>
      </div>

      <!-- No Data State -->
      <div v-else-if="responses.length === 0" class="no-data-state">
        <div class="empty-icon">📊</div>
        <p>No data available for this module.</p>
        <p class="hint">Submit some responses to see data here.</p>
      </div>

      <!-- Data Table View -->
      <div v-else-if="activeTab === 'table'" class="data-table-view">
        <div class="handsontable-container">
          <HotTable :settings="hotSettings" />
        </div>
        <div class="pagination">
          <div class="pagination-left">
            <label>Page Size:</label>
            <select class="page-size-select">
              <option>50</option>
              <option>100</option>
              <option>200</option>
              <option>500</option>
            </select>
          </div>
          <div class="pagination-right">
            <span class="page-info">{{ responses.length > 0 ? '1 to 50 of ' + responses.length : '0' }}</span>
            <button class="page-btn">«</button>
            <button class="page-btn">‹</button>
            <span class="page-number">Page 1 of 28</span>
            <button class="page-btn">›</button>
            <button class="page-btn">»</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.min.css';
import 'handsontable/styles/ht-theme-main.min.css';
import { API_BASE_URL } from '../utils';

// Register all Handsontable modules
registerAllModules();

// Props - receive eventId and moduleId from parent component
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
const activeTab = ref('table');

// Tab definitions
const tabs = [
  { id: 'table', label: 'Unified View' },
  { id: 'quiz', label: 'Quiz: Quiz Session' },
  { id: 'form1', label: 'Form: Welcome' },
  { id: 'form2', label: 'Form: Нужна информация' },
  { id: 'form3', label: 'Form: послепрод' },
  { id: 'teams', label: 'Teams: Teams' },
  { id: 'lottery', label: 'Lottery: Lottery' },
  { id: 'task', label: 'Task: Task' }
];

const hotSettings = reactive({
  data: [],
  colHeaders: true,
  rowHeaders: true,
  readOnly: true,
  licenseKey: 'non-commercial-and-evaluation',
  themeName: 'ht-theme-main',
  height: 'auto',
  stretchH: 'all',
  minRows: 10,
  rowHeights: 35,
  columnHeaderHeight: 40,
  autoWrapRow: true,
  manualColumnResize: true,
  manualRowResize: true,
});

const fetchData = async () => {
  if (!props.eventId || !props.moduleId) {
    error.value = 'Missing event or module ID';
    return;
  }

  loading.value = true;
  error.value = null;
  
  try {
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.moduleId}/responses`);
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
      } else if (key !== '_id' && key !== 'eventId' && key !== 'moduleId' && key !== '__v') {
        allKeys.add(key);
      }
    }
  });

  const headers = ['_id', 'userId', 'createdAt', ...Array.from(allKeys)].filter(Boolean);
  const tableData = [headers];

  rawData.forEach(item => {
    const row = [];
    headers.forEach(header => {
      if (header === 'createdAt' && item[header]) {
        row.push(new Date(item[header]).toLocaleString());
      } else if (item.data && item.data[header] !== undefined) {
        row.push(item.data[header]);
      } else {
        row.push(item[header] !== undefined ? item[header] : '');
      }
    });
    tableData.push(row);
  });

  hotSettings.data = tableData;
  hotSettings.colHeaders = headers;
};

onMounted(fetchData);

// Watch for changes in props and refetch data
watch(() => [props.eventId, props.moduleId], () => {
  fetchData();
});
</script>

<style scoped>
.data-view-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: #f5f5f5;
  color: #333;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background-color: white;
  border-bottom: 1px solid #ddd;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.header-left h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #222;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.export-btn {
  background-color: #007AFF;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.export-btn:hover {
  background-color: #0056b3;
}

.refresh-btn {
  background-color: #4CAF50;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.refresh-btn:hover {
  background-color: #45a049;
}

.record-count {
  font-size: 14px;
  color: #666;
}

/* Horizontal Tabs Styles */
.horizontal-tabs {
  display: flex;
  background-color: white;
  border-bottom: 1px solid #ddd;
  overflow-x: auto;
}

.tab-btn {
  padding: 12px 24px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: #666;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
  white-space: nowrap;
}

.tab-btn:hover {
  background-color: #f8f8f8;
  color: #333;
}

.tab-btn.active {
  color: #007AFF;
  border-bottom-color: #007AFF;
}

/* Tab Content Styles */
.tab-content {
  flex: 1;
  padding: 24px;
  background-color: #f5f5f5;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #888;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #ddd;
  border-top-color: #007AFF;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error State */
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #ff6b6b;
}

.error-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.retry-btn {
  margin-top: 16px;
  background-color: #ff6b6b;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.retry-btn:hover {
  background-color: #ff5252;
}

/* No Data State */
.no-data-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #888;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.hint {
  font-size: 14px;
  color: #666;
  margin-top: 8px;
}

/* Data Table View */
.data-table-view {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.handsontable-container {
  flex: 1;
  min-height: 400px;
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

/* Pagination */
.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background-color: white;
  border-top: 1px solid #eee;
  border-radius: 0 0 8px 8px;
  margin-top: -1px;
}

.pagination-left,
.pagination-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-size-select {
  padding: 4px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.page-btn {
  background-color: white;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 14px;
}

.page-btn:hover {
  background-color: #f5f5f5;
}

.page-number {
  font-size: 14px;
  color: #333;
}

.page-info {
  font-size: 14px;
  color: #666;
}
</style>
