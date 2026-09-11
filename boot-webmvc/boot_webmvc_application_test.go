package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// TestContextLoads is BootWebmvcApplicationTests.contextLoads: a @SpringBootTest
// that asserts nothing beyond the application context starting.
func TestContextLoads(t *testing.T) {
	if _, err := NewApplication(); err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
}

// newTestMvc builds the application and a MockMvc over it.
func newTestMvc(t *testing.T) *web.MockMvc {
	t.Helper()
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	return web.NewMockMvc(app)
}

// TestHello covers GET /mvc/hello, whose body and text/plain media type the
// Java module served from Spring's StringHttpMessageConverter.
func TestHello(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/mvc/hello"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "Hello SpringFox!" {
		t.Errorf("body = %q, want %q", res.Body, "Hello SpringFox!")
	}
	if res.ContentType != "text/plain;charset=UTF-8" {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, "text/plain;charset=UTF-8")
	}
}

// TestTestDoubleFormatting covers GET /mvc/hello/double. The Java handler
// concatenates a boxed Double into a String, so the response body carries
// java.lang.Double.toString's layout — behaviour the JDK used to supply and the
// port now implements itself.
func TestTestDoubleFormatting(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct {
		query string
		want  string
	}{
		{"", "Value 10.0"},
		{"?test=10", "Value 10.0"},
		{"?test=3.5", "Value 3.5"},
		{"?test=1", "Value 1.0"},
		{"?test=0.0001", "Value 1.0E-4"},
		{"?test=0.001", "Value 0.001"},
		{"?test=1e21", "Value 1.0E21"},
		{"?test=1234567", "Value 1234567.0"},
		{"?test=1e7", "Value 1.0E7"},
		{"?test=-0", "Value -0.0"},
		{"?test=NaN", "Value NaN"},
		{"?test=Infinity", "Value Infinity"},
		{"?test=-Infinity", "Value -Infinity"},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get("/mvc/hello/double" + c.query))
		if res.Status != http.StatusOK {
			t.Fatalf("%q: status = %d, want %d", c.query, res.Status, http.StatusOK)
		}
		if res.Body != c.want {
			t.Errorf("%q: body = %q, want %q", c.query, res.Body, c.want)
		}
	}
}

// TestTestDoubleRejectsNonNumeric covers Spring's type-conversion failure.
func TestTestDoubleRejectsNonNumeric(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/mvc/hello/double?test=abc"))
	if res.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusBadRequest)
	}
	if res.ContentType != web.MediaTypeJSON {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeJSON)
	}
}

// TestCsrfCookieIsIssued covers CookieCsrfTokenRepository.withHttpOnlyFalse():
// the token cookie is scoped to the context path and readable by JavaScript.
func TestCsrfCookieIsIssued(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/mvc/hello"))
	cookies := res.Headers.Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("no Set-Cookie header was written")
	}
	cookie := cookies[0]
	for _, want := range []string{web.CsrfCookieName + "=", "Path=" + contextPath} {
		if !strings.Contains(cookie, want) {
			t.Errorf("cookie %q does not contain %q", cookie, want)
		}
	}
	if strings.Contains(cookie, "HttpOnly") {
		t.Errorf("cookie %q is HttpOnly, but withHttpOnlyFalse() clears that flag", cookie)
	}
}

// TestCsrfRejectsUntokenedPost covers the 403 a state-changing request without
// the token gets.
func TestCsrfRejectsUntokenedPost(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/mvc/hello"))
	if res.Status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusForbidden)
	}
}

// TestCsrfAcceptsTokenedPost covers the other half of the CSRF contract: a
// request that echoes the token back is dispatched normally.
func TestCsrfAcceptsTokenedPost(t *testing.T) {
	mvc := newTestMvc(t)
	first := mvc.Perform(web.Get("/mvc/hello"))
	token := cookieValue(first.Headers.Values("Set-Cookie"), web.CsrfCookieName)
	if token == "" {
		t.Fatal("no CSRF token was issued")
	}
	res := mvc.Perform(web.Post("/mvc/hello").
		Header("Cookie", web.CsrfCookieName+"="+token).
		Header(web.CsrfHeaderName, token))
	// /mvc/hello maps GET only, so a tokened POST gets past the CSRF filter and
	// is rejected by the dispatcher instead.
	if res.Status != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusMethodNotAllowed)
	}
}

// TestDocumentationEndpoints covers the two api-docs endpoints and the
// swagger-resources listing under the configured base URL.
func TestDocumentationEndpoints(t *testing.T) {
	mvc := newTestMvc(t)
	for _, path := range []string{"/mvc/v2/api-docs", "/mvc/v3/api-docs"} {
		res := mvc.Perform(web.Get(path))
		if res.Status != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusOK)
		}
		if res.ContentType != web.MediaTypeJSON {
			t.Errorf("%s: Content-Type = %q, want %q", path, res.ContentType, web.MediaTypeJSON)
		}
	}
	res := mvc.Perform(web.Get("/mvc/documentation/swagger-resources"))
	want := `[{"name":"default","url":"/v3/api-docs","swaggerVersion":"3.0.3","location":"/v3/api-docs"}]`
	if res.Body != want {
		t.Errorf("swagger-resources = %s, want %s", res.Body, want)
	}
	// The listing lives under the base URL only.
	if bare := mvc.Perform(web.Get("/mvc/swagger-resources")); bare.Status != http.StatusNotFound {
		t.Errorf("/mvc/swagger-resources status = %d, want %d", bare.Status, http.StatusNotFound)
	}
}

// TestSecurityConfigurationIsPublished covers the SecurityConfiguration bean.
func TestSecurityConfigurationIsPublished(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/mvc/documentation/swagger-resources/configuration/security"))
	if res.Body != `{"enableCsrfSupport":true}` {
		t.Errorf("security configuration = %s, want %s", res.Body, `{"enableCsrfSupport":true}`)
	}
}

// cookieValue extracts a cookie value from a set of Set-Cookie headers.
func cookieValue(headers []string, name string) string {
	prefix := name + "="
	for _, h := range headers {
		if !strings.HasPrefix(h, prefix) {
			continue
		}
		value := h[len(prefix):]
		if i := strings.Index(value, ";"); i >= 0 {
			return value[:i]
		}
		return value
	}
	return ""
}
