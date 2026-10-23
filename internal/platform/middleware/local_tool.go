package middleware

import (
	"mime"
	"net"
	"net/http"
	"net/url"
)

// maxToolBody bounds request bodies of the local room-config tool.
const maxToolBody = 1 << 20 // 1 MiB

// LocalToolGuard protects a tool that edits files on the operator's machine
// from being driven by web pages the operator visits:
//
//   - the Host header must be a loopback name, which defeats DNS rebinding;
//   - writes must come from a loopback origin (the embedded UI or the Vite
//     dev server) or carry no Origin (curl) and be JSON, so cross-site
//     "simple" requests, which browsers send without a CORS preflight, are
//     refused;
//   - request bodies are capped at 1 MiB.
func LocalToolGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY") // no clickjacking of the editor

		if !isLoopbackHost(r.Host) {
			http.Error(w, "Forbidden host", http.StatusForbidden)
			return
		}

		if isWrite(r.Method) {
			if origin := r.Header.Get("Origin"); origin != "" && !isLoopbackOrigin(origin) {
				http.Error(w, "Cross-origin request refused", http.StatusForbidden)
				return
			}
			if r.Method != http.MethodDelete {
				if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
					http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
					return
				}
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxToolBody)
		}

		next.ServeHTTP(w, r)
	})
}

func isWrite(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

func hostname(hostport string) string {
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}
	return hostport
}

func isLoopbackHost(hostport string) bool {
	host := hostname(hostport)
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isLoopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && isLoopbackHost(u.Host)
}
