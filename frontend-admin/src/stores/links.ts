import api from "@/lib/api"
import { defineStore } from "pinia"
import { ref } from "vue"

export interface Link {
  id: number
  slug: string
  url: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export const useLinksStore = defineStore("links", () => {
  const links = ref<Link[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchLinks = async (
    params: {
      page?: number
      page_size?: number
      sort_by?: string
      sort_order?: "asc" | "desc"
      keyword?: string
      is_active?: boolean
    } = {},
  ) => {
    loading.value = true
    error.value = null
    try {
      const response = await api.get("/links", { params })
      // Handle both old array format (fallback) and new object format
      if (Array.isArray(response.data)) {
        links.value = response.data
        total.value = response.data.length // Best guess for old format
      } else {
        links.value = response.data.links || []
        total.value = response.data.total || 0
      }
    } catch (err: any) {
      error.value = err.message || "Failed to fetch links"
      console.error(err)
    } finally {
      loading.value = false
    }
  }

  const createLink = async (slug: string, url: string) => {
    await api.post("/links", { slug, url })
  }

  const updateLink = async (slug: string, data: { url?: string; is_active?: boolean }) => {
    await api.patch(`/links/${slug}`, data)
  }

  const deleteLink = async (slug: string) => {
    loading.value = true
    error.value = null
    try {
      await api.delete(`/links/${slug}`)
    } catch (err: any) {
      error.value = err.message || "Failed to delete link"
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    links,
    total,
    loading,
    error,
    fetchLinks,
    createLink,
    updateLink,
    deleteLink,
  }
})
