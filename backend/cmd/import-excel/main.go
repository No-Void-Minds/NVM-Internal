// import-excel memindahkan task dari "NVM - Marketing & Sales.xlsx" ke database (S12-01 sampai S12-04).
//
//	import-excel --file berkas.xlsx --map pemetaan-nama.csv --dry-run   # hanya laporan
//	import-excel --file berkas.xlsx --map pemetaan-nama.csv             # tulis, satu transaksi
//
// Butuh DATABASE_URL role pemilik tabel (nvm_internal_migrator di produksi), karena setval pada task_number_seq
// tidak diizinkan untuk role aplikasi.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
	"nvm-internal/internal/excelimport"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "impor gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	file := flag.String("file", "", "berkas .xlsx (sheet Task Tracker dan Completed Task)")
	mapFile := flag.String("map", "", "CSV pemetaan nama_excel,username")
	dryRun := flag.Bool("dry-run", false, "hanya membaca dan melaporkan, tidak menulis ke database")
	flag.Parse()
	if *file == "" || *mapFile == "" {
		flag.Usage()
		return fmt.Errorf("--file dan --map wajib diisi")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	mapping, err := excelimport.LoadMapping(*mapFile)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	r, err := excelimport.Prepare(ctx, pool, *file, mapping)
	if err != nil {
		return err
	}
	perSheet := map[string]int{}
	for _, t := range r.Tasks {
		perSheet[t.Sheet]++
	}
	fmt.Printf("Baris terbaca: %d\n", r.Read)
	fmt.Printf("Lolos: %d (Task Tracker %d, Completed Task %d)\n", len(r.Tasks), perSheet["Task Tracker"], perSheet["Completed Task"])
	fmt.Printf("Ditolak: %d alasan\n", len(r.Rejected))
	for _, p := range r.Rejected {
		fmt.Printf("  DITOLAK  %s baris %d  %s  %s\n", p.Sheet, p.Line, p.ID, p.Reason)
	}
	fmt.Printf("Peringatan: %d\n", len(r.Warnings))
	for _, p := range r.Warnings {
		fmt.Printf("  CATATAN  %s baris %d  %s  %s\n", p.Sheet, p.Line, p.ID, p.Reason)
	}

	if *dryRun {
		fmt.Println("Dry-run: tidak ada yang ditulis.")
		return nil
	}
	if len(r.Rejected) > 0 {
		// Impor sebagian akan membuat Manajer mencocokkan dua sumber; lebih aman perbaiki pemetaan lalu ulangi.
		return fmt.Errorf("masih ada baris yang ditolak; perbaiki Excel atau pemetaan, jalankan --dry-run lagi")
	}
	if err := excelimport.Apply(ctx, pool, r.Tasks); err != nil {
		return err
	}
	fmt.Printf("Selesai: %d task ditulis dengan is_migrated = true, task_number_seq disetel.\n", len(r.Tasks))
	return nil
}
