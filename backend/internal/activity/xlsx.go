package activity

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Berkas .xlsx ditulis langsung sebagai zip berisi XML dengan pustaka standar, sama seperti pembaca di
// internal/excelimport, supaya ekspor ini tidak menambah dependensi. Semua sel memakai inlineStr, jadi tidak
// ada tabel sharedStrings yang perlu dijaga.

const (
	styleNormal = 0
	styleBold   = 1
	styleWrap   = 2
)

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Daily Activity" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

// Tiga gaya saja: biasa, tebal untuk tiga baris kepala, dan rata atas dengan teks dilipat untuk isi.
const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>
<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>
<borders count="1"><border/></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="3">
<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>
<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf>
</cellXfs>
</styleSheet>`

func writeXLSX(rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"xl/workbook.xml", workbook},
		{"xl/_rels/workbook.xml.rels", workbookRels},
		{"xl/styles.xml", styles},
		{"xl/worksheets/sheet1.xml", sheetXML(rows)},
	}
	for _, p := range parts {
		w, err := zw.Create(p.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(p.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sheetXML(rows [][]string) string {
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	// Kolom Week, Date, dan Time sempit; kolom Activity dan Output lebar karena isinya kalimat.
	b.WriteString(`<sheetViews><sheetView workbookViewId="0">`)
	b.WriteString(`<pane xSplit="3" ySplit="3" topLeftCell="D4" activePane="bottomRight" state="frozen"/>`)
	b.WriteString(`</sheetView></sheetViews>`)
	b.WriteString(`<cols><col min="1" max="1" width="10" customWidth="1"/><col min="2" max="2" width="16" customWidth="1"/>`)
	b.WriteString(`<col min="3" max="3" width="13" customWidth="1"/>`)
	if width > 3 {
		fmt.Fprintf(&b, `<col min="4" max="%d" width="34" customWidth="1"/>`, width)
	}
	b.WriteString(`</cols><sheetData>`)
	for i, r := range rows {
		style := styleWrap
		if i < 3 {
			style = styleBold
		}
		fmt.Fprintf(&b, `<row r="%d">`, i+1)
		for c, v := range r {
			if v == "" {
				continue
			}
			fmt.Fprintf(&b, `<c r="%s%d" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`,
				colName(c), i+1, style, escape(v))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)

	merges := []string{}
	if width > 1 && len(rows) > 0 && rows[0][0] != "" {
		merges = append(merges, fmt.Sprintf("A1:%s1", colName(width-1)))
	}
	// Nama orang berdiri di atas dua kolomnya, Activity dan Output.
	if len(rows) > 1 {
		for c, v := range rows[1] {
			if v != "" && c+1 < width {
				merges = append(merges, fmt.Sprintf("%s2:%s2", colName(c), colName(c+1)))
			}
		}
	}
	if len(merges) > 0 {
		fmt.Fprintf(&b, `<mergeCells count="%d">`, len(merges))
		for _, m := range merges {
			fmt.Fprintf(&b, `<mergeCell ref="%s"/>`, m)
		}
		b.WriteString(`</mergeCells>`)
	}
	b.WriteString(`</worksheet>`)
	return b.String()
}

// colName mengubah indeks kolom 0 menjadi A, 26 menjadi AA, dan seterusnya.
func colName(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}

func escape(v string) string {
	var b bytes.Buffer
	// EscapeText juga mengubah baris baru menjadi entitas, jadi catatan yang punya baris baru tetap utuh.
	_ = xml.EscapeText(&b, []byte(v))
	return b.String()
}
