// Package excelimport membaca sheet Task Tracker dan Completed Task dari "NVM - Marketing & Sales.xlsx" (S12-01).
// Berkas .xlsx dibaca langsung sebagai zip berisi XML dengan pustaka standar, supaya tidak menambah dependensi
// untuk impor yang dijalankan sekali.
package excelimport

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// Batas ukuran per bagian XML, supaya berkas rusak atau zip bomb tidak menghabiskan memori.
const maxPartSize = 64 << 20

// readSheets mengembalikan isi setiap sheet yang namanya diminta, sebagai baris dan sel teks. Sheet yang tidak
// ada tidak masuk peta; pemanggil yang memutuskan apakah itu galat.
func readSheets(file string, names ...string) (map[string][][]string, error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("berkas bukan .xlsx yang bisa dibaca: %w", err)
	}
	defer func() { _ = zr.Close() }()
	parts := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		parts[f.Name] = f
	}

	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := decodePart(parts, "xl/workbook.xml", &wb); err != nil {
		return nil, err
	}
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := decodePart(parts, "xl/_rels/workbook.xml.rels", &rels); err != nil {
		return nil, err
	}
	targets := map[string]string{}
	for _, r := range rels.Rel {
		// Target bisa relatif ke xl/ ("worksheets/sheet1.xml") atau absolut ("/xl/worksheets/sheet1.xml").
		t := strings.TrimPrefix(r.Target, "/")
		if !strings.HasPrefix(t, "xl/") {
			t = path.Join("xl", t)
		}
		targets[r.ID] = t
	}

	shared, err := sharedStrings(parts)
	if err != nil {
		return nil, err
	}

	want := map[string]bool{}
	for _, n := range names {
		want[strings.ToLower(n)] = true
	}
	out := map[string][][]string{}
	for _, s := range wb.Sheets {
		if !want[strings.ToLower(strings.TrimSpace(s.Name))] {
			continue
		}
		rows, err := sheetRows(parts, targets[s.RID], shared)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", s.Name, err)
		}
		out[strings.TrimSpace(s.Name)] = rows
	}
	return out, nil
}

func decodePart(parts map[string]*zip.File, name string, v any) error {
	f, ok := parts[name]
	if !ok {
		return fmt.Errorf("bagian %s tidak ada di berkas", name)
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return xml.NewDecoder(io.LimitReader(rc, maxPartSize)).Decode(v)
}

// richText mengumpulkan teks sel: langsung di <t>, atau dipecah per format di <r><t>.
type richText struct {
	T string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (r richText) String() string {
	var b strings.Builder
	b.WriteString(r.T)
	for _, run := range r.R {
		b.WriteString(run.T)
	}
	return b.String()
}

func sharedStrings(parts map[string]*zip.File) ([]string, error) {
	if _, ok := parts["xl/sharedStrings.xml"]; !ok {
		return nil, nil
	}
	var sst struct {
		SI []richText `xml:"si"`
	}
	if err := decodePart(parts, "xl/sharedStrings.xml", &sst); err != nil {
		return nil, err
	}
	out := make([]string, len(sst.SI))
	for i, si := range sst.SI {
		out[i] = si.String()
	}
	return out, nil
}

func sheetRows(parts map[string]*zip.File, name string, shared []string) ([][]string, error) {
	var ws struct {
		Rows []struct {
			Cells []struct {
				Ref    string   `xml:"r,attr"`
				Type   string   `xml:"t,attr"`
				Value  string   `xml:"v"`
				Inline richText `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := decodePart(parts, name, &ws); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(ws.Rows))
	for _, r := range ws.Rows {
		var row []string
		for i, c := range r.Cells {
			col := i
			if c.Ref != "" {
				var err error
				if col, err = columnIndex(c.Ref); err != nil {
					return nil, err
				}
			}
			for len(row) <= col {
				row = append(row, "")
			}
			switch c.Type {
			case "s":
				n, err := strconv.Atoi(c.Value)
				if err != nil || n < 0 || n >= len(shared) {
					return nil, fmt.Errorf("sel %s menunjuk teks bersama yang tidak ada", c.Ref)
				}
				row[col] = shared[n]
			case "inlineStr":
				row[col] = c.Inline.String()
			default:
				// n (angka, termasuk tanggal sebagai nomor seri), str (hasil rumus), b (boolean).
				row[col] = c.Value
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// columnIndex mengubah referensi sel ("C12") menjadi indeks kolom dari 0.
func columnIndex(ref string) (int, error) {
	n := 0
	for _, ch := range ref {
		if ch >= 'A' && ch <= 'Z' {
			n = n*26 + int(ch-'A'+1)
			continue
		}
		break
	}
	if n == 0 {
		return 0, errors.New("referensi sel tidak dikenal: " + ref)
	}
	return n - 1, nil
}
