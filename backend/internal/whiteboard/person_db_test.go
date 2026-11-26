package whiteboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Revisi 51: kartu orang hanya menyimpan user_id dan tampilan; pengguna yang tidak ada ditolak; papan selalu memuat
// data profil terkini (nama, jabatan, divisi, foto); pengguna nonaktif tetap tampil dengan status nonaktif; nama
// terkini ikut pencarian; template membawa kartu orang; email, username, dan peran tidak pernah dikirim.
func TestWbKartuOrang(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Tim").Ref.ID
	content := func(id int64) string {
		return fmt.Sprintf(`{"user_id":%d,"view":"lengkap","size":"sedang","layout":"samping"}`, id)
	}

	_, err := f.svc.CreateCard(ctx, f.rina, b, CardInput{Kind: KindPerson, W: 260, H: 88, Content: json.RawMessage(content(987654321))})
	wantInvalid(t, err, "tidak ditemukan")
	_, err = f.svc.CreateCard(ctx, f.rina, b, CardInput{Kind: KindPerson, W: 260, H: 88, Content: json.RawMessage(`{"user_id":1,"view":"poster","size":"sedang","layout":"samping"}`)})
	wantInvalid(t, err, "Tampilan kartu orang")

	card := f.card(t, b, KindPerson, content(f.nala.ID))
	if strings.Contains(string(card.Content), "nala") || card.Person == nil || card.Person.Name != "nala" || card.Person.Division != "Marketing" || !card.Person.Active {
		t.Fatalf("kartu orang %s %+v", card.Content, card.Person)
	}

	// Profil berubah setelah kartu dibuat: papan memuat data terkini, tanpa mengubah isi kartu.
	if _, err := f.pool.Exec(ctx, `UPDATE users SET name = 'Nala Putri', job_title = 'Kepala Konten', is_active = false WHERE id = $1`, f.nala.ID); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.Board(ctx, f.rina, b)
	if err != nil || len(v.Cards) != 1 {
		t.Fatal(err)
	}
	p := v.Cards[0].Person
	if p == nil || p.Name != "Nala Putri" || p.JobTitle == nil || *p.JobTitle != "Kepala Konten" || p.Active {
		t.Fatalf("data orang tidak terkini: %+v", p)
	}
	raw, _ := json.Marshal(v.Cards[0])
	for _, bad := range []string{"username", "email", "role", "manager"} {
		if strings.Contains(string(raw), bad) {
			t.Errorf("kartu orang membocorkan %q: %s", bad, raw)
		}
	}

	// Daftar alat Tim memuat pengguna nonaktif dengan statusnya.
	people, err := f.svc.People(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pp := range people {
		if pp.ID == f.nala.ID {
			found = !pp.Active && pp.Name == "Nala Putri"
		}
	}
	if !found {
		t.Fatalf("pengguna nonaktif tidak ada di daftar Tim: %+v", people)
	}

	// Pencarian memakai nama terkini; isi kartu tidak berubah.
	hits, err := f.svc.SearchCards(ctx, "Nala Putri")
	if err != nil || len(hits) != 1 || hits[0].Kind != KindPerson {
		t.Fatalf("cari nama %+v %v", hits, err)
	}

	// Template membawa kartu orang dengan user_id yang sama.
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, true); err != nil {
		t.Fatal(err)
	}
	cp, err := f.svc.FromTemplate(ctx, f.rina, RootID, TemplateInput{TemplateID: b, X: 500, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	v, _ = f.svc.Board(ctx, f.rina, cp.Ref.ID)
	if len(v.Cards) != 1 || v.Cards[0].Person == nil || v.Cards[0].Person.ID != f.nala.ID {
		t.Fatalf("salinan template kartu orang %+v", v.Cards)
	}
}
