<template>
  <div class="screen-manager">
    <div class="header">
      <div class="header-left">
        <button class="back-btn" @click="goBack"><AppIcon name="arrowLeft" :size="18" /> {{ $t('admin.details.back_btn') }}</button>
        <h1>Slide Editor</h1>
      </div>
      <div class="header-right">
        <button 
          class="launch-btn" 
          :class="{ 'close-btn': isProjectorOpen }"
          @click="isProjectorOpen ? closeProjector() : openProjector()"
        >
          {{ isProjectorOpen ? $t('admin.details.close_projector') : $t('screen_manager.open_projector') }}
        </button>
        <button class="push-btn" @click="pushToScreen" :disabled="!activeSlide">
          Push Current Slide
        </button>
        <button class="save-btn" @click="saveSlides">
          Save All
        </button>
      </div>
    </div>

    <div class="editor-layout">
      <!-- Left Sidebar -->
      <div class="sidebar left-sidebar">
        <h3>Slides</h3>
        <ul class="slide-list">
          <li 
            v-for="slide in slides" 
            :key="slide.id"
            :class="{ active: activeSlideId === slide.id }"
            @click="selectSlide(slide.id)"
          >
            <div class="slide-item">
              <span>{{ slide.name || slide.type }}</span>
              <button 
                v-if="slide.type === 'custom'" 
                class="icon-btn delete-slide" 
                @click.stop="deleteSlide(slide.id)"
              >
                <AppIcon name="close" :size="16" />
              </button>
            </div>
          </li>
        </ul>
        <button class="add-slide-btn" @click="addCustomSlide">+ Add Custom Slide</button>
      </div>

      <!-- Center Canvas -->
      <div class="canvas-area">
        <div 
          class="canvas-container" 
          ref="canvasRef"
          :style="{ backgroundColor: activeSlide?.backgroundColor || '#000000' }"
          @click="deselectElement"
          @mousemove="onMouseMove"
          @mouseup="onMouseUp"
          @mouseleave="onMouseUp"
        >
          <img v-if="activeSlide?.backgroundImage" :src="getImageUrl(activeSlide.backgroundImage)" class="canvas-bg-image" decoding="async" loading="eager" />
          
          <div
            v-for="element in activeSlideElements"
            :key="element.id"
            class="canvas-element"
            :class="{ selected: selectedElementId === element.id }"
            :style="{
              left: element.x + '%',
              top: element.y + '%',
              width: element.width + '%',
              height: element.height + '%',
              color: element.color || '#ffffff',
              fontSize: getScaledFontSize(element.fontSize) + 'px',
              textAlign: element.textAlign || 'center',
              fontWeight: element.fontWeight || 'normal'
            }"
            @mousedown.stop="startDrag(element, $event)"
            @click.stop="selectElement(element.id)"
          >
            <div class="el-content">
              <div v-if="element.type === 'text'" class="el-text">{{ element.content || 'Text' }}</div>
               <img v-else-if="element.type === 'image'" :src="getImageUrl(element.content)" class="el-image" decoding="async" loading="eager" />
              <div v-else-if="element.type === 'qr'" class="el-qr">
                <QRCodeVue :value="qrCodeValue" :size="150" level="H" />
              </div>
            </div>
            <div class="resize-handle" v-if="selectedElementId === element.id" @mousedown.stop="startResize(element, $event)"></div>
          </div>
        </div>
      </div>

      <!-- Right Sidebar -->
      <div class="sidebar right-sidebar" v-if="activeSlide">
        <h3>Properties</h3>
        <div v-if="selectedElement" class="props-panel">
          <h4>Element</h4>
          <div v-if="selectedElement.type === 'text'">
             <label>Text Content</label>
             <textarea v-model="selectedElement.content" rows="3"></textarea>
             
             <label>Text Color</label>
             <input type="color" v-model="selectedElement.color" />

             <label>Font Size</label>
             <input type="range" min="10" max="200" v-model.number="selectedElement.fontSize" />
             <span>{{ selectedElement.fontSize }}px</span>
          </div>
          <div v-if="selectedElement.type === 'image'">
             <label>Image</label>
             <button class="action-btn" @click="showImageSelector('element')">Select an Image</button>
          </div>
          
          <div class="pos-inputs mt-3">
             <div class="input-group"><label>X(%)</label><input type="number" v-model="selectedElement.x" step="1"/></div>
             <div class="input-group"><label>Y(%)</label><input type="number" v-model="selectedElement.y" step="1"/></div>
             <div class="input-group"><label>W(%)</label><input type="number" v-model="selectedElement.width" step="1"/></div>
             <div class="input-group"><label>H(%)</label><input type="number" v-model="selectedElement.height" step="1"/></div>
          </div>
          <button v-if="activeSlide.type !== 'screensaver'" class="danger-btn mt-4" @click="deleteSelectedElement">Delete Element</button>
        </div>
        <div v-else class="props-panel">
          <h4>Slide</h4>
          <div class="input-group mb-2">
            <label>Name</label>
            <input type="text" v-model="activeSlide.name" :disabled="activeSlide.type !== 'custom'" />
          </div>
          <div class="input-group mb-2">
            <label>Background Color</label>
            <input type="color" v-model="activeSlide.backgroundColor" />
          </div>
          <div class="input-group mb-3">
            <label>Background Image</label>
            <button class="action-btn block-btn" @click="showImageSelector('bg')">Select Background</button>
            <button v-if="activeSlide.backgroundImage" @click="activeSlide.backgroundImage = ''" class="danger-btn mt-2 block-btn">Clear Image</button>
          </div>
          
          <hr v-if="activeSlide.type !== 'screensaver'" />
          <h4 v-if="activeSlide.type !== 'screensaver'">Add Elements</h4>
          <button class="action-btn block-btn mb-2" v-if="activeSlide.type !== 'screensaver'" @click="addElement('text')">+ Add Text</button>
          <button class="action-btn block-btn mb-2" v-if="activeSlide.type !== 'screensaver'" @click="addElement('image')">+ Add Image</button>
          <button class="action-btn block-btn" v-if="activeSlide.type === 'welcome'" @click="addElement('qr')">+ Add QR Code</button>
          
          <div v-if="activeSlide.type === 'custom'" class="mt-4">
            <hr />
            <h4 class="mt-3">Template Options</h4>
            <button class="action-btn block-btn mb-2" @click="saveAsTemplate">Save as Template</button>
            <button class="action-btn block-btn" @click="openTemplateSelector">Apply Template</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Templates Selector Modal -->
    <div v-if="isTemplateSelectorOpen" class="modal-overlay" @click.self="isTemplateSelectorOpen = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>Apply Template</h3>
          <button class="close-btn" @click="isTemplateSelectorOpen = false" aria-label="Close"><AppIcon name="close" :size="18" /></button>
        </div>
        <div class="modal-body">
          <div v-if="templates.length === 0" style="text-align:center;">No templates saved.</div>
          <div class="screensavers-grid">
            <div 
              v-for="t in templates" 
              :key="t.id" 
              class="screensaver-option template-option"
              @click="applyTemplate(t)"
            >
              <div class="template-box">
                <span class="template-name">{{ t.name }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Library Selector Modal -->
    <div v-if="isImageSelectorOpen" class="modal-overlay" @click.self="isImageSelectorOpen = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>Select Image</h3>
          <button class="close-btn" @click="isImageSelectorOpen = false" aria-label="Close"><AppIcon name="close" :size="18" /></button>
        </div>
        <div class="modal-body">
          <div class="screensavers-grid">
            <div 
              v-for="s in imageLibrary" 
              :key="s.id" 
              class="screensaver-option"
              @click="handleImageSelect(s.mediaId || s.id)"
            >
               <img :src="s.thumbnailUrl || getImageUrl(s.mediaId || s.id)" decoding="async" loading="lazy" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCodeVue from 'qrcode.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { getImageUrl, getScreensavers } from '../services/media'
import AppIcon from '../components/AppIcon.vue'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const { success, error } = useToast()

const eventId = route.params.id

const projectorWindow = ref(null)
const isProjectorOpen = computed(() => projectorWindow.value && !projectorWindow.value.closed)

const slides = ref([])
const activeSlideId = ref(null)

const activeSlide = computed(() => slides.value.find(s => s.id === activeSlideId.value))
const activeSlideElements = computed(() => activeSlide.value?.elements || [])
const qrCodeValue = computed(() => `${window.location.origin}/?eventId=${eventId}`)

const selectedElementId = ref(null)
const selectedElement = computed(() => activeSlideElements.value.find(e => e.id === selectedElementId.value))

const canvasRef = ref(null)
const isDragging = ref(false)
const isResizing = ref(false)
const dragItem = ref(null)
const dragStart = ref({ x: 0, y: 0, startX: 0, startY: 0, startW: 0, startH: 0 })

const isImageSelectorOpen = ref(false)
const imageLibrary = ref([])
const imageSelectorTarget = ref('') // 'bg' or 'element'

const isTemplateSelectorOpen = ref(false)
const templates = ref([])

const fetchTemplates = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/templates`, { headers: getAuthHeaders() })
    if (res.ok) {
      const allTpls = await res.json()
      templates.value = allTpls.filter(t => t.moduleType === 'slide')
    }
  } catch(err) {
    console.error(err)
  }
}

const saveAsTemplate = async () => {
  if(!activeSlide.value) return;
  const tplName = prompt("Enter a name for this template:", activeSlide.value.name);
  if(!tplName) return;

  try {
    const payload = {
      name: tplName,
      moduleType: 'slide',
      data: {
        backgroundColor: activeSlide.value.backgroundColor,
        backgroundImage: activeSlide.value.backgroundImage,
        elements: activeSlide.value.elements
      }
    };
    const res = await fetch(`${API_BASE_URL}/templates`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    });
    if(res.ok) {
      success('Saved as template successfully');
    }
  } catch(err) {
    error('Failed to save template');
  }
}

const openTemplateSelector = async () => {
  await fetchTemplates();
  isTemplateSelectorOpen.value = true; // Fix modal display
}

const applyTemplate = (tpl) => {
  if(!activeSlide.value || !tpl.data) return;
  
  if (confirm(`Apply template "${tpl.name}"? This will overwrite the current slide.`)) {
    activeSlide.value.backgroundColor = tpl.data.backgroundColor || '#1E1E1E';
    activeSlide.value.backgroundImage = tpl.data.backgroundImage || '';
    
    // Copy elements to generate new IDs so they don't crash
    const newElements = (tpl.data.elements || []).map(el => ({
      ...el,
      id: generateId()
    }))
    activeSlide.value.elements = newElements;
    
    isTemplateSelectorOpen.value = false;
    success('Template applied');
  }
}

onMounted(() => {
  fetchSlides()
})

const fetchSlides = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/slides`, {
        headers: getAuthHeaders()
    })
    if (res.ok) {
      slides.value = await res.json()
      if (slides.value.length > 0 && !activeSlideId.value) {
        activeSlideId.value = slides.value[0].id
      }
    }
  } catch (err) {
    console.error(err)
    error('Failed to load slides')
  }
}

const saveSlides = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/slides`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(slides.value)
    })
    if (res.ok) {
      success('Slides saved successfully')
    }
  } catch (err) {
    console.error(err)
    error('Failed to save slides')
  }
}

const pushToScreen = async () => {
  if (!activeSlide.value) return
  
  // First save to make sure changes are synced
  await saveSlides()

  const payload = {
    currentView: 'slide',
    activeSlide: activeSlide.value
  }
  
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screen-state`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })
    if (res.ok) {
      success('Pushed to Big Screen')
    }
  } catch (err) {
    error('Failed to push to screen')
  }
}

const selectSlide = (id) => {
  activeSlideId.value = id
  selectedElementId.value = null
}

const generateId = () => Math.random().toString(36).substr(2, 9)

const addCustomSlide = () => {
  const newSlide = {
    id: generateId(),
    type: 'custom',
    name: 'New Custom Slide',
    backgroundColor: '#1E1E1E',
    backgroundImage: '',
    elements: []
  }
  slides.value.push(newSlide)
  activeSlideId.value = newSlide.id
}

const deleteSlide = (id) => {
  if (confirm('Delete this slide?')) {
    slides.value = slides.value.filter(s => s.id !== id)
    if (activeSlideId.value === id) {
      activeSlideId.value = slides.value[0]?.id || null
    }
  }
}

const addElement = (type) => {
  if (!activeSlide.value) return
  const newEl = {
    id: generateId(),
    type,
    x: 40,
    y: 40,
    width: 20,
    height: 10,
    content: type === 'text' ? 'New Text' : '',
    color: '#ffffff',
    fontSize: 48,
    textAlign: 'center'
  }
  
  if (type === 'image') {
    newEl.width = 20
    newEl.height = 30
  }
  if (type === 'qr') {
    newEl.width = 15
    newEl.height = 25
  }

  if (!activeSlide.value.elements) activeSlide.value.elements = []
  activeSlide.value.elements.push(newEl)
  selectedElementId.value = newEl.id
}

const selectElement = (id) => {
  selectedElementId.value = id
}

const deselectElement = () => {
  selectedElementId.value = null
}

const deleteSelectedElement = () => {
  if (activeSlide.value && selectedElementId.value) {
    activeSlide.value.elements = activeSlide.value.elements.filter(e => e.id !== selectedElementId.value)
    selectedElementId.value = null
  }
}

// Mouse Drag/Resize logic processing percentage
const getCanvasRect = () => {
  if (!canvasRef.value) return null
  return canvasRef.value.getBoundingClientRect()
}

const startDrag = (item, e) => {
  if (activeSlide.value?.type === 'screensaver') return
  selectElement(item.id)
  isDragging.value = true
  dragItem.value = item
  dragStart.value = {
    x: e.clientX,
    y: e.clientY,
    startX: item.x,
    startY: item.y
  }
}

const startResize = (item, e) => {
  if (activeSlide.value?.type === 'screensaver') return
  selectElement(item.id)
  isResizing.value = true
  dragItem.value = item
  dragStart.value = {
    x: e.clientX,
    y: e.clientY,
    startW: item.width,
    startH: item.height
  }
}

const onMouseMove = (e) => {
  if ((!isDragging.value && !isResizing.value) || !dragItem.value) return
  const rect = getCanvasRect()
  if (!rect) return
  
  const dx = e.clientX - dragStart.value.x
  const dy = e.clientY - dragStart.value.y
  
  const dxPct = (dx / rect.width) * 100
  const dyPct = (dy / rect.height) * 100

  if (isDragging.value) {
    let newX = dragStart.value.startX + dxPct
    let newY = dragStart.value.startY + dyPct
    dragItem.value.x = Math.round(newX * 10) / 10
    dragItem.value.y = Math.round(newY * 10) / 10
  } else if (isResizing.value) {
    let newW = dragStart.value.startW + dxPct
    let newH = dragStart.value.startH + dyPct
    dragItem.value.width = Math.max(1, Math.round(newW * 10) / 10)
    dragItem.value.height = Math.max(1, Math.round(newH * 10) / 10)
  }
}

const onMouseUp = () => {
  isDragging.value = false
  isResizing.value = false
  dragItem.value = null
}

const getScaledFontSize = (size) => {
    if (!canvasRef.value) return size;
    const cw = canvasRef.value.clientWidth;
    // Base standard resolution 1920
    const ratio = cw / 1920;
    return size * ratio;
}

const showImageSelector = async (target) => {
  imageSelectorTarget.value = target
  try {
    imageLibrary.value = await getScreensavers()
    isImageSelectorOpen.value = true
  } catch (err) {
    error('Failed to load library')
  }
}

const handleImageSelect = (mediaId) => {
  if (imageSelectorTarget.value === 'bg') {
    activeSlide.value.backgroundImage = mediaId
  } else if (imageSelectorTarget.value === 'element' && selectedElement.value) {
    selectedElement.value.content = mediaId
  }
  isImageSelectorOpen.value = false
}

const goBack = () => {
  router.push(`/admin/event/${eventId}`)
}

const openProjector = () => {
  if (projectorWindow.value && !projectorWindow.value.closed) {
    projectorWindow.value.focus()
  } else {
    projectorWindow.value = window.open(`/projector/${eventId}`, 'projector', 'width=1200,height=800')
  }
}

const closeProjector = () => {
  if (projectorWindow.value && !projectorWindow.value.closed) {
    projectorWindow.value.close()
    projectorWindow.value = null
  }
}

</script>

<style scoped>
.screen-manager {
  min-height: 100vh;
  background-color: #0d0d0d;
  color: #fff;
  display: flex;
  flex-direction: column;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 30px;
  background-color: #1a1a1d;
  border-bottom: 1px solid #333;
}
.header-left { display: flex; align-items: center; gap: 20px; }
.header-left h1 { margin: 0; font-size: 1.5rem; font-weight: 500; }
.back-btn {
  background: transparent; border: 1px solid #444; color: #fff;
  padding: 6px 12px; border-radius: 4px; cursor: pointer;
}
.header-right { display: flex; gap: 10px; }
.launch-btn, .push-btn, .save-btn, .action-btn, .danger-btn, .add-slide-btn {
  padding: 8px 16px; border: none; border-radius: 6px; cursor: pointer; font-weight: 500;
  transition: opacity 0.2s;
}
.launch-btn { background: #3b82f6; color: white; }
.launch-btn.close-btn { background: #ef4444; }
.push-btn { background: #10b981; color: white; }
.push-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.save-btn { background: var(--ops-blue); color: white; }
.action-btn { background: #4b5563; color: white; }
.danger-btn { background: #ef4444; color: white; }
.add-slide-btn { background: #374151; color: white; width: 100%; margin-top: 10px; }
.icon-btn { background: transparent; color: #9ca3af; border: none; cursor: pointer; }
.icon-btn:hover { color: #f87171; }
.block-btn { display: block; width: 100%; }

.editor-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 250px;
  background: #151515;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow-y: auto;
}
.right-sidebar { border-left: 1px solid #333; border-right: none; }
.sidebar h3 { margin-top: 0; font-weight: 500; color: #9ca3af; font-size: 0.9rem; text-transform: uppercase; }

.slide-list { list-style: none; padding: 0; margin: 0; }
.slide-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 15px; background: #222; border-radius: 6px; margin-bottom: 8px; cursor: pointer;
}
.slide-list li.active .slide-item { background: #3b82f6; color: white; }

.canvas-area {
  flex: 1;
  display: flex;
  align-items: center; justify-content: center;
  background: #111;
  overflow: hidden;
  position: relative;
}

/* 16:9 Canvas */
.canvas-container {
  width: 90%;
  aspect-ratio: 16 / 9;
  background: #000;
  position: relative;
  box-shadow: 0 10px 30px rgba(0,0,0,0.5);
  border: 1px solid #333;
  overflow: hidden;
}

.canvas-bg-image {
  position: absolute;
  top: 0; left: 0; width: 100%; height: 100%;
  object-fit: cover;
  z-index: 0;
}

.canvas-element {
  position: absolute;
  z-index: 10;
  cursor: grab;
  user-select: none;
}
.canvas-element:active { cursor: grabbing; }
.canvas-element.selected { border: 2px dashed #3b82f6; }

.el-content { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }
.el-text { user-select: none; width: 100%; }
.el-image { width: 100%; height: 100%; object-fit: contain; pointer-events: none; }
.el-qr { background: white; padding: 10px; border-radius: 8px; width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }

.resize-handle {
  position: absolute;
  right: -5px; bottom: -5px;
  width: 10px; height: 10px;
  background: #3b82f6;
  border-radius: 50%;
  cursor: se-resize;
}

/* Props Panel */
.props-panel label { display: block; font-size: 0.8rem; color: #9ca3af; margin-bottom: 4px; margin-top: 10px; }
.props-panel input[type="text"], .props-panel input[type="number"], .props-panel textarea {
  width: 100%; background: #222; border: 1px solid #444; color: white; padding: 6px; border-radius: 4px;
}
.input-group { display: flex; flex-direction: column; margin-bottom: 10px; }
.pos-inputs { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

/* Modals */
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.8); z-index: 1000;
  display: flex; align-items: center; justify-content: center;
}
.modal-content { background: #1e1e1e; padding: 20px; border-radius: 12px; width: 90%; max-width: 800px; }
.screensavers-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 15px; margin-top: 15px;
}
.screensaver-option { cursor: pointer; }
.screensaver-option img { width: 100%; aspect-ratio: 16/9; object-fit: cover; border-radius: 6px; }

/* Templates */
.template-box {
  background: #333;
  aspect-ratio: 16/9;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  border: 1px solid #444;
  transition: all 0.2s;
}
.template-box:hover {
  border-color: #3b82f6;
  background: #2563eb;
}
.template-name {
  font-weight: 500;
  color: white;
  text-align: center;
  padding: 10px;
}
</style>
