<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <main class="stub-editor-page">
      <div class="stub-card">
        <h1>{{ $t('host.placeholder_template_editor') }}</h1>
        <p class="stub-meta">{{ tab }}{{ templateId ? ` / ${templateId}` : '' }}</p>
        <RouterLink :to="`/admin/library/${tab}`" class="stub-back-link">
          <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
          <span>{{ $t('library.back_btn') }}</span>
        </RouterLink>
      </div>
    </main>
  </AdminLayout>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'

const route = useRoute()
const { t } = useI18n()

const tab = computed(() => String(route.params.tab || 'quizzes'))
const templateId = computed(() => String(route.params.templateId || ''))

const breadcrumbs = computed(() => [
  { label: t('admin_shell.library'), to: `/admin/library/${tab.value}` },
  { label: t('host.placeholder_template_editor') }
])
</script>

<style scoped>
.stub-editor-page {
  max-width: 720px;
  margin: 32px auto;
  padding: 0 20px;
}

.stub-card {
  background: var(--admin-surface, #ffffff);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: 12px;
  padding: 24px;
}

.stub-card h1 {
  margin: 0 0 8px;
  font-size: 1.25rem;
}

.stub-meta {
  margin: 0 0 16px;
  color: var(--admin-text-muted, #475467);
}

.stub-back-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--admin-primary, #0969da);
  font-weight: 600;
  text-decoration: none;
}
</style>
