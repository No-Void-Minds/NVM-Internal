// Package linkpreview mengambil judul, deskripsi, dan gambar pratinjau sebuah tautan untuk kartu tautan Whiteboard
// (docs/whiteboard.md bagian 5). Pengambilan dilakukan server, jadi perlindungan SSRF ada di sini:
//   - hanya http dan https, port 80 atau 443, tanpa userinfo;
//   - nama host di-resolve sendiri, lalu yang disambungi adalah alamat IP hasil resolve itu, bukan nama host, jadi
//     DNS rebinding tidak bisa membelokkan sambungan setelah pemeriksaan;
//   - semua alamat hasil resolve diperiksa; satu saja yang terlarang (loopback, privat termasuk LAN kantor,
//     link-local dan metadata awan, CGNAT, multicast, alamat khusus) membuat tautan ditolak;
//   - hanya IPv4 yang disambungi: VM produksi tidak punya rute IPv6, jadi host yang hanya punya alamat IPv6 langsung
//     ditolak dengan pesan jelas, tidak menunggu batas waktu;
//   - setiap redirect (maksimal 3) melewati pemeriksaan yang sama;
//   - batas waktu total, batas ukuran body, hanya HTML, tanpa cookie, tanpa proxy dari env.
package linkpreview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	MaxURL       = 2048
	MaxBody      = 512 << 10
	MaxImage     = 5 << 20
	MaxRedirects = 3
	userAgent    = "Mozilla/5.0 (compatible; NVMInternalPreview/1.0)"
)

// Galat dengan pesan yang aman ditampilkan ke pengguna.
type Error struct {
	Blocked bool
	Message string
}

func (e *Error) Error() string { return e.Message }

func blocked(msg string) *Error { return &Error{Blocked: true, Message: msg} }
func failed(msg string) *Error  { return &Error{Message: msg} }

var (
	ErrBlocked  = blocked("Alamat ini menunjuk ke jaringan internal, jadi pratinjaunya tidak diambil.")
	ErrIPv6Only = failed("Situs ini hanya punya alamat IPv6, yang tidak bisa dijangkau server. Tautannya tetap tersimpan tanpa pratinjau.")
)

// Resolver mengembalikan alamat IP sebuah nama host. Bisa diganti di test.
type Resolver func(ctx context.Context, host string) ([]netip.Addr, error)

func defaultResolver(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

type Fetcher struct {
	Resolve Resolver
	// Timeout total satu pengambilan halaman, termasuk redirect.
	Timeout time.Duration
	// allowAddr dan anyPort hanya untuk test di paket ini: server uji berjalan di 127.0.0.1 dengan port acak.
	allowAddr func(netip.Addr) bool
	anyPort   bool
}

func New() *Fetcher {
	return &Fetcher{Resolve: defaultResolver, Timeout: 5 * time.Second}
}

var forbidden = func() []netip.Prefix {
	out := []netip.Prefix{}
	for _, p := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24",
		"203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4", "255.255.255.255/32",
		"::/128", "::1/128", "fc00::/7", "fe80::/10", "ff00::/8", "2001:db8::/32", "100::/64",
	} {
		out = append(out, netip.MustParsePrefix(p))
	}
	return out
}()

var nat64 = netip.MustParsePrefix("64:ff9b::/96")

// Forbidden: alamat yang tidak boleh disambungi pengambil pratinjau. IPv4 yang dibungkus IPv6 (::ffff:a.b.c.d)
// dan NAT64 (64:ff9b::a.b.c.d) diperiksa sebagai IPv4 di dalamnya.
func Forbidden(a netip.Addr) bool {
	a = a.Unmap()
	if a.Is6() && nat64.Contains(a) {
		b := a.As16()
		a = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	if !a.IsValid() || a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() ||
		a.IsInterfaceLocalMulticast() || a.IsMulticast() || a.IsUnspecified() {
		return true
	}
	for _, p := range forbidden {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func (f *Fetcher) allowed(a netip.Addr) bool {
	if f.allowAddr != nil {
		return f.allowAddr(a)
	}
	return !Forbidden(a)
}

// CheckURL memeriksa bentuk URL sebelum apa pun di-resolve.
func (f *Fetcher) CheckURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return blocked("Pratinjau hanya untuk alamat http:// atau https://.")
	}
	if u.User != nil {
		return blocked("Alamat dengan nama pengguna atau password tidak dipratinjau.")
	}
	if u.Hostname() == "" {
		return failed("Alamat tautan tidak lengkap.")
	}
	// IP langsung yang terlarang disebut sebagai jaringan internal, alasan yang lebih jelas daripada port.
	if a, err := netip.ParseAddr(u.Hostname()); err == nil && !f.allowed(a) {
		return ErrBlocked
	}
	if p := u.Port(); p != "" && !f.anyPort {
		if (u.Scheme == "http" && p != "80") || (u.Scheme == "https" && p != "443") {
			return blocked("Pratinjau hanya untuk port 80 atau 443.")
		}
	}
	return nil
}

// dial me-resolve host, memeriksa semua alamatnya, lalu menyambung ke alamat IPv4 pertama yang lolos.
func (f *Fetcher) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	var addrs []netip.Addr
	if ip, err := netip.ParseAddr(host); err == nil {
		addrs = []netip.Addr{ip}
	} else {
		addrs, err = f.Resolve(ctx, host)
		if err != nil {
			return nil, failed("Nama situs tidak ditemukan.")
		}
	}
	var v4 []netip.Addr
	for _, a := range addrs {
		if !f.allowed(a) {
			return nil, ErrBlocked
		}
		if a.Unmap().Is4() {
			v4 = append(v4, a.Unmap())
		}
	}
	if len(addrs) == 0 {
		return nil, failed("Nama situs tidak ditemukan.")
	}
	if len(v4) == 0 {
		return nil, ErrIPv6Only
	}
	d := net.Dialer{Timeout: 3 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
		// Lapis kedua: alamat yang benar-benar disambungi diperiksa lagi.
		ap, err := netip.ParseAddrPort(address)
		if err != nil || !f.allowed(ap.Addr()) {
			return ErrBlocked
		}
		return nil
	}}
	var last error
	for _, a := range v4 {
		c, err := d.DialContext(ctx, "tcp4", net.JoinHostPort(a.String(), port))
		if err == nil {
			return c, nil
		}
		last = err
	}
	var e *Error
	if errors.As(last, &e) {
		return nil, e
	}
	return nil, failed("Situs tidak bisa dihubungi.")
}

func (f *Fetcher) client() *http.Client {
	tr := &http.Transport{
		Proxy:                 nil,
		DialContext:           f.dial,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 4 * time.Second,
		MaxIdleConns:          4,
		IdleConnTimeout:       10 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > MaxRedirects {
				return failed("Tautan dialihkan terlalu banyak kali.")
			}
			return f.CheckURL(req.URL)
		},
	}
}

// Page adalah hasil pengambilan satu halaman.
type Page struct {
	FinalURL    string
	Title       string
	Description string
	SiteName    string
	ImageURL    string
}

func (f *Fetcher) get(ctx context.Context, raw string, max int64, accept string) (*http.Response, []byte, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > MaxURL {
		return nil, nil, failed("Alamat tautan tidak sah.")
	}
	if err := f.CheckURL(u); err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, failed("Alamat tautan tidak sah.")
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	res, err := f.client().Do(req)
	if err != nil {
		var e *Error
		if errors.As(err, &e) {
			return nil, nil, e
		}
		if ctx.Err() != nil {
			return nil, nil, failed("Situs terlalu lama menjawab.")
		}
		return nil, nil, failed("Situs tidak bisa dihubungi.")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 {
		return res, nil, failed(fmt.Sprintf("Situs menjawab dengan galat %d.", res.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, max+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, failed("Situs terlalu lama menjawab.")
		}
		return nil, nil, failed("Isi halaman gagal dibaca.")
	}
	if int64(len(body)) > max {
		return res, body[:max], nil
	}
	return res, body, nil
}

// Fetch mengambil satu halaman HTML dan membaca title serta meta og. Body di atas MaxBody dipotong (bagian head
// biasanya di awal); selain HTML ditolak.
func (f *Fetcher) Fetch(ctx context.Context, raw string) (Page, error) {
	ctx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()
	res, body, err := f.get(ctx, raw, MaxBody, "text/html,application/xhtml+xml")
	if err != nil {
		return Page{}, err
	}
	mt, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if mt != "text/html" && mt != "application/xhtml+xml" {
		return Page{}, failed("Tautan ini bukan halaman web, jadi tidak ada pratinjaunya.")
	}
	p := parse(body)
	p.FinalURL = res.Request.URL.String()
	if p.ImageURL != "" {
		if iu, err := res.Request.URL.Parse(p.ImageURL); err == nil {
			p.ImageURL = iu.String()
		} else {
			p.ImageURL = ""
		}
	}
	return p, nil
}

// FetchImage mengambil gambar og:image dengan pemeriksaan yang sama, maksimal MaxImage. Jenis gambarnya ditentukan
// pemanggil dari isi, bukan dari Content-Type.
func (f *Fetcher) FetchImage(ctx context.Context, raw string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()
	_, body, err := f.get(ctx, raw, MaxImage+1, "image/png,image/jpeg,image/webp,image/gif")
	if err != nil {
		return nil, err
	}
	if len(body) > MaxImage {
		return nil, failed("Gambar pratinjau terlalu besar.")
	}
	return body, nil
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func parse(body []byte) Page {
	var p Page
	z := html.NewTokenizer(bytes.NewReader(body))
	inTitle := false
	var title strings.Builder
	for {
		switch z.Next() {
		case html.ErrorToken:
			if p.Title == "" {
				p.Title = clip(title.String(), 300)
			}
			return p
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "title":
				inTitle = true
			case "body":
				// Meta di body tidak dipakai; berhenti lebih awal.
				if p.Title == "" {
					p.Title = clip(title.String(), 300)
				}
				return p
			case "meta":
				var prop, content string
				for hasAttr {
					var k, v []byte
					k, v, hasAttr = z.TagAttr()
					switch strings.ToLower(string(k)) {
					case "property", "name":
						prop = strings.ToLower(string(v))
					case "content":
						content = string(v)
					}
				}
				switch prop {
				case "og:title":
					p.Title = clip(content, 300)
				case "og:description":
					p.Description = clip(content, 500)
				case "description":
					if p.Description == "" {
						p.Description = clip(content, 500)
					}
				case "og:site_name":
					p.SiteName = clip(content, 120)
				case "og:image", "og:image:url", "og:image:secure_url":
					if p.ImageURL == "" && len(content) <= MaxURL {
						p.ImageURL = strings.TrimSpace(content)
					}
				}
			}
		case html.TextToken:
			if inTitle {
				title.Write(z.Text())
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			if string(name) == "title" {
				inTitle = false
			}
		}
	}
}

// Normalize membuang fragment dan menyeragamkan skema serta host, untuk kunci cache.
func Normalize(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String()
}
