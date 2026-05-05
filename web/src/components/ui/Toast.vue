import { ref } from 'vue'

const toastList = ref<Array<{ id: string; title?: string; description?: string; variant?: 'default' | 'destructive' }>>([])

export function useToast() {
  const toast = (opts: { title?: string; description?: string; variant?: 'default' | 'destructive' }) => {
    const id = Math.random().toString(36).substring(7)
    toastList.value.push({ id, ...opts })
    setTimeout(() => {
      toastList.value = toastList.value.filter(t => t.id !== id)
    }, 3000)
  }
  return { toast, toastList }
}
