<template>
  <div class="data-library-screen">
    <div class="page-header">
      <div class="header-left">
        <div class="language-switcher">
          <select v-model="$i18n.locale" @change="changeLanguage">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="ro">Română</option>
          </select>
        </div>
        <button class="back-link" @click="goBack">
          <span>{{ $t('data_library.back_to_event') }}</span>
        </button>
        <h1>{{ $t('data_library.title', { name: selectedEvent?.name }) }}</h1>
      </div>
    </div>

    <!-- Horizontal Tabs Navigation -->
    <div class="horizontal-tabs">
      <button 
        class="tab-btn"
        :class="{ active: activeTab === 'unified' }"
        @click="activeTab = 'unified'"
      >
        {{ $t('data_library.unified_view') }}
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
        <p>{{ $t('data_library.loading') }}</p>
      </div>

      <div v-else-if="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <p>{{ $t('data_library.error', { error: error }) }}</p>
        <button class="retry-btn" @click="fetchData">{{ $t('data_library.retry') }}</button>
      </div>

      <div v-else-if="!selectedEvent" class="no-event-state">
        <p>{{ $t('data_library.no_event') }}</p>
      </div>

      <div v-else-if="activeTab === 'unified'" class="unified-view">
        <div class="data-header">
          <div class="header-actions">
            <button class="export-btn">{{ $t('data_library.export_btn') }}</button>
            <button class="refresh-btn" @click="fetchData">{{ $t('data_library.refresh_btn') }}</button>
            <span class="record-count">{{ $t('data_library.users_count', { count: totalRecords }) }}</span>
          </div>
        </div>
        
        <div v-if="totalRecords === 0" class="no-data-state">
          <div class="empty-icon">📊</div>
          <p>{{ $t('data_library.no_data') }}</p>
          <p class="hint">{{ $t('data_library.hint_waiting') }}</p>
        </div>

        <div v-else class="data-table-container">
          <div class="handsontable-wrapper">
            <HotTable ref="hotTableComponent" :settings="hotSettings" />
          </div>
        </div>
      </div>

      <div v-else class="module-view">
        <div class="data-header">
          <div class="header-actions">
            <button class="export-btn">{{ $t('data_library.export_btn') }}</button>
            <button class="refresh-btn" @click="fetchData">{{ $t('data_library.refresh_btn') }}</button>
            <span class="record-count">{{ $t('data_library.records_count', { count: moduleResponses.length }) }}</span>
          </div>
        </div>
        
        <div v-if="moduleResponses.length === 0" class="no-data-state">
          <div class="empty-icon">Folder Empty</div>
          <p>{{ $t('data_library.no_data_module') }}</p>
          <p class="hint">{{ $t('data_library.hint_responses') }}</p>
        </div>

        <div v-else class="data-table-container">
          <div class="handsontable-wrapper">
            <HotTable ref="hotTableComponent" :settings="hotSettings" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.min.css';
import 'handsontable/styles/ht-theme-main.min.css';
import { API_BASE_URL, getAuthHeaders } from '../utils';

const { t, locale } = useI18n();
// Register all Handsontable modules
registerAllModules();

const router = useRouter();
const route = useRoute();

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

const events = ref([]);
const selectedEvent = ref(null);
const activeTab = ref('unified');
const loading = ref(true);
const error = ref(null);
const moduleResponses = ref([]);
const totalRecords = ref(0);
const hotTableComponent = ref(null);

// -----------------------------------------------------------------------------
// Helper Functions (Editing Logic)
// -----------------------------------------------------------------------------

const updateResponse = async (moduleId, responseId, data) => {
  if (!responseId) {
    console.warn("Cannot update response: No Response ID found");
    return;
  }
  try {
    console.log(`[UnifiedView] Updating response ${responseId} in module ${moduleId}`, data);
    const response = await fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${moduleId}/responses/${responseId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ data })
    });
    if (!response.ok) throw new Error('Failed to update response');
    console.log("[UnifiedView] Response updated successfully");
  } catch (e) {
    console.error("[UnifiedView] Error updating response:", e);
  }
};

const updateScore = async (moduleId, userId, score) => {
  try {
    console.log(`[UnifiedView] Updating score for user ${userId} in module ${moduleId}: ${score}`);
    const response = await fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${moduleId}/quiz-scores/${userId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ score })
    });
    if (!response.ok) throw new Error('Failed to update score');
    console.log("[UnifiedView] Score updated successfully");
  } catch (e) {
    console.error("[UnifiedView] Error updating score:", e);
  }
};

const handleTableChange = (changes, source) => {
  if (!changes || source === 'loadData') return;
  console.log("[UnifiedView] Table changed:", source, changes);

  changes.forEach(([row, prop, oldValue, newValue]) => {
    if (oldValue === newValue) return;

    // Access raw data. Because we bind hotSettings.data to a reactive ref or array, we can check it.
    // However, Handsontable manages its visual index.
    // Ideally we rely on the row index mapping to our source data.
    // Assuming no sorting/filtering for now.
    
    // We need to access the source data item corresponding to this row.
    const rowData = hotSettings.data[row]; 
    if (!rowData) {
        console.warn("[UnifiedView] Row data not found for row", row);
        return;
    }

    const userId = rowData.userId;
    console.log(`[UnifiedView] Change detected for user ${userId}, prop ${prop}: ${oldValue} -> ${newValue}`);

    if (activeTab.value === 'unified') {
        if (prop === 'name') {
            if (rowData.nameSourceModuleId && rowData.nameSourceKey) {
                const moduleId = rowData.nameSourceModuleId;
                const responseId = rowData.responseIds[moduleId];
                const key = rowData.nameSourceKey;
                
                const currentModuleData = rowData.formData[moduleId] || {};
                const updatedModuleData = { ...currentModuleData, [key]: newValue };
                
                updateResponse(moduleId, responseId, updatedModuleData);
            }
        } else if (prop.startsWith('scores.')) {
            const moduleId = prop.split('.')[1];
            updateScore(moduleId, userId, Number(newValue));
        } else if (prop.startsWith('formData.')) {
            const parts = prop.split('.');
            const moduleId = parts[1];
            const key = parts.slice(2).join('.');
            
            const responseId = rowData.responseIds[moduleId];
            const currentModuleData = rowData.formData[moduleId] || {};
            const updatedModuleData = { ...currentModuleData, [key]: newValue };
            
            updateResponse(moduleId, responseId, updatedModuleData);
        }
    } else {
        // Single Module Editing (TODO: Implement logic for specific single module updates if needed)
        // For now, Unified View is the priority for editing.
        console.log("[UnifiedView] Editing in Single Module View not fully implemented yet.");
    }
  });
};

const hotSettings = reactive({
  data: [],
  colHeaders: [],
  columns: [],
  rowHeaders: true,
  readOnly: false, 
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
  afterChange: handleTableChange
});

// -----------------------------------------------------------------------------
// Data Fetching
// -----------------------------------------------------------------------------

const fetchEvents = async () => {
  loading.value = true;
  error.value = null;
  
  try {
    const response = await fetch(`${API_BASE_URL}/events`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    events.value = await response.json();
    
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

const initializeUser = (map, userId) => {
  map.set(userId, {
    userId,
    name: '',
    scores: {},
    formData: {},
    responseIds: {}
  });
};

const fetchData = async () => {
  if (!selectedEvent.value) return;
  
  loading.value = true;
  error.value = null;
  // Clear table first
  if (hotTableComponent.value && hotTableComponent.value.hotInstance) {
      hotTableComponent.value.hotInstance.loadData([]);
  }
  hotSettings.data = [];
  hotSettings.colHeaders = [];
  hotSettings.columns = [];
  
  try {
    if (activeTab.value === 'unified') {
      await fetchUnifiedData();
    } else {
      await fetchModuleData();
    }
  } catch (e) {
    error.value = e.message;
    console.error("Failed to fetch responses:", e);
  } finally {
    loading.value = false;
  }
};

const fetchUnifiedData = async () => {
  console.log("[UnifiedView] Starting Unified Data Fetch");
  const promises = selectedEvent.value.modules.map(m => 
    fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${m.id}/responses`)
      .then(async res => {
        if (!res.ok) throw new Error(`Failed to fetch ${m.name} responses: ${res.statusText}`);
        return res.json();
      })
      .then(data => ({ moduleId: m.id, type: m.type, name: m.name, responses: data.responses || [] }))
  );

  const results = await Promise.all(promises);
  
  const userMap = new Map();

  results.forEach(({ moduleId, type, responses }) => {
    responses.forEach(res => {
      // Determine User ID
      if (type === 'quiz') {
        const quizResult = res;
        if (quizResult.players) {
           quizResult.players.forEach(p => {
             const userId = p.id;
             if (!userMap.has(userId)) initializeUser(userMap, userId);
             const userEntry = userMap.get(userId);
             userEntry.scores[moduleId] = (userEntry.scores[moduleId] || 0) + p.score;
             if (!userEntry.name && p.nickname) userEntry.name = p.nickname;
           });
        }
      } else {
        const userId = res.userId;
        if (!userId) return; // Skip responses without userId
        if (!userMap.has(userId)) initializeUser(userMap, userId);
        const userEntry = userMap.get(userId);

        userEntry.responseIds[moduleId] = res.id || res._id;

        if (!userEntry.formData[moduleId]) userEntry.formData[moduleId] = {};
        Object.assign(userEntry.formData[moduleId], res.data);

        // Name logic
        if (res.data && res.data['step_welcome_name']) {
           userEntry.name = res.data['step_welcome_name'];
           userEntry.nameSourceKey = 'step_welcome_name';
           userEntry.nameSourceModuleId = moduleId;
        }
      }
    });
  });

  // Construct Columns
  const headers = [t('data_library.headers.user_id'), t('data_library.headers.name'), t('data_library.headers.total_score')];
  const columns = [
    { data: 'userId', readOnly: true },
    { data: 'name', readOnly: false },
    { data: 'totalScore', type: 'numeric', readOnly: true }
  ];

  selectedEvent.value.modules.forEach(m => {
    if (m.type === 'quiz') {
      headers.push(`${m.name} (Score)`);
      columns.push({ 
        data: `scores.${m.id}`, 
        type: 'numeric',
        readOnly: false 
      });
    } else {
      const allKeys = new Set();
      userMap.forEach(u => {
        if (u.formData[m.id]) {
          Object.keys(u.formData[m.id]).forEach(k => allKeys.add(k));
        }
      });
      Array.from(allKeys).forEach(key => {
        headers.push(`${m.name}: ${key}`);
        columns.push({ 
          data: `formData.${m.id}.${key}`,
          readOnly: false 
        });
      });
    }
  });

  const flattenedData = Array.from(userMap.values()).map(u => {
    const totalScore = Object.values(u.scores).reduce((a, b) => a + b, 0);
    return { ...u, totalScore };
  });

  console.log("[UnifiedView] Aggregation complete. Records:", flattenedData.length);
  
  hotSettings.colHeaders = headers;
  hotSettings.columns = columns;
  hotSettings.data = flattenedData;
  hotSettings.readOnly = false; // Enable for Unified View

  totalRecords.value = flattenedData.length;
  
  if (hotTableComponent.value && hotTableComponent.value.hotInstance) {
      hotTableComponent.value.hotInstance.updateSettings({
          colHeaders: headers,
          columns: columns,
          data: flattenedData,
          readOnly: false
      });
  }
};

const fetchModuleData = async () => {
    const response = await fetch(`${API_BASE_URL}/events/${selectedEvent.value.id}/modules/${activeTab.value}/responses`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status} ${response.statusText}`);
    }
    const data = await response.json();
    const responses = data.responses || [];
    moduleResponses.value = responses;
    
    // Use legacy transform for single module
    const { headers, columns, data: rows } = transformDataForHandsontable(responses);
    
    hotSettings.colHeaders = headers;
    hotSettings.columns = columns;
    hotSettings.data = rows;
    hotSettings.readOnly = true; 

    if (hotTableComponent.value && hotTableComponent.value.hotInstance) {
      hotTableComponent.value.hotInstance.updateSettings({
          colHeaders: headers,
          columns: columns,
          data: rows,
          readOnly: true
      });
  }
};

// Legacy transform for Single Module View (updated to return settings)
const transformDataForHandsontable = (rawData) => {
  if (rawData.length === 0) {
    return { headers: [], columns: [], data: [] };
  }

  const allKeys = new Set();
  rawData.forEach(item => {
    for (const key in item) {
       if (key === 'data' && typeof item[key] === 'object' && item[key] !== null) {
        for (const dataKey in item[key]) {
          allKeys.add(dataKey);
        }
      } else if (['userId', 'createdAt', 'eventId', 'moduleId', '_id', 'id', '__v'].includes(key) === false) {
         allKeys.add(key); 
      }
    }
  });

  const headers = ['userId', 'createdAt', ...Array.from(allKeys)];
  const columns = headers.map(h => ({ data: h })); 
  
  const tableData = rawData.map(item => {
      const row = { userId: item.userId, createdAt: item.createdAt ? new Date(item.createdAt).toLocaleString() : '' };
      if (item.data) Object.assign(row, item.data);
      Object.keys(item).forEach(k => {
          if (!['userId', 'createdAt', 'data'].includes(k)) {
              if (k === 'players' && Array.isArray(item[k])) {
                  row[k] = item[k].map(p => `${p.nickname} (${p.score})`).join(', ');
              } else {
                  row[k] = item[k];
              }
          }
      });
      return row;
  });

  return { headers, columns, data: tableData };
};

const goBack = () => {
  if (selectedEvent.value?.id) {
    router.push(`/admin/event/${selectedEvent.value.id}`);
  } else {
    router.push('/admin');
  }
};

onMounted(fetchEvents);

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

.language-switcher {
  margin-right: 10px;
}

.language-switcher select {
  background-color: #252525;
  color: white;
  border: 1px solid #333333;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
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

/* PAGINATION REMOVED FOR UNIFIED VIEW - KEEP BASIC FOR NOW */

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
