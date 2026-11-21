<template>
  <div class="admin-layout">
    <header class="admin-topbar">
      <div class="admin-topbar-inner">
        <div class="admin-topbar-start">
          <RouterLink to="/admin/events" class="admin-brand">
            <AppIcon name="sliders" :size="18" aria-hidden="true" />
            <span class="admin-brand-text">{{ $t('admin_shell.brand') }}</span>
          </RouterLink>
          <button
            type="button"
            class="admin-menu-toggle"
            :aria-expanded="mobileMenuOpen ? 'true' : 'false'"
            aria-controls="admin-shell-nav-panel"
            :aria-label="mobileMenuOpen ? $t('admin_shell.menu_close') : $t('admin_shell.menu_open')"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <AppIcon :name="mobileMenuOpen ? 'close' : 'menu'" :size="20" aria-hidden="true" />
          </button>
        </div>

        <div
          id="admin-shell-nav-panel"
          class="admin-topbar-collapse"
          :class="{ 'is-open': mobileMenuOpen }"
        >
          <nav class="admin-primary-nav" :aria-label="$t('admin_shell.nav_label')">
            <RouterLink
              to="/admin/events"
              class="admin-nav-link"
              :class="{ active: isEventsActive }"
              :aria-current="isEventsActive ? 'page' : undefined"
            >
              {{ $t('admin_shell.events') }}
            </RouterLink>
            <RouterLink
              to="/admin/library/quizzes"
              class="admin-nav-link"
              :class="{ active: isLibraryActive }"
              :aria-current="isLibraryActive ? 'page' : undefined"
            >
              {{ $t('admin_shell.library') }}
            </RouterLink>
            <RouterLink
              v-if="session.user?.role === 'owner'"
              to="/admin/users"
              class="admin-nav-link"
              :class="{ active: isUsersActive }"
              :aria-current="isUsersActive ? 'page' : undefined"
            >
              {{ $t('admin_shell.users') }}
            </RouterLink>
          </nav>

          <div class="admin-utility-cluster">
            <select
              v-model="locale"
              class="admin-lang-select"
              :aria-label="$t('library.language_label')"
              @change="handleLanguageChange"
            >
              <option value="en">English</option>
              <option value="ru">Русский</option>
              <option value="ro">Română</option>
            </select>

            <AdminThemeToggle variant="inline" />

            <button
              type="button"
              class="admin-signout-btn"
              :disabled="signingOut"
              @click="handleSignOut"
            >
              <AppIcon name="logOut" :size="16" aria-hidden="true" />
              <span>{{ $t('auth.signOut') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="hasSecondaryBar" class="admin-subbar">
        <div class="admin-subbar-inner">
          <nav
            v-if="breadcrumbs.length"
            class="admin-breadcrumbs"
            :aria-label="$t('admin_shell.breadcrumb_label')"
          >
            <ol class="admin-breadcrumb-list">
              <li
                v-for="(crumb, index) in breadcrumbs"
                :key="`${crumb.label}-${index}`"
                class="admin-breadcrumb-item"
              >
                <AppIcon
                  v-if="index > 0"
                  name="chevronRight"
                  :size="14"
                  class="admin-breadcrumb-sep"
                  aria-hidden="true"
                />
                <RouterLink
                  v-if="crumb.to"
                  :to="crumb.to"
                  class="admin-breadcrumb-link"
                >
                  {{ crumb.label }}
                </RouterLink>
                <span
                  v-else
                  class="admin-breadcrumb-current"
                  :aria-current="index === breadcrumbs.length - 1 ? 'page' : undefined"
                >
                  {{ crumb.label }}
                </span>
              </li>
            </ol>
          </nav>

          <div v-if="$slots.actions" class="admin-subbar-actions">
            <slot name="actions" />
          </div>
        </div>
      </div>
    </header>

    <div class="admin-layout-content">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, useSlots, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'
import AdminThemeToggle from './AdminThemeToggle.vue'
import { session, signOut } from '../session.js'
import { safeLocalSet } from '../utils/storage.js'

const props = defineProps({
  breadcrumbs: {
    type: Array,
    default: () => []
  }
})

const slots = useSlots()
const route = useRoute()
const router = useRouter()
const { locale } = useI18n()

const mobileMenuOpen = ref(false)
const signingOut = ref(false)

const currentPath = computed(() => route?.path || '')

const isEventsActive = computed(() => {
  const path = currentPath.value
  return path === '/admin' || path.startsWith('/admin/events') || path.startsWith('/admin/event/')
})

const isLibraryActive = computed(() => {
  const path = currentPath.value
  return path.startsWith('/admin/library') || path.startsWith('/quiz/library') || path.startsWith('/data/library')
})

const isUsersActive = computed(() => currentPath.value.startsWith('/admin/users'))

const hasSecondaryBar = computed(() => props.breadcrumbs.length > 0 || Boolean(slots.actions))

watch(
  () => route?.fullPath,
  () => {
    mobileMenuOpen.value = false
  }
)

function handleLanguageChange() {
  safeLocalSet('userLanguage', locale.value)
}

async function handleSignOut() {
  if (signingOut.value) return
  signingOut.value = true
  try {
    await signOut()
    await router.push('/login')
  } catch {
    // Preserve current screen if signOut fails.
  } finally {
    signingOut.value = false
  }
}
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--admin-bg, #f5f6f8);
  color: var(--admin-text, #101828);
}

.admin-topbar {
  position: sticky;
  top: 0;
  z-index: 400;
  background: var(--admin-surface, #ffffff);
  border-bottom: 1px solid var(--admin-border, #dde1e7);
  padding-top: env(safe-area-inset-top, 0px);
}

.admin-topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 60px;
  padding: 8px clamp(16px, 3vw, 32px);
  box-sizing: border-box;
}

.admin-topbar-start {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.admin-brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 4px 8px 4px 2px;
  color: var(--admin-text, #101828);
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  text-decoration: none;
  border-radius: 8px;
}

.admin-brand:hover {
  color: var(--admin-primary, #0969da);
}

.admin-menu-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  min-height: 44px;
  padding: 0;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  cursor: pointer;
}

.admin-topbar-collapse {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex: 1;
  min-width: 0;
}

.admin-primary-nav {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.admin-nav-link {
  display: inline-flex;
  align-items: center;
  min-height: 38px;
  padding: 6px 12px;
  border-radius: 8px;
  color: var(--admin-text-muted, #475467);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
  transition: background-color 0.15s, color 0.15s;
}

.admin-nav-link:hover {
  background: var(--admin-surface-hover, #eef2f7);
  color: var(--admin-text, #101828);
}

.admin-nav-link.active {
  background: var(--admin-primary-soft, #e8f2ff);
  color: var(--admin-primary, #0969da);
}

.admin-utility-cluster {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.admin-lang-select {
  min-height: 38px;
  padding: 6px 10px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.admin-signout-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 38px;
  padding: 6px 12px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 0.15s;
}

.admin-signout-btn:hover:not(:disabled) {
  background: var(--admin-surface-hover, #eef2f7);
}

.admin-signout-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.admin-subbar {
  background: var(--admin-surface-subtle, #f8f9fb);
  border-top: 1px solid var(--admin-border, #dde1e7);
}

.admin-subbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px 16px;
  min-height: 46px;
  padding: 8px clamp(16px, 3vw, 32px);
  box-sizing: border-box;
}

.admin-breadcrumbs {
  min-width: 0;
}

.admin-breadcrumb-list {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  list-style: none;
  margin: 0;
  padding: 0;
}

.admin-breadcrumb-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 13px;
}

.admin-breadcrumb-sep {
  color: var(--admin-text-dim, #667085);
}

.admin-breadcrumb-link {
  color: var(--admin-primary, #0969da);
  font-weight: 600;
  text-decoration: none;
  border-radius: 6px;
  padding: 2px 4px;
}

.admin-breadcrumb-link:hover {
  background: var(--admin-surface-hover, #eef2f7);
  text-decoration: underline;
}

.admin-breadcrumb-current {
  color: var(--admin-text, #101828);
  font-weight: 600;
  overflow-wrap: anywhere;
}

.admin-subbar-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-left: auto;
}

.admin-layout-content {
  flex: 1;
  min-width: 0;
}

@media (max-width: 768px) {
  .admin-topbar-inner {
    flex-direction: column;
    align-items: stretch;
    gap: 0;
    padding: 6px max(12px, env(safe-area-inset-right, 0px)) 6px max(12px, env(safe-area-inset-left, 0px));
  }

  .admin-topbar-start {
    width: 100%;
    min-height: 48px;
  }

  .admin-brand {
    min-height: 44px;
  }

  .admin-menu-toggle {
    display: inline-flex;
  }

  .admin-topbar-collapse {
    display: none;
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: 10px 0 8px;
    border-top: 1px solid var(--admin-border, #dde1e7);
  }

  .admin-topbar-collapse.is-open {
    display: flex;
  }

  .admin-primary-nav {
    flex-direction: column;
    align-items: stretch;
    gap: 4px;
  }

  .admin-nav-link {
    min-height: 44px;
    padding: 10px 12px;
  }

  .admin-utility-cluster {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    padding-top: 8px;
    border-top: 1px solid var(--admin-border, #dde1e7);
  }

  .admin-lang-select,
  .admin-signout-btn {
    min-height: 44px;
    width: 100%;
  }

  .admin-utility-cluster :deep(.admin-theme-control) {
    width: 100%;
  }

  .admin-utility-cluster :deep(.admin-theme-option) {
    flex: 1;
    min-height: 44px;
  }

  .admin-subbar-inner {
    padding: 8px max(12px, env(safe-area-inset-right, 0px)) 8px max(12px, env(safe-area-inset-left, 0px));
  }

  .admin-breadcrumb-link {
    min-height: 44px;
    display: inline-flex;
    align-items: center;
  }

  .admin-subbar-actions {
    width: 100%;
    margin-left: 0;
  }
}

@media (pointer: coarse) {
  .admin-nav-link,
  .admin-lang-select,
  .admin-signout-btn,
  .admin-brand,
  .admin-breadcrumb-link {
    min-height: 44px;
  }

  .admin-breadcrumb-link {
    display: inline-flex;
    align-items: center;
  }

  .admin-utility-cluster :deep(.admin-theme-option) {
    min-height: 44px;
  }
}
</style>
