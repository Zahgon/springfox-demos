package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/springfox/springfox-demos/internal/restdocs"
	"github.com/springfox/springfox-demos/internal/web"
)

// restDocumentation is the @Rule JUnitRestDocumentation of
// SpringIntegrationWebMvcApplicationTest, and setUpRestDocs is its @Before
// setUp(): MockMvc built over the application context with the springfox
// template format configured for its snippets.
func setUpRestDocs(t *testing.T) (*web.MockMvc, *restdocs.Documentation) {
	t.Helper()
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	documentation := restdocs.NewDocumentation()
	documentation.OutputDirectory = filepath.Join(t.TempDir(), "generated-snippets")
	documentation.WithTemplateFormat(restdocs.SpringfoxTemplateFormat{})
	return web.NewMockMvc(app), documentation
}

// toLowerFlowFields are the responseFields descriptors both tests document.
func toLowerFlowFields() restdocs.ResponseFields {
	return restdocs.NewResponseFields(
		restdocs.FieldWithPath("bar").Describe("Name of the bar"),
		restdocs.FieldWithPath("foo").Describe("Specifies if this is a foo"),
		restdocs.FieldWithPath("count").Describe("Specifies how many foos there are"),
	)
}

// TestToLowerFlowAragorn is SpringIntegrationWebMvcApplicationTest.toLowerFlowAragorn.
func TestToLowerFlowAragorn(t *testing.T) {
	assertToLowerFlow(t, "Aragorn", "toLowerGatewayAragorn")
}

// TestToLowerFlowGimli is SpringIntegrationWebMvcApplicationTest.toLowerFlowGimli.
func TestToLowerFlowGimli(t *testing.T) {
	assertToLowerFlow(t, "Gimli", "toLowerGatewayGimli")
}

// assertToLowerFlow performs the request both tests share and documents it. The
// responseFields snippet is strict in both directions, so documenting exactly
// bar, foo and count asserts that the response has exactly those fields.
func assertToLowerFlow(t *testing.T, bar, identifier string) {
	t.Helper()
	mvc, documentation := setUpRestDocs(t)

	res := mvc.Perform(web.Post("/conversions/lower").
		ContentType(web.MediaTypeJSON).
		Content("{\n" +
			"  \"bar\": \"" + bar + "\",\n" +
			"  \"foo\": true,\n" +
			"  \"count\": 3\n" +
			"}"))

	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if err := documentation.Document(identifier, res.Body, toLowerFlowFields()); err != nil {
		t.Fatalf("documenting the response: %v", err)
	}

	snippet := documentation.SnippetPath(identifier, "response-fields")
	if _, err := os.Stat(snippet); err != nil {
		t.Fatalf("the snippet %s was not written: %v", snippet, err)
	}
}
