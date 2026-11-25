// Papan task realtime (revisi 47). Satu WebSocket per tab dipakai bersama papan, Arsip, detail task, dan dialog
// Pindahkan. Server hanya mengirim penanda bahwa task berubah; halaman mengambil ulang lewat REST, jadi filter,
// lingkup Task saya, batas kolom Completed, dan hak akses tetap dihitung server.
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { notifyUnauthorized, tabId } from '../api/client'
import { WbSocket } from './wbSocket'

export type TaskAksi = 'baru' | 'ubah' | 'pindah' | 'centang' | 'hapus'
export type TaskEvent = {
  task_id: string
  version: number
  aksi: TaskAksi
  pelaku: string
  pelaku_id?: number
  tab?: string
}

type Listener = { event: (e: TaskEvent) => void; reconnect: () => void }

// true hanya setelah koneksi terputus, bukan saat pertama menyambung, supaya penanda tidak berkedip saat halaman
// dibuka.
export const liveDisconnected = ref(false)

const listeners = new Set<Listener>()
let socket: WbSocket | null = null
let opened = false
let stopTimer: ReturnType<typeof setTimeout> | undefined

function socketURL(): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/api/tasks/ws`
}

function onOnline() {
  socket?.retryNow()
}

function start() {
  opened = false
  socket = new WbSocket(socketURL(), {
    message: (m) => {
      if (m.t !== 'task' || typeof m.task_id !== 'string') return
      const e = m as unknown as TaskEvent
      // Perubahan dari tab ini sendiri sudah ditangani halaman yang memicunya.
      if (e.tab === tabId) return
      for (const l of [...listeners]) l.event(e)
    },
    state: (s) => {
      if (s === 'open') liveDisconnected.value = false
      else if (s === 'closed') liveDisconnected.value = true
    },
    // Event selama terputus hilang, jadi setiap sambung ulang halaman memuat ulang sekali.
    open: () => {
      if (opened) for (const l of [...listeners]) l.reconnect()
      opened = true
    },
    sessionEnded: () => notifyUnauthorized(),
    // Batas tab sama dengan Whiteboard (32); penanda terputus tetap tampil dan halaman tetap bisa dimuat ulang.
    tooMany: () => {},
  })
  socket.start()
  window.addEventListener('online', onOnline)
}

function stop() {
  window.removeEventListener('online', onOnline)
  socket?.stop()
  socket = null
  liveDisconnected.value = false
}

// Dipanggil di setup komponen. Koneksi ditutup sedetik setelah pendengar terakhir pergi, supaya pindah dari papan
// ke Arsip (halaman lama dilepas sebelum yang baru dipasang) tidak menyambung ulang.
export function useTaskLive(event: (e: TaskEvent) => void, reconnect: () => void = () => {}) {
  const l: Listener = { event, reconnect }
  onMounted(() => {
    clearTimeout(stopTimer)
    listeners.add(l)
    if (!socket) start()
  })
  onBeforeUnmount(() => {
    listeners.delete(l)
    if (listeners.size === 0) stopTimer = setTimeout(stop, 1000)
  })
}

// Pengambilan ulang digabung: beberapa event dalam 300 ms (mis. centang beruntun) menjadi satu permintaan.
export function debounce(fn: () => void, ms = 300) {
  let t: ReturnType<typeof setTimeout> | undefined
  return {
    run: () => {
      clearTimeout(t)
      t = setTimeout(fn, ms)
    },
    cancel: () => clearTimeout(t),
  }
}

// Teks untuk penanda di detail dan dialog Pindahkan.
export function changedText(e: TaskEvent): string {
  const who = e.pelaku || 'Orang lain'
  if (e.aksi === 'hapus') return `${e.task_id} baru saja dihapus ${who}.`
  if (e.aksi === 'pindah') return `${e.task_id} baru saja dipindah ${who}.`
  return `${e.task_id} baru saja diubah ${who}.`
}
