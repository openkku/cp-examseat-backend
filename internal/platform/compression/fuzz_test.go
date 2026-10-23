package compression

import "testing"

// Accept-Encoding is attacker-controlled; negotiation must never panic and
// must only ever pick a supported encoding.
func FuzzNegotiate(f *testing.F) {
	for _, seed := range []string{"", "gzip", "br;q=0.5, zstd;q=0.9", "*;q=0", ";;;,,q=", "gzip;q=NaN", "zstd;q=1e309"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, header string) {
		switch enc := Negotiate(header); enc {
		case "", "zstd", "br", "gzip", "deflate":
		default:
			t.Fatalf("Negotiate(%q) = %q", header, enc)
		}
	})
}
