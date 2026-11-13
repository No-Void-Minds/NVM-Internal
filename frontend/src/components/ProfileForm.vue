<script setup lang="ts">
import { computed, ref } from 'vue'
import UserAvatar from './UserAvatar.vue'
import { ApiError } from '../api/client'
import { avatarTypes, deleteAvatar, maxAvatarBytes, updateProfile, uploadAvatar } from '../api/profile'
import { useSessionStore } from '../stores/session'
import { roleLabel } from '../utils/roles'

const emit = defineEmits<{ saved: [message: string] }>()

const session = useSessionStore()
const user = computed(() => session.user)

const name = ref(user.value?.name ?? '')
const jobTitle = ref(user.value?.job_title ?? '')
const errors = ref<{ name?: string; job_title?: string; form?: string }>({})
const saving = ref(false)
const saved = ref(false)

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

// Data sesi dimuat ulang supaya top bar, kartu profil, dan form langsung memakai nilai baru.
// Kalau muat ulang gagal, perubahan tetap sudah tersimpan, jadi tidak dilaporkan sebagai galat simpan.
async function refreshSession() {
  try {
    await session.fetchMe()
  } catch {
    // Nilai baru akan terlihat setelah halaman dimuat ulang.
  }
}

async function submit() {
  if (saving.value) return
  saved.value = false
  errors.value = {}
  if (!name.value.trim()) errors.value.name = 'Nama wajib diisi.'
  if ([...jobTitle.value.trim()].length > 100) errors.value.job_title = 'Jabatan maksimal 100 karakter.'
  if (errors.value.name || errors.value.job_title) return

  saving.value = true
  try {
    await updateProfile({ name: name.value, job_title: jobTitle.value })
    await refreshSession()
    name.value = user.value?.name ?? name.value
    jobTitle.value = user.value?.job_title ?? ''
    saved.value = true
    emit('saved', 'Profil tersimpan.')
  } catch (err) {
    const field = err instanceof ApiError ? err.field : null
    if (field === 'name' || field === 'job_title') errors.value[field] = messageOf(err)
    else errors.value.form = messageOf(err)
  } finally {
    saving.value = false
  }
}

const fileEl = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const progress = ref(0)
const removing = ref(false)
const photoError = ref('')
const photoNotice = ref('')

async function onFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  // Dikosongkan supaya memilih berkas yang sama lagi tetap memicu change.
  input.value = ''
  if (!file) return
  photoError.value = ''
  photoNotice.value = ''
  if (!avatarTypes.includes(file.type)) {
    photoError.value = 'Foto harus berupa berkas JPG atau PNG.'
    return
  }
  if (file.size > maxAvatarBytes) {
    photoError.value = 'Foto melebihi batas 2 MB. Pilih foto yang lebih kecil.'
    return
  }
  uploading.value = true
  progress.value = 0
  try {
    await uploadAvatar(file, (f) => (progress.value = Math.round(f * 100)))
    await refreshSession()
    photoNotice.value = 'Foto profil diganti.'
  } catch (err) {
    photoError.value = messageOf(err)
  } finally {
    uploading.value = false
  }
}

async function removePhoto() {
  if (removing.value) return
  photoError.value = ''
  photoNotice.value = ''
  removing.value = true
  try {
    await deleteAvatar()
    await refreshSession()
    photoNotice.value = 'Foto profil dihapus. Inisial namamu dipakai sebagai gantinya.'
  } catch (err) {
    photoError.value = messageOf(err)
  } finally {
    removing.value = false
  }
}

const inputClass = 'mt-1 block min-h-11 w-full min-w-0 rounded-card border bg-surface px-3'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60'
</script>

<template>
  <div v-if="user" class="space-y-6">
    <section aria-labelledby="profile-photo-title">
      <h3 id="profile-photo-title" class="font-semibold">Foto profil</h3>
      <div class="mt-3 flex flex-wrap items-center gap-4">
        <UserAvatar :user-id="user.id" :name="user.name" :avatar-id="user.avatar_id" size="xl" />
        <div class="flex min-w-0 flex-wrap gap-2">
          <input
            ref="fileEl"
            type="file"
            accept="image/jpeg,image/png"
            class="sr-only"
            tabindex="-1"
            aria-hidden="true"
            @change="onFile"
          />
          <button
            type="button"
            :class="buttonSecondary"
            :disabled="uploading || removing"
            aria-describedby="profile-photo-hint"
            @click="fileEl?.click()"
          >
            {{ user.avatar_id ? 'Ganti foto' : 'Pilih foto' }}
          </button>
          <button
            v-if="user.avatar_id"
            type="button"
            :class="buttonSecondary"
            :disabled="uploading || removing"
            @click="removePhoto"
          >
            {{ removing ? 'Menghapus...' : 'Hapus foto' }}
          </button>
        </div>
      </div>
      <p id="profile-photo-hint" class="mt-2 text-sm text-ink-muted">
        JPG atau PNG, maksimal 2 MB. Foto dipotong persegi dari bagian tengah.
      </p>
      <div v-if="uploading" class="mt-2" role="status">
        <p class="text-sm">Mengunggah foto... {{ progress }}%</p>
        <progress
          class="mt-1 h-2 w-full accent-accent"
          max="100"
          :value="progress"
          aria-label="Progres unggah foto"
        >
          {{ progress }}%
        </progress>
      </div>
      <p v-if="photoError" role="alert" class="mt-2 rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ photoError }}
      </p>
      <p v-else-if="photoNotice" role="status" class="mt-2 text-sm text-progress-fg">{{ photoNotice }}</p>
    </section>

    <form class="space-y-4" novalidate @submit.prevent="submit">
      <p v-if="errors.form" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ errors.form }}
      </p>
      <p
        v-if="saved"
        role="status"
        class="rounded-card border border-line bg-surface p-3 text-sm text-progress-fg"
      >
        Profil tersimpan.
      </p>

      <div>
        <label for="profile-name" class="block text-sm font-medium">Nama</label>
        <input
          id="profile-name"
          v-model="name"
          type="text"
          autocomplete="name"
          maxlength="100"
          :aria-invalid="errors.name ? 'true' : undefined"
          :aria-describedby="errors.name ? 'profile-name-error' : undefined"
          :class="[inputClass, errors.name ? 'border-danger' : 'border-line']"
        />
        <p v-if="errors.name" id="profile-name-error" class="mt-1 text-sm text-danger">{{ errors.name }}</p>
      </div>

      <div>
        <label for="profile-job" class="block text-sm font-medium">Jabatan</label>
        <input
          id="profile-job"
          v-model="jobTitle"
          type="text"
          autocomplete="organization-title"
          maxlength="100"
          :aria-invalid="errors.job_title ? 'true' : undefined"
          :aria-describedby="errors.job_title ? 'profile-job-error' : 'profile-job-hint'"
          :class="[inputClass, errors.job_title ? 'border-danger' : 'border-line']"
        />
        <p v-if="errors.job_title" id="profile-job-error" class="mt-1 text-sm text-danger">
          {{ errors.job_title }}
        </p>
        <p v-else id="profile-job-hint" class="mt-1 text-sm text-ink-muted">
          Tampil di kartu profil. Boleh dikosongkan.
        </p>
      </div>

      <dl
        class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-6 gap-y-2 rounded-card border border-line bg-canvas p-3"
      >
        <dt class="text-ink-muted">Username</dt>
        <dd class="min-w-0 break-words">{{ user.username }}</dd>
        <dt class="text-ink-muted">Divisi</dt>
        <dd>{{ user.division_name }}</dd>
        <dt class="text-ink-muted">Peran</dt>
        <dd>{{ roleLabel(user.role) }}</dd>
      </dl>
      <p class="text-sm text-ink-muted">
        Username, divisi, dan peran diatur Admin atau Manajer di halaman Pengguna.
      </p>

      <button
        type="submit"
        :disabled="saving"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ saving ? 'Menyimpan...' : 'Simpan profil' }}
      </button>
    </form>
  </div>
</template>
