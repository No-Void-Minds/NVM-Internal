<script setup lang="ts">
import { ref } from 'vue'
import FileDropzone from './FileDropzone.vue'
import { formatBytes, precheck } from '../utils/attachment'
import type { SubtaskForm } from '../utils/subtask'

// Satu form isi sub task (revisi 16), dipakai Tambah sub task di detail, Ubah sub task, dan tiap baris form
// Buat task, supaya aturan dan tampilannya tidak bercabang. Tombol simpan tetap milik pemakainya.
const props = withDefaults(
  defineProps<{
    // Awalan id unik per pemakaian, supaya label tetap menunjuk isiannya sendiri.
    idPrefix: string
    // Teks tambahan untuk label pembaca layar, mis. "nomor 2".
    labelSuffix?: string
    picOptions: { id: number; name: string }[]
    // Tenggat task induk: batas atas pemilih tanggal.
    taskDue?: string | null
    withFiles?: boolean
  }>(),
  { labelSuffix: '', taskDue: null, withFiles: false },
)
const form = defineModel<SubtaskForm>({ required: true })
const fileError = ref<string | null>(null)

function set<K extends keyof SubtaskForm>(key: K, value: SubtaskForm[K]) {
  form.value = { ...form.value, [key]: value }
}

function addFiles(files: File[]) {
  const ok: File[] = []
  const problems: string[] = []
  for (const f of files) {
    const p = precheck(f)
    if (p) problems.push(p)
    else ok.push(f)
  }
  fileError.value = problems.length ? problems.join(' ') : null
  set('files', [...form.value.files, ...ok])
}

function removeFile(i: number) {
  set(
    'files',
    form.value.files.filter((_, k) => k !== i),
  )
}

const id = (name: string) => `${props.idPrefix}-${name}`
const field = 'block min-h-11 w-full rounded-card border border-line bg-surface px-3 text-sm'
const label = 'block text-sm text-ink-muted'
</script>

<template>
  <div class="space-y-2">
    <div>
      <label :for="id('title')" :class="label"
        >Nama sub task (wajib)<span class="sr-only"> {{ labelSuffix }}</span></label
      >
      <input
        :id="id('title')"
        :value="form.title"
        type="text"
        maxlength="200"
        autocomplete="off"
        placeholder="Pekerjaan yang perlu dilakukan"
        :class="[field, 'mt-1']"
        @input="set('title', ($event.target as HTMLInputElement).value)"
      />
    </div>
    <div>
      <label :for="id('desc')" :class="label"
        >Deskripsi<span class="sr-only"> {{ labelSuffix }}</span></label
      >
      <textarea
        :id="id('desc')"
        :value="form.description"
        rows="2"
        maxlength="2000"
        class="mt-1 block w-full rounded-card border border-line bg-surface p-3 text-sm"
        @input="set('description', ($event.target as HTMLTextAreaElement).value)"
      ></textarea>
    </div>
    <div>
      <label :for="id('ref')" :class="label"
        >Tautan acuan<span class="sr-only"> {{ labelSuffix }}</span></label
      >
      <input
        :id="id('ref')"
        :value="form.reference_url"
        type="url"
        maxlength="2000"
        inputmode="url"
        placeholder="https://"
        :class="[field, 'mt-1']"
        @input="set('reference_url', ($event.target as HTMLInputElement).value)"
      />
    </div>
    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <div class="min-w-0">
        <label :for="id('due')" :class="label"
          >Tenggat<span class="sr-only"> {{ labelSuffix }}</span></label
        >
        <input
          :id="id('due')"
          :value="form.due_date"
          type="date"
          :max="taskDue || undefined"
          :aria-describedby="taskDue ? id('due-hint') : undefined"
          :class="[field, 'mt-1']"
          @input="set('due_date', ($event.target as HTMLInputElement).value)"
        />
        <p v-if="taskDue" :id="id('due-hint')" class="sr-only">Paling lambat sama dengan tenggat task.</p>
      </div>
      <div class="min-w-0">
        <label :for="id('pic')" :class="label"
          >Dikerjakan<span class="sr-only"> {{ labelSuffix }}</span></label
        >
        <select
          :id="id('pic')"
          :value="form.pic_id"
          :class="[field, 'mt-1']"
          @change="set('pic_id', Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="0">Belum ditugaskan</option>
          <option v-for="p in picOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
    </div>
    <div v-if="withFiles">
      <p :class="label">Lampiran acuan</p>
      <FileDropzone class="mt-1" @files="addFiles" />
      <p v-if="fileError" role="alert" class="mt-1 text-sm text-danger">{{ fileError }}</p>
      <ul v-if="form.files.length" class="mt-2 divide-y divide-line rounded-card border border-line">
        <li
          v-for="(f, i) in form.files"
          :key="`${f.name}-${f.size}-${f.lastModified}-${i}`"
          class="flex items-center gap-3 py-1 pr-1 pl-3 text-sm"
        >
          <span class="min-w-0 flex-1 truncate" :title="f.name">{{ f.name }}</span>
          <span class="shrink-0 text-ink-muted">{{ formatBytes(f.size) }}</span>
          <button
            type="button"
            class="inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            :aria-label="`Batalkan ${f.name}`"
            @click="removeFile(i)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
            >
              <path d="M6 6l12 12M18 6 6 18" />
            </svg>
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>
