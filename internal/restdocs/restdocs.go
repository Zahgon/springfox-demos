// Package restdocs reproduces the part of Spring REST Docs that
// springfox-integration-webmvc's tests use: the response-fields snippet, its
// strictness, and the springfox template format the module's build then copies
// onto the resource path.
//
// The original delegates this to spring-restdocs-mockmvc. The Go port
// implements it, so it needs tests of its own — see restdocs_test.go.
package restdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SpringfoxTemplateFormat is springfox.documentation.spring.web.SpringfoxTemplateFormat:
// snippets are written with the "springfox" file extension rather than
// Asciidoctor's ".adoc".
type SpringfoxTemplateFormat struct{}

// springfoxFormatName is both the format's id and its file extension.
const springfoxFormatName = "springfox"

// ID is TemplateFormat.getId().
func (SpringfoxTemplateFormat) ID() string { return springfoxFormatName }

// FileExtension is TemplateFormat.getFileExtension().
func (SpringfoxTemplateFormat) FileExtension() string { return springfoxFormatName }

// Documentation is org.springframework.restdocs.JUnitRestDocumentation: the
// output root the snippets are written under.
type Documentation struct {
	// OutputDirectory defaults to "target/generated-snippets", which is where
	// the module's build looks for them.
	OutputDirectory string
	format          SpringfoxTemplateFormat
}

// NewDocumentation creates a documentation context rooted at the default
// output directory.
func NewDocumentation() *Documentation {
	return &Documentation{OutputDirectory: filepath.Join("target", "generated-snippets")}
}

// WithTemplateFormat is
// documentationConfiguration(…).snippets().withTemplateFormat(…). Only the
// springfox format is used by these tests.
func (d *Documentation) WithTemplateFormat(f SpringfoxTemplateFormat) *Documentation {
	d.format = f
	return d
}

// FieldDescriptor is org.springframework.restdocs.payload.FieldDescriptor.
type FieldDescriptor struct {
	Path        string
	Description string
}

// FieldWithPath is PayloadDocumentation.fieldWithPath(path).
func FieldWithPath(path string) FieldDescriptor { return FieldDescriptor{Path: path} }

// Describe is FieldDescriptor.description(text).
func (f FieldDescriptor) Describe(text string) FieldDescriptor {
	f.Description = text
	return f
}

// ResponseFields is PayloadDocumentation.responseFields(descriptors...).
//
// It is strict in both directions, exactly as Spring REST Docs is: the snippet
// fails if the response carries a field that was not documented, and if a
// documented field is missing from the response.
type ResponseFields struct {
	Fields []FieldDescriptor
}

// NewResponseFields builds a response-fields snippet specification.
func NewResponseFields(fields ...FieldDescriptor) ResponseFields {
	return ResponseFields{Fields: fields}
}

// Document is MockMvcRestDocumentation.document(identifier, snippets...): it
// validates the response body against the documented fields and writes the
// snippet files.
func (d *Documentation) Document(identifier string, body string, fields ResponseFields) error {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return fmt.Errorf("restdocs: the response body is not a JSON object: %w", err)
	}

	documented := map[string]string{}
	for _, f := range fields.Fields {
		if _, ok := payload[f.Path]; !ok {
			return fmt.Errorf("restdocs: the documented field %q is missing from the response", f.Path)
		}
		documented[f.Path] = f.Description
	}
	for name := range payload {
		if _, ok := documented[name]; !ok {
			return fmt.Errorf("restdocs: the response field %q was not documented", name)
		}
	}

	dir := filepath.Join(d.OutputDirectory, identifier)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := "response-fields." + d.format.FileExtension()
	return os.WriteFile(filepath.Join(dir, name), []byte(renderSnippet(fields, payload)), 0o644)
}

// SnippetPath is where Document writes a snippet, which tests assert on.
func (d *Documentation) SnippetPath(identifier, snippet string) string {
	return filepath.Join(d.OutputDirectory, identifier, snippet+"."+d.format.FileExtension())
}

// renderSnippet writes the response-fields table, in the documented order.
func renderSnippet(fields ResponseFields, payload map[string]json.RawMessage) string {
	var b strings.Builder
	b.WriteString("|===\n|Path|Type|Description\n\n")
	for _, f := range fields.Fields {
		b.WriteString("|" + f.Path + "\n|" + jsonType(payload[f.Path]) + "\n|" + f.Description + "\n\n")
	}
	b.WriteString("|===\n")
	return b.String()
}

// The type names REST Docs writes in a snippet's Type column, taken from
// org.springframework.restdocs.payload.JsonFieldType.
const (
	TypeNull    = "Null"
	TypeBoolean = "Boolean"
	TypeString  = "String"
	TypeArray   = "Array"
	TypeObject  = "Object"
	TypeNumber  = "Number"
)

// jsonType names the REST Docs type of a raw JSON value.
func jsonType(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	switch {
	case s == "", s == "null":
		return TypeNull
	case s == "true", s == "false":
		return TypeBoolean
	case strings.HasPrefix(s, `"`):
		return TypeString
	case strings.HasPrefix(s, "["):
		return TypeArray
	case strings.HasPrefix(s, "{"):
		return TypeObject
	default:
		return TypeNumber
	}
}
