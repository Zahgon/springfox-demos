package staticdocs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const sampleDocument = `{
  "swagger": "2.0",
  "info": {
    "description": "Api Documentation",
    "version": "1.0",
    "title": "Api Documentation",
    "license": {"name": "Apache 2.0", "url": "http://www.apache.org/licenses/LICENSE-2.0"}
  },
  "host": "localhost:8080",
  "basePath": "/",
  "tags": [{"name": "pet-controller", "description": "Operations about pets"}],
  "paths": {
    "/api/pet/{petId}": {
      "get": {
        "tags": ["pet-controller"],
        "summary": "Find pet by ID",
        "description": "Returns a pet when ID < 10",
        "operationId": "getPetByIdUsingGET"
      }
    }
  },
  "definitions": {
    "Pet": {"title": "Pet", "properties": {"id": {"type": "integer"}, "name": {"type": "string"}}}
  }
}`

// TestGeneratesTheThreeAsciidocFiles covers the contract the ignored Groovy
// spec asserts: the writer produces exactly definitions.adoc, overview.adoc and
// paths.adoc.
func TestGeneratesTheThreeAsciidocFiles(t *testing.T) {
	dir := t.TempDir()
	if err := OutputDirectory(dir).Handle([]byte(sampleDocument)); err != nil {
		t.Fatalf("generating: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the output directory: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	want := []string{"definitions.adoc", "overview.adoc", "paths.adoc"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("generated %v, want %v", names, want)
	}
}

// TestOverviewContent covers the overview section.
func TestOverviewContent(t *testing.T) {
	dir := t.TempDir()
	if err := OutputDirectory(dir).Handle([]byte(sampleDocument)); err != nil {
		t.Fatalf("generating: %v", err)
	}
	content := read(t, dir, OverviewFile)
	for _, want := range []string{
		"= Api Documentation",
		"[[_overview]]",
		"__Version__ : 1.0",
		"__License__ : Apache 2.0",
		"__Host__ : localhost:8080",
		"* pet-controller : Operations about pets",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("the overview does not contain %q:\n%s", want, content)
		}
	}
}

// TestPathsContent covers the paths section.
func TestPathsContent(t *testing.T) {
	dir := t.TempDir()
	if err := OutputDirectory(dir).Handle([]byte(sampleDocument)); err != nil {
		t.Fatalf("generating: %v", err)
	}
	content := read(t, dir, PathsFile)
	for _, want := range []string{
		"[[_paths]]",
		"[[_getpetbyidusingget]]",
		"=== Find pet by ID",
		"GET /api/pet/{petId}",
		"Returns a pet when ID < 10",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("the paths document does not contain %q:\n%s", want, content)
		}
	}
}

// TestDefinitionsContent covers the definitions section.
func TestDefinitionsContent(t *testing.T) {
	dir := t.TempDir()
	if err := OutputDirectory(dir).Handle([]byte(sampleDocument)); err != nil {
		t.Fatalf("generating: %v", err)
	}
	content := read(t, dir, DefinitionsFile)
	for _, want := range []string{"[[_definitions]]", "[[_pet]]", "=== Pet", "* id", "* name"} {
		if !strings.Contains(content, want) {
			t.Errorf("the definitions document does not contain %q:\n%s", want, content)
		}
	}
}

// TestHandleRejectsAMalformedDocument covers the failure path.
func TestHandleRejectsAMalformedDocument(t *testing.T) {
	if err := OutputDirectory(t.TempDir()).Handle([]byte("{")); err == nil {
		t.Fatal("a malformed document was accepted")
	}
}

// TestHandleCreatesTheOutputDirectory covers the directory creation the ignored
// spec relies on when it points at a build directory that does not exist yet.
func TestHandleCreatesTheOutputDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "asciidoc", "generated", "swagger_adoc")
	if err := OutputDirectory(dir).Handle([]byte(sampleDocument)); err != nil {
		t.Fatalf("generating: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, OverviewFile)); err != nil {
		t.Fatalf("the nested output directory was not created: %v", err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(data)
}
