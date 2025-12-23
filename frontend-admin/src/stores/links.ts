import { defineStore } from "pinia"
import { ref } from "vue"
import api from "@/lib/api"

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
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchLinks = async (
    params: {
      page?: number
      page_size?: number
      sort_by?: string
      sort_order?: "asc" | "desc"
    } = {},
  ) => {
    loading.value = true
    error.value = null
    try {
      const response = await api.get("/links", { params })
      links.value = response.data
    } catch (err: any) {
      error.value = err.message || "Failed to fetch links"
      console.error(err)
    } finally {
      loading.value = false
    }
  }

  const createLink = async (slug: string, url: string) => {
    loading.value = true
    error.value = null
    try {
      await api.post("/links", { slug, url })
    } catch (err: any) {
      error.value = err.message || "Failed to create link"
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateLink = async (slug: string, data: { url?: string; is_active?: boolean }) => {
    loading.value = true
    error.value = null
    try {
      await api.patch(`/links/${slug}`, data)
    } catch (err: any) {
      error.value = err.message || "Failed to update link"
      throw err
    } finally {
      loading.value = false
    }
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
    loading,
    error,
    fetchLinks,
    createLink,
    updateLink,
    deleteLink,
  }
})
