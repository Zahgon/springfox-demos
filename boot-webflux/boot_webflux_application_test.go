package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// TestContextLoads is BootWebfluxApplicationTests.contextLoads: a
// @SpringBootTest that asserts nothing beyond the application context starting.
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

// TestFunctionalHelloWorldController covers the four annotated handlers,
// including WebFlux's plain-text encoding of a Mono and a Flux of String.
func TestFunctionalHelloWorldController(t *testing.T) {
	mvc := newTestMvc(t)
	cases := []struct {
		path        string
		wantStatus  int
		wantBody    string
		wantContent string
	}{
		{"/functional-hello/response-mono", http.StatusOK, "Hello SpringFox!", web.CharsetTextPlain},
		{"/functional-hello/mono?name=Frodo", http.StatusOK, "Hello Frodo!", web.CharsetTextPlain},
		// A plain String parameter is optional and binds to null when absent.
		{"/functional-hello/mono", http.StatusOK, "Hello null!", web.CharsetTextPlain},
		// A Flux<String> is written element after element with no separator.
		{"/functional-hello/flux?names=a&names=b", http.StatusOK, "Hello aHello b", web.CharsetTextPlain},
		{"/functional-hello/response-flux?names=a&names=b", http.StatusOK, "Hello aHello b", web.CharsetTextPlain},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get(c.path))
		if res.Status != c.wantStatus {
			t.Errorf("%s: status = %d, want %d", c.path, res.Status, c.wantStatus)
		}
		if res.Body != c.wantBody {
			t.Errorf("%s: body = %q, want %q", c.path, res.Body, c.wantBody)
		}
		if res.ContentType != c.wantContent {
			t.Errorf("%s: Content-Type = %q, want %q", c.path, res.ContentType, c.wantContent)
		}
	}
}

// TestFluxWithoutNamesFails covers the defect the original carries: `String...
// names` binds to null when the parameter is absent, and Flux.fromArray(null)
// throws, so the request fails with a 500 rather than returning an empty body.
func TestFluxWithoutNamesFails(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/functional-hello/flux"))
	if res.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
}

// TestResponseFluxRequiresNames covers the required @RequestParam.
func TestResponseFluxRequiresNames(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/functional-hello/response-flux"))
	if res.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusBadRequest)
	}
}

// TestGreetingRouterFunction covers the RouterFunction bean, whose accept
// predicate makes the route invisible to a client that will not take text/plain.
func TestGreetingRouterFunction(t *testing.T) {
	mvc := newTestMvc(t)

	res := mvc.Perform(web.Get("/hello").Accept(web.MediaTypeTextPlain))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	// Note the comma: the router function's greeting differs from the
	// annotated controller's.
	if res.Body != "Hello, SpringFox!" {
		t.Errorf("body = %q, want %q", res.Body, "Hello, SpringFox!")
	}
	if res.ContentType != web.MediaTypeTextPlain {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeTextPlain)
	}

	if json := mvc.Perform(web.Get("/hello").Accept(web.MediaTypeJSON)); json.Status != http.StatusNotFound {
		t.Errorf("with Accept: application/json, status = %d, want %d", json.Status, http.StatusNotFound)
	}
}

// TestReactiveErrorDocument covers the error document the reactive stack
// renders, which differs from the servlet stack's in key order and in its
// treatment of `message`.
func TestReactiveErrorDocument(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/nope"))
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusNotFound)
	}
	for _, want := range []string{`"path":"/nope"`, `"status":404`, `"error":"Not Found"`, `"message":null`, `"requestId":"`} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("error document %s does not contain %s", res.Body, want)
		}
	}
}

// TestDocumentationEndpoints covers the api-docs endpoints and the resource
// listing, which advertises only the OpenAPI version.
func TestDocumentationEndpoints(t *testing.T) {
	mvc := newTestMvc(t)
	for _, path := range []string{"/v2/api-docs", "/v3/api-docs"} {
		if res := mvc.Perform(web.Get(path)); res.Status != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusOK)
		}
	}
	res := mvc.Perform(web.Get("/documentation/swagger-resources"))
	want := `[{"name":"default","url":"/v3/api-docs","swaggerVersion":"3.0.3","location":"/v3/api-docs"}]`
	if res.Body != want {
		t.Errorf("swagger-resources = %s, want %s", res.Body, want)
	}
}

// TestNullPointerMessage covers the Java exception name the varargs failure
// carries, which is what makes the 500 recognisable in a log.
func TestNullPointerMessage(t *testing.T) {
	if got := (errNullPointer{}).Error(); got != "NullPointerException" {
		t.Errorf("Error() = %q, want %q", got, "NullPointerException")
	}
}
