package linkpreview

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

var loopback = netip.MustParseAddr("127.0.0.1")

// testFetcher memetakan nama host uji ke alamat tertentu. Server uji berjalan di 127.0.0.1 dengan port acak, jadi
// hanya alamat itu yang diizinkan selain alamat publik, dan port bebas.
func testFetcher(hosts map[string][]string) *Fetcher {
	f := New()
	f.Timeout = 2 * time.Second
	f.anyPort = true
	f.allowAddr = func(a netip.Addr) bool { return a == loopback || !Forbidden(a) }
	f.Resolve = func(_ context.Context, host string) ([]netip.Addr, error) {
		list, ok := hosts[host]
		if !ok {
			return nil, errors.New("tidak ada")
		}
		out := []netip.Addr{}
		for _, s := range list {
			out = append(out, netip.MustParseAddr(s))
		}
		return out, nil
	}
	return f
}

func serverPort(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, _ := url.Parse(srv.URL)
	return u.Port()
}

func wantErr(t *testing.T, err error, part string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || !strings.Contains(e.Message, part) {
		t.Fatalf("mau galat berisi %q, dapat %v", part, err)
	}
}

func TestForbidden(t *testing.T) {
	for _, s := range []string{
		"127.0.0.1", "10.1.2.3", "172.16.5.4", "172.31.255.255", "192.168.68.69", "169.254.169.254", "100.64.1.1",
		"0.0.0.0", "0.1.2.3", "224.0.0.1", "239.255.255.250", "255.255.255.255", "198.18.0.1", "192.0.2.10",
		"203.0.113.5", "240.0.0.1", "::", "::1", "fe80::1", "fc00::1", "fd12:3456::1", "ff02::1", "2001:db8::1",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "::ffff:169.254.169.254", "64:ff9b::a00:1", "64:ff9b::7f00:1",
	} {
		if !Forbidden(netip.MustParseAddr(s)) {
			t.Errorf("%s seharusnya terlarang", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700::1111", "64:ff9b::808:808"} {
		if Forbidden(netip.MustParseAddr(s)) {
			t.Errorf("%s seharusnya boleh", s)
		}
	}
}

func TestCheckURL(t *testing.T) {
	f := New()
	for raw, part := range map[string]string{
		"ftp://contoh.test/a":          "http",
		"javascript:alert(1)":          "http",
		"https://user:pw@contoh.test/": "nama pengguna",
		"https://contoh.test:8443/":    "port",
		"http://contoh.test:22/":       "port",
		"http:///tanpa-host":           "tidak lengkap",
	} {
		u, _ := url.Parse(raw)
		wantErr(t, f.CheckURL(u), part)
	}
	for _, raw := range []string{"http://127.0.0.1:8081/", "http://[::1]:3000/", "http://10.0.0.1/"} {
		u, _ := url.Parse(raw)
		if err := f.CheckURL(u); !errors.Is(err, ErrBlocked) {
			t.Errorf("%s: mau jaringan internal, dapat %v", raw, err)
		}
	}
	for _, raw := range []string{"https://contoh.test/", "http://contoh.test:80/a", "https://contoh.test:443/"} {
		u, _ := url.Parse(raw)
		if err := f.CheckURL(u); err != nil {
			t.Errorf("%s ditolak: %v", raw, err)
		}
	}
}

func TestFetchBerhasil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie terkirim")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Judul cadangan</title>
			<meta property="og:title" content="  Katalog   Desember ">
			<meta name="description" content="Deskripsi meta">
			<meta property="og:site_name" content="NVM Test">
			<meta property="og:image" content="/gambar.png"></head><body><meta property="og:title" content="di body"></body></html>`))
	}))
	defer srv.Close()
	f := testFetcher(map[string][]string{"situs.test": {"127.0.0.1"}})
	p, err := f.Fetch(context.Background(), "http://situs.test:"+serverPort(t, srv)+"/halaman#bagian")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Katalog Desember" || p.Description != "Deskripsi meta" || p.SiteName != "NVM Test" ||
		p.ImageURL != "http://situs.test:"+serverPort(t, srv)+"/gambar.png" {
		t.Fatalf("hasil %+v", p)
	}
}

// Setiap kelas alamat terlarang ditolak, termasuk bila hanya satu dari beberapa alamat hasil resolve yang
// terlarang, dan bila URL berisi IP langsung.
func TestFetchMenolakAlamatInternal(t *testing.T) {
	cases := map[string][]string{
		"loopback.test": {"127.0.0.1"},
		"lan.test":      {"192.168.68.69"},
		"privat.test":   {"10.0.0.5"},
		"metadata.test": {"169.254.169.254"},
		"cgnat.test":    {"100.64.0.1"},
		"campur.test":   {"93.184.216.34", "10.0.0.1"},
		"mapped.test":   {"::ffff:192.168.1.1"},
		"nat64.test":    {"64:ff9b::a00:1"},
		"ula.test":      {"fd00::1"},
	}
	f := New()
	f.Timeout = time.Second
	f.anyPort = true
	f.Resolve = func(_ context.Context, host string) ([]netip.Addr, error) {
		out := []netip.Addr{}
		for _, s := range cases[host] {
			out = append(out, netip.MustParseAddr(s))
		}
		return out, nil
	}
	for host := range cases {
		_, err := f.Fetch(context.Background(), "http://"+host+"/")
		if !errors.Is(err, ErrBlocked) {
			t.Errorf("%s: mau diblokir, dapat %v", host, err)
		}
	}
	for _, raw := range []string{"http://127.0.0.1/", "http://169.254.169.254/latest/meta-data/", "http://[::1]/", "http://0x7f000001/"} {
		if _, err := f.Fetch(context.Background(), raw); err == nil {
			t.Errorf("%s tidak ditolak", raw)
		}
	}
}

// Redirect ke alamat internal (IP langsung atau nama yang menunjuk ke jaringan internal) ditolak, dan lebih dari
// tiga redirect dihentikan.
func TestFetchRedirect(t *testing.T) {
	var port string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ke-ip":
			http.Redirect(w, r, "http://10.0.0.1/", http.StatusFound)
		case "/ke-nama":
			http.Redirect(w, r, "http://internal.test:"+port+"/", http.StatusFound)
		default:
			http.Redirect(w, r, "/putar", http.StatusFound)
		}
	}))
	defer srv.Close()
	port = serverPort(t, srv)
	f := testFetcher(map[string][]string{"situs.test": {"127.0.0.1"}, "internal.test": {"192.168.1.10"}})
	for _, path := range []string{"/ke-ip", "/ke-nama"} {
		_, err := f.Fetch(context.Background(), "http://situs.test:"+port+path)
		if !errors.Is(err, ErrBlocked) {
			t.Errorf("%s: mau diblokir, dapat %v", path, err)
		}
	}
	_, err := f.Fetch(context.Background(), "http://situs.test:"+port+"/putar")
	wantErr(t, err, "terlalu banyak")
}

// Host yang hanya punya alamat IPv6 gagal seketika dengan pesan jelas, tidak menunggu batas waktu. Host dengan
// alamat IPv4 dan IPv6 tetap tersambung lewat IPv4.
func TestFetchIPv6Saja(t *testing.T) {
	f := testFetcher(map[string][]string{"v6.test": {"2606:4700::1111"}})
	f.Timeout = 5 * time.Second
	start := time.Now()
	_, err := f.Fetch(context.Background(), "https://v6.test/")
	if !errors.Is(err, ErrIPv6Only) {
		t.Fatalf("mau ErrIPv6Only, dapat %v", err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("butuh %v, mau gagal seketika", d)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<title>Dua tumpukan</title>`))
	}))
	defer srv.Close()
	f = testFetcher(map[string][]string{"dua.test": {"2606:4700::1111", "127.0.0.1"}})
	p, err := f.Fetch(context.Background(), "http://dua.test:"+serverPort(t, srv)+"/")
	if err != nil || p.Title != "Dua tumpukan" {
		t.Fatalf("dual stack: %+v %v", p, err)
	}
}

func TestFetchBatas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4"))
		case "/lambat":
			time.Sleep(2 * time.Second)
		case "/besar":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<head><title>Besar</title></head>` + strings.Repeat("a", 2*MaxBody)))
		case "/gambar-besar":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(make([]byte, MaxImage+10))
		case "/404":
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base := "http://situs.test:" + serverPort(t, srv)
	f := testFetcher(map[string][]string{"situs.test": {"127.0.0.1"}})
	_, err := f.Fetch(context.Background(), base+"/pdf")
	wantErr(t, err, "bukan halaman web")
	_, err = f.Fetch(context.Background(), base+"/404")
	wantErr(t, err, "404")
	p, err := f.Fetch(context.Background(), base+"/besar")
	if err != nil || p.Title != "Besar" {
		t.Fatalf("body besar: %+v %v", p, err)
	}
	_, err = f.FetchImage(context.Background(), base+"/gambar-besar")
	wantErr(t, err, "terlalu besar")
	f.Timeout = 300 * time.Millisecond
	start := time.Now()
	_, err = f.Fetch(context.Background(), base+"/lambat")
	wantErr(t, err, "terlalu lama")
	if d := time.Since(start); d > time.Second {
		t.Fatalf("batas waktu %v tidak dipatuhi (%v)", f.Timeout, d)
	}
	_, err = f.Fetch(context.Background(), "http://tidak-ada.test/")
	wantErr(t, err, "tidak ditemukan")
}

func TestNormalize(t *testing.T) {
	if got := Normalize("HTTPS://Contoh.TEST/Jalan?a=1#bagian"); got != "https://contoh.test/Jalan?a=1" {
		t.Fatalf("normalize %s", got)
	}
}
