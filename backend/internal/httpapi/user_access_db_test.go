package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// Revisi 38: akses tambahan per pengguna untuk Timeline dan Important Notes. Diberikan dan
// dicabut lewat PUT /api/users/:id, langsung berlaku di permintaan berikutnya pada sesi yang sama, dan tercatat
// di Audit Log. Admin tidak pernah mendapat akses tulis.
func TestAksesTambahanPerPengguna(t *testing.T) {
	tm := newTaskTeam(t)
	ctx := context.Background()
	var adminID int64
	if err := tm.pool.QueryRow(ctx, `SELECT id FROM users WHERE username = 'admin.a'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}

	event := `{"category":"launching","title":"Rilis Staf","start_date":"2026-11-20"}`
	note := `{"title":"Catatan Staf","body":"Isi"}`
	write := func(c *http.Client) (int, int) {
		t.Helper()
		return tm.sendJSON(t, c, http.MethodPost, "/api/timeline/events", event, nil),
			tm.sendJSON(t, c, http.MethodPost, "/api/timeline/notes", note, nil)
	}
	setAccess := func(id int64, name, role, access string) {
		t.Helper()
		body := fmt.Sprintf(`{"name":%q,"role":%q,"division_id":%d%s}`, name, role, tm.division, access)
		if code := tm.sendJSON(t, tm.manager, http.MethodPut, fmt.Sprintf("/api/users/%d", id), body, nil); code != http.StatusNoContent {
			t.Fatalf("ubah akses %s: %d", name, code)
		}
	}

	// Staf tanpa akses: ditolak menulis.
	if ev, nt := write(tm.pic); ev != http.StatusForbidden || nt != http.StatusForbidden {
		t.Fatalf("Staf tanpa akses: event %d catatan %d, mau 403", ev, nt)
	}

	// Diberi kedua akses: sesi yang sama langsung boleh, tanpa login ulang.
	setAccess(tm.picID, "staf.a", "staff", `,"can_manage_timeline":true,"can_manage_notes":true`)
	if ev, nt := write(tm.pic); ev != http.StatusCreated || nt != http.StatusCreated {
		t.Fatalf("Staf dengan akses: event %d catatan %d, mau 201", ev, nt)
	}
	var canWrite struct {
		CanWrite bool `json:"can_write"`
	}
	tm.sendJSON(t, tm.pic, http.MethodGet, "/api/timeline/notes", "", &canWrite)
	if !canWrite.CanWrite {
		t.Fatal("can_write catatan harus true untuk Staf dengan akses")
	}
	// Staf lain tetap tanpa akses.
	if ev, nt := write(tm.other); ev != http.StatusForbidden || nt != http.StatusForbidden {
		t.Fatalf("Staf lain: event %d catatan %d, mau 403", ev, nt)
	}

	// Mencabut satu akses saja; akses yang tidak dikirim tidak berubah.
	setAccess(tm.picID, "staf.a", "staff", `,"can_manage_timeline":false`)
	if ev, nt := write(tm.pic); ev != http.StatusForbidden || nt != http.StatusCreated {
		t.Fatalf("setelah Kelola Timeline dicabut: event %d (mau 403) catatan %d (mau 201)", ev, nt)
	}
	setAccess(tm.picID, "staf.a", "staff", "")
	if _, nt := write(tm.pic); nt != http.StatusCreated {
		t.Fatalf("PUT tanpa kolom akses tidak boleh mencabut Kelola Important Notes: catatan %d", nt)
	}

	// Audit mencatat akses apa, sebelum dan sesudah, dan siapa pelakunya.
	var before, after bool
	if err := tm.pool.QueryRow(ctx, `SELECT (old_value->>'can_manage_timeline')::bool, (new_value->>'can_manage_timeline')::bool
		FROM audit_logs WHERE action = 'user_updated' AND entity_id = $1 AND new_value ? 'can_manage_timeline'
		ORDER BY id DESC LIMIT 1`, fmt.Sprint(tm.picID)).Scan(&before, &after); err != nil || !before || after {
		t.Fatalf("audit pencabutan: sebelum %v sesudah %v err %v", before, after, err)
	}

	// Admin: akses yang dikirim diabaikan dan tetap ditolak menulis. Manajer selalu boleh.
	setAccess(adminID, "admin.a", "admin", `,"can_manage_timeline":true,"can_manage_notes":true`)
	if ev, nt := write(tm.admin); ev != http.StatusForbidden || nt != http.StatusForbidden {
		t.Fatalf("Admin: event %d catatan %d, mau 403", ev, nt)
	}
	if ev, nt := write(tm.manager); ev != http.StatusCreated || nt != http.StatusCreated {
		t.Fatalf("Manajer: event %d catatan %d, mau 201", ev, nt)
	}

	// Staf yang dijadikan Manajer kehilangan akses tersimpannya (CHECK users_access_staff_only).
	setAccess(tm.picID, "staf.a", "manager", "")
	var stored bool
	if err := tm.pool.QueryRow(ctx, `SELECT can_manage_notes FROM users WHERE id = $1`, tm.picID).Scan(&stored); err != nil || stored {
		t.Fatalf("akses tersimpan untuk Manajer: %v err %v", stored, err)
	}
}
