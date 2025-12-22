<script setup lang="ts">
import { ref, watch, computed } from "vue"
import { toast } from "vue-sonner"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { useLinksStore, type Link } from "@/stores/links"

const props = defineProps<{
  open: boolean
  linkToEdit?: Link | null
}>()

const emit = defineEmits<{
  (e: "update:open", value: boolean): void
  (e: "saved"): void
}>()

const store = useLinksStore()
const isEditMode = computed(() => !!props.linkToEdit)

const form = ref({
  slug: "",
  url: "",
  is_active: true,
})

const isLoading = ref(false)
const errorMessage = ref("")

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      if (props.linkToEdit) {
        form.value = {
          slug: props.linkToEdit.slug,
          url: props.linkToEdit.url,
          is_active: props.linkToEdit.is_active,
        }
      } else {
        form.value = {
          slug: "",
          url: "",
          is_active: true,
        }
      }
      errorMessage.value = ""
    }
  },
)

const handleSubmit = async () => {
  errorMessage.value = ""

  const slug = form.value.slug.trim()
  const url = form.value.url.trim()

  if (!slug) {
    errorMessage.value = "Slug is required"
    return
  }

  if (slug.length > 32) {
    errorMessage.value = "Slug must be 32 characters or less"
    return
  }

  const slugRegex = /^[a-zA-Z0-9-_]+$/
  if (!slugRegex.test(slug)) {
    errorMessage.value = "Slug must contain only letters, numbers, hyphens, and underscores"
    return
  }

  if (!url) {
    errorMessage.value = "Target URL is required"
    return
  }

  if (url.length > 2048) {
    errorMessage.value = "Target URL must be 2048 characters or less"
    return
  }

  try {
    const urlObj = new URL(url)
    if (urlObj.protocol !== "http:" && urlObj.protocol !== "https:") {
      errorMessage.value = "Target URL must start with http:// or https://"
      return
    }
  } catch (e) {
    errorMessage.value = "Target URL must be a valid URL"
    return
  }

  isLoading.value = true
  try {
    if (isEditMode.value) {
      await store.updateLink(props.linkToEdit!.slug, {
        url: url,
        is_active: form.value.is_active,
      })
    } else {
      await store.createLink(slug, url)
    }
    toast.success(isEditMode.value ? "Link updated successfully" : "Link created successfully")
    emit("saved")
    emit("update:open", false)
  } catch (e: any) {
    const msg = e.response?.data?.error || e.message || "An error occurred"
    errorMessage.value = msg
    toast.error(msg)
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ isEditMode ? "Edit Link" : "Create New Link" }}</DialogTitle>
        <DialogDescription>
          {{ isEditMode ? "Make changes to your link here." : "Add a new short link." }}
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="slug" class="text-right">Slug</Label>
          <Input id="slug" v-model="form.slug" class="col-span-3" :disabled="isEditMode" placeholder="e.g. twitter" />
        </div>
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="url" class="text-right">Target URL</Label>
          <Input id="url" v-model="form.url" class="col-span-3" placeholder="https://example.com" />
        </div>

        <div v-if="isEditMode" class="grid grid-cols-4 items-center gap-4">
          <Label for="active" class="text-right">Active</Label>
          <div class="col-span-3 flex items-center space-x-2">
            <Switch id="active" v-model="form.is_active" />
            <span class="text-sm text-muted-foreground">{{ form.is_active ? "Enabled" : "Disabled" }}</span>
          </div>
        </div>

        <div v-if="errorMessage" class="text-red-500 text-sm font-medium text-center">
          {{ errorMessage }}
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="$emit('update:open', false)">Cancel</Button>
        <Button type="submit" @click="handleSubmit" :disabled="isLoading">
          {{ isLoading ? "Saving..." : "Save changes" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
