package excelimport

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sheets yang dibaca, sesuai SRS bagian 2.1.
var Sheets = []string{"Task Tracker", "Completed Task"}

// Kolom dicari dari baris header berdasarkan nama (SRS 2.1), bukan posisi, karena sheet Completed Task punya
// kotak pencarian di atas tabelnya.
const (
	colID          = "idtask"
	colTitle       = "judultask"
	colRequestBy   = "requestby"
	colSolvedBy    = "solvedby"
	colUrgency     = "urgency"
	colStatus      = "status"
	colRequestDate = "requestdate"
	colSolvedDate  = "solveddate"
	colDeliverable = "deliverable"
)

var officeTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}
	return loc
}()

// Problem adalah satu baris yang ditolak atau perlu diperhatikan, dengan alasan yang bisa dibaca Manajer.
type Problem struct {
	Sheet  string
	Line   int
	ID     string
	Reason string
}

type Task struct {
	Sheet       string
	Line        int
	ID          string
	Number      int64
	Title       string
	Description string
	Status      string
	Urgency     string
	RequestedBy int64
	DivisionID  int64
	RequestDate time.Time
	SolvedDate  *time.Time
	Deliverable *string
}

type Report struct {
	Read     int
	Tasks    []Task
	Rejected []Problem
	Warnings []Problem
}

type account struct {
	id         int64
	divisionID int64
}

// LoadMapping membaca CSV pemetaan nama Excel ke username (kolom nama_excel,username). Baris dengan username
// kosong dianggap belum dipetakan, supaya template yang setengah terisi tidak diam-diam dipakai.
func LoadMapping(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV pemetaan tidak terbaca: %w", err)
	}
	out := map[string]string{}
	for i, r := range records {
		if i == 0 || len(r) < 2 {
			continue
		}
		name, user := normName(r[0]), strings.TrimSpace(r[1])
		if name == "" || user == "" {
			continue
		}
		out[name] = user
	}
	return out, nil
}

func normName(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func normHeader(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }

// Prepare membaca berkas, memvalidasi setiap baris, dan mencocokkannya dengan database tanpa menulis apa pun.
func Prepare(ctx context.Context, pool *pgxpool.Pool, file string, mapping map[string]string) (Report, error) {
	var r Report
	sheets, err := readSheets(file, Sheets...)
	if err != nil {
		return r, err
	}
	for _, name := range Sheets {
		if _, ok := sheets[name]; !ok {
			return r, fmt.Errorf("sheet %q tidak ditemukan di berkas", name)
		}
	}

	accounts, err := loadAccounts(ctx, pool)
	if err != nil {
		return r, err
	}

	// Baris mentah dari kedua sheet, dengan nomor baris Excel (mulai 1) untuk laporan.
	type rawRow struct {
		sheet string
		line  int
		cells map[string]string
	}
	var raws []rawRow
	for _, name := range Sheets {
		rows := sheets[name]
		header, at := findHeader(rows)
		if header == nil {
			return r, fmt.Errorf("sheet %q: baris header dengan kolom ID Task dan Judul Task tidak ditemukan", name)
		}
		for i := at + 1; i < len(rows); i++ {
			cells := map[string]string{}
			for col, key := range header {
				if col < len(rows[i]) {
					cells[key] = strings.TrimSpace(rows[i][col])
				}
			}
			if cells[colID] == "" && cells[colTitle] == "" {
				continue
			}
			raws = append(raws, rawRow{sheet: name, line: i + 1, cells: cells})
		}
	}
	r.Read = len(raws)

	// ID kembar di dalam berkas ditolak semuanya (SRS 3.3), bukan memilih salah satu.
	count := map[string]int{}
	for _, raw := range raws {
		if id, err := normID(raw.cells[colID]); err == nil {
			count[id]++
		}
	}

	var ids []string
	for _, raw := range raws {
		t, problems, warnings := buildTask(raw.sheet, raw.line, raw.cells, mapping, accounts)
		r.Warnings = append(r.Warnings, warnings...)
		if t != nil && count[t.ID] > 1 {
			problems = append(problems, Problem{raw.sheet, raw.line, t.ID, fmt.Sprintf("ID %s muncul %d kali di berkas", t.ID, count[t.ID])})
		}
		if len(problems) > 0 {
			r.Rejected = append(r.Rejected, problems...)
			continue
		}
		r.Tasks = append(r.Tasks, *t)
		ids = append(ids, t.ID)
	}

	// ID yang sudah ada di database (misalnya impor kedua kalinya) ikut ditolak.
	existing, err := existingIDs(ctx, pool, ids)
	if err != nil {
		return r, err
	}
	kept := r.Tasks[:0]
	for _, t := range r.Tasks {
		if existing[t.ID] {
			r.Rejected = append(r.Rejected, Problem{t.Sheet, t.Line, t.ID, "ID sudah ada di database"})
			continue
		}
		kept = append(kept, t)
	}
	r.Tasks = kept
	return r, nil
}

func findHeader(rows [][]string) (map[int]string, int) {
	for i, row := range rows {
		header := map[int]string{}
		for col, v := range row {
			if h := normHeader(v); h != "" {
				header[col] = h
			}
		}
		var hasID, hasTitle bool
		for _, h := range header {
			hasID = hasID || h == colID
			hasTitle = hasTitle || h == colTitle
		}
		if hasID && hasTitle {
			return header, i
		}
	}
	return nil, 0
}

var idPattern = regexp.MustCompile(`(?i)^T-?\s*([0-9]{1,8})$`)

// normID menerima "T-0012", "T-12", atau "T12" dan menulisnya ulang dalam format aplikasi (T- dan minimal 4 digit).
func normID(v string) (string, error) {
	m := idPattern.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return "", fmt.Errorf("ID %q bukan format T-nnnn", v)
	}
	n, _ := strconv.ParseInt(m[1], 10, 64)
	if n <= 0 {
		return "", fmt.Errorf("ID %q harus lebih dari 0", v)
	}
	return fmt.Sprintf("T-%04d", n), nil
}

var statusMap = map[string]string{
	"requested": "requested", "onprogress": "on_progress", "ready": "ready", "completed": "completed",
}

var urgencyMap = map[string]string{"low": "low", "medium": "medium", "high": "high", "urgent": "urgent"}

func buildTask(sheet string, line int, c map[string]string, mapping map[string]string, accounts map[string]account) (*Task, []Problem, []Problem) {
	var problems, warnings []Problem
	bad := func(id, reason string) { problems = append(problems, Problem{sheet, line, id, reason}) }
	warn := func(id, reason string) { warnings = append(warnings, Problem{sheet, line, id, reason}) }

	id, err := normID(c[colID])
	if err != nil {
		bad(c[colID], err.Error())
		return nil, problems, warnings
	}
	t := &Task{Sheet: sheet, Line: line, ID: id}
	t.Number, _ = strconv.ParseInt(strings.TrimPrefix(id, "T-"), 10, 64)

	t.Title = strings.TrimSpace(c[colTitle])
	switch {
	case t.Title == "":
		bad(id, "Judul Task kosong")
	case utf8.RuneCountInString(t.Title) > 200:
		bad(id, "Judul Task lebih dari 200 karakter")
	}

	status := strings.ReplaceAll(normHeader(c[colStatus]), "_", "")
	switch {
	case status == "" && sheet == "Completed Task":
		t.Status = "completed"
		warn(id, "Status kosong di sheet Completed Task, dianggap Completed")
	case statusMap[status] != "":
		t.Status = statusMap[status]
	default:
		bad(id, fmt.Sprintf("Status %q tidak dikenal (Requested, On Progress, Ready, Completed)", c[colStatus]))
	}
	if t.Urgency = urgencyMap[normHeader(c[colUrgency])]; t.Urgency == "" {
		bad(id, fmt.Sprintf("Urgency %q tidak dikenal (Low, Medium, High, Urgent)", c[colUrgency]))
	}

	requester, reason := lookup(c[colRequestBy], mapping, accounts)
	if reason != "" {
		bad(id, "Request By: "+reason)
	}
	t.RequestedBy, t.DivisionID = requester.id, requester.divisionID

	notes := []string{"Data migrasi dari Excel."}
	if name := strings.TrimSpace(c[colSolvedBy]); name != "" {
		// Task hasil migrasi tidak punya sub task (S12-01), jadi tidak punya PIC. Solved By hanya menentukan divisi
		// task dan dicatat di deskripsi supaya informasinya tidak hilang.
		solver, reason := lookup(name, mapping, accounts)
		if reason != "" {
			bad(id, "Solved By: "+reason)
		} else {
			t.DivisionID = solver.divisionID
		}
		notes = append(notes, "Solved By: "+name+".")
	}

	if d, err := parseDay(c[colRequestDate]); err != nil {
		bad(id, "Request Date: "+err.Error())
	} else if d == nil {
		bad(id, "Request Date kosong")
	} else {
		t.RequestDate = *d
	}
	solved, err := parseDay(c[colSolvedDate])
	switch {
	case err != nil:
		bad(id, "Solved Date: "+err.Error())
	// Dua aturan CHECK tasks_solved_only_when_completed. S12-03: ditolak dan dilaporkan, diputuskan saat dry-run.
	case t.Status == "completed" && solved == nil:
		bad(id, "Completed tanpa Solved Date")
	case t.Status != "completed" && t.Status != "" && solved != nil:
		bad(id, "Solved Date terisi padahal status bukan Completed")
	default:
		t.SolvedDate = solved
	}

	if d := strings.TrimSpace(c[colDeliverable]); d != "" {
		lower := strings.ToLower(d)
		if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
			t.Deliverable = &d
		} else {
			// CHECK tasks_deliverable_url_scheme hanya menerima http(s); teks lain dipindah ke deskripsi.
			notes = append(notes, "Deliverable (teks dari Excel): "+d)
			warn(id, "Deliverable bukan tautan http(s), dipindah ke deskripsi")
		}
	}
	t.Description = strings.Join(notes, "\n")
	return t, problems, warnings
}

func lookup(name string, mapping map[string]string, accounts map[string]account) (account, string) {
	n := normName(name)
	if n == "" {
		return account{}, "kosong"
	}
	user, ok := mapping[n]
	if !ok {
		return account{}, fmt.Sprintf("nama %q belum ada di CSV pemetaan", strings.TrimSpace(name))
	}
	a, ok := accounts[strings.ToLower(user)]
	if !ok {
		return account{}, fmt.Sprintf("username %q (untuk %q) tidak ada di database", user, strings.TrimSpace(name))
	}
	return a, ""
}

var dayLayouts = []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006"}

// parseDay membaca tanggal sebagai hari WIB: nomor seri Excel (sel bertipe tanggal) atau teks dd/mm/yyyy dan
// yyyy-mm-dd. Format Indonesia dibaca hari dulu, bukan bulan dulu.
func parseDay(v string) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if serial, err := strconv.ParseFloat(v, 64); err == nil {
		// Excel menghitung hari sejak 30 Desember 1899 (termasuk tahun kabisat 1900 yang keliru).
		if serial < 1 || serial > 2958465 {
			return nil, fmt.Errorf("nomor tanggal %q di luar jangkauan", v)
		}
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, officeTZ)
		d := base.AddDate(0, 0, int(math.Floor(serial)))
		return &d, nil
	}
	for _, layout := range dayLayouts {
		if d, err := time.ParseInLocation(layout, v, officeTZ); err == nil {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("tanggal %q tidak dikenal (pakai dd/mm/yyyy atau yyyy-mm-dd)", v)
}

func loadAccounts(ctx context.Context, pool *pgxpool.Pool) (map[string]account, error) {
	rows, err := pool.Query(ctx, `SELECT id, username, division_id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]account{}
	for rows.Next() {
		var a account
		var username string
		if err := rows.Scan(&a.id, &username, &a.divisionID); err != nil {
			return nil, err
		}
		out[strings.ToLower(username)] = a
	}
	return out, rows.Err()
}

func existingIDs(ctx context.Context, pool *pgxpool.Pool, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `SELECT id FROM tasks WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// Apply menulis task yang lolos dalam satu transaksi: semuanya masuk atau tidak ada yang masuk. Setelah itu
// task_number_seq disetel ke ID terbesar, supaya task baru melanjutkan nomornya (SRS 3.3 langkah 3).
func Apply(ctx context.Context, pool *pgxpool.Pool, tasks []Task) error {
	if len(tasks) == 0 {
		return errors.New("tidak ada task yang lolos validasi")
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var maxNumber int64
		for _, t := range tasks {
			_, err := tx.Exec(ctx, `
				INSERT INTO tasks (id, title, description, division_id, requested_by, urgency, status,
				                   request_date, solved_date, deliverable_url, is_migrated)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true)`,
				t.ID, t.Title, t.Description, t.DivisionID, t.RequestedBy, t.Urgency, t.Status,
				t.RequestDate, t.SolvedDate, t.Deliverable)
			if err != nil {
				return fmt.Errorf("%s (%s baris %d): %w", t.ID, t.Sheet, t.Line, err)
			}
			detail, _ := json.Marshal(map[string]any{"source": "excel", "sheet": t.Sheet, "line": t.Line})
			// Pelaku kosong: Audit Log menampilkannya sebagai "Sistem".
			if _, err := tx.Exec(ctx, `
				INSERT INTO audit_logs (action, entity, entity_id, new_value) VALUES ('task_created', 'task', $1, $2)`,
				t.ID, string(detail)); err != nil {
				return err
			}
			maxNumber = max(maxNumber, t.Number)
		}
		// greatest dengan nilai sekarang supaya urutan tidak pernah mundur bila sudah ada task yang lebih baru.
		_, err := tx.Exec(ctx, `
			SELECT setval('task_number_seq', greatest($1, (SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM task_number_seq)))`,
			maxNumber)
		return err
	})
}
