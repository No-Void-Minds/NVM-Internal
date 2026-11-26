package whiteboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// Revisi 54: daftar putih jenis bentuk dan varian di server harus sama persis dengan katalog yang dipakai palet dan
// renderer klien, supaya bentuk baru tidak bisa ditambah di satu sisi saja.
func TestKatalogBentuk(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "src", "utils", "wbShapeCatalog.json"))
	if err != nil {
		t.Fatalf("katalog bentuk klien tidak terbaca: %v", err)
	}
	var cat struct {
		Shapes []struct {
			ID       string `json:"id"`
			Variants []struct {
				ID string `json:"id"`
			} `json:"variants"`
		} `json:"shapes"`
	}
	if err := json.Unmarshal(raw, &cat); err != nil {
		t.Fatal(err)
	}
	client := map[string]bool{}
	for _, s := range cat.Shapes {
		if client[s.ID] {
			t.Errorf("id bentuk ganda di katalog: %s", s.ID)
		}
		client[s.ID] = true
		var vs []string
		for _, v := range s.Variants {
			vs = append(vs, v.ID)
		}
		if !slices.Equal(vs, ShapeVariants[s.ID]) {
			t.Errorf("varian %s: klien %v, server %v", s.ID, vs, ShapeVariants[s.ID])
		}
	}
	var onlyServer, onlyClient []string
	for id := range ShapeTypes {
		if !client[id] {
			onlyServer = append(onlyServer, id)
		}
	}
	for id := range client {
		if !ShapeTypes[id] {
			onlyClient = append(onlyClient, id)
		}
	}
	for id := range ShapeVariants {
		if !client[id] {
			onlyServer = append(onlyServer, id+" (varian)")
		}
	}
	sort.Strings(onlyServer)
	sort.Strings(onlyClient)
	if len(onlyServer)+len(onlyClient) > 0 {
		t.Fatalf("hanya di server: %s; hanya di klien: %s", strings.Join(onlyServer, ", "), strings.Join(onlyClient, ", "))
	}

	// Varian divalidasi per jenis bentuk.
	shape := func(s, v string) json.RawMessage {
		return json.RawMessage(`{"shape":"` + s + `","variant":"` + v + `","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`)
	}
	for _, ok := range [][2]string{{"bpmn_task", "user"}, {"bpmn_gateway", "event"}, {"bpmn_end", ""}, {"rect", ""}} {
		if _, _, err := CleanShape(shape(ok[0], ok[1])); err != nil {
			t.Errorf("%v ditolak: %v", ok, err)
		}
	}
	for _, bad := range [][2]string{{"bpmn_end", "timer"}, {"bpmn_task", "robot"}, {"rect", "user"}} {
		if _, _, err := CleanShape(shape(bad[0], bad[1])); err == nil {
			t.Errorf("%v diterima", bad)
		}
	}
}
