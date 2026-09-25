<template>
  <div class="simple-quiz-judge">
    <h1>Judge workspace</h1>
    <p>{{ eventId }} / {{ moduleId }}</p>
    <p>Full judge UI lands in the next commit.</p>
  </div>
</template>
<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
</script>
<style scoped>
.simple-quiz-judge { padding: 24px; color: #fff; background: #1C1C1E; min-height: 100vh; }
</style>
