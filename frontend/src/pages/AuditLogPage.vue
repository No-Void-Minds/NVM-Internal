<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ApiError } from '../api/client'
import { listAuditLogs, type AuditCategory, type AuditEntry } from '../api/audit'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { listUsers, type UserItem } from '../api/users'
import { describeAudit } from '../utils/auditText'
import { formatDateTime } from '../utils/format'

const monthLabel = new Intl.DateTimeFormat('id-ID', { month: 'short', year: 'numeric' })

// Enam bulan terakhir cukup untuk pemakaian harian; audit login sendiri hanya disimpan 1 tahun (NFR-RET-03).
const months = (() => {
  const now = new Date()
  return Array.from({ length: 6 }, (_, i) => {
    const start = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const end = new Date(now.getFullYear(), now.getMonth() - i + 1, 1)
    return {
      key: `${start.getFullYear()}-${start.getMonth() + 1}`,
      label: monthLabel.format(start),
      start,
      end,
    }
  })
})()

const categories: { value: AuditCategory; label: string }[] = [
  { value: 'auth', label: 'Masuk dan keamanan' },
  { value: 'account', label: 'Perubahan akun' },
  { value: 'division', label: 'Perubahan divisi' },
  { value: 'label', label: 'Perubahan label' },
  { value: 'task', label: 'Task' },
]

const monthFilter = ref('')
const userFilter = ref<number | ''>('')
const categoryFilter = ref<AuditCategory | ''>('')
const page = ref(1)

const entries = ref<AuditEntry[]>([])
const hasMore = ref(false)
const users = ref<UserItem[]>([])
const divisions = ref<DivisionItem[]>([])
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
let requestSeq = 0

const hasFilter = computed(() => !!monthFilter.value || userFilter.value !== '' || !!categoryFilter.value)

function divisionName(id: number): string {
  return divisions.value.find((d) => d.id === id)?.name ?? `divisi #${id}`
}

function actorName(e: AuditEntry): string {
  if (e.actor) return e.actor.name
  // Login gagal dengan username tak terdaftar tidak punya pelaku; selain itu baris tanpa pelaku dibuat sistem (seed).
  return e.action === 'login_failed' ? 'Tidak dikenal' : 'Sistem'
}

async function load() {
  // Nomor urut mencegah respons filter lama yang datang terlambat menimpa hasil filter terbaru.
  const seq = ++requestSeq
  loadState.value = 'loading'
  const month = months.find((m) => m.key === monthFilter.value)
  try {
    const res = await listAuditLogs({
      page: page.value,
      from: month?.start.toISOString(),
      to: month?.end.toISOString(),
      userId: userFilter.value === '' ? undefined : userFilter.value,
      category: categoryFilter.value || undefined,
    })
    if (seq !== requestSeq) return
    entries.value = res.entries
    hasMore.value = res.has_more
    loadState.value = 'ready'
  } catch (err) {
    if (seq !== requestSeq) return
    loadError.value =
      err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
    loadState.value = 'error'
  }
}

watch([monthFilter, userFilter, categoryFilter], () => {
  page.value = 1
  void load()
})

function goTo(p: number) {
  page.value = p
  void load()
}

function clearFilters() {
  monthFilter.value = ''
  userFilter.value = ''
  categoryFilter.value = ''
}

onMounted(async () => {
  void load()
  // Daftar pengguna dan divisi hanya pelengkap filter dan teks; kalau gagal, audit log tetap tampil.
  try {
    ;[users.value, divisions.value] = await Promise.all([listUsers(), listDivisions()])
  } catch {
    users.value = []
  }
})

const selectClass = 'min-h-11 w-full rounded-card border border-line bg-surface px-3'
const pagerButton =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60'
</script>

<template>
  <section>
    <h1 class="text-2xl font-semibold">Audit Log</h1>
    <p class="mt-1 text-ink-muted">
      Catatan siapa melakukan apa, kapan, dan dari alamat IP mana. Hanya bisa dibaca.
    </p>

    <div class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
      <div>
        <label for="audit-month" class="block text-sm font-medium">Rentang tanggal</label>
        <select id="audit-month" v-model="monthFilter" :class="selectClass" class="mt-1">
          <option value="">Semua tanggal</option>
          <option v-for="m in months" :key="m.key" :value="m.key">{{ m.label }}</option>
        </select>
      </div>
      <div>
        <label for="audit-user" class="block text-sm font-medium">Pengguna</label>
        <select id="audit-user" v-model="userFilter" :class="selectClass" class="mt-1">
          <option value="">Semua pengguna</option>
          <option v-for="u in users" :key="u.id" :value="u.id">{{ u.name }}</option>
        </select>
      </div>
      <div>
        <label for="audit-category" class="block text-sm font-medium">Jenis aksi</label>
        <select id="audit-category" v-model="categoryFilter" :class="selectClass" class="mt-1">
          <option value="">Semua jenis aksi</option>
          <option v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</option>
        </select>
      </div>
    </div>

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted" role="status">Memuat audit log...</p>

    <div
      v-else-if="loadState === 'error'"
      role="alert"
      class="mt-6 rounded-card bg-danger-bg p-4 text-danger"
    >
      <p class="font-medium">{{ loadError }}</p>
      <button
        type="button"
        class="mt-3 inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
        @click="load()"
      >
        Muat ulang
      </button>
    </div>

    <div
      v-else-if="entries.length === 0"
      class="mt-6 rounded-card border border-line bg-surface p-6 text-center"
    >
      <p>{{ hasFilter ? 'Tidak ada catatan yang cocok dengan filter ini.' : 'Belum ada catatan.' }}</p>
      <button v-if="hasFilter" type="button" :class="pagerButton" class="mt-3" @click="clearFilters">
        Hapus filter
      </button>
    </div>

    <template v-else>
      <!-- Kalimat aksi butuh ruang lebar; tabel baru dipakai mulai 1024px, di bawahnya tiap catatan jadi kartu. -->
      <table class="mt-4 hidden w-full border-collapse border border-line bg-surface text-left lg:table">
        <thead class="border-b border-line text-sm text-ink-muted">
          <tr>
            <th scope="col" class="px-4 py-3 font-medium whitespace-nowrap">Waktu</th>
            <th scope="col" class="px-4 py-3 font-medium">Pengguna</th>
            <th scope="col" class="px-4 py-3 font-medium">Aksi</th>
            <th scope="col" class="px-4 py-3 font-medium">Alamat IP</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in entries" :key="e.id" class="border-b border-line align-top last:border-b-0">
            <td class="px-4 py-3 whitespace-nowrap text-ink-muted">{{ formatDateTime(e.occurred_at) }}</td>
            <td class="px-4 py-3">{{ actorName(e) }}</td>
            <td class="px-4 py-3">{{ describeAudit(e, divisionName) }}</td>
            <td class="px-4 py-3 text-ink-muted">{{ e.ip || '-' }}</td>
          </tr>
        </tbody>
      </table>

      <ul class="mt-4 space-y-3 lg:hidden">
        <li v-for="e in entries" :key="e.id" class="rounded-card border border-line bg-surface p-4">
          <p class="text-sm text-ink-muted">{{ formatDateTime(e.occurred_at) }}</p>
          <p class="mt-1 font-medium">{{ actorName(e) }}</p>
          <p class="mt-1 break-words">{{ describeAudit(e, divisionName) }}</p>
          <p v-if="e.ip" class="mt-1 text-sm text-ink-muted">IP {{ e.ip }}</p>
        </li>
      </ul>

      <nav aria-label="Halaman audit log" class="mt-4 flex items-center justify-between gap-3">
        <button type="button" :class="pagerButton" :disabled="page === 1" @click="goTo(page - 1)">
          Lebih baru
        </button>
        <span class="text-sm text-ink-muted">Halaman {{ page }}</span>
        <button type="button" :class="pagerButton" :disabled="!hasMore" @click="goTo(page + 1)">
          Lebih lama
        </button>
      </nav>
    </template>
  </section>
</template>
