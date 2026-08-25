import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getSyllabus, type SyllabusCategory } from '../api/teaching'

export const useSyllabusStore = defineStore('syllabus', () => {
  const categories = ref<SyllabusCategory[]>([])
  const loaded = ref(false)

  async function ensureLoaded() {
    if (loaded.value) return
    categories.value = await getSyllabus()
    loaded.value = true
  }

  return { categories, loaded, ensureLoaded }
})
