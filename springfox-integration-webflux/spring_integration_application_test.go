package main

import (
	"net/http"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// TestContextLoads is SpringIntegrationApplicationTests.contextLoads.
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

// TestToUpperFlow covers the text/plain POST gateway.
func TestToUpperFlow(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/conversions/upper").
		ContentType(web.MediaTypeTextPlain).
		Content("aragorn"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "ARAGORN" {
		t.Errorf("body = %q, want %q", res.Body, "ARAGORN")
	}
	// The reactive stack writes text/plain without a charset parameter.
	if res.ContentType != web.MediaTypeTextPlain {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeTextPlain)
	}
}

// TestToUpperGetFlow covers the GET gateway.
func TestToUpperGetFlow(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct{ path, want string }{
		{"/conversions/pathvariable/upper?toConvert=Gimli", "GIMLI"},
		{"/conversions/pathvariable/lower?toConvert=Gimli", "gimli"},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get(c.path))
		if res.Status != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", c.path, res.Status, http.StatusOK)
		}
		if res.Body != c.want {
			t.Errorf("%s: body = %q, want %q", c.path, res.Body, c.want)
		}
	}
	if missing := mvc.Perform(web.Get("/conversions/pathvariable/upper")); missing.Status != http.StatusBadRequest {
		t.Errorf("without toConvert, status = %d, want %d", missing.Status, http.StatusBadRequest)
	}
}

// TestToLowerFlowFailsWithClassCast pins the defect the original carries:
// toLowerFlow deserialises a Foo and then hands it to a handler whose erased
// parameter type is String, so the request fails with a 500. The port
// reproduces the failure rather than repairing it.
func TestToLowerFlowFailsWithClassCast(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/conversions/lower").
		ContentType(web.MediaTypeJSON).
		Content(`{"bar":"Aragorn"}`))
	if res.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
}

// TestConvertControllerIsNotMapped pins the other deliberate non-behaviour:
// the application class carries no controller stereotype, so its @PostMapping
// method is never mapped.
func TestConvertControllerIsNotMapped(t *testing.T) {
	res := newTestMvc(t).Perform(web.Post("/conversion/controller").
		ContentType(web.MediaTypeJSON).
		Content(`{"barf":"dragons"}`))
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusNotFound)
	}
	// The method itself still exists and still echoes its argument.
	if got := convert(Baz{Barf: "dragons"}); got.Barf != "dragons" {
		t.Errorf("convert returned %+v, want barf dragons", got)
	}
}

// TestActuatorEndpoints covers the three endpoints
// spring-boot-starter-actuator contributes.
func TestActuatorEndpoints(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct{ path, want string }{
		{"/actuator/health", `{"status":"UP"}`},
		{"/actuator/info", `{}`},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get(c.path))
		if res.Status != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", c.path, res.Status, http.StatusOK)
		}
		if res.Body != c.want {
			t.Errorf("%s: body = %s, want %s", c.path, res.Body, c.want)
		}
	}
	links := mvc.Perform(web.Get("/actuator"))
	want := `{"_links":{"self":{"href":"http://localhost:8080/actuator","templated":false},` +
		`"health-path":{"href":"http://localhost:8080/actuator/health/{*path}","templated":true},` +
		`"health":{"href":"http://localhost:8080/actuator/health","templated":false},` +
		`"info":{"href":"http://localhost:8080/actuator/info","templated":false}}}`
	if links.Body != want {
		t.Errorf("/actuator = %s, want %s", links.Body, want)
	}
}
