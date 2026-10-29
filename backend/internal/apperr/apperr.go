// Package apperr berisi jenis galat yang dipakai bersama oleh paket domain.
package apperr

// Validation membawa pesan yang aman ditampilkan langsung ke pengguna; handler memetakannya ke 400.
type Validation struct{ Message string }

func (e *Validation) Error() string { return e.Message }

func Invalid(msg string) error { return &Validation{Message: msg} }
