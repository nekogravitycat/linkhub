<script setup lang="ts">
import "vue-sonner/style.css"
import { onMounted } from "vue"
import { Toaster } from "@/components/ui/sonner"

onMounted(() => {
  document.documentElement.classList.add("dark")
})
</script>

<template>
  <div class="min-h-screen bg-background font-sans antialiased text-foreground">
    <RouterView />
    <Toaster />
  </div>
</template>
