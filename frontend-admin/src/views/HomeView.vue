<script setup lang="ts">
import { onMounted, ref } from "vue"
import { storeToRefs } from "pinia"
import { useLinksStore, type Link } from "@/stores/links"
import LinkDialog from "@/components/LinkDialog.vue"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { MoreHorizontal, Plus, Copy, Pencil, Trash2, ExternalLink } from "lucide-vue-next"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { toast } from "vue-sonner"

const linksStore = useLinksStore()
const { links, loading, error } = storeToRefs(linksStore)

const isDialogOpen = ref(false)
const selectedLink = ref<Link | null>(null)
const isDeleteDialogOpen = ref(false)
const linkToDelete = ref<Link | null>(null)

const BASE_SHORT_URL = "https://t.gravitycat.tw"

onMounted(() => {
  linksStore.fetchLinks()
})

const openCreateDialog = () => {
  selectedLink.value = null
  isDialogOpen.value = true
}

const openEditDialog = (link: Link) => {
  selectedLink.value = link
  isDialogOpen.value = true
}

const copyToClipboard = async (slug: string) => {
  const url = `${BASE_SHORT_URL}/${slug}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success("Shorten link copied!")
  } catch (err) {
    console.error("Failed to copy", err)
    toast.error("Failed to copy link")
  }
}

const deleteLink = (link: Link) => {
  linkToDelete.value = link
  isDeleteDialogOpen.value = true
}

const confirmDelete = async () => {
  if (linkToDelete.value) {
    try {
      await linksStore.deleteLink(linkToDelete.value.slug)
      toast.success("Link deleted successfully")
      isDeleteDialogOpen.value = false
      linkToDelete.value = null
    } catch (error) {
      toast.error("Failed to delete link")
    }
  }
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}
</script>

<template>
  <div class="w-full max-w-7xl mx-auto py-10 px-6">
    <!-- Header -->
    <div class="flex flex-row items-center justify-between mb-8 gap-4">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">Linkhub</h1>
        <p class="text-muted-foreground">Manage your shortened links.</p>
      </div>
      <Button @click="openCreateDialog"> <Plus class="mr-2 h-4 w-4" /> Create </Button>
    </div>

    <!-- Error State -->
    <div v-if="error" class="bg-destructive/15 text-destructive p-4 rounded-md mb-6">Error: {{ error }}</div>

    <!-- Data Table -->
    <div class="rounded-md border bg-card text-card-foreground shadow-sm overflow-hidden overflow-x-auto">
      <Table class="table-fixed">
        <TableHeader>
          <TableRow>
            <TableHead class="w-[150px]">Slug</TableHead>
            <TableHead class="hidden md:table-cell">Target URL</TableHead>
            <TableHead class="w-[100px]">Status</TableHead>
            <TableHead class="hidden md:table-cell w-[150px]">Created</TableHead>
            <TableHead class="w-[80px] text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="loading && links.length === 0">
            <TableCell colspan="5" class="h-24 text-center">Loading...</TableCell>
          </TableRow>

          <TableRow v-else-if="links.length === 0">
            <TableCell colspan="5" class="h-24 text-center text-muted-foreground">No links found.</TableCell>
          </TableRow>

          <TableRow v-for="link in links" :key="link.id" @click="openEditDialog(link)" class="cursor-pointer">
            <TableCell class="font-medium">
              <a
                :href="`${BASE_SHORT_URL}/${link.slug}`"
                target="_blank"
                class="hover:underline flex items-center gap-1 max-w-full pointer-events-none md:pointer-events-auto"
                :title="link.slug"
                @click.stop
              >
                <span class="truncate">/{{ link.slug }}</span>
                <ExternalLink class="h-3 w-3 opacity-50 shrink-0 hidden md:block" />
              </a>
            </TableCell>
            <TableCell class="hidden md:table-cell truncate" :title="link.url">
              {{ link.url }}
            </TableCell>
            <TableCell>
              <Badge :variant="link.is_active ? 'default' : 'secondary'">
                {{ link.is_active ? "Active" : "Inactive" }}
              </Badge>
            </TableCell>
            <TableCell class="hidden md:table-cell text-muted-foreground">
              {{ formatDate(link.created_at) }}
            </TableCell>
            <TableCell class="text-right">
              <div class="flex items-center justify-end gap-1">
                <Button variant="ghost" size="icon" class="h-8 w-8" @click.stop="copyToClipboard(link.slug)">
                  <Copy class="h-4 w-4" />
                  <span class="sr-only">Copy short URL</span>
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button variant="ghost" class="h-8 w-8 p-0" @click.stop>
                      <span class="sr-only">Open menu</span>
                      <MoreHorizontal class="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem @click="openEditDialog(link)"> <Pencil class="mr-2 h-4 w-4" /> Edit </DropdownMenuItem>
                    <DropdownMenuItem @click="deleteLink(link)" class="text-destructive focus:text-destructive">
                      <Trash2 class="mr-2 h-4 w-4" /> Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <LinkDialog v-model:open="isDialogOpen" :link-to-edit="selectedLink" @saved="linksStore.fetchLinks()" />

    <AlertDialog :open="isDeleteDialogOpen" @update:open="isDeleteDialogOpen = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete the link
            <span class="font-bold">/{{ linkToDelete?.slug }}</span
            >.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel @click="linkToDelete = null">Cancel</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-destructive-foreground hover:bg-destructive/90" @click="confirmDelete">
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
