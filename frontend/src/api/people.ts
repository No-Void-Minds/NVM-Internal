import { apiGet } from './client'
import { listPeople } from './board'
import type { Role } from '../utils/roles'

// Kandidat PIC di form task: hanya kolom yang benar-benar dipakai formnya. Manajer memakai daftar dari
// /api/users, Staf memakai /api/people yang terbuka untuk semua yang login (revisi 27). status opsional,
// karena /api/people hanya berisi akun aktif.
export type PicCandidate = {
  id: number
  name: string
  role: string
  division_name: string
  avatar_id: string | null
  status?: string
}

// Isi kartu profil dari GET /api/people/:id. Sengaja tanpa username: endpoint ini terbuka untuk semua yang login.
export type PersonCard = {
  id: number
  name: string
  division_name: string
  job_title: string | null
  role: Role
  is_active: boolean
  avatar_url: string | null
}

// Cache per id selama halaman terbuka. Yang gagal dibuang dari cache supaya Coba lagi benar-benar meminta ulang.
const cards = new Map<number, Promise<PersonCard>>()

export function getPersonCard(id: number): Promise<PersonCard> {
  let p = cards.get(id)
  if (!p) {
    p = apiGet<PersonCard>(`/people/${id}`)
    p.catch(() => cards.delete(id))
    cards.set(id, p)
  }
  return p
}

let people: Promise<Map<number, string>> | null = null

// Filter PIC papan memakai username (S4-06), jadi tautan "Lihat task-nya di papan" mencarinya di /people,
// daftar yang memang sudah dipakai filter itu. null bila orangnya bukan calon PIC (Admin, akun nonaktif).
export async function boardUsername(id: number): Promise<string | null> {
  if (!people) {
    people = listPeople().then((rows) => new Map(rows.map((r) => [r.id, r.username])))
    people.catch(() => (people = null))
  }
  return (await people).get(id) ?? null
}
