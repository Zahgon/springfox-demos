package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

// CsrfCookieName and CsrfHeaderName are Spring Security's defaults for
// CookieCsrfTokenRepository.
const (
	CsrfCookieName = "XSRF-TOKEN"
	CsrfHeaderName = "X-XSRF-TOKEN"
	CsrfParamName  = "_csrf"
)

// CsrfFilter reproduces
//
//	http.csrf().csrfTokenRepository(CookieCsrfTokenRepository.withHttpOnlyFalse())
//
// It issues a token cookie on every response — scoped to the context path, and
// readable from JavaScript because withHttpOnlyFalse() clears the HttpOnly flag
// — and rejects a state-changing request that does not echo the token back.
func CsrfFilter(contextPath string) Filter {
	cookiePath := contextPath
	if cookiePath == "" {
		cookiePath = "/"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := newCsrfToken()
			if c, err := r.Cookie(CsrfCookieName); err == nil && c.Value != "" {
				token = c.Value
			}
			http.SetCookie(w, &http.Cookie{
				Name:     CsrfCookieName,
				Value:    token,
				Path:     cookiePath,
				HttpOnly: false,
			})
			if !csrfExempt(r.Method) && !csrfTokenMatches(r, token) {
				writeCsrfDenied(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// csrfExempt lists the methods Spring Security treats as safe.
func csrfExempt(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func csrfTokenMatches(r *http.Request, token string) bool {
	if v := r.Header.Get(CsrfHeaderName); v != "" {
		return v == token
	}
	if v := r.URL.Query().Get(CsrfParamName); v != "" {
		return v == token
	}
	return false
}

func writeCsrfDenied(w http.ResponseWriter, r *http.Request) {
	payload := mustMarshalOrdered([]kv{
		{"timestamp", nowTimestamp()},
		{"status", http.StatusForbidden},
		{"error", "Forbidden"},
		{"message", ""},
		{"path", r.URL.Path},
	})
	w.Header().Set("Content-Type", MediaTypeJSON)
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write(payload)
}

func newCsrfToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}
	// Format the bytes as a random (version 4) UUID, the shape
	// CookieCsrfTokenRepository produces.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	h := hex.EncodeToString(buf[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
