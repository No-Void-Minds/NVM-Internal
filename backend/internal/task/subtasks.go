package task

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
	"slices"
	"strings"
	"unicode/utf8"
)

// Batas ini mencegah request yang tidak masuk akal. Angkanya bisa dinaikkan kalau Manajer butuh lebih.
const (
	maxSubtasks     = 50
	maxSubtaskTitle = 200
	maxSubtaskDesc  = 2000
)

// SubtaskInput adalah satu baris sub task dari form Manajer. ID 0 berarti sub task baru.
// Sejak revisi 6 form tidak memilih PIC per sub task: PIC dipilih di task induk.
type SubtaskInput struct {
	ID    int64
	Title string
}

// subtaskSnapshot dipakai untuk audit: keadaan sub task sebelum dan sesudah perubahan.
type subtaskSnapshot struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// cleanSubtaskDesc memakai batas yang sama dengan deskripsi task, supaya satu halaman cukup membacanya.
func cleanSubtaskDesc(desc string) (string, error) {
	d := strings.TrimSpace(desc)
	if utf8.RuneCountInString(d) > maxSubtaskDesc {
		return "", apperr.Invalid(fmt.Sprintf("Deskripsi sub task maksimal %d karakter.", maxSubtaskDesc))
	}
	return d, nil
}

// cleanSubtaskTitle dipakai form Manajer dan penambahan sub task oleh PIC, supaya aturan namanya satu.
func cleanSubtaskTitle(title, label string) (string, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return "", apperr.Invalid(fmt.Sprintf("Nama sub task %s wajib diisi.", label))
	}
	if utf8.RuneCountInString(t) > maxSubtaskTitle {
		return "", apperr.Invalid(fmt.Sprintf("Nama sub task %s maksimal %d karakter.", label, maxSubtaskTitle))
	}
	return t, nil
}

func cleanSubtasks(in []SubtaskInput) ([]SubtaskInput, error) {
	if len(in) > maxSubtasks {
		return nil, apperr.Invalid(fmt.Sprintf("Satu task paling banyak punya %d sub task.", maxSubtasks))
	}
	out := make([]SubtaskInput, 0, len(in))
	seenID := make(map[int64]bool, len(in))
	for i, s := range in {
		title, err := cleanSubtaskTitle(s.Title, fmt.Sprintf("nomor %d", i+1))
		if err != nil {
			return nil, err
		}
		if s.ID != 0 {
			if seenID[s.ID] {
				return nil, apperr.Invalid("Ada sub task yang terkirim dua kali.")
			}
			seenID[s.ID] = true
		}
		out = append(out, SubtaskInput{ID: s.ID, Title: title})
	}
	return out, nil
}

// userRef membungkus id akun untuk kolom yang boleh kosong. Nol berarti tidak diketahui, yang terjadi
// pada sub task buatan sebelum revisi 7.
func userRef(id int64) pgtype.Int8 {
	return pgtype.Int8{Int64: id, Valid: id != 0}
}

// without mengembalikan isi a yang tidak ada di b.
func without(a, b []int64) []int64 {
	var out []int64
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

// loadSubtasks membaca sub task task ini beserta PIC-nya, urut sesuai posisi.
func loadSubtasks(ctx context.Context, q *sqlc.Queries, taskID string) ([]subtaskSnapshot, error) {
	rows, err := q.ListSubtasks(ctx, []string{taskID})
	if err != nil {
		return nil, err
	}
	out := make([]subtaskSnapshot, len(rows))
	for i, r := range rows {
		out[i] = subtaskSnapshot{ID: r.ID, Title: r.Title}
	}
	return out, nil
}

// SubtaskView adalah sub task untuk ditampilkan: nama dan siapa saja PIC-nya.
type SubtaskView struct {
	ID             int64
	Title          string
	Description    string
	ReferenceURL   string
	DeliverableURL string
	CreatedBy      int64
	CreatedByName  string
	DoneAt         pgtype.Timestamptz
	DoneBy         string
	DueDate        pgtype.Date
	Assignees      []sqlc.ListSubtaskAssigneesRow
}

func loadSubtaskViews(ctx context.Context, q *sqlc.Queries, taskID string) ([]SubtaskView, error) {
	rows, err := q.ListSubtasks(ctx, []string{taskID})
	if err != nil {
		return nil, err
	}
	out := make([]SubtaskView, len(rows))
	ids := make([]int64, len(rows))
	index := make(map[int64]int, len(rows))
	for i, r := range rows {
		out[i] = SubtaskView{
			ID: r.ID, Title: r.Title, Description: r.Description,
			ReferenceURL: r.ReferenceUrl, DeliverableURL: r.DeliverableUrl,
			CreatedBy: r.CreatedBy.Int64, CreatedByName: r.CreatedByName.String,
			DoneAt: r.DoneAt, DoneBy: r.DoneByName.String, DueDate: r.DueDate,
			Assignees: []sqlc.ListSubtaskAssigneesRow{},
		}
		ids[i] = r.ID
		index[r.ID] = i
	}
	if len(ids) == 0 {
		return out, nil
	}
	pics, err := q.ListSubtaskAssignees(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range pics {
		i := index[p.SubtaskID]
		out[i].Assignees = append(out[i].Assignees, p)
	}
	return out, nil
}

// subtasksChanged membandingkan daftar di database dengan daftar dari form tanpa menulis apa pun,
// supaya simpan tanpa perubahan tidak menaikkan versi task. PIC dibandingkan sebagai himpunan.
func subtasksChanged(cur []subtaskSnapshot, in []SubtaskInput) bool {
	if len(cur) != len(in) {
		return true
	}
	for i := range cur {
		if in[i].ID != cur[i].ID || in[i].Title != cur[i].Title {
			return true
		}
	}
	return false
}

// syncSubtasks menyamakan database dengan daftar dari form Manajer. Sub task yang ID-nya tidak ikut
// terkirim dihapus, ID 0 dibuat baru, sisanya diperbarui. Pemilik sub task tidak disentuh di sini. Keunikan posisi baru dicek saat commit (constraint DEFERRABLE),
// jadi urutan penulisan di sini tidak bisa bentrok.
func syncSubtasks(ctx context.Context, q *sqlc.Queries, taskID string, creatorID int64, cur []subtaskSnapshot, in []SubtaskInput) ([]subtaskSnapshot, error) {
	existing := make(map[int64]subtaskSnapshot, len(cur))
	for _, c := range cur {
		existing[c.ID] = c
	}
	wanted := make(map[int64]bool, len(in))
	for _, s := range in {
		if s.ID == 0 {
			continue
		}
		if _, ok := existing[s.ID]; !ok {
			return nil, apperr.Invalid("Ada sub task yang bukan milik task ini atau sudah dihapus. Muat ulang lalu coba lagi.")
		}
		wanted[s.ID] = true
	}
	var gone []int64
	for _, c := range cur {
		if !wanted[c.ID] {
			gone = append(gone, c.ID)
		}
	}
	if len(gone) > 0 {
		if err := q.DeleteSubtasks(ctx, sqlc.DeleteSubtasksParams{TaskID: taskID, Ids: gone}); err != nil {
			return nil, err
		}
	}

	after := make([]subtaskSnapshot, 0, len(in))
	for i, s := range in {
		id := s.ID
		if id == 0 {
			var err error
			id, err = q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{
				TaskID: taskID, Title: s.Title, Position: int32(i), CreatedBy: userRef(creatorID),
			})
			if err != nil {
				return nil, err
			}
		} else {
			err := q.UpdateSubtask(ctx, sqlc.UpdateSubtaskParams{ID: id, TaskID: taskID, Title: s.Title, Position: int32(i)})
			if err != nil {
				return nil, err
			}
		}
		after = append(after, subtaskSnapshot{ID: id, Title: s.Title})
	}
	return after, nil
}

// subtaskEvent satu baris riwayat. Nilainya map supaya bentuknya sama dengan audit lain.
type subtaskEvent struct {
	action string
	oldVal map[string]any
	newVal map[string]any
}

func snapMap(s subtaskSnapshot) map[string]any {
	return map[string]any{"id": s.ID, "title": s.Title}
}

// subtaskEvents menurunkan kejadian audit dengan membandingkan keadaan sebelum dan sesudah.
// Urutan hanya dibandingkan di antara sub task yang bertahan, supaya menambah atau menghapus
// satu sub task tidak ikut tercatat sebagai pengurutan ulang.
func subtaskEvents(before, after []subtaskSnapshot) []subtaskEvent {
	prev := make(map[int64]subtaskSnapshot, len(before))
	for _, b := range before {
		prev[b.ID] = b
	}
	kept := make(map[int64]bool, len(after))
	for _, a := range after {
		kept[a.ID] = true
	}

	var ev []subtaskEvent
	var orderBefore, orderAfter []int64
	for _, b := range before {
		if !kept[b.ID] {
			ev = append(ev, subtaskEvent{"subtask_removed", snapMap(b), nil})
		} else {
			orderBefore = append(orderBefore, b.ID)
		}
	}
	for _, a := range after {
		b, existed := prev[a.ID]
		if !existed {
			ev = append(ev, subtaskEvent{"subtask_added", nil, snapMap(a)})
			continue
		}
		orderAfter = append(orderAfter, a.ID)
		if b.Title != a.Title {
			ev = append(ev, subtaskEvent{"subtask_renamed",
				map[string]any{"id": b.ID, "title": b.Title}, map[string]any{"id": a.ID, "title": a.Title}})
		}
	}
	if !slices.Equal(orderBefore, orderAfter) {
		ev = append(ev, subtaskEvent{"subtask_reordered",
			map[string]any{"order": orderBefore}, map[string]any{"order": orderAfter}})
	}
	return ev
}
