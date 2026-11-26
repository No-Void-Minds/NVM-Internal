package whiteboard

import (
	"encoding/json"
	"testing"
)

// Gaya kosong dan nilai bawaan menjadi {}; potongan gambar harus di dalam gambar, potongan penuh tidak disimpan.
func TestCleanStyleDanCrop(t *testing.T) {
	for in, want := range map[string]string{
		``:                                 `{}`,
		`null`:                             `{}`,
		`{"opacity":100,"radius":"kecil"}`: `{}`,
		`{"border":"tanpa","border_color":"biru"}`: `{"border":"tanpa"}`,
	} {
		got, err := CleanStyle(json.RawMessage(in))
		if err != nil || string(got) != want {
			t.Errorf("%q: %s %v", in, got, err)
		}
	}
	for _, bad := range []Crop{{X: -0.1, W: 0.5, H: 0.5}, {X: 0.6, W: 0.5, H: 0.5}, {W: 0.01, H: 0.5}} {
		if _, err := cleanCrop(&bad); err == nil {
			t.Errorf("potongan %+v diterima", bad)
		}
	}
	if c, err := cleanCrop(&Crop{W: 1, H: 1}); c != nil || err != nil {
		t.Errorf("potongan penuh disimpan: %+v %v", c, err)
	}
	if _, _, err := CleanContent(KindImage, json.RawMessage(`{"caption":"","crop":{"x":0.5,"y":0.5,"w":0.6,"h":0.2}}`)); err == nil {
		t.Error("potongan di luar gambar diterima lewat isi kartu")
	}
}
