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

// TestPetstoreIsTheOnlyApiSurface covers the component scan of
// springfox.petstore.controller, which is everything this module maps.
func TestPetstoreIsTheOnlyApiSurface(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Get("/spring-xml-swagger/api/user/login?username=a&password=b"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if !strings.HasPrefix(res.Body, "logged in user session:") {
		t.Errorf("body = %q, want a logged-in session line", res.Body)
	}
	if res.ContentType != web.MediaTypeJSON {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeJSON)
	}
	if empty := mvc.Perform(web.Get("/spring-xml-swagger/api/pet/findByStatus?status=available")); empty.Body != "[]" {
		t.Errorf("findByStatus = %q, want an empty array", empty.Body)
	}
}

// TestOnlyTheSwagger2EndpointIsServed covers @EnableSwagger2 on
// ApplicationSwaggerConfig: the module serves /v2/api-docs and not /v3.
func TestOnlyTheSwagger2EndpointIsServed(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Get("/spring-xml-swagger/v2/api-docs"))
	if res.Status != http.StatusOK {
		t.Fatalf("v2 status = %d, want %d", res.Status, http.StatusOK)
	}
	for _, want := range []string{
		`"swagger":"2.0"`,
		`"/spring-xml-swagger/api/pet/{petId}"`,
		`"pet-controller"`,
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("the document does not contain %s", want)
		}
	}
	if v3 := mvc.Perform(web.Get("/spring-xml-swagger/v3/api-docs")); v3.Status != http.StatusNotFound {
		t.Errorf("v3 status = %d, want %d", v3.Status, http.StatusNotFound)
	}
}

// TestSwaggerUiTrailingSlashIsNotForwarded pins a difference from the two
// code-configured modules: the XML declares only <mvc:resources>, with no view
// controller, so /swagger-ui/ resolves to nothing.
func TestSwaggerUiTrailingSlashIsNotForwarded(t *testing.T) {
	mvc := newTestMvc(t)
	if res := mvc.Perform(web.Get("/spring-xml-swagger/swagger-ui/")); res.Status != http.StatusNotFound {
		t.Errorf("/swagger-ui/ status = %d, want %d", res.Status, http.StatusNotFound)
	}
	if res := mvc.Perform(web.Get("/spring-xml-swagger/swagger-ui/index.html")); res.Status != http.StatusOK {
		t.Errorf("/swagger-ui/index.html status = %d, want %d", res.Status, http.StatusOK)
	}
}

// TestApplicationInitializerDeclarations pins the bootstrap values the
// original's ApplicationInitializer declares.
func TestApplicationInitializerDeclarations(t *testing.T) {
	var init ApplicationInitializer
	if init.ServletName() != "dispatcher" {
		t.Errorf("servlet name = %q, want %q", init.ServletName(), "dispatcher")
	}
	if init.LoadOnStartup() != 1 {
		t.Errorf("load on startup = %d, want 1", init.LoadOnStartup())
	}
	if init.Mapping() != "/*" {
		t.Errorf("mapping = %q, want %q", init.Mapping(), "/*")
	}
}
