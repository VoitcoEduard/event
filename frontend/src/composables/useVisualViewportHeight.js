import { ref, onMounted, onUnmounted, computed } from 'vue'

/**
 * Tracks `window.visualViewport.height` so fixed/absolutely positioned
 * elements can clear the virtual keyboard on mobile devices.
 *
 * Mobile-only: on desktop viewports (width >= 768) the height string is
 * '100vh'-equivalent (undefined → consumers fall back to normal layout).
 *
 * Returns { viewportHeight: ComputedRef<string|undefined>, isMobileViewport }
 * where viewportHeight is a CSS length like '1234px'.
 */
export function useVisualViewportHeight() {
  const viewportHeight = ref(null)
  const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1024)

  const isMobileViewport = computed(() => windowWidth.value < 768)

  const update = () => {
    if (window.visualViewport) {
      viewportHeight.value = `${window.visualViewport.height}px`
    }
    windowWidth.value = window.innerWidth
  }

  const onResize = () => update()
  const onScroll = () => update()

  onMounted(() => {
    if (!window.visualViewport) return
    update()
    window.visualViewport.addEventListener('resize', onResize)
    window.visualViewport.addEventListener('scroll', onScroll)
  })

  onUnmounted(() => {
    if (!window.visualViewport) return
    window.visualViewport.removeEventListener('resize', onResize)
    window.visualViewport.removeEventListener('scroll', onScroll)
  })

  return { viewportHeight, isMobileViewport }
}
