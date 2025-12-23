<script setup lang="ts">
import { Toaster } from "@/components/ui/sonner"
import { onMounted } from "vue"
import "vue-sonner/style.css"

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
