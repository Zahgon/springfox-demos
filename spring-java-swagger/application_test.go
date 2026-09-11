package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

func newTestMvc(t *testing.T) *web.MockMvc {
	t.Helper()
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	return web.NewMockMvc(app)
}

// TestContextLoads gives the module the construction guarantee the boot
// modules' contextLoads tests give. The Java module had no test of its own.
func TestContextLoads(t *testing.T) {
	if _, err := NewApplication(); err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
}

// TestSayHello covers GET /hello under the WAR's context path. The container
// the original deploys into negotiates ISO-8859-1 rather than UTF-8.
func TestSayHello(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/spring-java-swagger/hello"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "Hello World" {
		t.Errorf("body = %q, want %q", res.Body, "Hello World")
	}
	if res.ContentType != "text/plain;charset=iso-8859-1" {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, "text/plain;charset=iso-8859-1")
	}
}

// TestSayHelloIsMappedForEveryMethod covers the bare @RequestMapping("/hello"),
// which declares no `method` attribute and is therefore mapped for all of them.
func TestSayHelloIsMappedForEveryMethod(t *testing.T) {
	mvc := newTestMvc(t)
	for _, method := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodTrace,
	} {
		res := mvc.Perform(web.RequestFor(method, "/spring-java-swagger/hello"))
		if res.Status != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", method, res.Status, http.StatusOK)
		}
	}
}

// TestOnlyTheOpenAPIEndpointIsServed covers @EnableOpenApi: the module serves
// /v3/api-docs and not /v2/api-docs.
func TestOnlyTheOpenAPIEndpointIsServed(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Get("/spring-java-swagger/v3/api-docs"))
	if res.Status != http.StatusOK {
		t.Fatalf("v3 status = %d, want %d", res.Status, http.StatusOK)
	}
	for _, want := range []string{
		`"openapi":"3.0.3"`,
		`"/spring-java-swagger/hello"`,
		`"servers":[{"url":"http://localhost:8080","description":"Inferred Url"}]`,
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("the document does not contain %s", want)
		}
	}
	if v2 := mvc.Perform(web.Get("/spring-java-swagger/v2/api-docs")); v2.Status != http.StatusNotFound {
		t.Errorf("v2 status = %d, want %d", v2.Status, http.StatusNotFound)
	}
}

// TestSwaggerUIIsMounted covers SpringConfig's resource handler and view
// controller.
func TestSwaggerUIIsMounted(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/spring-java-swagger/swagger-ui/"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if !strings.Contains(res.Body, "<title>Swagger UI</title>") {
		t.Error("the forwarded view is not the swagger-ui index page")
	}
}

// TestServletInitializerDeclarations pins the bootstrap values the original's
// ServletInitializer declares.
func TestServletInitializerDeclarations(t *testing.T) {
	var init ServletInitializer
	mappings := init.ServletMappings()
	if len(mappings) != 1 || mappings[0] != "/" {
		t.Errorf("servlet mappings = %v, want [/]", mappings)
	}
	if init.RootConfigClasses() != nil {
		t.Error("the root configuration classes should be nil")
	}
}
