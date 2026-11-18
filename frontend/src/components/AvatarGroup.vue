<script setup lang="ts">
import { computed, useId } from 'vue'
import PersonChip from './PersonChip.vue'
import UserAvatar from './UserAvatar.vue'
import type { TaskPerson } from '../api/tasks'

// Satu kelompok bulatan berlabel di kartu papan (revisi 22): "PIC" atau "Sub task". Maksimal empat bulatan,
// sisanya "+n" dengan nama lengkap di title dan aria-label.
const props = defineProps<{ label: string; people: TaskPerson[] }>()

const MAX = 4
const shown = computed(() => props.people.slice(0, MAX))
const rest = computed(() => props.people.slice(MAX))
const restNames = computed(() => rest.value.map((p) => p.name).join(', '))
const labelId = useId()
</script>

<template>
  <div class="min-w-0">
    <p :id="labelId" class="text-xs text-ink-muted">{{ label }}</p>
    <!-- Bulatan tidak bertumpuk: area klik 44 px tiap bulatan butuh jarak 12 px di antara bulatan 32 px, supaya
         tidak ada bulatan yang tertutup area klik tetangganya (revisi 14). z-10 di atas lapisan judul kartu. -->
    <ul class="mt-1 flex flex-wrap items-center gap-3" :aria-labelledby="labelId">
      <li v-for="p in shown" :key="p.id" :title="p.name">
        <PersonChip :user-id="p.id" :name="p.name" class="z-10 rounded-full">
          <UserAvatar :user-id="p.id" :name="p.name" :avatar-id="p.avatar_id" size="md" decorative />
        </PersonChip>
      </li>
      <li v-if="rest.length">
        <!-- relative z-10 supaya title tetap muncul saat disorot, bukan tertutup lapisan klik judul. -->
        <span
          role="img"
          :title="restNames"
          :aria-label="`${rest.length} lainnya: ${restNames}`"
          class="relative z-10 inline-flex size-8 items-center justify-center rounded-full border border-line font-mono text-xs"
        >
          +{{ rest.length }}
        </span>
      </li>
    </ul>
  </div>
</template>
