<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import { createTask, setTaskLabels, updateTask, type TaskDetail } from '../api/tasks'
import type { UserItem } from '../api/users'
import type { LabelItem } from '../api/labels'
import { urgencies, urgencyLabel, type Urgency } from '../utils/task'

const props = defineProps<{
  open: boolean
  task: TaskDetail | null
  users: UserItem[]
  labels: LabelItem[]
}>()
const emit = defineEmits<{ close: []; saved: [id: string, message: string] }>()

const isEdit = computed(() => props.task !== null)

const title = ref('')
const urgency = ref<Urgency>('medium')
// Urutan sesuai klik: PIC pertama menentukan divisi task (keputusan Sprint 3).
const picIds = ref<number[]>([])
const labelIds = ref<number[]>([])
const description = ref('')
const startDate = ref('')
const dueDate = ref('')
const deliverableUrl = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

// Hanya akun aktif dengan peran Staf atau Manajer yang bisa jadi PIC; Admin tidak memindahkan task.
const picCandidates = computed(() =>
  props.users.filter((u) => u.status !== 'nonaktif' && (u.role === 'staff' || u.role === 'manager')),
)
const labelOptions = computed(() =>
  props.labels.filter((l) => l.is_active || props.task?.labels.some((t) => t.id === l.id)),
)
const firstPicDivision = computed(
  () => props.users.find((u) => u.id === picIds.value[0])?.division_name ?? null,
)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    const t = props.task
    error.value = null
    title.value = t?.title ?? ''
    urgency.value = t?.urgency ?? 'medium'
    picIds.value = t?.assignees.map((a) => a.id) ?? []
    labelIds.value = t?.labels.map((l) => l.id) ?? []
    description.value = t?.description ?? ''
    startDate.value = t?.start_date ?? ''
    dueDate.value = t?.due_date ?? ''
    deliverableUrl.value = t?.deliverable_url ?? ''
  },
)

function togglePic(id: number) {
  picIds.value = picIds.value.includes(id) ? picIds.value.filter((p) => p !== id) : [...picIds.value, id]
}

function toggleLabel(id: number) {
  labelIds.value = labelIds.value.includes(id)
    ? labelIds.value.filter((l) => l !== id)
    : [...labelIds.value, id]
}

function validate(): string | null {
  if (!title.value.trim()) return 'Judul wajib diisi.'
  if (picIds.value.length === 0) return 'Pilih minimal satu PIC.'
  if (startDate.value && dueDate.value && dueDate.value < startDate.value) {
    return 'Tenggat tidak boleh lebih awal dari tanggal mulai.'
  }
  const url = deliverableUrl.value.trim()
  if (url && !/^https?:\/\//i.test(url)) return 'Tautan deliverable harus diawali http:// atau https://.'
  return null
}

function sameIds(a: number[], b: number[]) {
  return a.length === b.length && [...a].sort().every((v, i) => v === [...b].sort()[i])
}

async function submit() {
  if (submitting.value) return
  error.value = validate()
  if (error.value) return

  const fields = {
    title: title.value.trim(),
    description: description.value.trim(),
    urgency: urgency.value,
    pic_ids: picIds.value,
    start_date: startDate.value,
    due_date: dueDate.value,
    deliverable_url: deliverableUrl.value.trim(),
  }
  submitting.value = true
  try {
    const t = props.task
    if (t) {
      const { version } = await updateTask(t.id, t.version, fields)
      // Label punya endpoint sendiri karena Staf PIC juga boleh mengubahnya; dikirim hanya bila berubah.
      if (
        !sameIds(
          labelIds.value,
          t.labels.map((l) => l.id),
        )
      )
        await setTaskLabels(t.id, version, labelIds.value)
      emit('saved', t.id, `${t.id} disimpan.`)
    } else {
      const { id } = await createTask({ ...fields, label_ids: labelIds.value })
      emit('saved', id, `${id} dibuat dan masuk ke Requested.`)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  } finally {
    submitting.value = false
  }
}

const inputClass = 'mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3'
const optionClass =
  'flex min-h-11 cursor-pointer items-center justify-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas has-[:checked]:border-accent has-[:checked]:bg-progress-bg has-[:checked]:text-progress-fg has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent'
</script>

<template>
  <BaseDialog
    :open="open"
    :title="isEdit ? `Ubah ${task?.id}` : 'Buat task'"
    variant="panel"
    @close="emit('close')"
  >
    <form id="task-form" class="space-y-5" novalidate @submit.prevent="submit">
      <p v-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">{{ error }}</p>

      <div>
        <label for="task-title" class="block text-sm font-medium">Judul (wajib)</label>
        <input
          id="task-title"
          v-model="title"
          type="text"
          maxlength="200"
          autocomplete="off"
          autofocus
          :class="inputClass"
        />
      </div>

      <fieldset>
        <legend class="text-sm font-medium">Urgency (wajib)</legend>
        <div class="mt-1 grid grid-cols-4 gap-2">
          <label v-for="u in urgencies" :key="u" :class="optionClass">
            <input v-model="urgency" type="radio" name="task-urgency" :value="u" class="sr-only" />
            {{ urgencyLabel[u] }}
          </label>
        </div>
      </fieldset>

      <fieldset>
        <legend class="text-sm font-medium">PIC (wajib, bisa lebih dari satu)</legend>
        <ul class="mt-1 divide-y divide-line rounded-card border border-line">
          <li v-for="u in picCandidates" :key="u.id">
            <label class="flex min-h-11 cursor-pointer items-center gap-3 px-3 hover:bg-canvas">
              <input
                type="checkbox"
                class="size-5 accent-accent"
                :checked="picIds.includes(u.id)"
                @change="togglePic(u.id)"
              />
              <span class="min-w-0 flex-1">{{ u.name }}</span>
              <span class="text-sm text-ink-muted">{{ u.division_name }}</span>
            </label>
          </li>
        </ul>
        <p class="mt-1 text-sm text-ink-muted" aria-live="polite">
          {{
            firstPicDivision
              ? `Divisi task: ${firstPicDivision} (mengikuti PIC pertama yang dipilih).`
              : 'Divisi task mengikuti PIC pertama yang dipilih.'
          }}
        </p>
      </fieldset>

      <details class="rounded-card border border-line" :open="isEdit">
        <summary class="flex min-h-11 cursor-pointer items-center px-3 font-medium">Detail tambahan</summary>
        <div class="space-y-4 border-t border-line p-3">
          <div>
            <label for="task-description" class="block text-sm font-medium">Deskripsi</label>
            <textarea
              id="task-description"
              v-model="description"
              rows="4"
              maxlength="5000"
              class="mt-1 block w-full rounded-card border border-line bg-surface px-3 py-2"
            />
          </div>

          <fieldset v-if="labelOptions.length > 0">
            <legend class="text-sm font-medium">Label</legend>
            <div class="mt-1 flex flex-wrap gap-2">
              <label
                v-for="l in labelOptions"
                :key="l.id"
                class="inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-chip border border-line px-3 text-sm has-[:checked]:border-accent has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent"
                :style="{ backgroundColor: l.color }"
              >
                <input
                  type="checkbox"
                  class="size-4 accent-accent"
                  :checked="labelIds.includes(l.id)"
                  @change="toggleLabel(l.id)"
                />
                {{ l.name }}
              </label>
            </div>
          </fieldset>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label for="task-start" class="block text-sm font-medium">Tanggal mulai</label>
              <input id="task-start" v-model="startDate" type="date" :class="inputClass" />
            </div>
            <div>
              <label for="task-due" class="block text-sm font-medium">Tenggat</label>
              <input
                id="task-due"
                v-model="dueDate"
                type="date"
                :min="startDate || undefined"
                :class="inputClass"
              />
            </div>
          </div>

          <div>
            <label for="task-link" class="block text-sm font-medium">Tautan deliverable</label>
            <input
              id="task-link"
              v-model="deliverableUrl"
              type="url"
              inputmode="url"
              placeholder="https://"
              :class="inputClass"
            />
          </div>
        </div>
      </details>
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
        form="task-form"
        :disabled="submitting"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Menyimpan...' : isEdit ? 'Simpan perubahan' : 'Buat task' }}
      </button>
    </template>
  </BaseDialog>
</template>
