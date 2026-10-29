<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import BaseDialog from '../components/BaseDialog.vue'
import RowMenu, { type MenuItem } from '../components/RowMenu.vue'
import UserFormPanel from '../components/UserFormPanel.vue'
import { ApiError } from '../api/client'
import { listDivisions, type DivisionItem } from '../api/divisions'
import {
  listUsers,
  resetUserPassword,
  setUserActive,
  unlockUser,
  type UserItem,
  type UserStatus,
} from '../api/users'
import { useSessionStore } from '../stores/session'
import { formatDateTime } from '../utils/format'
import { MIN_PASSWORD_LENGTH } from '../utils/password'
import { roleLabel, type Role } from '../utils/roles'

const session = useSessionStore()

const users = ref<UserItem[]>([])
const divisions = ref<DivisionItem[]>([])
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const notice = ref<{ kind: 'success' | 'error'; text: string } | null>(null)

const search = ref('')
const roleFilter = ref<Role | ''>('')
const divisionFilter = ref<number | ''>('')
const statusFilter = ref<UserStatus | ''>('')

const statusLabel: Record<UserStatus, string> = { aktif: 'Aktif', terkunci: 'Terkunci', nonaktif: 'Nonaktif' }
const statusClass: Record<UserStatus, string> = {
  aktif: 'bg-progress-bg text-progress-fg',
  terkunci: 'bg-ready-bg text-ready-fg',
  nonaktif: 'bg-requested-bg text-requested-fg',
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return users.value.filter(
    (u) =>
      (!q || u.name.toLowerCase().includes(q) || u.username.toLowerCase().includes(q)) &&
      (!roleFilter.value || u.role === roleFilter.value) &&
      (divisionFilter.value === '' || u.division_id === divisionFilter.value) &&
      (!statusFilter.value || u.status === statusFilter.value),
  )
})
const hasFilter = computed(
  () => !!search.value || !!roleFilter.value || divisionFilter.value !== '' || !!statusFilter.value,
)

function clearFilters() {
  search.value = ''
  roleFilter.value = ''
  divisionFilter.value = ''
  statusFilter.value = ''
}

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

// quiet dipakai setelah aksi, supaya tabel tidak berkedip ke tampilan memuat.
async function load(quiet = false) {
  if (!quiet) loadState.value = 'loading'
  try {
    const [u, d] = await Promise.all([listUsers(), listDivisions()])
    users.value = u
    divisions.value = d
    loadState.value = 'ready'
  } catch (err) {
    if (quiet) {
      notice.value = { kind: 'error', text: messageOf(err) }
      return
    }
    loadError.value = messageOf(err)
    loadState.value = 'error'
  }
}
onMounted(() => load())

async function afterChange(text: string, changedId: number | null) {
  notice.value = { kind: 'success', text }
  await load(true)
  // Mengubah akun sendiri (nama atau peran) harus langsung terlihat di header dan menu.
  if (changedId === session.user?.id) await session.fetchMe()
}

const panelOpen = ref(false)
const editing = ref<UserItem | null>(null)

function openCreate() {
  editing.value = null
  notice.value = null
  panelOpen.value = true
}
function openEdit(u: UserItem) {
  editing.value = u
  notice.value = null
  panelOpen.value = true
}
async function onSaved(text: string) {
  const changedId = editing.value?.id ?? null
  panelOpen.value = false
  await afterChange(text, changedId)
}

const resetTarget = ref<UserItem | null>(null)
const resetPassword = ref('')
const resetError = ref<string | null>(null)
const resetting = ref(false)

function openReset(u: UserItem) {
  resetTarget.value = u
  resetPassword.value = ''
  resetError.value = null
  notice.value = null
}
async function submitReset() {
  const target = resetTarget.value
  if (!target || resetting.value) return
  if ([...resetPassword.value].length < MIN_PASSWORD_LENGTH) {
    resetError.value = `Password minimal ${MIN_PASSWORD_LENGTH} karakter.`
    return
  }
  resetting.value = true
  try {
    await resetUserPassword(target.id, resetPassword.value)
    resetTarget.value = null
    await afterChange(
      `Password ${target.name} diatur ulang. Berikan password barunya secara langsung.`,
      target.id,
    )
  } catch (err) {
    resetError.value = messageOf(err)
  } finally {
    resetting.value = false
  }
}

const deactivateTarget = ref<UserItem | null>(null)
const deactivating = ref(false)
const deactivateError = ref<string | null>(null)

function openDeactivate(u: UserItem) {
  deactivateTarget.value = u
  deactivateError.value = null
  notice.value = null
}
async function submitDeactivate() {
  const target = deactivateTarget.value
  if (!target || deactivating.value) return
  deactivating.value = true
  try {
    await setUserActive(target.id, false)
    deactivateTarget.value = null
    await afterChange(`${target.name} dinonaktifkan dan sudah keluar dari semua perangkat.`, target.id)
  } catch (err) {
    deactivateError.value = messageOf(err)
  } finally {
    deactivating.value = false
  }
}

async function quickAction(run: () => Promise<void>, text: string, id: number) {
  notice.value = null
  try {
    await run()
    await afterChange(text, id)
  } catch (err) {
    notice.value = { kind: 'error', text: messageOf(err) }
  }
}

function actionsFor(u: UserItem): MenuItem[] {
  const items: MenuItem[] = [{ label: 'Ubah', action: () => openEdit(u) }]
  if (u.status !== 'nonaktif') items.push({ label: 'Atur ulang password', action: () => openReset(u) })
  if (u.status === 'terkunci') {
    items.push({
      label: 'Buka kunci',
      action: () => quickAction(() => unlockUser(u.id), `Kunci akun ${u.name} dibuka.`, u.id),
    })
  }
  if (u.status === 'nonaktif') {
    items.push({
      label: 'Aktifkan',
      action: () => quickAction(() => setUserActive(u.id, true), `${u.name} aktif kembali.`, u.id),
    })
  } else if (u.id !== session.user?.id) {
    items.push({ label: 'Nonaktifkan', danger: true, action: () => openDeactivate(u) })
  }
  return items
}

const selectClass = 'min-h-11 w-full rounded-card border border-line bg-surface px-3'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">Pengguna</h1>
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px"
        @click="openCreate"
      >
        Tambah pengguna
      </button>
    </div>

    <div aria-live="polite">
      <p
        v-if="notice"
        :role="notice.kind === 'error' ? 'alert' : 'status'"
        class="mt-4 rounded-card p-3 text-sm"
        :class="
          notice.kind === 'error'
            ? 'bg-danger-bg text-danger'
            : 'border border-line bg-surface text-progress-fg'
        "
      >
        {{ notice.text }}
      </p>
    </div>

    <div
      class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))]"
    >
      <div>
        <label for="user-search" class="sr-only">Cari nama atau username</label>
        <input
          id="user-search"
          v-model="search"
          type="search"
          placeholder="Cari nama atau username"
          :class="selectClass"
        />
      </div>
      <div>
        <label for="filter-role" class="sr-only">Filter peran</label>
        <select id="filter-role" v-model="roleFilter" :class="selectClass">
          <option value="">Semua peran</option>
          <option value="admin">Admin</option>
          <option value="manager">Manajer</option>
          <option value="staff">Staf</option>
        </select>
      </div>
      <div>
        <label for="filter-division" class="sr-only">Filter divisi</label>
        <select id="filter-division" v-model="divisionFilter" :class="selectClass">
          <option value="">Semua divisi</option>
          <option v-for="d in divisions" :key="d.id" :value="d.id">{{ d.name }}</option>
        </select>
      </div>
      <div>
        <label for="filter-status" class="sr-only">Filter status</label>
        <select id="filter-status" v-model="statusFilter" :class="selectClass">
          <option value="">Semua status</option>
          <option value="aktif">Aktif</option>
          <option value="terkunci">Terkunci</option>
          <option value="nonaktif">Nonaktif</option>
        </select>
      </div>
    </div>

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat daftar pengguna...</p>

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

    <template v-else>
      <p class="mt-4 text-sm text-ink-muted">
        Menampilkan {{ filtered.length }} dari {{ users.length }} pengguna
      </p>

      <div
        v-if="filtered.length === 0"
        class="mt-3 rounded-card border border-line bg-surface p-6 text-center"
      >
        <p>Tidak ada pengguna yang cocok dengan filter ini.</p>
        <button v-if="hasFilter" type="button" :class="buttonSecondary" class="mt-3" @click="clearFilters">
          Hapus filter
        </button>
      </div>

      <template v-else>
        <!-- Tujuh kolom baru muat tanpa terpotong mulai sekitar 1024px; di bawah itu tiap akun jadi kartu. -->
        <table class="mt-3 hidden w-full border-collapse border border-line bg-surface text-left lg:table">
          <thead class="border-b border-line text-sm text-ink-muted">
            <tr>
              <th scope="col" class="px-4 py-3 font-medium">Nama</th>
              <th scope="col" class="px-4 py-3 font-medium">Username</th>
              <th scope="col" class="px-4 py-3 font-medium">Peran</th>
              <th scope="col" class="px-4 py-3 font-medium">Divisi</th>
              <th scope="col" class="px-4 py-3 font-medium">Status</th>
              <th scope="col" class="px-4 py-3 font-medium">Login terakhir</th>
              <th scope="col" class="px-4 py-3"><span class="sr-only">Aksi</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in filtered" :key="u.id" class="border-b border-line last:border-b-0">
              <td class="px-4 py-2">{{ u.name }}</td>
              <td class="px-4 py-2 text-ink-muted">{{ u.username }}</td>
              <td class="px-4 py-2">{{ roleLabel(u.role) }}</td>
              <td class="px-4 py-2">{{ u.division_name }}</td>
              <td class="px-4 py-2">
                <span class="rounded-chip px-2 py-0.5 text-sm font-medium" :class="statusClass[u.status]">
                  {{ statusLabel[u.status] }}
                </span>
              </td>
              <td class="px-4 py-2 text-ink-muted">
                {{ u.last_login_at ? formatDateTime(u.last_login_at) : 'Belum pernah' }}
              </td>
              <td class="px-2 py-1 text-right">
                <RowMenu :items="actionsFor(u)" :label="`Aksi untuk ${u.name}`" />
              </td>
            </tr>
          </tbody>
        </table>

        <ul class="mt-3 space-y-3 lg:hidden">
          <li v-for="u in filtered" :key="u.id" class="rounded-card border border-line bg-surface p-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="font-medium">{{ u.name }}</p>
                <p class="break-words text-sm text-ink-muted">{{ u.username }}</p>
              </div>
              <RowMenu :items="actionsFor(u)" :label="`Aksi untuk ${u.name}`" />
            </div>
            <p class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
              <span class="rounded-chip px-2 py-0.5 font-medium" :class="statusClass[u.status]">
                {{ statusLabel[u.status] }}
              </span>
              <span>{{ roleLabel(u.role) }}</span>
              <span>{{ u.division_name }}</span>
            </p>
            <p class="mt-2 text-sm text-ink-muted">
              Login terakhir: {{ u.last_login_at ? formatDateTime(u.last_login_at) : 'belum pernah' }}
            </p>
          </li>
        </ul>
      </template>
    </template>

    <UserFormPanel
      :open="panelOpen"
      :user="editing"
      :divisions="divisions"
      @close="panelOpen = false"
      @saved="onSaved"
    />

    <BaseDialog
      :open="resetTarget !== null"
      :title="`Atur ulang password ${resetTarget?.name ?? ''}`"
      @close="resetTarget = null"
    >
      <form id="reset-form" class="space-y-3" novalidate @submit.prevent="submitReset">
        <p class="text-sm text-ink-muted">
          Semua sesi akun ini akan berakhir, kuncinya dibuka, dan password ini wajib diganti saat login
          berikutnya.
        </p>
        <p v-if="resetError" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
          {{ resetError }}
        </p>
        <div>
          <label for="reset-password" class="block text-sm font-medium">Password sementara</label>
          <input
            id="reset-password"
            v-model="resetPassword"
            type="text"
            autocomplete="off"
            autofocus
            class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          />
        </div>
      </form>
      <template #footer>
        <button type="button" :class="buttonSecondary" @click="resetTarget = null">Batal</button>
        <button
          type="submit"
          form="reset-form"
          :disabled="resetting"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
        >
          {{ resetting ? 'Menyimpan...' : 'Atur ulang' }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :open="deactivateTarget !== null"
      :title="`Nonaktifkan ${deactivateTarget?.name ?? ''}?`"
      @close="deactivateTarget = null"
    >
      <p>
        {{ deactivateTarget?.name }} akan langsung keluar dari semua perangkat. Task lamanya tetap tercatat.
      </p>
      <p v-if="deactivateError" role="alert" class="mt-3 rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ deactivateError }}
      </p>
      <template #footer>
        <button type="button" :class="buttonSecondary" @click="deactivateTarget = null">Batal</button>
        <button
          type="button"
          :disabled="deactivating"
          class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
          @click="submitDeactivate"
        >
          {{ deactivating ? 'Memproses...' : 'Nonaktifkan' }}
        </button>
      </template>
    </BaseDialog>
  </section>
</template>
