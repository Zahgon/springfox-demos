package restdocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestDocumentation(t *testing.T) *Documentation {
	t.Helper()
	d := NewDocumentation()
	d.OutputDirectory = filepath.Join(t.TempDir(), "generated-snippets")
	return d.WithTemplateFormat(SpringfoxTemplateFormat{})
}

// TestSnippetIsWrittenWithTheSpringfoxExtension covers SpringfoxTemplateFormat,
// whose file extension the module's build depends on when it copies the
// snippets onto the resource path.
func TestSnippetIsWrittenWithTheSpringfoxExtension(t *testing.T) {
	d := newTestDocumentation(t)
	fields := NewResponseFields(
		FieldWithPath("bar").Describe("Name of the bar"),
		FieldWithPath("foo").Describe("Specifies if this is a foo"),
		FieldWithPath("count").Describe("Specifies how many foos there are"),
	)
	body := `{"bar":"aragorn","foo":false,"count":3}`
	if err := d.Document("toLowerGatewayAragorn", body, fields); err != nil {
		t.Fatalf("documenting: %v", err)
	}

	path := d.SnippetPath("toLowerGatewayAragorn", "response-fields")
	if !strings.HasSuffix(path, ".springfox") {
		t.Errorf("the snippet path %q does not use the springfox extension", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the snippet was not written: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"|bar\n|String\n|Name of the bar",
		"|foo\n|Boolean\n|Specifies if this is a foo",
		"|count\n|Number\n|Specifies how many foos there are",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("the snippet does not contain:\n%s\ngot:\n%s", want, content)
		}
	}
}

// TestSpringfoxTemplateFormat covers the format's identity: SpringfoxTemplateFormat
// reports the same string as its id and as its file extension, which is what
// makes the snippet land at response-fields.springfox.
func TestSpringfoxTemplateFormat(t *testing.T) {
	var format SpringfoxTemplateFormat
	if format.ID() != "springfox" {
		t.Errorf("ID() = %q, want %q", format.ID(), "springfox")
	}
	if format.FileExtension() != "springfox" {
		t.Errorf("FileExtension() = %q, want %q", format.FileExtension(), "springfox")
	}
}

// TestResponseFieldsIsStrictAboutUndocumentedFields covers the half of REST
// Docs' strictness that makes the two ported tests an assertion about the
// response's exact shape.
func TestResponseFieldsIsStrictAboutUndocumentedFields(t *testing.T) {
	d := newTestDocumentation(t)
	fields := NewResponseFields(FieldWithPath("bar").Describe("Name of the bar"))
	err := d.Document("x", `{"bar":"a","foo":false}`, fields)
	if err == nil {
		t.Fatal("an undocumented field was accepted")
	}
	if !strings.Contains(err.Error(), `"foo"`) {
		t.Errorf("the error does not name the undocumented field: %v", err)
	}
}

// TestResponseFieldsIsStrictAboutMissingFields covers the other half.
func TestResponseFieldsIsStrictAboutMissingFields(t *testing.T) {
	d := newTestDocumentation(t)
	fields := NewResponseFields(
		FieldWithPath("bar").Describe("Name of the bar"),
		FieldWithPath("missing").Describe("Not in the response"),
	)
	err := d.Document("x", `{"bar":"a"}`, fields)
	if err == nil {
		t.Fatal("a missing documented field was accepted")
	}
	if !strings.Contains(err.Error(), `"missing"`) {
		t.Errorf("the error does not name the missing field: %v", err)
	}
}

// TestDocumentRejectsANonObjectBody covers the failure REST Docs reports when
// responseFields is applied to something that is not a JSON object.
func TestDocumentRejectsANonObjectBody(t *testing.T) {
	d := newTestDocumentation(t)
	if err := d.Document("x", `["a"]`, NewResponseFields()); err == nil {
		t.Fatal("a JSON array was accepted as a response-fields payload")
	}
}

// TestJSONTypeNames covers the type column of the snippet table.
func TestJSONTypeNames(t *testing.T) {
	cases := []struct{ raw, want string }{
		{`"a"`, "String"},
		{`1`, "Number"},
		{`1.5`, "Number"},
		{`true`, "Boolean"},
		{`false`, "Boolean"},
		{`null`, "Null"},
		{`[1]`, "Array"},
		{`{"a":1}`, "Object"},
	}
	for _, c := range cases {
		if got := jsonType([]byte(c.raw)); got != c.want {
			t.Errorf("jsonType(%s) = %q, want %q", c.raw, got, c.want)
		}
	}
}
