// Package task berisi aturan dan penyimpanan task.
package task

import (
	"errors"
	"strings"
	"unicode/utf8"
)

type Status string

const (
	Requested  Status = "requested"
	OnProgress Status = "on_progress"
	Ready      Status = "ready"
	Completed  Status = "completed"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleStaff   Role = "staff"
)

var (
	// ErrInvalidTransition berarti perpindahan itu tidak ada di alur status, siapa pun pelakunya.
	ErrInvalidTransition = errors.New("perpindahan status tidak dikenal")
	// ErrTransitionForbidden berarti perpindahannya ada, tetapi bukan hak pelaku ini.
	ErrTransitionForbidden = errors.New("tidak punya hak memindahkan task ini")
	ErrNoteRequired        = errors.New("catatan revisi wajib diisi")
	ErrNoteTooLong         = errors.New("catatan revisi terlalu panjang")
)

const MaxNoteLength = 500

type edge struct{ from, to Status }

// Alur status (BR-03). staffPIC menandai perpindahan yang juga boleh dilakukan Staf yang menjadi PIC;
// Manajer boleh semua perpindahan di sini (FR-TASK-07); Admin tidak boleh satu pun (FR-TASK-19).
var edges = map[edge]struct{ staffPIC, needsNote bool }{
	{Requested, OnProgress}: {staffPIC: true},
	{OnProgress, Ready}:     {staffPIC: true},
	{Ready, OnProgress}:     {staffPIC: true, needsNote: true}, // revisi (FR-TASK-22)
	{Ready, Completed}:      {},                                // hanya Manajer, setelah memeriksa hasil
	{Completed, OnProgress}: {},                                // buka kembali, hanya Manajer
}

// CheckTransition memeriksa apakah pelaku boleh memindahkan task dari from ke to.
// isPIC hanya berpengaruh untuk Staf. Catatan yang dikembalikan sudah dirapikan.
func CheckTransition(role Role, isPIC bool, from, to Status, note string) (string, error) {
	rule, ok := edges[edge{from, to}]
	if !ok {
		return "", ErrInvalidTransition
	}

	switch role {
	case RoleManager:
	case RoleStaff:
		if !isPIC || !rule.staffPIC {
			return "", ErrTransitionForbidden
		}
	default:
		return "", ErrTransitionForbidden
	}

	note = strings.TrimSpace(note)
	if rule.needsNote && note == "" {
		return "", ErrNoteRequired
	}
	if utf8.RuneCountInString(note) > MaxNoteLength {
		return "", ErrNoteTooLong
	}
	return note, nil
}
