package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

// writeTaskError meneruskan pesan penolakan dari galatnya sendiri (ForbiddenError), juga bila galat itu
// dibungkus. Galat ErrForbidden polos mendapat pesan cadangan yang tidak menyebut aksi apa pun.
func TestWriteTaskErrorForbiddenMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"pindah status", &task.ForbiddenError{Message: "Kamu tidak punya hak memindahkan task ini ke Ready."},
			"Kamu tidak punya hak memindahkan task ini ke Ready."},
		{"dibungkus", fmt.Errorf("transaksi: %w", task.ErrNotSubtaskPIC), task.ErrNotSubtaskPIC.Error()},
		{"polos", task.ErrForbidden, "Kamu tidak punya hak melakukan ini pada task ini."},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		(&Server{}).writeTaskError(c, tc.err)
		var body errorBody
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if w.Code != http.StatusForbidden || body.Error.Message != tc.want {
			t.Errorf("%s: %d %q, mau 403 %q", tc.name, w.Code, body.Error.Message, tc.want)
		}
	}
}
