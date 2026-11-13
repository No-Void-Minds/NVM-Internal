package account

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"

	"github.com/disintegration/imaging"
)

const (
	MaxAvatarSize = 2 << 20
	// Berkas 2 MB bisa berisi gambar puluhan ribu piksel per sisi yang setelah di-decode memakan gigabyte memori.
	// Batas dicek dari header gambar sebelum decode penuh.
	maxAvatarPixels = 40_000_000
	avatarSide      = 256
	avatarQuality   = 85
)

var ErrAvatarTooLarge = errors.New("foto melebihi 2 MB")

// AvatarRejected berarti isi berkas bukan foto yang bisa dipakai. Pesannya aman ditampilkan.
type AvatarRejected struct{ Message string }

func (e *AvatarRejected) Error() string { return e.Message }

var (
	errNotImage    = &AvatarRejected{Message: "Foto harus berupa berkas JPG atau PNG."}
	errBrokenImage = &AvatarRejected{Message: "Berkas foto rusak atau isinya tidak sesuai dengan tipenya. Coba simpan ulang fotonya."}
	errHugeImage   = &AvatarRejected{Message: "Resolusi foto terlalu besar, maksimal 40 megapiksel. Kecilkan dulu fotonya."}
)

// sniff menentukan tipe dari isi berkas, bukan dari nama atau Content-Type kiriman browser.
func sniff(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg"
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	}
	return ""
}

// ProcessAvatar mengubah unggahan menjadi JPEG persegi 256 px. Gambar disimpan ulang dari piksel, jadi metadata
// EXIF (termasuk koordinat lokasi foto ponsel) tidak ikut. Arah putar dari EXIF diterapkan lebih dulu; tanpa itu
// foto ponsel yang diambil tegak akan tersimpan miring setelah EXIF-nya hilang.
func ProcessAvatar(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxAvatarSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxAvatarSize {
		return nil, ErrAvatarTooLarge
	}
	kind := sniff(data)
	if kind == "" {
		return nil, errNotImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != kind || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errBrokenImage
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxAvatarPixels {
		return nil, errHugeImage
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, errBrokenImage
	}
	img = imaging.Fill(img, avatarSide, avatarSide, imaging.Center, imaging.Lanczos)
	// JPEG tidak punya transparansi; tanpa latar putih, area transparan PNG menjadi hitam.
	flat := imaging.Overlay(imaging.New(avatarSide, avatarSide, color.White), img, image.Pt(0, 0), 1)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, flat, imaging.JPEG, imaging.JPEGQuality(avatarQuality)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
