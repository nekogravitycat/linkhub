<script setup lang="ts">
import { Toaster } from "@/components/ui/sonner"
import { onMounted, onUnmounted } from "vue"
import "vue-sonner/style.css"

const updateTheme = (e: MediaQueryListEvent | MediaQueryList) => {
  if (e.matches) {
    document.documentElement.classList.add("dark")
  } else {
    document.documentElement.classList.remove("dark")
  }
}

onMounted(() => {
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)")
  updateTheme(mediaQuery)
  mediaQuery.addEventListener("change", updateTheme)
})

onUnmounted(() => {
  window.matchMedia("(prefers-color-scheme: dark)").removeEventListener("change", updateTheme)
})
</script>

<template>
  <div class="min-h-dvh bg-background font-sans antialiased text-foreground">
    <RouterView />
    <Toaster />
  </div>
</template>
