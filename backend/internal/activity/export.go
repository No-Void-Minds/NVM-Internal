package activity

import (
	"context"
	"fmt"
	"time"

	"nvm-internal/internal/apperr"
)

// exportDay adalah satu hari di sheet ekspor: tanggalnya dan isian setiap orang, satu Cell per slot.
type exportDay struct {
	Date  time.Time
	Cells map[int64][]Cell
}

// Export menyusun .xlsx dengan tata letak spreadsheet lama: kolom Week, Date, dan Time di kiri, lalu dua
// kolom Activity dan Output per orang. Nama hari dan bulan memakai bahasa Inggris seperti spreadsheet itu.
func (s *Service) Export(ctx context.Context, from, to string) ([]byte, string, error) {
	start, err := time.Parse(dayLayout, from)
	if err != nil {
		return nil, "", apperr.Invalid("Tanggal awal tidak valid. Pakai bentuk 2026-11-18.")
	}
	end, err := time.Parse(dayLayout, to)
	if err != nil {
		return nil, "", apperr.Invalid("Tanggal akhir tidak valid. Pakai bentuk 2026-11-18.")
	}
	if end.Before(start) {
		return nil, "", apperr.Invalid("Tanggal akhir tidak boleh lebih awal dari tanggal awal.")
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > MaxExportDays {
		return nil, "", apperr.Invalid(fmt.Sprintf("Rentang ekspor maksimal %d hari.", MaxExportDays))
	}

	people, err := s.people(ctx, from, to)
	if err != nil {
		return nil, "", err
	}
	cells, err := s.rows(ctx, from, to)
	if err != nil {
		return nil, "", err
	}
	byDate := map[string]map[int64][]Cell{}
	for _, r := range cells {
		key := r.ActivityDate.Time.Format(dayLayout)
		if byDate[key] == nil {
			byDate[key] = map[int64][]Cell{}
		}
		if byDate[key][r.UserID] == nil {
			byDate[key][r.UserID] = emptyCells()
		}
		fill(byDate[key][r.UserID], r)
	}

	list := make([]exportDay, 0, days)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		day := exportDay{Date: d, Cells: byDate[d.Format(dayLayout)]}
		// Sabtu dan Minggu boleh diisi, tetapi baru muncul di ekspor kalau memang ada isinya (keputusan revisi 21).
		if weekend(d) && !hasContent(day.Cells) {
			continue
		}
		list = append(list, day)
	}

	name := fmt.Sprintf("aktivitas-harian-%s-sampai-%s.xlsx", from, to)
	data, err := writeXLSX(exportRows(exportTitle(start, end), people, list))
	return data, name, err
}

func weekend(d time.Time) bool {
	return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday
}

func hasContent(cells map[int64][]Cell) bool {
	for _, list := range cells {
		for _, c := range list {
			if c.Activity != "" || c.Output != "" {
				return true
			}
		}
	}
	return false
}

func exportTitle(start, end time.Time) string {
	first := start.Format("January 2006")
	last := end.Format("January 2006")
	if first == last {
		return "Daily Activity Monitoring - " + first
	}
	return "Daily Activity Monitoring - " + first + " sampai " + last
}

// exportRows menyusun seluruh isi sheet sebagai teks, supaya tata letaknya bisa diuji tanpa membuka .xlsx.
// Tiga baris pertama: judul, nama orang di atas dua kolomnya, lalu kepala kolom.
func exportRows(title string, people []Person, days []exportDay) [][]string {
	width := 3 + 2*len(people)
	row := func() []string { return make([]string, width) }

	head := row()
	head[0] = title
	names := row()
	header := row()
	header[0], header[1], header[2] = "Week", "Date", "Time"
	for i, p := range people {
		names[3+2*i] = p.Name
		header[3+2*i], header[4+2*i] = "Activity", "Output"
	}
	out := [][]string{head, names, header}

	lastWeek := ""
	for _, d := range days {
		_, week := d.Date.ISOWeek()
		weekLabel := fmt.Sprintf("Week %d", week)
		// Nomor minggu dan tanggal hanya ditulis sekali, di baris slot pertamanya, seperti spreadsheet lama.
		for slot := range Slots {
			r := row()
			if slot == 0 {
				if weekLabel != lastWeek {
					r[0] = weekLabel
					lastWeek = weekLabel
				}
				r[1] = fmt.Sprintf("%s - %d", d.Date.Weekday(), d.Date.Day())
			}
			r[2] = Slots[slot]
			for i, p := range people {
				cells := d.Cells[p.ID]
				if cells == nil {
					continue
				}
				r[3+2*i], r[4+2*i] = cells[slot].Activity, cells[slot].Output
			}
			out = append(out, r)
		}
	}
	return out
}
