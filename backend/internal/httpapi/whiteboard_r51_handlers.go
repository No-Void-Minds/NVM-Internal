package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Whiteboard revisi 51: kartu anggota tim.

// wbPeople: daftar alat Tim (aktif dan nonaktif), hanya nama, foto, jabatan, dan divisi.
func (s *Server) wbPeople(c *gin.Context) {
	out, err := s.whiteboard.People(c.Request.Context())
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"people": out})
}

// wbPersonChanged dipanggil setelah profil, foto, divisi, atau status aktif pengguna berubah, supaya kartu orang di
// semua papan yang terbuka ikut berubah. Galatnya diabaikan: perubahan profil sudah tersimpan, dan papan yang
// dimuat ulang tetap membaca data terkini.
func (s *Server) wbPersonChanged(ctx context.Context, id int64) {
	if p, err := s.whiteboard.Person(ctx, id); err == nil {
		s.hub.PersonChanged(p)
	}
}
