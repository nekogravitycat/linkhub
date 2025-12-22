<script setup lang="ts">
import { onMounted } from "vue"

onMounted(() => {
  document.documentElement.classList.add("dark")
})
</script>

<template>
  <div class="min-h-screen bg-background font-sans antialiased text-foreground">
    <RouterView />
  </div>
</template>
