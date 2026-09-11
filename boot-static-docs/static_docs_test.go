package main

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/staticdocs"
	"github.com/springfox/springfox-demos/internal/web"
)

// TestGeneratesThePetstoreApiAsciidoc is
// StaticDocsTest."generates the petstore api asciidoc".
//
// The Groovy spec carries spock.lang.Ignore, so it never runs. It is ported as
// a skipped test rather than a deleted or an enabled one, so that the intent
// survives and re-enabling it is a one-line change.
func TestGeneratesThePetstoreApiAsciidoc(t *testing.T) {
	t.Skip("@Ignore in the original StaticDocsTest")

	// setup:
	outDir := os.Getenv("asciiDocOutputDir")
	if outDir == "" {
		// The default in the original spec, typo and all.
		outDir = filepath.Join("build", "aciidoc")
	}
	resultHandler := staticdocs.OutputDirectory(outDir)

	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	mvc := web.NewMockMvc(app)

	// when:
	res := mvc.Perform(web.Get("/v2/api-docs").Accept(web.MediaTypeJSON))
	if err := resultHandler.Handle([]byte(res.Body)); err != nil {
		t.Fatalf("generating the asciidoc: %v", err)
	}

	// andExpect(status().isOk())
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}

	// then:
	var list []string
	entries, err := os.ReadDir(resultHandler.OutputDir)
	if err != nil {
		t.Fatalf("reading the output directory: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			list = append(list, e.Name())
		}
	}
	sort.Strings(list)
	want := []string{"definitions.adoc", "overview.adoc", "paths.adoc"}
	if len(list) != len(want) {
		t.Fatalf("generated %v, want %v", list, want)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("generated %v, want %v", list, want)
		}
	}
}

// TestContextLoads has no counterpart in the original — boot-static-docs had no
// context test — but it gives the module the same construction guarantee the
// other modules' contextLoads tests give.
func TestContextLoads(t *testing.T) {
	if _, err := NewApplication(); err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
}

// TestPetstoreApiDocsAreServed covers the documentation endpoint the ignored
// spec would have consumed, so that the module's actual behaviour is tested
// even while that spec stays skipped.
func TestPetstoreApiDocsAreServed(t *testing.T) {
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	res := web.NewMockMvc(app).Perform(web.Get("/v2/api-docs"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.ContentType != web.MediaTypeJSON {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeJSON)
	}
	for _, want := range []string{`"swagger":"2.0"`, `"/api/pet/{petId}"`, `"pet-controller"`} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("the document does not contain %s", want)
		}
	}
}

// TestAsciidocGeneration exercises the Swagger-to-Asciidoc writer end to end:
// the original delegated this to swagger2markup, so the port owes it a test.
func TestAsciidocGeneration(t *testing.T) {
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("the application context failed to start: %v", err)
	}
	res := web.NewMockMvc(app).Perform(web.Get("/v2/api-docs"))

	outDir := t.TempDir()
	if err := staticdocs.OutputDirectory(outDir).Handle([]byte(res.Body)); err != nil {
		t.Fatalf("generating the asciidoc: %v", err)
	}
	for _, name := range []string{staticdocs.OverviewFile, staticdocs.PathsFile, staticdocs.DefinitionsFile} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("%s was not generated: %v", name, err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
