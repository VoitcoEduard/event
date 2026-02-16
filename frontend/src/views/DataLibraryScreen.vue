<template>
  <div class="data-library-screen">
    <div class="page-header">
      <div class="header-left">
        <button class="back-link" @click="goBack">
          <span>← Back to event</span>
        </button>
        <h1>Event Data Tables: {{ selectedEvent?.name }}</h1>
      </div>
    </div>

    <!-- Horizontal Tabs Navigation -->
    <div class="horizontal-tabs">
      <button 
        class="tab-btn"
        :class="{ active: activeTab === 'unified' }"
        @click="activeTab = 'unified'"
      >
        Unified View
      </button>
      <button 
        v-for="module in selectedEvent?.modules || []" 
        :key="module.id" 
        class="tab-btn"
        :class="{ active: activeTab === module.id }"
        @click="activeTab = module.id"
      >
        {{ module.type }}: {{ module.name }}
      </button>
    </div>

    <!-- Tab Content Area -->
    <div class="tab-content">
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>Loading data...</p>
      </div>

      <div v-else-if="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <p>Error: {{ error }}</p>
        <button class="retry-btn" @click="fetchData">Retry</button>
      </div>

      <div v-else-if="!selectedEvent" class="no-event-state">
        <p>Please select an event to view data tables</p>
      </div>

      <div v-else-if="activeTab === 'unified'" class="unified-view">
        <div class="data-header">
          <div class="header-actions">
            <button class="export-btn">Export Current Tab</button>
            <button class="export-btn">Export Selected Modules</button>
            <button class="refresh-btn">Refresh</button>
            <span class="record-count">{{ totalRecords }} records</span>
          </div>
        </div>
        
        <div v-if="totalRecords === 0" class="no-data-state">
          <div class="empty-icon">📊</div>
          <p>No data generated yet</p>
          <p class="hint">Waiting for participants to submit responses...</p>
        </div>

        <div v-else class="data-table-container">
          <div class="handsontable-wrapper">
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
              <span class="page-info">1 to 50 of {{ totalRecords }}</span>
              <button class="page-btn">«</button>
              <button class="page-btn">‹</button>
              <span class="page-number">Page 1 of {{ totalPages }}</span>
              <button class="page-btn">›</button>
              <button class="page-btn">»</button>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="module-view">
        <div class="data-header">
          <div class="header-actions">
            <button class="export-btn">Export Current Tab</button>
            <button class="export-btn">Export Selected Modules</button>
            <button class="refresh-btn">Refresh</button>
            <span class="record-count">{{ moduleResponses.length }} records</span>
          </div>
        </div>
        
        <div v-if="moduleResponses.length === 0" class="no-data-state">
          <div class="empty-icon">Folder Empty</div>
          <p>No data generated for this module</p>
          <p class="hint">Waiting for responses...</p>
        </div>

        <div v-else class="data-table-container">
          <div class="handsontable-wrapper">
            <HotTable :settings="getModuleTableSettings()" />
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
              <span class="page-info">1 to 50 of {{ moduleResponses.length }}</span>
              <button class="page-btn">«</button>
              <button class="page-btn">‹</button>
              <span class="page-number">Page 1 of {{ modulePages }}</span>
              <button class="page-btn">›</button>
              <button class="page-btn">»</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.min.css';
import 'handsontable/styles/ht-theme-main.min.css';
import { API_BASE_URL } from '../utils';

// Register all Handsontable modules
registerAllModules();

const router = useRouter();
const route = useRoute();

const events = ref([]);
const selectedEvent = ref(null);
const activeTab = ref('unified');
const loading = ref(true);
const error = ref(null);
const moduleResponses = ref([]);
const totalRecords = ref(0);
const totalPages = ref(0);
const modulePages = ref(0);

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

const fetchEvents = async () => {
  loading.value = true;
  error.value = null;
  
  try {
    const response = await fetch(`${API_BASE_URL}/events`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    events.value = await response.json();
    
    // Pre-select event based on route or first available
    if (route.query.eventId) {
      selectedEvent.value = events.value.find(e => e.id === route.query.eventId);
    } else if (events.value.length > 0) {
      selectedEvent.value = events.value[0];
    }
  } catch (e) {
    error.value = e.message;
    console.error("Failed to fetch events:", e);
  } finally {
    loading.value = false;
  }
};

const fetchData = async () => {
  if (!selectedEvent.value) return;
  
  loading.value = true;
  error.value = null;
  
  try {
    if (activeTab.value === 'unified') {
      // Fetch all module responses for the selected event
      const allResponses = [];
      for (const module of selectedEvent.value.modules) {
        const response = await fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${module.id}/responses`);
        if (response.ok) {
          const data = await response.json();
          if (data.responses && Array.isArray(data.responses)) {
            allResponses.push(...data.responses);
          }
        }
      }
      moduleResponses.value = allResponses;
      totalRecords.value = allResponses.length;
      totalPages.value = Math.ceil(allResponses.length / 50);
      transformDataForHandsontable(allResponses);
    } else {
      // Fetch responses for specific module
      const response = await fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${activeTab.value}/responses`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      const responses = data.responses || [];
      moduleResponses.value = responses;
      modulePages.value = Math.ceil(responses.length / 50);
    }
  } catch (e) {
    error.value = e.message;
    console.error("Failed to fetch responses:", e);
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
      let value;
      if (header === 'createdAt' && item[header]) {
        value = new Date(item[header]).toLocaleString();
      } else if (item.data && item.data[header] !== undefined) {
        value = item.data[header];
      } else {
        value = item[header] !== undefined ? item[header] : '';
      }
      
      row.push(formatCellValue(header, value));
    });
    tableData.push(row);
  });

  hotSettings.data = tableData;
  hotSettings.colHeaders = headers;
};

const formatCellValue = (header, value) => {
  if (value === null || value === undefined) return '';
  
  // Specific formatting for known complex columns
  if (header === 'players' && Array.isArray(value)) {
    return value.map(p => `${p.nickname || 'Unknown'} (${p.score || 0} pts)`).join(', ');
  }
  
  // Generic object handling
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value);
    } catch (e) {
      return String(value);
    }
  }
  
  return value;
};

const getModuleTableSettings = () => {
  const module = selectedEvent.value?.modules.find(m => m.id === activeTab.value);
  if (!module) return {};
  
  // Simple implementation - reuse hotSettings for now
  return hotSettings;
};

const goBack = () => {
  if (selectedEvent.value?.id) {
    router.push(`/admin/event/${selectedEvent.value.id}`);
  } else {
    router.push('/admin');
  }
};

onMounted(fetchEvents);

// Watch for changes in selected event and active tab
watch([selectedEvent, activeTab], () => {
  if (selectedEvent.value) {
    fetchData();
  }
});
</script>

<style scoped>
.data-library-screen {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: #121212;
  color: #ffffff;
}

.page-header {
  background-color: #1E1E1E;
  border-bottom: 1px solid #333;
  box-shadow: 0 2px 4px rgba(0,0,0,0.3);
  padding: 20px 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-link {
  background: none;
  border: none;
  color: #007AFF;
  font-size: 14px;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.back-link:hover {
  text-decoration: underline;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: #ffffff;
}

/* Horizontal Tabs Styles */
.horizontal-tabs {
  display: flex;
  background-color: #1E1E1E;
  border-bottom: 1px solid #333;
  overflow-x: auto;
  flex-wrap: wrap;
}

.tab-btn {
  padding: 12px 24px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: #888;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
  white-space: nowrap;
}

.tab-btn:hover {
  background-color: #252525;
  color: #fff;
}

.tab-btn.active {
  color: #007AFF;
  border-bottom-color: #007AFF;
}

/* Tab Content Styles */
.tab-content {
  flex: 1;
  padding: 24px;
  background-color: #121212;
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
  border: 3px solid #333;
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
  color: #FF6B6B;
}

.error-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.retry-btn {
  margin-top: 16px;
  background-color: #FF6B6B;
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

/* No Event State */
.no-event-state {
  text-align: center;
  padding: 60px;
  color: #888;
}

/* No Data State */
.no-data-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 400px;
  color: #888;
  background-color: #1E1E1E; /* Match table container bg */
  border-radius: 8px;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.hint {
  font-size: 14px;
  color: #666;
  margin-top: 8px;
}

/* Data Header */
.data-header {
  margin-bottom: 20px;
}

.header-actions {
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
  background-color: #30D158;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.refresh-btn:hover {
  background-color: #28a745;
}

.record-count {
  font-size: 14px;
  color: #888;
}

/* Data Table Container */
.data-table-container {
  background-color: #1E1E1E;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  overflow: hidden;
}

.handsontable-wrapper {
  min-height: 400px;
}

/* Pagination */
.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background-color: #1E1E1E;
  border-top: 1px solid #333;
}

.pagination-left,
.pagination-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-size-select {
  padding: 4px 8px;
  border: 1px solid #333;
  border-radius: 4px;
  font-size: 14px;
  background-color: #252525;
  color: #fff;
}

.page-btn {
  background-color: #252525;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 14px;
  color: #fff;
}

.page-btn:hover {
  background-color: #333;
}

.page-number {
  font-size: 14px;
  color: #fff;
}

.page-info {
  font-size: 14px;
  color: #888;
}

/* Handsontable Dark Mode Overrides */
:deep(.handsontable) {
  color: #ffffff;
}

:deep(.handsontable table) {
  color: #ffffff;
  background-color: #1E1E1E;
}

:deep(.handsontable th),
:deep(.handsontable td) {
  background-color: #1E1E1E;
  color: #ffffff !important;
  border-bottom: 1px solid #333;
  border-right: 1px solid #333;
}

/* Row headers (index column) */
:deep(.handsontable th) {
  background-color: #252525;
  color: #cccccc;
  font-weight: 600;
  border-bottom: 1px solid #333;
  border-right: 1px solid #333;
}

/* Specific overrides for row headers in fixed columns */
:deep(.ht_clone_left th),
:deep(.ht_clone_top_left_corner th),
:deep(.ht_clone_bottom_left_corner th) {
  background-color: #252525 !important;
  color: #cccccc !important;
}

/* Ensure ALL th elements get the dark treatment */
:deep(.handsontable tbody th),
:deep(.handsontable thead th) {
  background-color: #252525 !important;
  color: #cccccc !important;
  border-color: #333 !important;
}

:deep(.handsontable tr:first-child th),
:deep(.handsontable tr:first-child td) {
  border-top: 1px solid #333;
}

:deep(.handsontable th:first-child),
:deep(.handsontable td:first-child) {
  border-left: 1px solid #333;
}

:deep(.handsontable .wtBorder) {
  background-color: #007AFF !important; /* Selection border color */
}

:deep(.handsontable .area) {
  background-color: rgba(0, 122, 255, 0.1) !important; /* Selection area background */
}

/* Fix for white background in empty areas */
:deep(.wtHider),
:deep(.wtSpreader),
:deep(.wtHolder) {
  background-color: #1E1E1E !important;
}

/* Background for the corner */
:deep(.ht_master .wtHolder),
:deep(.ht_clone_left .wtHolder),
:deep(.ht_clone_top .wtHolder) {
   background-color: #1E1E1E !important;
}

/* Ensure input editors are dark */
:deep(.handsontableInput) {
  background-color: #1E1E1E !important;
  color: #ffffff !important;
  box-shadow: none !important; 
}
</style>
