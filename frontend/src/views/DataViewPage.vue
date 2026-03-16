<template>
  <div class="data-view-container">
    <div class="page-header">
      <div class="header-left">
        <div class="language-switcher">
          <select v-model="$i18n.locale" @change="changeLanguage">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="ro">Română</option>
          </select>
        </div>
        <h1>{{ $t('data_library.title', { name: eventName || $t('common.loading') }) }}</h1>
      </div>
      <div class="header-right">
        <button class="export-btn">{{ $t('data_library.export_btn') }}</button>
        <button class="refresh-btn" @click="fetchData">{{ $t('data_library.refresh_btn') }}</button>
        <span class="record-count">{{ $t('data_library.records_count', { count: recordCount }) }}</span>
      </div>
    </div>

    <!-- Horizontal Tabs Navigation -->
    <div class="horizontal-tabs">
      <button 
        v-for="tab in tabs" 
        :key="tab.id" 
        class="tab-btn"
        :class="{ active: activeTab === tab.id }"
        @click="setActiveTab(tab.id)"
      >
        {{ tab.label === 'Unified View' ? $t('data_library.unified_view') : tab.label }}
      </button>
    </div>

    <!-- Tab Content Area -->
    <div class="tab-content">
      <!-- Loading State -->
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>{{ $t('data_library.loading') }}</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <p>{{ $t('data_library.error', { error: error }) }}</p>
        <button class="retry-btn" @click="fetchData">{{ $t('data_library.retry') }}</button>
      </div>

      <!-- No Data State (only if not loading and no error and no records) -->
      <div v-else-if="!loading && !error && recordCount === 0" class="no-data-state">
        <div class="empty-icon">📊</div>
        <p>{{ $t('data_library.no_data') }}</p>
        <p class="hint">{{ $t('data_library.hint_waiting') }}</p>
      </div>

      <!-- Data Table View -->
      <div v-else class="data-table-view">
        <div class="handsontable-container">
          <HotTable ref="hotTableComponent" :settings="hotSettings" />
        </div>
        <!-- Pagination reserved for future use -->
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.min.css';
import 'handsontable/styles/ht-theme-main.min.css';
import { API_BASE_URL, getAuthHeaders } from '../utils';
import { extractLabel } from '../utils/labelExtractor';

const { t, locale } = useI18n();
// Register all Handsontable modules
registerAllModules();

const route = useRoute();
const router = useRouter();

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}
const routeEventId = computed(() => route.query.eventId);
const routeModuleId = computed(() => route.query.moduleId);

const eventName = ref('');
const tabs = ref([]);
const activeTab = ref('unified'); // Default to unified view
const loading = ref(false);
const error = ref(null);

// Data for Handsontable
const tableData = ref([]);
const tableHeaders = ref([]);
const tableColumns = ref([]); // Column definitions for Handsontable

const hotTableComponent = ref(null);
const eventModules = ref([]); 

// -----------------------------------------------------------------------------
// Helper functions defined BEFORE usage in hotSettings to avoid hoisting issues
// -----------------------------------------------------------------------------

const updateResponse = async (moduleId, responseId, data) => {
  if (!responseId) {
    console.warn("Cannot update response: No Response ID found");
    return;
  }
  try {
    console.log(`[UnifiedView] Updating response ${responseId} in module ${moduleId}`, data);
    const response = await fetch(`${API_BASE_URL}/events/${routeEventId.value}/modules/${moduleId}/responses/${responseId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data })
    });
    if (!response.ok) throw new Error('Failed to update response');
    console.log("[UnifiedView] Response updated successfully");
  } catch (e) {
    console.error("[UnifiedView] Error updating response:", e);
  }
};

const getModuleFieldMapping = (moduleDef) => {
  const mapping = {};
  console.log('[UnifiedView] Generating mapping for module:', moduleDef ? moduleDef.name : 'undefined');
  if (moduleDef && moduleDef.type === 'form' && moduleDef.data && moduleDef.data.timeline) {
    console.log('[UnifiedView] Timeline found with steps:', moduleDef.data.timeline.length);
    moduleDef.data.timeline.forEach(step => {
       if(step.type === 'field') {
          // Always generate label from question text to ensure consistency
          const label = step.data.label ? extractLabel(step.data.label) : 'Question';
          mapping[step.id] = label;
          console.log(`[UnifiedView] Mapped field ${step.id} -> ${label}`);
       }
    });
  } else {
    console.log('[UnifiedView] No timeline found or module is not a form');
    if (moduleDef) console.log('Module type:', moduleDef.type, 'Data:', moduleDef.data);
  }
  return mapping;
};

const updateScore = async (moduleId, userId, score) => {
  try {
    console.log(`[UnifiedView] Updating score for user ${userId} in module ${moduleId}: ${score}`);
    const response = await fetch(`${API_BASE_URL}/events/${routeEventId.value}/modules/${moduleId}/quiz-scores/${userId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
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
    if (oldValue === newValue) return; // No change

    const rowData = tableData.value[row];
    if (!rowData) {
        console.warn("[UnifiedView] Row data not found for row", row);
        return;
    }
    const userId = rowData.userId;
    console.log(`[UnifiedView] Change detected for user ${userId}, prop ${prop}: ${oldValue} -> ${newValue}`);

    // Determine what changed
    if (prop === 'name') {
      if (rowData.nameSourceModuleId && rowData.nameSourceKey) {
        const moduleId = rowData.nameSourceModuleId;
        const responseId = rowData.responseIds[moduleId];
        const key = rowData.nameSourceKey;
        
        const currentModuleData = rowData.formData[moduleId] || {};
        const updatedModuleData = { ...currentModuleData, [key]: newValue };
        
        updateResponse(moduleId, responseId, updatedModuleData);
      } else {
        console.warn("[UnifiedView] Cannot update name: No source module tracked.");
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
  });
};

// -----------------------------------------------------------------------------
// HotSettings Initialization
// -----------------------------------------------------------------------------

const hotSettings = reactive({
  data: tableData,
  colHeaders: tableHeaders,
  columns: tableColumns,
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

const recordCount = computed(() => tableData.value.length);

const initializeUser = (map, userId) => {
  map.set(userId, {
    userId,
    name: '',
    scores: {},
    formData: {},
    responseIds: {}
  });
};

// -----------------------------------------------------------------------------
// Data Fetching Logic
// -----------------------------------------------------------------------------

const fetchEventDetails = async () => {
  if (!routeEventId.value) return;
  try {
    const response = await fetch(`${API_BASE_URL}/events/${routeEventId.value}`);
    if (!response.ok) throw new Error('Failed to fetch event details');
    const event = await response.json();
    eventName.value = event.name;
    eventModules.value = event.modules || [];
    console.log("[UnifiedView] Event Details Loaded:", eventName.value, eventModules.value);

    // Build Tabs
    const newTabs = [{ id: 'unified', label: 'Unified View' }];
    event.modules.forEach(m => {
      newTabs.push({
        id: m.id,
        label: `${m.type === 'form' ? 'Form' : 'Quiz'}: ${m.name}`
      });
    });
    tabs.value = newTabs;
    
    // Sync active tab with route or default
    if (routeModuleId.value) {
      if (event.modules.find(m => m.id === routeModuleId.value)) {
        activeTab.value = routeModuleId.value;
      } else {
         activeTab.value = 'unified';
      }
    } else {
      activeTab.value = 'unified';
    }
    console.log("[UnifiedView] Active Tab set to:", activeTab.value);

  } catch (e) {
    console.error("[UnifiedView] Error fetching event:", e);
    error.value = "Failed to load event details.";
  }
};

const setActiveTab = (tabId) => {
  console.log("[UnifiedView] Switching tab to:", tabId);
  activeTab.value = tabId;
  const query = { ...route.query };
  if (tabId === 'unified') {
    delete query.moduleId;
  } else {
    query.moduleId = tabId;
  }
  router.replace({ query });
};

const fetchData = async () => {
  if (!routeEventId.value) return;
  console.log("[UnifiedView] Fetching data for tab:", activeTab.value);
  loading.value = true;
  error.value = null;
  
  // Clear data first
  tableData.value = [];
  tableHeaders.value = [];
  tableColumns.value = [];

  try {
    // RELOAD SCHEMA: Fetch event details to ensure we have the latest form definitions (labels/IDs)
    await fetchEventDetails();

    if (activeTab.value === 'unified') {
      await fetchUnifiedData();
    } else {
      await fetchModuleData(activeTab.value);
    }
  } catch (e) {
    error.value = e.message;
    console.error("[UnifiedView] Error fetching data:", e);
  } finally {
    loading.value = false;
  }
};

const fetchUnifiedData = async () => {
  console.log("[UnifiedView] Starting Unified Data Fetch...");
  const promises = eventModules.value.map(m => 
    fetch(`${API_BASE_URL}/events/${routeEventId.value}/modules/${m.id}/responses`, { headers: getAuthHeaders() })
      .then(async res => {
        if (!res.ok) throw new Error(`Failed to fetch ${m.name} responses: ${res.statusText}`);
        return res.json();
      })
      .then(data => ({ moduleId: m.id, type: m.type, responses: data.responses || [] }))
  );

  const results = await Promise.all(promises);
  console.log("[UnifiedView] Fetched raw results from modules:", results);
  
  const userMap = new Map(); 

  results.forEach(({ moduleId, type, responses }) => {
    responses.forEach(res => {
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
        if (!userId) return;
        if (!userMap.has(userId)) initializeUser(userMap, userId);
        const userEntry = userMap.get(userId);

        userEntry.responseIds[moduleId] = res.id || res._id;

        if (!userEntry.formData[moduleId]) userEntry.formData[moduleId] = {};
        Object.assign(userEntry.formData[moduleId], res.data);

        if (res.data && res.data['step_welcome_name']) {
           userEntry.name = res.data['step_welcome_name'];
           userEntry.nameSourceKey = 'step_welcome_name';
           userEntry.nameSourceModuleId = moduleId;
        }
      }
    });
  });

  console.log("[UnifiedView] Aggregated User Map size:", userMap.size);

  const headers = [t('data_library.headers.user_id'), t('data_library.headers.name'), t('data_library.headers.total_score')];
  const columns = [
    { data: 'userId', readOnly: true },
    { data: 'name', readOnly: false }, 
    { data: 'totalScore', type: 'numeric', readOnly: true }
  ];

  eventModules.value.forEach(m => {
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
      
      const fieldLabels = getModuleFieldMapping(m);

      Array.from(allKeys).forEach(key => {
        const label = fieldLabels[key] || key;
        headers.push(`${m.name}: ${label}`);
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

  console.log("[UnifiedView] Constructed Headers:", headers);
  console.log("[UnifiedView] Constructed Columns:", columns);
  console.log("[UnifiedView] Flattened Data Sample:", flattenedData.slice(0, 2));

  tableHeaders.value = headers;
  tableColumns.value = columns;
  tableData.value = flattenedData;
  
  // Force update settings to be safe
  if (hotTableComponent.value && hotTableComponent.value.hotInstance) {
    console.log("[UnifiedView] Updating HotTable Settings with new data/columns...");
    hotTableComponent.value.hotInstance.updateSettings({
      colHeaders: headers,
      columns: columns,
      data: flattenedData,
      readOnly: false 
    });
  }
};

const fetchModuleData = async (moduleId) => {
  console.log(`[UnifiedView] Fetching Single Module Data: ${moduleId}`);
  const response = await fetch(`${API_BASE_URL}/events/${routeEventId.value}/modules/${moduleId}/responses`, { headers: getAuthHeaders() });
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status} ${response.statusText}`);
  }
  const result = await response.json();
  const rawData = result.responses || [];

  if (rawData.length === 0) {
    tableData.value = [];
    tableHeaders.value = [];
    tableColumns.value = [];
    return;
  }

  const allKeys = new Set();
  rawData.forEach(item => {
    if (item.data) {
       Object.keys(item.data).forEach(k => allKeys.add(k));
    }
    Object.keys(item).forEach(k => {
      if (k !== 'data' && k !== '_id' && k !== 'eventId' && k !== 'moduleId' && k !== '__v') {
        allKeys.add(k);
      }
    });
  });

  const moduleDef = eventModules.value.find(m => m.id === moduleId);
  const fieldLabels = getModuleFieldMapping(moduleDef);

  const headers = ['ID', ...Array.from(allKeys).map(k => fieldLabels[k] || k)];
  const columns = headers.map((h, index) => {
      // First column is ID
      if (index === 0) return { data: 'ID' };
      // Other columns, find the key that maps to this header or use the header itself (if it was a key)
      // Actually we need to map data keys to columns, but since we flatmapped keys, we can just use the key for data access?
      // Wait, Handsontable needs 'data' property to look up in the row object.
      // If we change the header, the data property in the row object (which is the key) doesn't change.
      // So we need to separate 'data' accessor from 'title' (header).
      
      // But here headers array is determining the visual order AND the accessor if we use simple array of objects.
      // In the Unified view we did: headers.push(label) AND columns.push({ data: key }).
      
      // Here in fetchModuleData we construct 'columns' based on 'headers'.
      return { data: Array.from(allKeys)[index - 1] }; // -1 because index 0 is ID
  }); 
  
  // Re-write this part to be clearer and correct
  const sortedKeys = Array.from(allKeys);
  const displayHeaders = ['ID', ...sortedKeys.map(k => fieldLabels[k] || k)];
  const displayColumns = [{ data: 'ID' }, ...sortedKeys.map(k => ({ data: k }))];

  const rows = rawData.map(item => {
    const row = { ID: item._id || item.id };
    if (item.data) Object.assign(row, item.data);
    Object.keys(item).forEach(k => {
       if (k !== 'data') row[k] = item[k];
    });
    return row;
  });

  console.log("[UnifiedView] Single Module Headers:", displayHeaders);
  console.log("[UnifiedView] Single Module Data Sample:", rows.slice(0, 2));

  tableHeaders.value = displayHeaders;
  tableColumns.value = displayColumns;
  
   if (hotTableComponent.value && hotTableComponent.value.hotInstance) {
    hotTableComponent.value.hotInstance.updateSettings({
      colHeaders: displayHeaders,
      columns: displayColumns,
      data: rows,
      readOnly: true 
    });
  } else {
    tableData.value = rows;
  }
};

onMounted(() => {
  fetchEventDetails().then(() => {
    fetchData(); 
  });
});

watch(activeTab, () => {
  fetchData();
});
</script>

<style scoped>
.data-view-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: #121212;
  color: #ffffff;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #333;
  box-shadow: 0 2px 4px rgba(0,0,0,0.3);
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

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.export-btn, .refresh-btn, .retry-btn {
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  color: white;
  transition: opacity 0.2s;
}

.export-btn { background-color: #007AFF; }
.refresh-btn { background-color: #30D158; }
.retry-btn { background-color: #FF6B6B; }

.export-btn:hover, .refresh-btn:hover, .retry-btn:hover {
  opacity: 0.9;
}

.record-count {
  font-size: 14px;
  color: #888;
}

/* Horizontal Tabs Styles */
.horizontal-tabs {
  display: flex;
  background-color: #1E1E1E;
  border-bottom: 1px solid #333;
  overflow-x: auto;
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
.loading-state, .error-state, .no-data-state {
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

.error-icon { font-size: 48px; margin-bottom: 16px; color: #FF6B6B;}
.empty-icon { font-size: 64px; margin-bottom: 16px; }

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
  background-color: #1E1E1E;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  overflow: hidden; /* Ensure table stays within round corners */
}
</style>
