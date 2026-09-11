package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/api"
)

func newRoutedApp(stack Stack) *App {
	app := NewApp("test", stack)
	app.Handle(Route{
		Method: http.MethodGet, Pattern: "/items/{id}",
		Handler: func(r *Request) (ResponseEntity, error) { return OKText(r.PathVar("id")), nil },
	})
	app.Handle(Route{
		Method: http.MethodPost, Pattern: "/items",
		Consumes: []string{MediaTypeJSON},
		Handler:  func(*Request) (ResponseEntity, error) { return OKEmpty(), nil },
	})
	app.Handle(Route{
		Method: http.MethodGet, Pattern: "/items/search",
		Params:  []ParamCondition{{Name: "x", Value: "TX"}},
		Handler: func(*Request) (ResponseEntity, error) { return OKText("tx"), nil },
	})
	app.Handle(Route{
		Method: http.MethodGet, Pattern: "/search",
		Params:  []ParamCondition{{Name: "x", Value: "TX"}},
		Handler: func(*Request) (ResponseEntity, error) { return OKText("tx"), nil },
	})
	app.Handle(Route{
		Method: http.MethodGet, Pattern: "/boom",
		Handler: func(*Request) (ResponseEntity, error) { return ResponseEntity{}, errBoom{} },
	})
	return app
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// TestPathMatchingPrefersLiteralSegments covers the ordering Spring's
// AntPatternComparator imposes: a literal segment beats a template variable.
func TestPathMatchingPrefersLiteralSegments(t *testing.T) {
	mvc := NewMockMvc(newRoutedApp(StackServlet))
	if res := mvc.Perform(Get("/items/search?x=TX")); res.Body != "tx" {
		t.Errorf("the literal route lost to the template route: %q", res.Body)
	}
	if res := mvc.Perform(Get("/items/7")); res.Body != "7" {
		t.Errorf("the template route did not bind the variable: %q", res.Body)
	}
	// When the literal route's parameter condition fails, the template route
	// takes over — as Spring's second matching stage does.
	if res := mvc.Perform(Get("/items/search?x=CA")); res.Body != "search" {
		t.Errorf("the template route did not take over: %q", res.Body)
	}
}

// TestDispatcherStatuses covers the status each matching stage reports when it
// fails, which mirrors the exception the DispatcherServlet would have raised.
func TestDispatcherStatuses(t *testing.T) {
	mvc := NewMockMvc(newRoutedApp(StackServlet))
	cases := []struct {
		name       string
		build      func() *RequestBuilder
		wantStatus int
	}{
		{"no path match", func() *RequestBuilder { return Get("/nope") }, http.StatusNotFound},
		{"no method match", func() *RequestBuilder { return Delete("/items/7") }, http.StatusMethodNotAllowed},
		{"no parameter match", func() *RequestBuilder { return Get("/search?x=CA") }, http.StatusBadRequest},
		{"no consumes match", func() *RequestBuilder {
			return Post("/items").ContentType(MediaTypeTextPlain).Content("x")
		}, http.StatusUnsupportedMediaType},
		{"handler failure", func() *RequestBuilder { return Get("/boom") }, http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if res := mvc.Perform(c.build()); res.Status != c.wantStatus {
				t.Errorf("status = %d, want %d", res.Status, c.wantStatus)
			}
		})
	}
}

// TestHTTPErrorMessage covers the error text the dispatcher's own failures
// carry, which is what reaches a log when one is reported.
func TestHTTPErrorMessage(t *testing.T) {
	cases := []struct {
		err  *HTTPError
		want string
	}{
		{ErrBadRequest, "400 Bad Request"},
		{ErrNotFound, "404 Not Found"},
		{ErrUnsupported, "415 Unsupported Media Type"},
		{ErrNotAllowed, "405 Method Not Allowed"},
		{ErrInternalError, "500 Internal Server Error"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
		if _, ok := AsHTTPError(c.err); !ok {
			t.Errorf("%s was not recognised as an HTTP error", c.want)
		}
	}
}

// TestContextPathIsStripped covers server.servlet.context-path.
func TestContextPathIsStripped(t *testing.T) {
	app := newRoutedApp(StackServlet)
	app.ContextPath = "/mvc"
	mvc := NewMockMvc(app)
	if res := mvc.Perform(Get("/mvc/items/7")); res.Body != "7" {
		t.Errorf("under the context path, body = %q, want %q", res.Body, "7")
	}
	if res := mvc.Perform(Get("/items/7")); res.Status != http.StatusNotFound {
		t.Errorf("outside the context path, status = %d, want %d", res.Status, http.StatusNotFound)
	}
}

// TestErrorDocumentPerStack covers the two error documents, which differ in key
// order and in how they report `message`.
func TestErrorDocumentPerStack(t *testing.T) {
	servlet := NewMockMvc(newRoutedApp(StackServlet)).Perform(Get("/nope"))
	if !strings.HasPrefix(servlet.Body, `{"timestamp":"`) ||
		!strings.Contains(servlet.Body, `"status":404,"error":"Not Found","message":"","path":"/nope"`) {
		t.Errorf("the servlet error document is %s", servlet.Body)
	}

	reactive := NewMockMvc(newRoutedApp(StackReactive)).Perform(Get("/nope"))
	if !strings.Contains(reactive.Body, `"path":"/nope","status":404,"error":"Not Found","message":null,"requestId":"`) {
		t.Errorf("the reactive error document is %s", reactive.Body)
	}
	// A handler failure reports an empty message rather than a null one.
	boom := NewMockMvc(newRoutedApp(StackReactive)).Perform(Get("/boom"))
	if !strings.Contains(boom.Body, `"message":""`) {
		t.Errorf("the reactive 500 document is %s", boom.Body)
	}

	war := NewMockMvc(newRoutedApp(StackWAR)).Perform(Get("/nope"))
	if war.Body != "Not Found" || war.ContentType != "text/plain;charset=iso-8859-1" {
		t.Errorf("the WAR error response is %q with %q", war.Body, war.ContentType)
	}
}

// TestResponseEntityContentNegotiation covers the Content-Type each body shape
// gets, including the body-less entity that carries none at all.
func TestResponseEntityContentNegotiation(t *testing.T) {
	app := NewApp("test", StackServlet)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/text",
		Handler: func(*Request) (ResponseEntity, error) { return OKText("hi"), nil }})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/json",
		Handler: func(*Request) (ResponseEntity, error) { return OK(map[string]int{"a": 1}), nil }})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/empty",
		Handler: func(*Request) (ResponseEntity, error) { return OKEmpty(), nil }})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/produces",
		Produces: []string{MediaTypeJSON, MediaTypeXML},
		Handler:  func(*Request) (ResponseEntity, error) { return OKText("SUCCESS"), nil }})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/wildcard",
		Produces: []string{MediaTypeAll},
		Handler:  func(*Request) (ResponseEntity, error) { return OKText("hi"), nil }})

	mvc := NewMockMvc(app)
	cases := []struct{ path, body, contentType string }{
		{"/text", "hi", CharsetTextPlain},
		{"/json", `{"a":1}`, MediaTypeJSON},
		{"/empty", "", ""},
		// A declared `produces` wins over the body kind's default.
		{"/produces", "SUCCESS", MediaTypeJSON},
		// "*/*" is how Springfox spells "no produces"; it never reaches the wire.
		{"/wildcard", "hi", CharsetTextPlain},
	}
	for _, c := range cases {
		res := mvc.Perform(Get(c.path))
		if res.Body != c.body {
			t.Errorf("%s: body = %q, want %q", c.path, res.Body, c.body)
		}
		if res.ContentType != c.contentType {
			t.Errorf("%s: Content-Type = %q, want %q", c.path, res.ContentType, c.contentType)
		}
	}
}

// TestAcceptPredicate covers RequestPredicates.accept, which a wildcard Accept
// header satisfies, and the stricter negotiation the error controller needs.
func TestAcceptPredicate(t *testing.T) {
	app := NewApp("test", StackServlet)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/hello",
		Accept:  []string{MediaTypeTextPlain},
		Handler: func(*Request) (ResponseEntity, error) { return OKText("hi"), nil }})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/strict",
		Accept: []string{MediaTypeTextHTML}, AcceptExplicit: true,
		Handler: func(*Request) (ResponseEntity, error) { return OKText("html"), nil }})
	mvc := NewMockMvc(app)

	if res := mvc.Perform(Get("/hello").Accept(MediaTypeTextPlain)); res.Status != http.StatusOK {
		t.Errorf("with text/plain, status = %d", res.Status)
	}
	if res := mvc.Perform(Get("/hello").Accept("*/*")); res.Status != http.StatusOK {
		t.Errorf("with a wildcard Accept, status = %d", res.Status)
	}
	if res := mvc.Perform(Get("/hello").Accept(MediaTypeJSON)); res.Status != http.StatusNotFound {
		t.Errorf("with application/json, status = %d, want %d", res.Status, http.StatusNotFound)
	}
	if res := mvc.Perform(Get("/strict").Accept("*/*")); res.Status != http.StatusNotFound {
		t.Errorf("a strict route matched a wildcard Accept: %d", res.Status)
	}
	if res := mvc.Perform(Get("/strict").Accept(MediaTypeTextHTML)); res.Status != http.StatusOK {
		t.Errorf("a strict route rejected an explicit Accept: %d", res.Status)
	}
}

// TestRequestBinding covers the query and body helpers the handlers use.
func TestRequestBinding(t *testing.T) {
	app := NewApp("test", StackServlet)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/q",
		Handler: func(r *Request) (ResponseEntity, error) {
			all := r.QueryAll("n")
			m := r.QueryMap()
			return OKText(strings.Join(all, ",") + "|" + m["n"] + "|" + r.QueryOr("missing", "def")), nil
		}})
	app.Handle(Route{Method: http.MethodPost, Pattern: "/b",
		Handler: func(r *Request) (ResponseEntity, error) {
			var v struct {
				A int `json:"a"`
			}
			if err := r.BindJSON(&v); err != nil {
				return ResponseEntity{}, err
			}
			return OK(v), nil
		}})
	mvc := NewMockMvc(app)

	if res := mvc.Perform(Get("/q?n=1&n=2")); res.Body != "1,2|1|def" {
		t.Errorf("query binding = %q", res.Body)
	}
	if res := mvc.Perform(Post("/b").Content(`{"a":5}`)); res.Body != `{"a":5}` {
		t.Errorf("body binding = %q", res.Body)
	}
	// A malformed body is Spring's HttpMessageNotReadableException, a 400.
	if res := mvc.Perform(Post("/b").Content("{")); res.Status != http.StatusBadRequest {
		t.Errorf("a malformed body gave %d, want %d", res.Status, http.StatusBadRequest)
	}
	if res := mvc.Perform(Post("/b")); res.Status != http.StatusBadRequest {
		t.Errorf("an empty body gave %d, want %d", res.Status, http.StatusBadRequest)
	}
}

// TestOperationMetadataIsCarriedOnRoutes checks that a route's documentation
// metadata survives registration, since the documentation generators read
// nothing else.
func TestOperationMetadataIsCarriedOnRoutes(t *testing.T) {
	app := NewApp("test", StackServlet)
	app.Handle(Route{Method: http.MethodGet, Pattern: "/a",
		Handler: func(*Request) (ResponseEntity, error) { return OKEmpty(), nil },
		Op:      api.Operation{Name: "a", Tag: "t"}})
	app.Handle(Route{Method: http.MethodGet, Pattern: "/b/{id}",
		Handler: func(*Request) (ResponseEntity, error) { return OKEmpty(), nil },
		Op:      api.Operation{Name: "b", Tag: "t"}})
	app.Seal()

	routes := app.Routes()
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2", len(routes))
	}
	// Routes() reports registration order even after Seal reorders for matching.
	if routes[0].Op.Name != "a" || routes[1].Op.Name != "b" {
		t.Errorf("Routes() returned %q then %q, want registration order",
			routes[0].Op.Name, routes[1].Op.Name)
	}
}
