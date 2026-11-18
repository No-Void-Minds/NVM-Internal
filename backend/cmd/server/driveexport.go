package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
	"nvm-internal/internal/driveexport"
	"nvm-internal/internal/storage"
)

// runDriveExport menjalankan satu langkah salinan ke Google Drive (revisi 25). Dipisah tiga langkah karena
// unggahannya dikerjakan container rclone di antaranya:
//
//	stage [--dry-run]  menyusun pohon staging dan manifest; dump dibaca dari tar di stdin
//	record             mencatat drive_exports setelah rclone berhasil, lalu membuang staging
//	fail <pesan>       hanya menulis baris status gagal, supaya halaman Status menunjukkannya
//
// Pemisahan stage dan record yang membuat aturan "setiap berkas hanya diunggah sekali" tidak berubah jadi
// "hanya dicoba sekali": kalau rclone gagal, tidak ada baris yang tercatat dan malam berikutnya dicoba lagi.
func runDriveExport(cfg config.Config, log *slog.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("langkah wajib: stage, record, atau fail")
	}
	step, rest := args[0], args[1:]

	fs := flag.NewFlagSet("drive-export", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "susun staging dan cetak daftar path, tanpa menulis ke database")
	if err := fs.Parse(rest); err != nil {
		return err
	}

	staging := envOr("DRIVE_STAGING_DIR", "/data/drive-staging")
	// Kunci publik age. Kosong berarti dump tidak diunggah, dan alasannya masuk log.
	recipient := strings.TrimSpace(os.Getenv("DRIVE_BACKUP_AGE_RECIPIENT"))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	store, err := storage.Open(cfg.StorageDir)
	if err != nil {
		return err
	}
	svc := driveexport.NewService(pool, store, staging, recipient, log)

	switch step {
	case "stage":
		// Dump sampai lewat stdin sebagai tar. Tanpa pipe (mis. uji kering di tangan), stdin dilewati.
		var dumps *os.File
		if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
			dumps = os.Stdin
		}
		m, err := svc.Stage(ctx, dumps, *dryRun)
		if err != nil {
			return err
		}
		// stdout hanya daftar path, karena drive-export.sh membacanya sebagai daftar berkas.
		for _, p := range m.Listing() {
			fmt.Println(p)
		}
		// Ringkasan ke stderr: jenis yang kosong langsung kelihatan, dan kalau tidak ada berkas baru,
		// alasannya ikut tercetak, bukan hanya angka nol.
		printSummary(m)
		log.Info("staging siap", "berkas_baru", len(m.Files), "berkas_ditimpa", len(m.Extra),
			"bytes", m.Bytes, "dry_run", *dryRun)
		return nil

	case "record":
		m, err := svc.ReadManifest()
		if err != nil {
			return err
		}
		if err := svc.Record(ctx, m); err != nil {
			return err
		}
		log.Info("salinan Drive dicatat", "berkas", len(m.Files)+len(m.Extra), "bytes", m.Bytes)
		return nil

	case "fail":
		message := strings.Join(rest, " ")
		if message == "" {
			message = "tidak ada pesan"
		}
		if err := svc.Fail(ctx, message); err != nil {
			return err
		}
		log.Error("salinan Drive gagal", "pesan", message)
		return nil

	default:
		return fmt.Errorf("langkah %q tidak dikenal", step)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// printSummary menulis ringkasan per jenis dan alasan yang dilewati ke stderr.
func printSummary(m driveexport.Manifest) {
	fmt.Fprintln(os.Stderr, "ringkasan staging:")
	for _, kind := range driveexport.KindOrder {
		fmt.Fprintf(os.Stderr, "  %-16s %d\n", kind, m.Counts[kind])
	}
	if len(m.Skipped) == 0 {
		return
	}
	// Urut menurut alasan, supaya laporannya sama di setiap run dan mudah dibandingkan.
	alasan := make([]string, 0, len(m.Skipped))
	for a, n := range m.Skipped {
		if n > 0 {
			alasan = append(alasan, a)
		}
	}
	if len(alasan) == 0 {
		return
	}
	sort.Strings(alasan)
	fmt.Fprintln(os.Stderr, "dilewati:")
	for _, a := range alasan {
		fmt.Fprintf(os.Stderr, "  %-32s %d\n", a, m.Skipped[a])
	}
}
