package main

import (
	"net/http"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// TestContextLoads is SpringIntegrationWebMvcApplicationTests.contextLoads.
func TestContextLoads(t *testing.T) {
	if _, err := NewApplication(); err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
}

func newTestMvc(t *testing.T) *web.MockMvc {
	t.Helper()
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	return web.NewMockMvc(app)
}

// TestToUpperGateway covers the text/plain POST gateway.
func TestToUpperGateway(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Post("/conversions/upper").
		ContentType(web.MediaTypeTextPlain).
		Content("aragorn"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "ARAGORN" {
		t.Errorf("body = %q, want %q", res.Body, "ARAGORN")
	}
	if res.ContentType != web.CharsetTextPlain {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.CharsetTextPlain)
	}
	// The gateway declares consumes = text/plain.
	wrong := mvc.Perform(web.Post("/conversions/upper").
		ContentType(web.MediaTypeJSON).
		Content("aragorn"))
	if wrong.Status != http.StatusUnsupportedMediaType {
		t.Errorf("with application/json, status = %d, want %d",
			wrong.Status, http.StatusUnsupportedMediaType)
	}
}

// TestToUpperLowerGateway covers the GET gateway, whose header expression
// compares the path variable against the literal "upper", case-sensitively.
func TestToUpperLowerGateway(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct{ path, want string }{
		{"/conversions/pathvariable/upper?toConvert=Gimli", "GIMLI"},
		{"/conversions/pathvariable/lower?toConvert=Gimli", "gimli"},
		// Any value other than the exact literal takes the lower-casing branch.
		{"/conversions/pathvariable/UPPER?toConvert=Gimli", "gimli"},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get(c.path))
		if res.Status != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", c.path, res.Status, http.StatusOK)
		}
		if res.Body != c.want {
			t.Errorf("%s: body = %q, want %q", c.path, res.Body, c.want)
		}
		// No request Content-Type means Spring MVC falls back to ISO-8859-1.
		if res.ContentType != web.CharsetTextPlainLatin1 {
			t.Errorf("%s: Content-Type = %q, want %q", c.path, res.ContentType, web.CharsetTextPlainLatin1)
		}
	}
	missing := mvc.Perform(web.Get("/conversions/pathvariable/upper"))
	if missing.Status != http.StatusBadRequest {
		t.Errorf("without toConvert, status = %d, want %d", missing.Status, http.StatusBadRequest)
	}
}

// TestConvertController covers the @PostMapping on the application class, which
// is a @RestController in this module.
func TestConvertController(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/conversion/controller").
		ContentType(web.MediaTypeJSON).
		Content(`{"gnarf":"dragons"}`))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != `{"gnarf":"dragons"}` {
		t.Errorf("body = %s, want %s", res.Body, `{"gnarf":"dragons"}`)
	}
	if res.ContentType != web.MediaTypeJSON {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeJSON)
	}
}

// TestFooConstructorDiscardsClientValues covers the observable consequence of
// the one-argument Foo constructor: the reply always carries foo=false and
// count=3, whatever the client sent.
func TestFooConstructorDiscardsClientValues(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/conversions/lower").
		ContentType(web.MediaTypeJSON).
		Content(`{"bar":"Aragorn","foo":true,"count":99}`))
	want := `{"bar":"aragorn","foo":false,"count":3}`
	if res.Body != want {
		t.Errorf("body = %s, want %s", res.Body, want)
	}
}
