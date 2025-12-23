<script setup lang="ts">
import LinkDialog from "@/components/LinkDialog.vue"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { type Link, useLinksStore } from "@/stores/links"
import { useDebounceFn } from "@vueuse/core"
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Copy,
  ExternalLink,
  Filter,
  MoreHorizontal,
  Pencil,
  Plus,
  Search,
  Trash2,
} from "lucide-vue-next"
import { storeToRefs } from "pinia"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"

const linksStore = useLinksStore()
const { links, total, loading: tableLoading, error } = storeToRefs(linksStore)

const isDialogOpen = ref(false)
const selectedLink = ref<Link | null>(null)
const isDeleteDialogOpen = ref(false)
const linkToDelete = ref<Link | null>(null)

const BASE_SHORT_URL = import.meta.env.VITE_SHORT_BASE_URL || "https://example.com"

const sortBy = ref<"created_at" | "updated_at" | "slug">("created_at")
const sortOrder = ref<"asc" | "desc">("desc")
const page = ref(1)
const pageSize = ref(10)
const keyword = ref("")
const filterStatus = ref<"all" | "active" | "inactive">("all")
const isLoading = ref(false)

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))
const startItem = computed(() => (total.value === 0 ? 0 : (page.value - 1) * pageSize.value + 1))
const endItem = computed(() => Math.min(page.value * pageSize.value, total.value))

const handlePageChange = (newPage: number) => {
  if (newPage < 1 || newPage > totalPages.value || isLoading.value) return
  page.value = newPage
  fetchData()
}

const handlePageSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  fetchData()
}

const handleJumpToPage = (event: Event) => {
  const target = event.target as HTMLInputElement
  let newPage = parseInt(target.value)
  if (isNaN(newPage)) newPage = 1
  if (newPage < 1) newPage = 1
  if (newPage > totalPages.value) newPage = totalPages.value

  // Force update input value if auto-corrected
  target.value = newPage.toString()

  if (newPage !== page.value) {
    handlePageChange(newPage)
  }
}

const handleSearch = useDebounceFn(async () => {
  if (isLoading.value) return

  const term = keyword.value.trim()

  // Validation
  if (term.length > 0 && term.length < 3) {
    toast.error("Search term must be at least 3 characters.")
    return
  }
  if (term.length > 100) {
    toast.error("Search term is too long (max 100 characters).")
    return
  }

  // Lock Cycle
  isLoading.value = true
  try {
    page.value = 1
    await fetchData()
  } catch (err) {
    console.error("Search failed", err)
    toast.error("Search failed. Please try again.")
  } finally {
    isLoading.value = false
  }
}, 300)

const handleFilterChange = (status: "all" | "active" | "inactive") => {
  filterStatus.value = status
  page.value = 1
  fetchData()
}

const fetchData = async () => {
  await linksStore.fetchLinks({
    page: page.value,
    page_size: pageSize.value,
    sort_by: sortBy.value,
    sort_order: sortOrder.value,
    keyword: keyword.value || undefined,
    is_active: filterStatus.value === "all" ? undefined : filterStatus.value === "active",
  })
}

const handleSort = (field: "created_at" | "updated_at" | "slug") => {
  if (sortBy.value === field) {
    if (sortOrder.value === "desc") {
      sortOrder.value = "asc"
    } else {
      // Third click: Reset to default sorting (created_at DESC)
      sortBy.value = "created_at"
      sortOrder.value = "desc"
    }
  } else {
    sortBy.value = field
    sortOrder.value = "desc"
  }
  page.value = 1 // strict reset
  fetchData()
}

onMounted(() => {
  fetchData()
})

const openCreateDialog = () => {
  // delay 5ms to prevent keyboard showing up on iOS devices
  setTimeout(() => {
    selectedLink.value = null
    isDialogOpen.value = true
  }, 5)
}

const openEditDialog = (link: Link) => {
  // delay 5ms to prevent keyboard showing up on iOS devices
  setTimeout(() => {
    selectedLink.value = link
    isDialogOpen.value = true
  }, 5)
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
      await fetchData()
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

    <!-- Search and Filter Toolbar -->
    <div class="flex flex-col sm:flex-row gap-4 mb-6">
      <div class="relative flex-1">
        <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          v-model="keyword"
          type="search"
          enterkeyhint="search"
          placeholder="Search by slug or URL..."
          class="pl-8"
          :disabled="isLoading"
          @keydown.enter="handleSearch"
        />
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="outline" class="w-full sm:w-[150px] justify-start">
            <Filter class="mr-2 h-4 w-4" />
            {{ filterStatus === "all" ? "All Status" : filterStatus === "active" ? "Active" : "Inactive" }}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem @click="handleFilterChange('all')">All Status</DropdownMenuItem>
          <DropdownMenuItem @click="handleFilterChange('active')">Active</DropdownMenuItem>
          <DropdownMenuItem @click="handleFilterChange('inactive')">Inactive</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Data Table -->
    <div class="rounded-md border bg-card text-card-foreground shadow-sm overflow-hidden overflow-x-auto">
      <Table class="table-fixed relative transition-opacity duration-300" :class="{ 'opacity-50 pointer-events-none': tableLoading }">
        <TableHeader>
          <TableRow>
            <TableHead class="sm:w-[150px] cursor-pointer hover:bg-muted/50 transition-colors select-none" @click="handleSort('slug')">
              <div class="flex items-center gap-2">
                Slug
                <component
                  :is="sortBy === 'slug' ? (sortOrder === 'asc' ? ArrowUp : ArrowDown) : ArrowUpDown"
                  class="h-4 w-4"
                  :class="sortBy === 'slug' ? 'text-primary' : 'text-muted-foreground/30'"
                />
              </div>
            </TableHead>
            <TableHead class="hidden sm:table-cell">Target URL</TableHead>
            <TableHead class="w-[100px]">Status</TableHead>
            <TableHead
              class="hidden md:table-cell w-[150px] cursor-pointer hover:bg-muted/50 transition-colors select-none"
              @click="handleSort('created_at')"
            >
              <div class="flex items-center gap-2">
                Created
                <component
                  :is="sortBy === 'created_at' ? (sortOrder === 'asc' ? ArrowUp : ArrowDown) : ArrowUpDown"
                  class="h-4 w-4"
                  :class="sortBy === 'created_at' ? 'text-primary' : 'text-muted-foreground/30'"
                />
              </div>
            </TableHead>
            <TableHead
              class="hidden lg:table-cell w-[150px] cursor-pointer hover:bg-muted/50 transition-colors select-none"
              @click="handleSort('updated_at')"
            >
              <div class="flex items-center gap-2">
                Updated
                <component
                  :is="sortBy === 'updated_at' ? (sortOrder === 'asc' ? ArrowUp : ArrowDown) : ArrowUpDown"
                  class="h-4 w-4"
                  :class="sortBy === 'updated_at' ? 'text-primary' : 'text-muted-foreground/30'"
                />
              </div>
            </TableHead>
            <TableHead class="w-[80px] text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="tableLoading && links.length === 0">
            <TableCell colspan="6" class="h-24 text-center">Loading...</TableCell>
          </TableRow>

          <TableRow v-else-if="!links || links.length === 0">
            <TableCell colspan="6" class="h-24 text-center text-muted-foreground">No links found.</TableCell>
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
                <ExternalLink class="h-3 w-3 opacity-50 shrink-0 hidden sm:block" />
              </a>
            </TableCell>
            <TableCell class="hidden sm:table-cell truncate" :title="link.url">
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
            <TableCell class="hidden lg:table-cell text-muted-foreground">
              {{ formatDate(link.updated_at) }}
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

    <!-- Pagination -->
    <div class="flex items-center justify-between px-2 py-4">
      <div class="flex flex-1 items-center gap-4 text-sm text-muted-foreground">
        <div class="flex items-center gap-2">
          <span>Rows per page</span>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="outline" size="sm" class="h-8 gap-1">
                {{ pageSize }}
                <ChevronDown class="h-3 w-3 opacity-50" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem v-for="size in [10, 20, 50, 100]" :key="size" @click="handlePageSizeChange(size)">
                {{ size }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        <div class="hidden sm:block">Showing {{ startItem }} to {{ endItem }} of {{ total }} entries</div>
      </div>
      <div class="flex items-center space-x-2">
        <Button variant="outline" class="hidden h-8 w-8 p-0 lg:flex" :disabled="page === 1 || isLoading" @click="handlePageChange(1)">
          <span class="sr-only">Go to first page</span>
          <ChevronsLeft class="h-4 w-4" />
        </Button>
        <Button variant="outline" class="h-8 w-8 p-0" :disabled="page === 1 || isLoading" @click="handlePageChange(page - 1)">
          <span class="sr-only">Go to previous page</span>
          <ChevronLeft class="h-4 w-4" />
        </Button>
        <div class="flex items-center justify-center text-sm font-medium">
          <span class="mr-2">Page</span>
          <Input
            :model-value="page"
            class="h-8 w-12 text-center p-0"
            @change="handleJumpToPage"
            @keydown.enter="handleJumpToPage"
            :disabled="isLoading"
          />
          <span class="ml-2">of {{ totalPages }}</span>
        </div>
        <Button variant="outline" class="h-8 w-8 p-0" :disabled="page >= totalPages || isLoading" @click="handlePageChange(page + 1)">
          <span class="sr-only">Go to next page</span>
          <ChevronRight class="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          class="hidden h-8 w-8 p-0 lg:flex"
          :disabled="page >= totalPages || isLoading"
          @click="handlePageChange(totalPages)"
        >
          <span class="sr-only">Go to last page</span>
          <ChevronsRight class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <LinkDialog v-model:open="isDialogOpen" :link-to-edit="selectedLink" @saved="fetchData()" />

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
