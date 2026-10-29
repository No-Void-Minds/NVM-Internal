<script setup lang="ts">
import { onMounted, ref } from 'vue'
import BaseDialog from '../components/BaseDialog.vue'
import RowMenu, { type MenuItem } from '../components/RowMenu.vue'
import { ApiError } from '../api/client'
import {
  createDivision,
  listDivisions,
  renameDivision,
  setDivisionActive,
  type DivisionItem,
} from '../api/divisions'

const divisions = ref<DivisionItem[]>([])
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const notice = ref<{ kind: 'success' | 'error'; text: string } | null>(null)

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

async function load(quiet = false) {
  if (!quiet) loadState.value = 'loading'
  try {
    divisions.value = await listDivisions()
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

async function afterChange(text: string) {
  notice.value = { kind: 'success', text }
  await load(true)
}

// Satu dialog dipakai untuk tambah dan ganti nama; editing null berarti tambah.
const formOpen = ref(false)
const editing = ref<DivisionItem | null>(null)
const name = ref('')
const formError = ref<string | null>(null)
const saving = ref(false)

function openForm(d: DivisionItem | null) {
  editing.value = d
  name.value = d?.name ?? ''
  formError.value = null
  notice.value = null
  formOpen.value = true
}

async function submitForm() {
  if (saving.value) return
  const value = name.value.trim()
  if (!value) {
    formError.value = 'Nama divisi wajib diisi.'
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await renameDivision(editing.value.id, value)
    } else {
      await createDivision(value)
    }
    formOpen.value = false
    await afterChange(
      editing.value ? `Nama divisi diganti menjadi ${value}.` : `Divisi ${value} ditambahkan.`,
    )
  } catch (err) {
    formError.value = messageOf(err)
  } finally {
    saving.value = false
  }
}

const deactivateTarget = ref<DivisionItem | null>(null)
const deactivating = ref(false)
const deactivateError = ref<string | null>(null)

async function submitDeactivate() {
  const target = deactivateTarget.value
  if (!target || deactivating.value) return
  deactivating.value = true
  try {
    await setDivisionActive(target.id, false)
    deactivateTarget.value = null
    await afterChange(`Divisi ${target.name} dinonaktifkan.`)
  } catch (err) {
    deactivateError.value = messageOf(err)
  } finally {
    deactivating.value = false
  }
}

async function activate(d: DivisionItem) {
  notice.value = null
  try {
    await setDivisionActive(d.id, true)
    await afterChange(`Divisi ${d.name} aktif kembali.`)
  } catch (err) {
    notice.value = { kind: 'error', text: messageOf(err) }
  }
}

function actionsFor(d: DivisionItem): MenuItem[] {
  const items: MenuItem[] = [{ label: 'Ganti nama', action: () => openForm(d) }]
  if (d.is_active) {
    items.push({
      label: 'Nonaktifkan',
      danger: true,
      action: () => {
        deactivateTarget.value = d
        deactivateError.value = null
        notice.value = null
      },
    })
  } else {
    items.push({ label: 'Aktifkan', action: () => activate(d) })
  }
  return items
}

const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section class="max-w-3xl">
    <h1 class="text-2xl font-semibold">Pengaturan</h1>

    <div class="mt-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-lg font-semibold">Divisi</h2>
        <p class="text-sm text-ink-muted">
          Pilihan divisi untuk setiap akun. Divisi tidak bisa dihapus, hanya dinonaktifkan.
        </p>
      </div>
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px"
        @click="openForm(null)"
      >
        Tambah divisi
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

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat daftar divisi...</p>

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

    <p
      v-else-if="divisions.length === 0"
      class="mt-4 rounded-card border border-line bg-surface p-6 text-center"
    >
      Belum ada divisi. Tambahkan divisi pertama sebelum membuat akun.
    </p>

    <ul v-else class="mt-4 divide-y divide-line rounded-card border border-line bg-surface">
      <li v-for="d in divisions" :key="d.id" class="flex items-center gap-3 px-4 py-2">
        <div class="min-w-0 flex-1">
          <p class="font-medium">{{ d.name }}</p>
          <p class="text-sm text-ink-muted">{{ d.active_users }} pengguna aktif</p>
        </div>
        <span
          class="rounded-chip px-2 py-0.5 text-sm font-medium"
          :class="d.is_active ? 'bg-progress-bg text-progress-fg' : 'bg-requested-bg text-requested-fg'"
        >
          {{ d.is_active ? 'Aktif' : 'Nonaktif' }}
        </span>
        <RowMenu :items="actionsFor(d)" :label="`Aksi untuk divisi ${d.name}`" />
      </li>
    </ul>

    <BaseDialog
      :open="formOpen"
      :title="editing ? 'Ganti nama divisi' : 'Tambah divisi'"
      @close="formOpen = false"
    >
      <form id="division-form" class="space-y-3" novalidate @submit.prevent="submitForm">
        <p v-if="formError" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
          {{ formError }}
        </p>
        <div>
          <label for="division-name" class="block text-sm font-medium">Nama divisi</label>
          <input
            id="division-name"
            v-model="name"
            type="text"
            maxlength="50"
            autocomplete="off"
            autofocus
            class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          />
        </div>
      </form>
      <template #footer>
        <button type="button" :class="buttonSecondary" @click="formOpen = false">Batal</button>
        <button
          type="submit"
          form="division-form"
          :disabled="saving"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
        >
          {{ saving ? 'Menyimpan...' : 'Simpan' }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :open="deactivateTarget !== null"
      :title="`Nonaktifkan divisi ${deactivateTarget?.name ?? ''}?`"
      @close="deactivateTarget = null"
    >
      <p>
        Divisi ini tidak bisa dipilih lagi untuk akun baru. {{ deactivateTarget?.active_users ?? 0 }} pengguna
        aktif di dalamnya tetap di divisi ini.
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
