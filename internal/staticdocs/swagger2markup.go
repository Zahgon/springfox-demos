// Package staticdocs reproduces the part of springfox-staticdocs and
// swagger2markup that boot-static-docs depends on: turning a Swagger 2.0
// document into Asciidoc source at build time.
//
// The original delegates this to a third-party pair of libraries. The Go port
// implements it, so it needs tests of its own — see swagger2markup_test.go.
package staticdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The three files swagger2markup writes.
const (
	OverviewFile    = "overview.adoc"
	PathsFile       = "paths.adoc"
	DefinitionsFile = "definitions.adoc"
)

// ResultHandler is springfox.documentation.staticdocs.Swagger2MarkupResultHandler:
// it takes the body of a /v2/api-docs response and writes Asciidoc into an
// output directory.
type ResultHandler struct {
	// OutputDir is the directory the .adoc files are written to.
	OutputDir string
}

// OutputDirectory starts a handler for an output directory, mirroring
// Swagger2MarkupResultHandler.outputDirectory(dir).build().
func OutputDirectory(dir string) *ResultHandler { return &ResultHandler{OutputDir: dir} }

// Handle converts a Swagger 2.0 document and writes the three Asciidoc files.
func (h *ResultHandler) Handle(document []byte) error {
	var doc swaggerDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		return fmt.Errorf("staticdocs: parsing the swagger document: %w", err)
	}
	if err := os.MkdirAll(h.OutputDir, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		OverviewFile:    doc.overview(),
		PathsFile:       doc.paths(),
		DefinitionsFile: doc.definitions(),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(h.OutputDir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// swaggerDocument is the subset of a Swagger 2.0 document the markup uses.
type swaggerDocument struct {
	Info struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Version     string `json:"version"`
		License     struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"license"`
	} `json:"info"`
	Host        string                                 `json:"host"`
	BasePath    string                                 `json:"basePath"`
	Tags        []struct{ Name, Description string }   `json:"tags"`
	Paths       map[string]map[string]swaggerOperation `json:"paths"`
	Definitions map[string]swaggerDefinition           `json:"definitions"`
}

type swaggerOperation struct {
	Tags        []string `json:"tags"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	OperationID string   `json:"operationId"`
}

type swaggerDefinition struct {
	Title      string                     `json:"title"`
	Properties map[string]json.RawMessage `json:"properties"`
}

func (d swaggerDocument) overview() string {
	var b strings.Builder
	b.WriteString("= " + d.Info.Title + "\n\n\n")
	b.WriteString("[[_overview]]\n== Overview\n")
	if d.Info.Description != "" {
		b.WriteString(d.Info.Description + "\n")
	}
	b.WriteString("\n\n=== Version information\n[%hardbreaks]\n__Version__ : " + d.Info.Version + "\n")
	if d.Info.License.Name != "" {
		b.WriteString("\n\n=== License information\n[%hardbreaks]\n__License__ : " +
			d.Info.License.Name + "\n__License URL__ : " + d.Info.License.URL + "\n")
	}
	b.WriteString("\n\n=== URI scheme\n[%hardbreaks]\n__Host__ : " + d.Host + "\n")
	if d.BasePath != "" {
		b.WriteString("__BasePath__ : " + d.BasePath + "\n")
	}
	if len(d.Tags) > 0 {
		b.WriteString("\n\n=== Tags\n")
		for _, t := range d.Tags {
			b.WriteString("* " + t.Name + " : " + t.Description + "\n")
		}
	}
	return b.String()
}

func (d swaggerDocument) paths() string {
	var b strings.Builder
	b.WriteString("\n[[_paths]]\n== Paths\n")
	for _, path := range sortedKeys(d.Paths) {
		for _, method := range sortedKeys(d.Paths[path]) {
			op := d.Paths[path][method]
			b.WriteString("\n[[_" + strings.ToLower(op.OperationID) + "]]\n=== " + op.Summary + "\n")
			b.WriteString("....\n" + strings.ToUpper(method) + " " + path + "\n....\n")
			if op.Description != "" {
				b.WriteString("\n==== Description\n" + op.Description + "\n")
			}
			if len(op.Tags) > 0 {
				b.WriteString("\n==== Tags\n")
				for _, t := range op.Tags {
					b.WriteString("* " + t + "\n")
				}
			}
		}
	}
	return b.String()
}

func (d swaggerDocument) definitions() string {
	var b strings.Builder
	b.WriteString("\n[[_definitions]]\n== Definitions\n")
	for _, name := range sortedKeys(d.Definitions) {
		def := d.Definitions[name]
		title := def.Title
		if title == "" {
			title = name
		}
		b.WriteString("\n[[_" + strings.ToLower(name) + "]]\n=== " + title + "\n")
		for _, prop := range sortedKeys(def.Properties) {
			b.WriteString("* " + prop + "\n")
		}
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
