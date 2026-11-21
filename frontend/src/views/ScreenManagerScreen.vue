<template>
  <div ref="screenManagerRef" class="screen-manager">
    <header class="header">
      <div class="header-left">
        <button type="button" class="back-btn" @click="goBack">
          <AppIcon name="arrowLeft" :size="18" aria-hidden="true" />
          <span>{{ $t('library.back_btn') }}</span>
        </button>
        <h1>{{ $t('screen_manager.editor.title') }}</h1>
      </div>
      <div class="header-right">
        <button
          type="button"
          class="launch-btn"
          :class="{ 'close-btn': isProjectorOpen }"
          @click="isProjectorOpen ? closeProjector() : openProjector()"
        >
          {{ isProjectorOpen ? $t('admin.details.close_projector') : $t('screen_manager.open_projector') }}
        </button>
        <button type="button" class="push-btn" @click="pushToScreen" :disabled="!activeSlide">
          {{ $t('screen_manager.editor.push_current') }}
        </button>
        <button type="button" class="save-btn" @click="saveSlides">
          {{ $t('screen_manager.editor.save_all') }}
        </button>
      </div>
    </header>

    <main class="editor-layout">
      <!-- Center Canvas -->
      <div class="canvas-area">
        <div
          class="canvas-container"
          ref="canvasRef"
          :style="{ backgroundColor: activeSlide?.backgroundColor || '#000000' }"
          @click="onCanvasClick"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
        >
          <img
            v-if="activeSlide?.backgroundImage"
            :src="getImageUrl(activeSlide.backgroundImage)"
            alt=""
            class="canvas-bg-image"
            decoding="async"
            loading="eager"
          />

          <div
            v-for="element in activeSlideElements"
            :key="element.id"
            class="canvas-element"
            :class="{ selected: selectedElementId === element.id }"
            tabindex="0"
            :aria-label="getElementAriaLabel(element)"
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
            @pointerdown.stop="startDrag(element, $event)"
            @pointermove.stop="onPointerMove"
            @pointerup.stop="onPointerUp"
            @pointercancel.stop="onPointerUp"
            @lostpointercapture.stop="onPointerUp"
            @click.stop="selectElement(element.id)"
            @keydown="onElementKeydown(element, $event)"
          >
            <div class="el-content">
              <div v-if="element.type === 'text'" class="el-text">{{ element.content || $t('screen_manager.editor.text') }}</div>
              <img
                v-else-if="element.type === 'image'"
                :src="getImageUrl(element.content)"
                :alt="$t('screen_manager.editor.image')"
                class="el-image"
                decoding="async"
                loading="eager"
              />
              <div v-else-if="element.type === 'qr'" class="el-qr">
                <QRCodeVue :value="qrCodeValue" :size="150" level="H" />
              </div>
            </div>
            <div
              v-if="selectedElementId === element.id"
              class="resize-handle"
              @pointerdown.stop="startResize(element, $event)"
              @pointermove.stop="onPointerMove"
              @pointerup.stop="onPointerUp"
              @pointercancel.stop="onPointerUp"
              @lostpointercapture.stop="onPointerUp"
              @click.stop
            ></div>
          </div>
        </div>
        <div class="sr-only" aria-live="polite">{{ liveAnnouncement }}</div>
      </div>

      <!-- Mobile Panel Switcher (visible <= 900px) -->
      <div class="mobile-panel-tabs" role="tablist" :aria-label="$t('screen_manager.editor.title')">
        <button
          id="tab-slides"
          ref="slidesTabRef"
          type="button"
          role="tab"
          class="mobile-tab-btn"
          :aria-selected="mobileTab === 'slides' ? 'true' : 'false'"
          aria-controls="panel-slides"
          :tabindex="mobileTab === 'slides' ? 0 : -1"
          @click="mobileTab = 'slides'"
          @keydown.left.prevent="switchMobileTab('properties')"
          @keydown.right.prevent="switchMobileTab('properties')"
        >
          {{ $t('screen_manager.editor.mobile_slides') }}
        </button>
        <button
          id="tab-properties"
          ref="propertiesTabRef"
          type="button"
          role="tab"
          class="mobile-tab-btn"
          :aria-selected="mobileTab === 'properties' ? 'true' : 'false'"
          aria-controls="panel-properties"
          :tabindex="mobileTab === 'properties' ? 0 : -1"
          @click="mobileTab = 'properties'"
          @keydown.left.prevent="switchMobileTab('slides')"
          @keydown.right.prevent="switchMobileTab('slides')"
        >
          {{ $t('screen_manager.editor.mobile_properties') }}
        </button>
      </div>

      <!-- Left Sidebar -->
      <div
        id="panel-slides"
        class="sidebar left-sidebar"
        :class="{ 'mobile-panel-hidden': mobileTab !== 'slides' }"
        :role="isCompactLayout ? 'tabpanel' : undefined"
        :aria-labelledby="isCompactLayout ? 'tab-slides' : undefined"
      >
        <h3>{{ $t('screen_manager.editor.slides') }}</h3>
        <ul class="slide-list">
          <li
            v-for="slide in slides"
            :key="slide.id"
            :class="{ active: activeSlideId === slide.id }"
          >
            <div class="slide-item" @click="selectSlide(slide.id)">
              <button
                type="button"
                class="slide-select-btn"
                :aria-current="activeSlideId === slide.id ? 'true' : undefined"
                @click.stop="selectSlide(slide.id)"
              >
                {{ slide.name || slide.type }}
              </button>
              <button
                v-if="slide.type === 'custom'"
                type="button"
                class="icon-btn delete-slide"
                :aria-label="`${$t('admin.delete_btn')}: ${slide.name || slide.type}`"
                @click.stop="deleteSlide(slide.id)"
              >
                <AppIcon name="close" :size="16" aria-hidden="true" />
              </button>
            </div>
          </li>
        </ul>
        <button type="button" class="add-slide-btn" @click="addCustomSlide">{{ $t('screen_manager.editor.add_custom_slide') }}</button>
      </div>

      <!-- Right Sidebar -->
      <div
        id="panel-properties"
        class="sidebar right-sidebar"
        :class="{ 'mobile-panel-hidden': mobileTab !== 'properties', 'desktop-hidden': !activeSlide }"
        :role="isCompactLayout ? 'tabpanel' : undefined"
        :aria-labelledby="isCompactLayout ? 'tab-properties' : undefined"
      >
        <template v-if="activeSlide">
          <h3>{{ $t('screen_manager.editor.properties') }}</h3>
          <div v-if="selectedElement" class="props-panel">
            <h4>{{ $t('screen_manager.editor.element') }}</h4>
            <div v-if="selectedElement.type === 'text'">
              <label for="el-text-content">{{ $t('screen_manager.editor.text_content') }}</label>
              <textarea id="el-text-content" v-model="selectedElement.content" rows="3"></textarea>

              <label for="el-text-color">{{ $t('screen_manager.editor.text_color') }}</label>
              <input id="el-text-color" type="color" v-model="selectedElement.color" />

              <label for="el-font-size">{{ $t('screen_manager.editor.font_size') }}</label>
              <input id="el-font-size" type="range" min="10" max="200" v-model.number="selectedElement.fontSize" />
              <span>{{ selectedElement.fontSize }}px</span>
            </div>
            <div v-if="selectedElement.type === 'image'">
              <label>{{ $t('screen_manager.editor.image') }}</label>
              <button type="button" class="action-btn" @click="showImageSelector('element')">{{ $t('screen_manager.editor.select_image') }}</button>
            </div>

            <div class="pos-inputs mt-3">
              <div class="input-group">
                <label for="el-pos-x">X(%)</label>
                <input id="el-pos-x" type="number" v-model.number="selectedElement.x" min="0" max="100" step="1" />
              </div>
              <div class="input-group">
                <label for="el-pos-y">Y(%)</label>
                <input id="el-pos-y" type="number" v-model.number="selectedElement.y" min="0" max="100" step="1" />
              </div>
              <div class="input-group">
                <label for="el-pos-w">W(%)</label>
                <input id="el-pos-w" type="number" v-model.number="selectedElement.width" min="5" max="100" step="1" />
              </div>
              <div class="input-group">
                <label for="el-pos-h">H(%)</label>
                <input id="el-pos-h" type="number" v-model.number="selectedElement.height" min="5" max="100" step="1" />
              </div>
            </div>
            <button v-if="activeSlide.type !== 'screensaver'" type="button" class="danger-btn mt-4" @click="deleteSelectedElement">{{ $t('screen_manager.editor.delete_element') }}</button>
          </div>
          <div v-else class="props-panel">
            <h4>{{ $t('screen_manager.editor.slide') }}</h4>
            <div class="input-group mb-2">
              <label for="slide-name-input">{{ $t('screen_manager.editor.name') }}</label>
              <input id="slide-name-input" type="text" v-model="activeSlide.name" :disabled="activeSlide.type !== 'custom'" />
            </div>
            <div class="input-group mb-2">
              <label for="slide-bg-color">{{ $t('screen_manager.editor.background_color') }}</label>
              <input id="slide-bg-color" type="color" v-model="activeSlide.backgroundColor" />
            </div>
            <div class="input-group mb-3">
              <label>{{ $t('screen_manager.editor.background_image') }}</label>
              <button type="button" class="action-btn block-btn" @click="showImageSelector('bg')">{{ $t('screen_manager.editor.select_background') }}</button>
              <button v-if="activeSlide.backgroundImage" type="button" @click="activeSlide.backgroundImage = ''" class="danger-btn mt-2 block-btn">{{ $t('screen_manager.editor.clear_image') }}</button>
            </div>

            <hr v-if="activeSlide.type !== 'screensaver'" />
            <h4 v-if="activeSlide.type !== 'screensaver'">{{ $t('screen_manager.editor.add_elements') }}</h4>
            <button type="button" class="action-btn block-btn mb-2" v-if="activeSlide.type !== 'screensaver'" @click="addElement('text')">{{ $t('screen_manager.editor.add_text') }}</button>
            <button type="button" class="action-btn block-btn mb-2" v-if="activeSlide.type !== 'screensaver'" @click="addElement('image')">{{ $t('screen_manager.editor.add_image') }}</button>
            <button type="button" class="action-btn block-btn" v-if="activeSlide.type === 'welcome'" @click="addElement('qr')">{{ $t('screen_manager.editor.add_qr') }}</button>

            <div v-if="activeSlide.type === 'custom'" class="mt-4">
              <hr />
              <h4 class="mt-3">{{ $t('screen_manager.editor.template_options') }}</h4>
              <button type="button" class="action-btn block-btn mb-2" @click="saveAsTemplate">{{ $t('screen_manager.editor.save_template') }}</button>
              <button type="button" class="action-btn block-btn" @click="openTemplateSelector">{{ $t('screen_manager.editor.apply_template') }}</button>
            </div>
          </div>
        </template>
      </div>
    </main>

    <!-- Templates Selector Modal -->
    <div v-if="isTemplateSelectorOpen" class="modal-overlay" @click.self="closeTemplateSelector">
      <div
        ref="templateModalRef"
        class="modal-content"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('screen_manager.editor.apply_template')"
        tabindex="-1"
        @keydown.escape.stop.prevent="closeTemplateSelector"
      >
        <div class="modal-header">
          <h3>{{ $t('screen_manager.editor.apply_template') }}</h3>
          <button type="button" class="close-btn" @click="closeTemplateSelector" :aria-label="$t('common.close')">
            <AppIcon name="close" :size="18" aria-hidden="true" />
          </button>
        </div>
        <div class="modal-body">
          <div v-if="templates.length === 0" class="modal-empty">{{ $t('screen_manager.editor.no_templates') }}</div>
          <div class="screensavers-grid">
            <button
              v-for="tpl in templates"
              :key="tpl.id"
              type="button"
              class="screensaver-option template-option"
              @click="applyTemplate(tpl)"
            >
              <div class="template-box">
                <span class="template-name">{{ tpl.name }}</span>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Library Selector Modal -->
    <div v-if="isImageSelectorOpen" class="modal-overlay" @click.self="closeImageSelector">
      <div
        ref="imageModalRef"
        class="modal-content"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('screen_manager.editor.select_image')"
        tabindex="-1"
        @keydown.escape.stop.prevent="closeImageSelector"
      >
        <div class="modal-header">
          <h3>{{ $t('screen_manager.editor.select_image') }}</h3>
          <button type="button" class="close-btn" @click="closeImageSelector" :aria-label="$t('common.close')">
            <AppIcon name="close" :size="18" aria-hidden="true" />
          </button>
        </div>
        <div class="modal-body">
          <div class="screensavers-grid">
            <button
              v-for="s in imageLibrary"
              :key="s.id"
              type="button"
              class="screensaver-option"
              :aria-label="s.name || s.filename || $t('screen_manager.editor.select_image')"
              @click="handleImageSelect(s.mediaId || s.id)"
            >
              <img
                :src="s.thumbnailUrl || getImageUrl(s.mediaId || s.id)"
                :alt="s.name || s.filename || $t('screen_manager.editor.image')"
                decoding="async"
                loading="lazy"
              />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
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

const screenManagerRef = ref(null)
const isCompactLayout = ref(typeof window !== 'undefined' ? window.innerWidth <= 900 : false)
const mobileTab = ref('slides')
const slidesTabRef = ref(null)
const propertiesTabRef = ref(null)

const projectorWindow = ref(null)
const isProjectorOpen = ref(false)
let projectorPollTimer = null

const eventPin = ref('')
const slides = ref([])
const activeSlideId = ref(null)

const activeSlide = computed(() => slides.value.find(s => s.id === activeSlideId.value))
const activeSlideElements = computed(() => activeSlide.value?.elements || [])
const qrCodeValue = computed(() => `${window.location.origin}/?pin=${eventPin.value || eventId}`)

const selectedElementId = ref(null)
const selectedElement = computed(() => activeSlideElements.value.find(e => e.id === selectedElementId.value))

const canvasRef = ref(null)
const canvasWidth = ref(0)
const liveAnnouncement = ref('')

const isDragging = ref(false)
const isResizing = ref(false)
const dragItem = ref(null)
const activePointerId = ref(null)
const dragStart = ref({ x: 0, y: 0, startX: 0, startY: 0, startW: 0, startH: 0 })
let didMoveDuringDrag = false
let suppressCanvasClick = false

const isImageSelectorOpen = ref(false)
const imageLibrary = ref([])
const imageSelectorTarget = ref('') // 'bg' or 'element'
const imageModalRef = ref(null)

const isTemplateSelectorOpen = ref(false)
const templates = ref([])
const templateModalRef = ref(null)

let modalTriggerEl = null
let layoutObserver = null

const round1 = (value) => Math.round(value * 10) / 10
const clamp = (value, min, max) => Math.min(max, Math.max(min, value))

const switchMobileTab = (tab) => {
  mobileTab.value = tab
  nextTick(() => {
    const target = tab === 'slides' ? slidesTabRef.value : propertiesTabRef.value
    target?.focus?.()
  })
}

const getElementLabel = (element) => {
  if (!element) return ''
  if (element.type === 'text') {
    const content = element.content?.trim()
    return content ? `${t('screen_manager.editor.text')}: ${content}` : t('screen_manager.editor.text')
  }
  if (element.type === 'image') {
    return t('screen_manager.editor.image')
  }
  if (element.type === 'qr') {
    return 'QR'
  }
  return element.type || t('screen_manager.editor.element')
}

const getElementAriaLabel = (element) => {
  const label = getElementLabel(element)
  return `${label}, X ${element.x}%, Y ${element.y}%, W ${element.width}%, H ${element.height}%`
}

const fetchTemplates = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/templates`, { headers: getAuthHeaders() })
    if (res.ok) {
      const allTpls = await res.json()
      templates.value = allTpls.filter(tpl => tpl.moduleType === 'slide')
    }
  } catch (err) {
    console.error(err)
  }
}

const saveAsTemplate = async () => {
  if (!activeSlide.value) return
  const tplName = prompt(t('screen_manager.editor.template_name_prompt'), activeSlide.value.name)
  if (!tplName) return

  try {
    const payload = {
      name: tplName,
      moduleType: 'slide',
      data: {
        backgroundColor: activeSlide.value.backgroundColor,
        backgroundImage: activeSlide.value.backgroundImage,
        elements: activeSlide.value.elements
      }
    }
    const res = await fetch(`${API_BASE_URL}/templates`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })
    if (!res.ok) {
      error(t('screen_manager.editor.template_save_failed'))
      return
    }
    success(t('screen_manager.editor.template_saved'))
  } catch (err) {
    error(t('screen_manager.editor.template_save_failed'))
  }
}

const openTemplateSelector = async () => {
  await fetchTemplates()
  isTemplateSelectorOpen.value = true
}

const closeTemplateSelector = () => {
  isTemplateSelectorOpen.value = false
}

const closeImageSelector = () => {
  isImageSelectorOpen.value = false
}

const applyTemplate = (tpl) => {
  if (!activeSlide.value || !tpl.data) return

  if (confirm(t('screen_manager.editor.apply_confirm', { name: tpl.name }))) {
    activeSlide.value.backgroundColor = tpl.data.backgroundColor || '#1E1E1E'
    activeSlide.value.backgroundImage = tpl.data.backgroundImage || ''

    // Copy elements to generate new IDs so they don't crash
    const newElements = (tpl.data.elements || []).map(el => ({
      ...el,
      id: generateId()
    }))
    activeSlide.value.elements = newElements

    isTemplateSelectorOpen.value = false
    success(t('screen_manager.editor.template_applied'))
  }
}

const handleGlobalKeydown = (e) => {
  if (e.key !== 'Escape') return
  if (isImageSelectorOpen.value) {
    e.preventDefault()
    closeImageSelector()
    return
  }
  if (isTemplateSelectorOpen.value) {
    e.preventDefault()
    closeTemplateSelector()
  }
}

watch(
  [isTemplateSelectorOpen, isImageSelectorOpen],
  async ([tplOpen, imgOpen], [prevTpl, prevImg]) => {
    const isAnyOpen = tplOpen || imgOpen
    const wasAnyOpen = prevTpl || prevImg

    if (isAnyOpen && !wasAnyOpen) {
      modalTriggerEl = document.activeElement instanceof HTMLElement ? document.activeElement : null
    }

    if (tplOpen) {
      await nextTick()
      templateModalRef.value?.focus?.()
    } else if (imgOpen) {
      await nextTick()
      imageModalRef.value?.focus?.()
    } else if (wasAnyOpen && !isAnyOpen) {
      await nextTick()
      modalTriggerEl?.focus?.()
      modalTriggerEl = null
    }
  }
)

const updateLayoutMetrics = () => {
  if (screenManagerRef.value && screenManagerRef.value.clientWidth > 0) {
    isCompactLayout.value = screenManagerRef.value.clientWidth <= 900
  } else if (typeof window !== 'undefined') {
    isCompactLayout.value = window.innerWidth <= 900
  }
  if (canvasRef.value) {
    canvasWidth.value = canvasRef.value.clientWidth
  }
}

onMounted(() => {
  fetchEventMeta()
  fetchSlides()
  updateLayoutMetrics()
  window.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('resize', updateLayoutMetrics)
  window.addEventListener('focus', syncProjectorState)

  if (typeof ResizeObserver !== 'undefined') {
    layoutObserver = new ResizeObserver(() => {
      updateLayoutMetrics()
    })
    if (screenManagerRef.value) layoutObserver.observe(screenManagerRef.value)
    if (canvasRef.value) layoutObserver.observe(canvasRef.value)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('resize', updateLayoutMetrics)
  window.removeEventListener('focus', syncProjectorState)
  stopProjectorPoll()
  layoutObserver?.disconnect()
  layoutObserver = null
})

const fetchEventMeta = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}`, {
      headers: getAuthHeaders()
    })
    if (res.ok) {
      const data = await res.json()
      eventPin.value = data?.pin || ''
    }
  } catch (err) {
    console.error(err)
  }
}

const fetchSlides = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/slides`, {
      headers: getAuthHeaders()
    })
    if (!res.ok) {
      error(t('screen_manager.editor.load_failed'))
      return
    }
    slides.value = await res.json()
    if (slides.value.length > 0 && !activeSlideId.value) {
      activeSlideId.value = slides.value[0].id
    }
  } catch (err) {
    console.error(err)
    error(t('screen_manager.editor.load_failed'))
  }
}

const saveSlides = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/slides`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(slides.value)
    })
    if (!res.ok) {
      error(t('screen_manager.editor.save_failed'))
      return false
    }
    success(t('screen_manager.editor.saved'))
    return true
  } catch (err) {
    console.error(err)
    error(t('screen_manager.editor.save_failed'))
    return false
  }
}

const pushToScreen = async () => {
  if (!activeSlide.value) return

  // First save to make sure changes are synced
  const saved = await saveSlides()
  if (!saved) return

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
    if (!res.ok) {
      error(t('screen_manager.editor.push_failed'))
      return
    }
    success(t('screen_manager.editor.pushed'))
  } catch (err) {
    error(t('screen_manager.editor.push_failed'))
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
    name: t('screen_manager.editor.new_custom_slide'),
    backgroundColor: '#1E1E1E',
    backgroundImage: '',
    elements: []
  }
  slides.value.push(newSlide)
  activeSlideId.value = newSlide.id
  selectedElementId.value = null
}

const deleteSlide = (id) => {
  if (confirm(t('screen_manager.editor.delete_slide_confirm'))) {
    slides.value = slides.value.filter(s => s.id !== id)
    if (activeSlideId.value === id) {
      activeSlideId.value = slides.value[0]?.id || null
      selectedElementId.value = null
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
  mobileTab.value = 'properties'
}

const selectElement = (id) => {
  selectedElementId.value = id
  mobileTab.value = 'properties'
}

const deselectElement = () => {
  selectedElementId.value = null
}

const onCanvasClick = () => {
  if (suppressCanvasClick) {
    suppressCanvasClick = false
    return
  }
  deselectElement()
}

const deleteSelectedElement = () => {
  if (activeSlide.value && selectedElementId.value) {
    activeSlide.value.elements = activeSlide.value.elements.filter(e => e.id !== selectedElementId.value)
    selectedElementId.value = null
  }
}

// Pointer Events Drag/Resize logic processing percentage coordinates
const getCanvasRect = () => {
  if (!canvasRef.value) return null
  return canvasRef.value.getBoundingClientRect()
}

const startDrag = (item, e) => {
  if (activeSlide.value?.type === 'screensaver') return
  if (e.pointerType === 'mouse' && e.button !== 0) return
  if (activePointerId.value !== null && activePointerId.value !== e.pointerId) return

  try {
    e.currentTarget?.setPointerCapture?.(e.pointerId)
  } catch {
    // Ignore if synthetic event has no active pointer
  }

  activePointerId.value = e.pointerId ?? null
  didMoveDuringDrag = false
  selectElement(item.id)
  isDragging.value = true
  isResizing.value = false
  dragItem.value = item
  dragStart.value = {
    x: e.clientX,
    y: e.clientY,
    startX: item.x,
    startY: item.y,
    startW: item.width,
    startH: item.height
  }
}

const startResize = (item, e) => {
  if (activeSlide.value?.type === 'screensaver') return
  if (e.pointerType === 'mouse' && e.button !== 0) return
  if (activePointerId.value !== null && activePointerId.value !== e.pointerId) return

  try {
    e.currentTarget?.setPointerCapture?.(e.pointerId)
  } catch {
    // Ignore if synthetic event has no active pointer
  }

  activePointerId.value = e.pointerId ?? null
  didMoveDuringDrag = false
  selectElement(item.id)
  isResizing.value = true
  isDragging.value = false
  dragItem.value = item
  dragStart.value = {
    x: e.clientX,
    y: e.clientY,
    startX: item.x,
    startY: item.y,
    startW: item.width,
    startH: item.height
  }
}

const onPointerMove = (e) => {
  if ((!isDragging.value && !isResizing.value) || !dragItem.value) return
  if (activePointerId.value !== null && e.pointerId !== undefined && e.pointerId !== activePointerId.value) return

  const rect = getCanvasRect()
  if (!rect || !rect.width || !rect.height) return

  const dx = e.clientX - dragStart.value.x
  const dy = e.clientY - dragStart.value.y
  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    didMoveDuringDrag = true
  }

  const dxPct = (dx / rect.width) * 100
  const dyPct = (dy / rect.height) * 100

  if (isDragging.value) {
    const maxX = Math.max(0, 100 - dragItem.value.width)
    const maxY = Math.max(0, 100 - dragItem.value.height)
    dragItem.value.x = clamp(round1(dragStart.value.startX + dxPct), 0, maxX)
    dragItem.value.y = clamp(round1(dragStart.value.startY + dyPct), 0, maxY)
  } else if (isResizing.value) {
    dragItem.value.width = clamp(round1(dragStart.value.startW + dxPct), 5, 100)
    dragItem.value.height = clamp(round1(dragStart.value.startH + dyPct), 5, 100)
  }
}

const onPointerUp = (e) => {
  if (activePointerId.value !== null && e?.pointerId !== undefined && e.pointerId !== activePointerId.value) return

  if (didMoveDuringDrag) {
    suppressCanvasClick = true
    setTimeout(() => {
      suppressCanvasClick = false
    }, 0)
  }

  try {
    if (e?.pointerId !== undefined && e?.currentTarget?.hasPointerCapture?.(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId)
    }
  } catch {
    // Ignore if capture was already released
  }

  isDragging.value = false
  isResizing.value = false
  dragItem.value = null
  activePointerId.value = null
  didMoveDuringDrag = false
}

const onElementKeydown = (element, e) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    deselectElement()
    liveAnnouncement.value = `${getElementLabel(element)} deselected`
    return
  }

  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    selectElement(element.id)
    liveAnnouncement.value = getElementAriaLabel(element)
    return
  }

  if (!['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) return
  if (activeSlide.value?.type === 'screensaver') return

  e.preventDefault()
  selectElement(element.id)

  const step = e.shiftKey ? 5 : 1
  const maxX = Math.max(0, 100 - element.width)
  const maxY = Math.max(0, 100 - element.height)

  if (e.key === 'ArrowLeft') {
    element.x = clamp(round1(element.x - step), 0, maxX)
  } else if (e.key === 'ArrowRight') {
    element.x = clamp(round1(element.x + step), 0, maxX)
  } else if (e.key === 'ArrowUp') {
    element.y = clamp(round1(element.y - step), 0, maxY)
  } else if (e.key === 'ArrowDown') {
    element.y = clamp(round1(element.y + step), 0, maxY)
  }

  liveAnnouncement.value = `${getElementLabel(element)}: X ${element.x}%, Y ${element.y}%`
}

const getScaledFontSize = (size) => {
  const cw = canvasWidth.value || canvasRef.value?.clientWidth
  if (!cw) return size
  // Base standard resolution 1920
  const ratio = cw / 1920
  return size * ratio
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

const stopProjectorPoll = () => {
  if (projectorPollTimer !== null) {
    clearInterval(projectorPollTimer)
    projectorPollTimer = null
  }
}

const syncProjectorState = () => {
  if (projectorWindow.value && !projectorWindow.value.closed) {
    isProjectorOpen.value = true
  } else {
    projectorWindow.value = null
    isProjectorOpen.value = false
    stopProjectorPoll()
  }
}

const startProjectorPoll = () => {
  stopProjectorPoll()
  if (typeof window !== 'undefined') {
    projectorPollTimer = window.setInterval(syncProjectorState, 500)
  }
}

const openProjector = () => {
  if (projectorWindow.value && !projectorWindow.value.closed) {
    projectorWindow.value.focus()
    isProjectorOpen.value = true
    return
  }
  const win = window.open(`/projector/${eventId}`, 'projector', 'width=1200,height=800')
  if (win && !win.closed) {
    projectorWindow.value = win
    isProjectorOpen.value = true
    startProjectorPoll()
  } else {
    projectorWindow.value = null
    isProjectorOpen.value = false
    stopProjectorPoll()
  }
}

const closeProjector = () => {
  if (projectorWindow.value && !projectorWindow.value.closed) {
    projectorWindow.value.close()
  }
  projectorWindow.value = null
  isProjectorOpen.value = false
  stopProjectorPoll()
}
</script>

<style scoped>
.screen-manager {
  container: screen-manager / inline-size;
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  background-color: #0d0d0d;
  color: #fff;
  display: flex;
  flex-direction: column;
}

.screen-manager,
.screen-manager * {
  box-sizing: border-box;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.mt-2 { margin-top: 8px; }
.mt-3 { margin-top: 12px; }
.mt-4 { margin-top: 16px; }
.mb-2 { margin-bottom: 8px; }
.mb-3 { margin-bottom: 12px; }

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 15px 30px;
  background-color: #1a1a1d;
  border-bottom: 1px solid #333;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
  min-width: 0;
}
.header-left h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 500;
  text-wrap: balance;
}
.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid #444;
  color: #fff;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
}
.header-right {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.launch-btn, .push-btn, .save-btn, .action-btn, .danger-btn, .add-slide-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: opacity 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.launch-btn { background: #3b82f6; color: white; }
.launch-btn.close-btn { background: #ef4444; }
.push-btn { background: #10b981; color: white; }
.push-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.save-btn { background: var(--ops-blue); color: white; }
.action-btn { background: #4b5563; color: white; }
.danger-btn { background: #ef4444; color: white; }
.add-slide-btn { background: #374151; color: white; width: 100%; margin-top: 10px; }
.icon-btn {
  background: transparent;
  color: #9ca3af;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 4px;
}
.icon-btn:hover { color: #f87171; }
.block-btn { display: block; width: 100%; }

:global(html[data-admin-appearance="true"] .screen-manager .launch-btn.close-btn) {
  background: var(--admin-danger-solid) !important;
  border-color: var(--admin-danger-solid) !important;
  color: #fff !important;
}

.editor-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.left-sidebar { order: 1; }
.canvas-area { order: 2; min-width: 0; }
.right-sidebar { order: 3; }
.mobile-panel-tabs { display: none; }

.sidebar {
  width: 250px;
  flex-shrink: 0;
  background: #151515;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow-y: auto;
}
.right-sidebar {
  border-left: 1px solid #333;
  border-right: none;
  padding-bottom: 76px;
}
.right-sidebar.desktop-hidden {
  display: none;
}
.sidebar h3 {
  margin-top: 0;
  font-weight: 500;
  color: #9ca3af;
  font-size: 0.9rem;
  text-transform: uppercase;
}

.slide-list { list-style: none; padding: 0; margin: 0; }
.slide-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 6px 10px 6px 12px;
  background: #222;
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: pointer;
}
.slide-select-btn {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: none;
  color: inherit;
  font: inherit;
  text-align: left;
  padding: 4px 2px;
  cursor: pointer;
  overflow-wrap: break-word;
}
.slide-list li.active .slide-item { background: #3b82f6; color: white; }

:global(html[data-admin-appearance="true"] .screen-manager .slide-list li.active .slide-item) {
  background: var(--admin-primary-solid) !important;
  color: #fff !important;
}
:global(html[data-admin-appearance="true"] .screen-manager .slide-list .slide-select-btn) {
  background: transparent !important;
  border-color: transparent !important;
  color: inherit !important;
}

.canvas-area {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #111;
  overflow: hidden;
  position: relative;
  padding: 20px;
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
  touch-action: pan-y;
}

.canvas-bg-image {
  position: absolute;
  top: 0; left: 0; width: 100%; height: 100%;
  object-fit: cover;
  z-index: 0;
  pointer-events: none;
}

.canvas-element {
  position: absolute;
  z-index: 10;
  cursor: grab;
  touch-action: none;
  -webkit-user-select: none;
  user-select: none;
}
.canvas-element:active { cursor: grabbing; }
.canvas-element:focus-visible {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
}
.canvas-element.selected { border: 2px dashed #3b82f6; }

.el-content { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }
.el-text {
  -webkit-user-select: none;
  user-select: none;
  width: 100%;
  overflow-wrap: break-word;
}
.el-image { width: 100%; height: 100%; object-fit: contain; pointer-events: none; }
.el-qr {
  background: white;
  padding: clamp(2px, 1vw, 10px);
  border-radius: 8px;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.el-qr :deep(canvas),
.el-qr :deep(svg) {
  max-width: 100%;
  max-height: 100%;
  width: auto !important;
  height: auto !important;
}

.resize-handle {
  position: absolute;
  right: -7px;
  bottom: -7px;
  width: 14px;
  height: 14px;
  background: #3b82f6;
  border: 2px solid #ffffff;
  border-radius: 50%;
  cursor: se-resize;
  touch-action: none;
  -webkit-user-select: none;
  user-select: none;
}
.resize-handle::after {
  content: "";
  position: absolute;
  inset: -14px;
}

/* Props Panel */
.props-panel h4 {
  margin: 0 0 8px;
}
.props-panel label {
  display: block;
  font-size: 0.8rem;
  color: #9ca3af;
  margin-bottom: 4px;
  margin-top: 10px;
}
.props-panel input[type="text"],
.props-panel input[type="number"],
.props-panel textarea {
  width: 100%;
  min-inline-size: 0;
  background: #222;
  border: 1px solid #444;
  color: white;
  padding: 8px;
  border-radius: 4px;
}
.props-panel input[type="range"] {
  width: 100%;
}
.input-group { display: flex; flex-direction: column; margin-bottom: 10px; }
.pos-inputs { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }

/* Modals */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.8);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
}
.modal-content {
  background: #1e1e1e;
  padding: 20px;
  border-radius: 12px;
  width: calc(100% - 24px);
  max-width: 800px;
  max-height: 88dvh;
  display: flex;
  flex-direction: column;
  overscroll-behavior: contain;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--admin-border, #333);
  flex-shrink: 0;
}
.modal-header h3 {
  margin: 0;
  font-size: 1.15rem;
}
.modal-header .close-btn {
  min-block-size: 44px;
  min-inline-size: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: #9ca3af;
  border-radius: 8px;
  cursor: pointer;
}
.modal-header .close-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #ffffff;
}
.modal-body {
  overflow-y: auto;
  flex: 1;
  overscroll-behavior: contain;
  padding-top: 14px;
}
.modal-empty {
  text-align: center;
  padding: 20px 8px;
}
.screensavers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 130px), 1fr));
  gap: 15px;
  margin-top: 4px;
}
.screensaver-option {
  cursor: pointer;
  background: transparent;
  border: none;
  padding: 0;
  width: 100%;
  text-align: inherit;
  color: inherit;
  border-radius: 8px;
}
.screensaver-option img {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  border-radius: 6px;
}

:global(html[data-admin-appearance="true"] .screen-manager .screensaver-option) {
  background: transparent !important;
  border-color: transparent !important;
  padding: 0;
}

/* Templates */
.template-box {
  background: #333;
  aspect-ratio: 16 / 9;
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

/* Mobile Compact Layout */
@container screen-manager (inline-size <= 900px) {
  .header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: max(12px, env(safe-area-inset-top, 0px)) max(14px, env(safe-area-inset-right, 0px)) 12px max(14px, env(safe-area-inset-left, 0px));
  }
  .header-left {
    gap: 12px;
    flex-wrap: wrap;
  }
  .header-left h1 {
    font-size: clamp(1.1rem, 4vw, 1.45rem);
    text-wrap: balance;
  }
  .back-btn {
    min-block-size: 44px;
  }
  .header-right {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    width: 100%;
  }
  .header-right button {
    flex: 1 1 120px;
    min-block-size: 44px;
    justify-content: center;
    text-wrap: balance;
    line-height: 1.2;
  }

  .editor-layout {
    flex-direction: column;
    overflow: visible;
    flex: 1 0 auto;
  }
  .canvas-area {
    order: 1;
    flex: none;
    padding: 12px;
    overflow: visible;
  }
  .canvas-container {
    width: 100%;
  }

  .mobile-panel-tabs {
    order: 2;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    padding: 10px max(16px, env(safe-area-inset-right, 0px)) 10px max(16px, env(safe-area-inset-left, 0px));
    background: #151515;
    border-top: 1px solid var(--admin-border, #333);
  }
  .mobile-tab-btn {
    min-block-size: 44px;
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid #444;
    background: #222;
    color: #d1d5db;
    font-size: 0.95rem;
    font-weight: 600;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    text-align: center;
  }
  .mobile-tab-btn[aria-selected="true"] {
    background: #3b82f6;
    border-color: #3b82f6;
    color: #ffffff;
  }

  .left-sidebar { order: 3; }
  .right-sidebar { order: 4; }
  .sidebar {
    width: 100%;
    border-left: none;
    border-right: none;
    border-bottom: none;
    border-top: 1px solid var(--admin-border, #333);
    padding: 16px 16px calc(84px + env(safe-area-inset-bottom, 0px));
    overflow-y: visible;
  }
  .sidebar.mobile-panel-hidden,
  .right-sidebar.desktop-hidden.mobile-panel-hidden {
    display: none;
  }
  .right-sidebar.desktop-hidden {
    display: flex;
  }

  .props-panel input[type="text"],
  .props-panel input[type="number"],
  .props-panel textarea {
    font-size: 16px;
  }
}

:global(html[data-admin-appearance="true"] .screen-manager .mobile-panel-tabs) {
  background: var(--admin-surface) !important;
  border-top-color: var(--admin-border) !important;
}
:global(html[data-admin-appearance="true"] .screen-manager .mobile-tab-btn) {
  background: var(--admin-surface-subtle) !important;
  border-color: var(--admin-border-strong) !important;
  color: var(--admin-text-muted) !important;
}
:global(html[data-admin-appearance="true"] .screen-manager .mobile-tab-btn[aria-selected="true"]) {
  background: var(--admin-primary-solid) !important;
  border-color: var(--admin-primary-solid) !important;
  color: #fff !important;
}

@media (pointer: coarse) {
  .screen-manager button,
  .screen-manager input,
  .screen-manager textarea,
  .screen-manager .slide-item {
    min-block-size: 44px;
  }
  .screen-manager .delete-slide,
  .screen-manager .modal-header .close-btn,
  .screen-manager input[type="color"] {
    min-inline-size: 44px;
  }
}
</style>
