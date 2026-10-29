<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import BaseDialog from './BaseDialog.vue'
import RowMenu, { type MenuItem } from './RowMenu.vue'
import { ApiError } from '../api/client'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { createLabel, listLabels, setLabelActive, updateLabel, type LabelItem } from '../api/labels'

// Nama warna ditampilkan di samping kotak warna supaya pilihan tidak dibedakan lewat warna saja.
const colorName: Record<string, string> = {
  '#E6E1D7': 'Pasir',
  '#D5E3C8': 'Hijau',
  '#D3E2EC': 'Biru',
  '#E1DAEC': 'Ungu',
  '#F2D8D8': 'Merah muda',
  '#F3E5BF': 'Kuning',
  '#F2D9C2': 'Jingga',
  '#DADDE0': 'Abu',
}

const labels = ref<LabelItem[]>([])
const palette = ref<string[]>([])
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
    const [l, d] = await Promise.all([listLabels(), listDivisions()])
    labels.value = l.labels
    palette.value = l.palette
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

async function afterChange(text: string) {
  notice.value = { kind: 'success', text }
  await load(true)
}

const formOpen = ref(false)
const editing = ref<LabelItem | null>(null)
const name = ref('')
const color = ref('')
// String kosong berarti "Semua divisi"; select HTML tidak bisa menyimpan null.
const divisionId = ref<number | ''>('')
const formError = ref<string | null>(null)
const saving = ref(false)

const divisionOptions = computed(() =>
  divisions.value.filter((d) => d.is_active || d.id === editing.value?.division_id),
)

function openForm(l: LabelItem | null) {
  editing.value = l
  name.value = l?.name ?? ''
  color.value = l?.color ?? palette.value[0] ?? ''
  divisionId.value = l?.division_id ?? ''
  formError.value = null
  notice.value = null
  formOpen.value = true
}

async function submitForm() {
  if (saving.value) return
  const value = name.value.trim()
  if (!value) {
    formError.value = 'Nama label wajib diisi.'
    return
  }
  const input = {
    name: value,
    color: color.value,
    division_id: divisionId.value === '' ? null : divisionId.value,
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateLabel(editing.value.id, input)
    } else {
      await createLabel(input)
    }
    formOpen.value = false
    await afterChange(editing.value ? `Label ${value} disimpan.` : `Label ${value} ditambahkan.`)
  } catch (err) {
    formError.value = messageOf(err)
  } finally {
    saving.value = false
  }
}

async function toggleActive(l: LabelItem) {
  notice.value = null
  try {
    await setLabelActive(l.id, !l.is_active)
    await afterChange(
      l.is_active
        ? `Label ${l.name} dinonaktifkan. Task lama tetap memakainya.`
        : `Label ${l.name} aktif kembali.`,
    )
  } catch (err) {
    notice.value = { kind: 'error', text: messageOf(err) }
  }
}

function actionsFor(l: LabelItem): MenuItem[] {
  return [
    { label: 'Ubah', action: () => openForm(l) },
    l.is_active
      ? { label: 'Nonaktifkan', danger: true, action: () => toggleActive(l) }
      : { label: 'Aktifkan', action: () => toggleActive(l) },
  ]
}

const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-lg font-semibold">Label</h2>
        <p class="text-sm text-ink-muted">
          Penanda task seperti di Trello. Label nonaktif tidak bisa dipasang lagi, tetapi tetap tampil di task
          lama.
        </p>
      </div>
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px"
        @click="openForm(null)"
      >
        Tambah label
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

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat daftar label...</p>

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
      v-else-if="labels.length === 0"
      class="mt-4 rounded-card border border-line bg-surface p-6 text-center"
    >
      Belum ada label. Tambahkan label pertama, misalnya jenis pekerjaan atau platform.
    </p>

    <ul v-else class="mt-4 divide-y divide-line rounded-card border border-line bg-surface">
      <li v-for="l in labels" :key="l.id" class="flex items-center gap-3 px-4 py-2">
        <div class="min-w-0 flex-1">
          <span
            class="inline-block rounded-chip px-2 py-0.5 font-medium"
            :style="{ backgroundColor: l.color }"
          >
            {{ l.name }}
          </span>
          <p class="mt-1 text-sm text-ink-muted">
            {{ l.division_name ?? 'Semua divisi' }} · dipakai {{ l.task_count }} task
          </p>
        </div>
        <span
          class="rounded-chip px-2 py-0.5 text-sm font-medium"
          :class="l.is_active ? 'bg-progress-bg text-progress-fg' : 'bg-requested-bg text-requested-fg'"
        >
          {{ l.is_active ? 'Aktif' : 'Nonaktif' }}
        </span>
        <RowMenu :items="actionsFor(l)" :label="`Aksi untuk label ${l.name}`" />
      </li>
    </ul>

    <BaseDialog :open="formOpen" :title="editing ? 'Ubah label' : 'Tambah label'" @close="formOpen = false">
      <form id="label-form" class="space-y-4" novalidate @submit.prevent="submitForm">
        <p v-if="formError" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
          {{ formError }}
        </p>

        <div>
          <label for="label-name" class="block text-sm font-medium">Nama label</label>
          <input
            id="label-name"
            v-model="name"
            type="text"
            maxlength="50"
            autocomplete="off"
            autofocus
            class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          />
        </div>

        <fieldset>
          <legend class="text-sm font-medium">Warna</legend>
          <div class="mt-1 grid grid-cols-2 gap-2 sm:grid-cols-4">
            <label
              v-for="c in palette"
              :key="c"
              class="flex min-h-11 cursor-pointer items-center gap-2 rounded-card border border-line px-2 text-sm has-[:checked]:border-accent has-[:checked]:font-medium has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent"
            >
              <input v-model="color" type="radio" name="label-color" :value="c" class="sr-only" />
              <span
                aria-hidden="true"
                class="size-5 shrink-0 rounded-chip border border-line"
                :style="{ backgroundColor: c }"
              />
              {{ colorName[c] ?? c }}
            </label>
          </div>
        </fieldset>

        <div>
          <label for="label-division" class="block text-sm font-medium">Divisi</label>
          <select
            id="label-division"
            v-model="divisionId"
            aria-describedby="label-division-hint"
            class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          >
            <option value="">Semua divisi</option>
            <option v-for="d in divisionOptions" :key="d.id" :value="d.id">
              {{ d.name }}{{ d.is_active ? '' : ' (nonaktif)' }}
            </option>
          </select>
          <p id="label-division-hint" class="mt-1 text-sm text-ink-muted">
            Staf hanya bisa memasang label divisinya sendiri dan label untuk semua divisi.
          </p>
        </div>

        <p class="text-sm">
          Pratinjau:
          <span class="ml-1 rounded-chip px-2 py-0.5 font-medium" :style="{ backgroundColor: color }">
            {{ name.trim() || 'Nama label' }}
          </span>
        </p>
      </form>
      <template #footer>
        <button type="button" :class="buttonSecondary" @click="formOpen = false">Batal</button>
        <button
          type="submit"
          form="label-form"
          :disabled="saving"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
        >
          {{ saving ? 'Menyimpan...' : 'Simpan' }}
        </button>
      </template>
    </BaseDialog>
  </section>
</template>
