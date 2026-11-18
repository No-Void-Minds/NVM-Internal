<script setup lang="ts">
import { useId } from 'vue'
import type { RevisionSubtask } from '../api/tasks'

// Isian jalur revisi PIC sub task (revisi 22), dipakai dialog Pindahkan dan panel detail. Daftar pilihan
// datang dari server (revision_subtasks), jadi frontend tidak menentukan sendiri sub task mana yang boleh.
defineProps<{ choices: RevisionSubtask[] }>()
const selected = defineModel<number[]>('selected', { required: true })
const reason = defineModel<string>('reason', { required: true })

const hintId = useId()
const reasonId = useId()
const reasonHintId = useId()
</script>

<template>
  <div class="space-y-4">
    <fieldset :aria-describedby="hintId">
      <legend class="text-sm font-medium">Sub task yang direvisi</legend>
      <p :id="hintId" class="text-sm text-ink-muted">
        Centangnya dibuka lagi. Sub task lain, termasuk milik orang lain, tetap tercentang.
      </p>
      <div class="mt-1">
        <label v-for="s in choices" :key="s.id" class="flex min-h-11 cursor-pointer items-center gap-3">
          <input v-model="selected" type="checkbox" :value="s.id" class="size-5 shrink-0 accent-mark" />
          <span class="min-w-0 break-words">{{ s.title }}</span>
        </label>
      </div>
    </fieldset>
    <div>
      <label :for="reasonId" class="block text-sm font-medium">Alasan revisi (opsional)</label>
      <textarea
        :id="reasonId"
        v-model="reason"
        rows="3"
        maxlength="500"
        :aria-describedby="reasonHintId"
        class="mt-1 block w-full rounded-card border border-line bg-surface p-3"
      />
      <p :id="reasonHintId" class="mt-1 text-sm text-ink-muted">
        Tercatat di riwayat task. Maksimal 500 karakter.
      </p>
    </div>
  </div>
</template>
