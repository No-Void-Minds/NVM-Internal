<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import { createTimelineEvent, updateTimelineEvent, type TimelineEventInput } from '../api/timeline'
import type { Person } from '../api/board'
import {
  absenceTypes,
  categories,
  categoryLabel,
  type AbsenceType,
  type TimelineCategory,
  type TimelineEvent,
} from '../utils/timeline'

// Satu form untuk Buat kartu dan Ubah (revisi 30). Hanya Manajer yang membukanya; backend tetap memeriksa haknya.
const props = defineProps<{
  open: boolean
  // null untuk kartu baru.
  event: TimelineEvent | null
  // Tanggal mulai bawaan untuk kartu baru, mis. dari "Tambah di tanggal ini".
  date: string
  people: Person[] | null
  peopleError: string
}>()
const emit = defineEmits<{ close: []; saved: [date: string, message: string] }>()

const uid = useId()
const category = ref<TimelineCategory>('launching')
const title = ref('')
const description = ref('')
const startDate = ref('')
const endDate = ref('')
const userId = ref(0)
const absence = ref<AbsenceType | ''>('')
const error = ref('')
const submitting = ref(false)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    const e = props.event
    category.value = e?.category ?? 'launching'
    title.value = e?.title ?? ''
    description.value = e?.description ?? ''
    startDate.value = e?.start_date ?? props.date
    endDate.value = e?.end_date ?? ''
    userId.value = e?.user?.id ?? 0
    absence.value = e?.absence_type ?? ''
    error.value = ''
  },
  { immediate: true },
)

const isAbsence = computed(() => category.value === 'kehadiran')
// Akun yang sudah nonaktif tetap tampil di pilihan saat mengubah kartu lamanya, supaya pilihannya tidak kosong.
const options = computed(() => {
  const list = props.people ?? []
  const u = props.event?.user
  return u && !list.some((p) => p.id === u.id) ? [{ id: u.id, name: `${u.name} (nonaktif)` }, ...list] : list
})

async function submit() {
  submitting.value = true
  error.value = ''
  const input: TimelineEventInput = {
    category: category.value,
    title: isAbsence.value ? '' : title.value,
    description: description.value,
    start_date: startDate.value,
    end_date: endDate.value,
    user_id: isAbsence.value ? userId.value : 0,
    absence_type: isAbsence.value ? absence.value : '',
  }
  try {
    if (props.event) await updateTimelineEvent(props.event.id, input)
    else await createTimelineEvent(input)
    emit('saved', startDate.value, props.event ? 'Kartu disimpan.' : 'Kartu dibuat.')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Kartu gagal disimpan. Periksa koneksi, lalu coba lagi.'
  } finally {
    submitting.value = false
  }
}

const inputClass = 'mt-1 block min-h-11 w-full min-w-0 rounded-card border border-line bg-surface px-3'
const optionClass =
  'flex min-h-11 cursor-pointer items-center justify-center rounded-card border border-line px-2 text-sm font-medium hover:bg-canvas has-[:checked]:border-mark has-[:checked]:bg-progress-bg has-[:checked]:text-progress-fg has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-mark'
</script>

<template>
  <BaseDialog
    :open="open"
    :title="event ? 'Ubah kartu' : 'Buat kartu'"
    variant="panel"
    @close="emit('close')"
  >
    <form :id="`${uid}-form`" class="space-y-5" novalidate @submit.prevent="submit">
      <p v-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">{{ error }}</p>

      <fieldset>
        <legend class="text-sm font-medium">Jenis kartu</legend>
        <div class="mt-1 grid grid-cols-3 gap-2">
          <label v-for="c in categories" :key="c" :class="optionClass">
            <input v-model="category" type="radio" :name="`${uid}-category`" :value="c" class="sr-only" />
            {{ categoryLabel[c] }}
          </label>
        </div>
      </fieldset>

      <div v-if="!isAbsence">
        <label :for="`${uid}-title`" class="block text-sm font-medium">Judul (wajib)</label>
        <input
          :id="`${uid}-title`"
          v-model="title"
          type="text"
          maxlength="200"
          autocomplete="off"
          :class="inputClass"
        />
      </div>

      <div v-else>
        <label :for="`${uid}-person`" class="block text-sm font-medium">Nama (wajib)</label>
        <select
          :id="`${uid}-person`"
          v-model.number="userId"
          :class="inputClass"
          :disabled="!people && !peopleError"
        >
          <option :value="0">
            {{ people ? 'Pilih akun' : peopleError ? 'Daftar akun gagal dimuat' : 'Memuat akun...' }}
          </option>
          <option v-for="p in options" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <p v-if="peopleError" role="alert" class="mt-1 text-sm text-danger">{{ peopleError }}</p>
      </div>

      <div class="grid grid-cols-[repeat(auto-fit,minmax(9rem,1fr))] gap-3">
        <div class="min-w-0">
          <label :for="`${uid}-start`" class="block text-sm font-medium">Tanggal mulai (wajib)</label>
          <input :id="`${uid}-start`" v-model="startDate" type="date" :class="inputClass" />
        </div>
        <div class="min-w-0">
          <label :for="`${uid}-end`" class="block text-sm font-medium">Sampai</label>
          <input
            :id="`${uid}-end`"
            v-model="endDate"
            type="date"
            :min="startDate || undefined"
            :aria-describedby="`${uid}-end-hint`"
            :class="inputClass"
          />
        </div>
      </div>
      <p :id="`${uid}-end-hint`" class="-mt-3 text-sm text-ink-muted">
        Kosongkan untuk satu hari. Paling panjang 62 hari.
      </p>

      <fieldset v-if="isAbsence">
        <legend class="text-sm font-medium">Alasan (wajib)</legend>
        <div class="mt-1 grid grid-cols-2 gap-2">
          <label v-for="a in absenceTypes" :key="a.value" :class="optionClass">
            <input v-model="absence" type="radio" :name="`${uid}-absence`" :value="a.value" class="sr-only" />
            {{ a.label }}
          </label>
        </div>
        <p class="mt-2 text-sm text-ink-muted">
          Sakit, izin pribadi, dan cuti hanya terlihat oleh Manajer dan orangnya. Yang lain melihat "Tidak
          hadir".
        </p>
      </fieldset>

      <div>
        <label :for="`${uid}-desc`" class="block text-sm font-medium">{{
          isAbsence ? 'Keterangan' : 'Deskripsi'
        }}</label>
        <textarea
          :id="`${uid}-desc`"
          v-model="description"
          rows="4"
          maxlength="2000"
          class="mt-1 block w-full min-w-0 rounded-card border border-line bg-surface px-3 py-2"
        />
      </div>
    </form>
    <template #footer>
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px"
        @click="emit('close')"
      >
        Batal
      </button>
      <button
        type="submit"
        :form="`${uid}-form`"
        :disabled="submitting"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Menyimpan...' : event ? 'Simpan kartu' : 'Buat kartu' }}
      </button>
    </template>
  </BaseDialog>
</template>
