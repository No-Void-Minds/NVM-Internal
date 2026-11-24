<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import { createUser, updateUser, type UserAccess, type UserItem } from '../api/users'
import type { DivisionItem } from '../api/divisions'
import { roleLabel, type Role } from '../utils/roles'
import { MIN_PASSWORD_LENGTH } from '../utils/password'

const props = defineProps<{ open: boolean; user: UserItem | null; divisions: DivisionItem[] }>()
const emit = defineEmits<{ close: []; saved: [message: string] }>()

const roles: Role[] = ['admin', 'manager', 'staff']
const isEdit = computed(() => props.user !== null)

const name = ref('')
const username = ref('')
const role = ref<Role>('staff')
const divisionId = ref<number | null>(null)
const password = ref('')
const access = ref<UserAccess>({ can_manage_timeline: false, can_manage_notes: false })
const showPassword = ref(false)

// Akses tambahan (revisi 38). Hanya berlaku untuk Staf: Manajer selalu punya keduanya, Admin hanya membaca
// Timeline dan Important Notes. Untuk dua peran itu kotaknya nonaktif, dan backend menyimpan false.
const accessItems: { key: keyof UserAccess; label: string; hint: string }[] = [
  {
    key: 'can_manage_timeline',
    label: 'Kelola Timeline',
    hint: 'Menambah, mengubah, dan menghapus agenda Launching, Produksi, dan Kehadiran.',
  },
  {
    key: 'can_manage_notes',
    label: 'Kelola Important Notes',
    hint: 'Menulis, mengubah, mengurutkan, dan menghapus Important Notes.',
  },
]
const accessNote = computed(() =>
  role.value === 'manager'
    ? 'Manajer selalu punya akses ini.'
    : role.value === 'admin'
      ? 'Admin hanya membaca Timeline dan Important Notes, jadi akses ini tidak berlaku.'
      : '',
)
const submitting = ref(false)
const error = ref<string | null>(null)

// Divisi nonaktif disembunyikan, kecuali divisi yang sedang dipakai akun yang diubah.
const divisionOptions = computed(() =>
  props.divisions.filter((d) => d.is_active || d.id === props.user?.division_id),
)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    error.value = null
    showPassword.value = false
    password.value = ''
    name.value = props.user?.name ?? ''
    username.value = props.user?.username ?? ''
    role.value = props.user?.role ?? 'staff'
    access.value = {
      can_manage_timeline: props.user?.can_manage_timeline ?? false,
      can_manage_notes: props.user?.can_manage_notes ?? false,
    }
    divisionId.value = props.user?.division_id ?? divisionOptions.value[0]?.id ?? null
  },
)

function validate(): string | null {
  if (!name.value.trim()) return 'Nama wajib diisi.'
  if (!isEdit.value && !/^[A-Za-z0-9._-]{3,50}$/.test(username.value.trim())) {
    return 'Username 3 sampai 50 karakter: huruf, angka, titik, garis bawah, atau tanda hubung, tanpa spasi.'
  }
  if (divisionId.value === null) return 'Pilih divisi.'
  if (!isEdit.value && [...password.value].length < MIN_PASSWORD_LENGTH) {
    return `Password awal minimal ${MIN_PASSWORD_LENGTH} karakter.`
  }
  return null
}

async function submit() {
  if (submitting.value) return
  error.value = validate()
  if (error.value || divisionId.value === null) return

  submitting.value = true
  const staff = role.value === 'staff'
  const profile = {
    name: name.value.trim(),
    role: role.value,
    division_id: divisionId.value,
    can_manage_timeline: staff && access.value.can_manage_timeline,
    can_manage_notes: staff && access.value.can_manage_notes,
  }
  try {
    if (props.user) {
      await updateUser(props.user.id, profile)
      emit('saved', `Data ${profile.name} disimpan.`)
    } else {
      await createUser({ ...profile, username: username.value.trim(), password: password.value })
      emit('saved', `Akun ${profile.name} dibuat. Berikan password awalnya secara langsung.`)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  } finally {
    submitting.value = false
  }
}

const inputClass = 'mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3'
const optionClass =
  'flex min-h-11 cursor-pointer items-center justify-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas has-[:checked]:border-mark has-[:checked]:bg-progress-bg has-[:checked]:text-progress-fg has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-mark'
</script>

<template>
  <BaseDialog
    :open="open"
    :title="isEdit ? 'Ubah pengguna' : 'Tambah pengguna'"
    variant="panel"
    @close="emit('close')"
  >
    <form id="user-form" class="space-y-5" novalidate @submit.prevent="submit">
      <p v-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">{{ error }}</p>

      <div>
        <label for="user-name" class="block text-sm font-medium">Nama</label>
        <input id="user-name" v-model="name" type="text" autocomplete="off" autofocus :class="inputClass" />
      </div>

      <div>
        <label for="user-username" class="block text-sm font-medium">Username</label>
        <input
          v-if="!isEdit"
          id="user-username"
          v-model="username"
          type="text"
          autocomplete="off"
          autocapitalize="none"
          spellcheck="false"
          aria-describedby="user-username-hint"
          :class="inputClass"
        />
        <p v-else id="user-username" class="mt-1 min-h-11 content-center break-words text-ink-muted">
          {{ username }}
        </p>
        <p id="user-username-hint" class="mt-1 text-sm text-ink-muted">
          {{ isEdit ? 'Username tidak bisa diubah.' : 'Contoh: staf.a. Dipakai untuk masuk.' }}
        </p>
      </div>

      <fieldset>
        <legend class="text-sm font-medium">Peran</legend>
        <div class="mt-1 grid grid-cols-3 gap-2">
          <label v-for="r in roles" :key="r" :class="optionClass">
            <input v-model="role" type="radio" name="user-role" :value="r" class="sr-only" />
            {{ roleLabel(r) }}
          </label>
        </div>
      </fieldset>

      <fieldset>
        <legend class="text-sm font-medium">Divisi</legend>
        <div class="mt-1 grid grid-cols-2 gap-2">
          <label v-for="d in divisionOptions" :key="d.id" :class="optionClass">
            <input v-model="divisionId" type="radio" name="user-division" :value="d.id" class="sr-only" />
            {{ d.name }}{{ d.is_active ? '' : ' (nonaktif)' }}
          </label>
        </div>
      </fieldset>

      <fieldset aria-describedby="user-access-note">
        <legend class="text-sm font-medium">Akses tambahan</legend>
        <p id="user-access-note" class="mt-1 text-sm text-ink-muted">
          {{ accessNote || 'Berlaku langsung, tanpa perlu masuk ulang.' }}
        </p>
        <div class="mt-1 space-y-1">
          <label
            v-for="a in accessItems"
            :key="a.key"
            class="flex min-h-11 items-start gap-3 rounded-card py-2"
            :class="role === 'staff' ? 'cursor-pointer' : 'cursor-not-allowed'"
          >
            <input
              v-if="role === 'staff'"
              v-model="access[a.key]"
              type="checkbox"
              :name="`user-${a.key}`"
              class="mt-0.5 size-5 shrink-0 accent-mark"
            />
            <input
              v-else
              type="checkbox"
              :name="`user-${a.key}`"
              :checked="role === 'manager'"
              disabled
              class="mt-0.5 size-5 shrink-0 accent-mark"
            />
            <span class="min-w-0">
              <span class="block text-sm font-medium">{{ a.label }}</span>
              <span class="block text-sm text-ink-muted">{{ a.hint }}</span>
            </span>
          </label>
        </div>
      </fieldset>

      <div v-if="!isEdit">
        <label for="user-password" class="block text-sm font-medium">Password awal</label>
        <div class="relative">
          <input
            id="user-password"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
            aria-describedby="user-password-hint"
            :class="[inputClass, 'pr-12']"
          />
          <button
            type="button"
            :aria-pressed="showPassword"
            aria-label="Tampilkan password"
            class="absolute inset-y-0 right-0 inline-flex w-11 items-center justify-center text-ink-muted hover:text-ink"
            @click="showPassword = !showPassword"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z" />
              <circle cx="12" cy="12" r="3" />
              <path v-if="showPassword" d="M3 3l18 18" />
            </svg>
          </button>
        </div>
        <p id="user-password-hint" class="mt-1 text-sm text-ink-muted">
          Minimal {{ MIN_PASSWORD_LENGTH }} karakter. Pengguna wajib menggantinya saat login pertama.
        </p>
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
        form="user-form"
        :disabled="submitting"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Menyimpan...' : isEdit ? 'Simpan perubahan' : 'Buat akun' }}
      </button>
    </template>
  </BaseDialog>
</template>
