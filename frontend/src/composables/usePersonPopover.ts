import { shallowRef } from 'vue'

export type OpenPerson = { id: number; name: string; trigger: HTMLElement; token: number }

// Satu popover untuk seluruh aplikasi: membuka yang baru otomatis memindahkannya dari pemicu lama.
const current = shallowRef<OpenPerson | null>(null)
let token = 0

export function usePersonPopover() {
  function open(id: number, name: string, trigger: HTMLElement) {
    current.value = { id, name, trigger, token: ++token }
  }
  function close(returnFocus: boolean) {
    const was = current.value
    current.value = null
    if (returnFocus && was?.trigger.isConnected) was.trigger.focus()
  }
  return { current, open, close }
}
