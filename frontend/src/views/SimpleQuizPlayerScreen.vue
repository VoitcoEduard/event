<template>
  <div class="simple-quiz-player">
    <h1>Simple Quiz — Player</h1>
    <p>{{ eventId }} / {{ moduleId }}</p>
    <p>Full player UI lands in the next commit.</p>
  </div>
</template>
<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const eventId = route.params.eventId
const moduleId = route.params.moduleId
</script>
<style scoped>
.simple-quiz-player { padding: 24px; color: #fff; background: #46178F; min-height: 100vh; }
</style>
