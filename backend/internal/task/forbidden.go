package task

// ForbiddenError adalah penolakan hak akses yang membawa pesan untuk pengguna sesuai aksinya. errors.Is(err,
// ErrForbidden) tetap benar, jadi pemeriksaan lama tidak berubah; handler membaca Message lewat errors.As.
// Dulu semua penolakan memakai ErrForbidden polos dan handler memberinya satu pesan bersama tentang centang sub
// task, sehingga Staf yang ditolak memindahkan task membaca pesan yang tidak ada hubungannya.
type ForbiddenError struct{ Message string }

func (e *ForbiddenError) Error() string        { return e.Message }
func (e *ForbiddenError) Is(target error) bool { return target == ErrForbidden }

func forbidden(msg string) error { return &ForbiddenError{Message: msg} }

var statusName = map[Status]string{
	Requested:  "Requested",
	OnProgress: "On Progress",
	Ready:      "Ready",
	Completed:  "Completed",
}

// Pesan per aksi. Hanya aksi atas sub task yang menyebut sub task.
var (
	errCreateForbidden      = forbidden("Hanya Staf dan Manajer yang bisa membuat task.")
	errUpdateForbidden      = forbidden("Task ini hanya bisa diubah Manajer, atau pembuatnya selama masih Requested.")
	errDeleteForbidden      = forbidden("Task ini hanya bisa dihapus Manajer, atau pembuatnya selama masih Requested.")
	errLabelsForbidden      = forbidden("Hanya PIC task ini dan Manajer yang bisa mengubah labelnya.")
	errDeliverableForbidden = forbidden("Hanya PIC task ini dan Manajer yang bisa mengubah tautan deliverable.")
	errStopSeriesForbidden  = forbidden("Hanya Manajer yang bisa menghentikan pengulangan task.")
	errAddSubtaskForbidden  = forbidden("Hanya PIC task ini dan Manajer yang bisa menambah sub task.")
)

func moveForbidden(to Status) error {
	name, ok := statusName[to]
	if !ok {
		name = string(to)
	}
	return forbidden("Kamu tidak punya hak memindahkan task ini ke " + name + ".")
}

// subtaskForbidden: mengubah atau menghapus sub task yang sudah ada (verb "mengubah" atau "menghapus").
func subtaskForbidden(verb string) error {
	return forbidden("Hanya PIC task ini, Manajer, atau PIC sub task itu yang bisa " + verb + " sub task ini.")
}

// attachmentForbidden memakai nama tempat berkas, supaya lampiran brief dan deliverable task tidak disebut
// sebagai sub task.
func attachmentForbidden(verb string, kind AttachmentKind, subtaskID int64) error {
	switch {
	case subtaskID != 0:
		return forbidden("Hanya PIC task ini, Manajer, atau PIC sub task itu yang bisa " + verb + " berkas sub task ini.")
	case kind == KindBrief:
		return forbidden("Hanya Manajer yang bisa " + verb + " lampiran brief task ini.")
	default:
		return forbidden("Hanya PIC task ini dan Manajer yang bisa " + verb + " berkas deliverable task ini.")
	}
}
