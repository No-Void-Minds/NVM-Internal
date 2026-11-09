package task

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// Batas ini mencegah request yang tidak masuk akal. Angkanya bisa dinaikkan kalau Manajer butuh lebih.
const (
	maxSubtasks     = 50
	maxSubtaskTitle = 200
)

// SubtaskInput adalah satu baris sub task dari form. ID 0 berarti sub task baru.
type SubtaskInput struct {
	ID     int64
	Title  string
	PICIDs []int64
}

// subtaskSnapshot dipakai untuk audit: keadaan sub task sebelum dan sesudah perubahan.
type subtaskSnapshot struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	PICIDs []int64 `json:"pic_ids"`
}

// cleanSubtasks merapikan nama, membuang PIC ganda per sub task, dan mengembalikan gabungan PIC unik
// untuk divalidasi sekali saja ke database.
func cleanSubtasks(in []SubtaskInput) ([]SubtaskInput, []int64, error) {
	if len(in) > maxSubtasks {
		return nil, nil, apperr.Invalid(fmt.Sprintf("Satu task paling banyak punya %d sub task.", maxSubtasks))
	}
	out := make([]SubtaskInput, 0, len(in))
	seenID := make(map[int64]bool, len(in))
	seenPIC := map[int64]bool{}
	var all []int64
	for i, s := range in {
		title := strings.TrimSpace(s.Title)
		if title == "" {
			return nil, nil, apperr.Invalid(fmt.Sprintf("Nama sub task nomor %d wajib diisi.", i+1))
		}
		if utf8.RuneCountInString(title) > maxSubtaskTitle {
			return nil, nil, apperr.Invalid(fmt.Sprintf("Nama sub task nomor %d maksimal %d karakter.", i+1, maxSubtaskTitle))
		}
		if s.ID != 0 {
			if seenID[s.ID] {
				return nil, nil, apperr.Invalid("Ada sub task yang terkirim dua kali.")
			}
			seenID[s.ID] = true
		}
		pics := uniqueIDs(s.PICIDs)
		for _, p := range pics {
			if !seenPIC[p] {
				seenPIC[p] = true
				all = append(all, p)
			}
		}
		out = append(out, SubtaskInput{ID: s.ID, Title: title, PICIDs: pics})
	}
	return out, all, nil
}

// firstPIC mengikuti urutan form: sub task paling awal yang punya PIC, lalu PIC pertama di dalamnya.
func firstPIC(subs []SubtaskInput) (int64, bool) {
	for _, s := range subs {
		if len(s.PICIDs) > 0 {
			return s.PICIDs[0], true
		}
	}
	return 0, false
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

func addAssignees(ctx context.Context, q *sqlc.Queries, actorID, subtaskID int64, ids []int64) error {
	for _, uid := range ids {
		if err := q.AddSubtaskAssignee(ctx, sqlc.AddSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: uid, AssignedBy: actorID}); err != nil {
			return err
		}
	}
	return nil
}

func insertSubtasks(ctx context.Context, q *sqlc.Queries, actorID int64, taskID string, subs []SubtaskInput) ([]subtaskSnapshot, error) {
	snaps := make([]subtaskSnapshot, 0, len(subs))
	for i, s := range subs {
		id, err := q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{TaskID: taskID, Title: s.Title, Position: int32(i)})
		if err != nil {
			return nil, err
		}
		if err := addAssignees(ctx, q, actorID, id, s.PICIDs); err != nil {
			return nil, err
		}
		snaps = append(snaps, subtaskSnapshot{ID: id, Title: s.Title, PICIDs: append([]int64{}, s.PICIDs...)})
	}
	return snaps, nil
}

// loadSubtasks membaca sub task task ini beserta PIC-nya, urut sesuai posisi.
func loadSubtasks(ctx context.Context, q *sqlc.Queries, taskID string) ([]subtaskSnapshot, error) {
	rows, err := q.ListSubtasks(ctx, []string{taskID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []subtaskSnapshot{}, nil
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	pics, err := q.ListSubtaskAssignees(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64][]int64, len(rows))
	for _, p := range pics {
		byID[p.SubtaskID] = append(byID[p.SubtaskID], p.ID)
	}
	out := make([]subtaskSnapshot, len(rows))
	for i, r := range rows {
		out[i] = subtaskSnapshot{ID: r.ID, Title: r.Title, PICIDs: append([]int64{}, byID[r.ID]...)}
	}
	return out, nil
}

// SubtaskView adalah sub task untuk ditampilkan: nama dan siapa saja PIC-nya.
type SubtaskView struct {
	ID        int64
	Title     string
	Assignees []sqlc.ListSubtaskAssigneesRow
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
		out[i] = SubtaskView{ID: r.ID, Title: r.Title, Assignees: []sqlc.ListSubtaskAssigneesRow{}}
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
		if len(without(in[i].PICIDs, cur[i].PICIDs)) > 0 || len(without(cur[i].PICIDs, in[i].PICIDs)) > 0 {
			return true
		}
	}
	return false
}

// syncSubtasks menyamakan database dengan daftar dari form. Sub task yang ID-nya tidak ikut terkirim dihapus,
// ID 0 dibuat baru, sisanya diperbarui. Keunikan posisi baru dicek saat commit (constraint DEFERRABLE),
// jadi urutan penulisan di sini tidak bisa bentrok.
func syncSubtasks(ctx context.Context, q *sqlc.Queries, actorID int64, taskID string, cur []subtaskSnapshot, in []SubtaskInput) ([]subtaskSnapshot, error) {
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
			id, err = q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{TaskID: taskID, Title: s.Title, Position: int32(i)})
			if err != nil {
				return nil, err
			}
			if err := addAssignees(ctx, q, actorID, id, s.PICIDs); err != nil {
				return nil, err
			}
		} else {
			err := q.UpdateSubtask(ctx, sqlc.UpdateSubtaskParams{ID: id, TaskID: taskID, Title: s.Title, Position: int32(i)})
			if err != nil {
				return nil, err
			}
			old := existing[id].PICIDs
			if removed := without(old, s.PICIDs); len(removed) > 0 {
				if err := q.RemoveSubtaskAssignees(ctx, sqlc.RemoveSubtaskAssigneesParams{SubtaskID: id, UserIds: removed}); err != nil {
					return nil, err
				}
			}
			if err := addAssignees(ctx, q, actorID, id, without(s.PICIDs, old)); err != nil {
				return nil, err
			}
		}
		after = append(after, subtaskSnapshot{ID: id, Title: s.Title, PICIDs: append([]int64{}, s.PICIDs...)})
	}
	return after, nil
}
