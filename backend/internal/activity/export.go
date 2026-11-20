package activity

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// exportDay adalah satu hari di sheet ekspor: tanggalnya dan isian setiap orang, satu Cell per slot.
type exportDay struct {
	Date  time.Time
	Cells map[int64][]Cell
}

// Export menyusun .xlsx dengan tata letak spreadsheet lama: kolom Week, Date, dan Time di kiri, lalu dua
// kolom Activity dan Output per orang. Nama hari dan bulan memakai bahasa Inggris seperti spreadsheet itu.
func (s *Service) Export(ctx context.Context, from, to string) ([]byte, string, error) {
	start, end, err := parseSpan(from, to)
	if err != nil {
		return nil, "", err
	}
	// Isi, aturan kolom, dan aturan akhir pekan sama dengan halaman Aktivitas, jadi ekspor memakai collect.
	people, dates, err := s.collect(ctx, from, to)
	if err != nil {
		return nil, "", err
	}

	list := make([]exportDay, 0, len(dates))
	for _, key := range dates {
		d, err := time.Parse(dayLayout, key)
		if err != nil {
			return nil, "", err
		}
		cells := map[int64][]Cell{}
		for _, p := range people {
			cells[p.ID] = p.Days[key]
		}
		list = append(list, exportDay{Date: d, Cells: cells})
	}

	name := fmt.Sprintf("aktivitas-harian-%s-sampai-%s.xlsx", from, to)
	data, err := writeXLSX(exportRows(exportTitle(start, end), people, list), exportMerges(people, list))
	return data, name, err
}

func weekend(d time.Time) bool {
	return d.Weekday() == time.Saturday || d.Weekday() == time.Sunday
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
				// Jam tersambung dikosongkan: isinya sudah di jam kepala, yang digabung ke bawah oleh exportMerges.
				if cells[slot].Linked {
					continue
				}
				r[3+2*i], r[4+2*i] = cells[slot].Activity, outputWithFiles(cells[slot])
			}
			out = append(out, r)
		}
	}
	return out
}

// exportMerges menggabungkan kolom Activity dan Output seseorang ke bawah sepanjang rantai jamnya (revisi 33),
// jadi 09.00-10.00 dan 10.00-11.00 yang tersambung tampil sebagai satu blok, bukan isi yang tertulis dua kali.
// Nomor baris mengikuti exportRows: tiga baris kepala, lalu satu baris per slot per hari.
func exportMerges(people []Person, days []exportDay) []string {
	out := []string{}
	for d, day := range days {
		for i, p := range people {
			cells := day.Cells[p.ID]
			for _, c := range cells {
				n := 0
				for _, o := range cells {
					if o.Head == c.Slot {
						n++
					}
				}
				if c.Linked || n < 2 {
					continue
				}
				top := 4 + d*len(Slots) + c.Slot - 1
				for _, col := range []string{colName(3 + 2*i), colName(4 + 2*i)} {
					out = append(out, fmt.Sprintf("%s%d:%s%d", col, top, col, top+n-1))
				}
			}
		}
	}
	return out
}

// outputWithFiles menambahkan nama lampiran di bawah teks Output, dipisah baris baru. Berkasnya sendiri tidak
// disematkan: sheet dengan video 95 MB di dalamnya tidak bisa dibuka siapa pun (revisi 24).
func outputWithFiles(c Cell) string {
	if len(c.Files) == 0 {
		return c.Output
	}
	names := make([]string, 0, len(c.Files)+1)
	if c.Output != "" {
		names = append(names, c.Output)
	}
	for _, f := range c.Files {
		names = append(names, f.Name)
	}
	return strings.Join(names, "\n")
}
