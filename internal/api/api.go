// Package api holds the specification-neutral description of an application's
// HTTP surface.
//
// In the Java original this description did not exist as data: Springfox
// recovered it at runtime by reflecting over Spring's handler mappings and over
// the `@Api…` annotations attached to controllers, parameters and models. Go
// has neither annotations nor the reflective type metadata Springfox relies on,
// so the same information is declared explicitly next to each route. The
// documentation generators in internal/springfox consume nothing else.
package api

// Type is a JSON-schema primitive type name.
type Type string

// The JSON-schema primitives Springfox emits.
const (
	TypeString  Type = "string"
	TypeInteger Type = "integer"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
	TypeArray   Type = "array"
	TypeObject  Type = "object"
	TypeFile    Type = "file"
)

// Schema describes a value. Exactly one of Ref and Type is meaningful: a Ref
// names an entry in the document's model registry, a Type describes an inline
// primitive, array or map.
type Schema struct {
	Ref                  string
	Type                 Type
	Format               string
	Items                *Schema
	AdditionalProperties *Schema
	Enum                 []string
	Default              any
}

// Parameter locations, matching the Swagger "in" vocabulary.
const (
	InPath     = "path"
	InQuery    = "query"
	InBody     = "body"
	InFormData = "formData"
	InHeader   = "header"
)

// Parameter describes one operation parameter.
type Parameter struct {
	Name        string
	In          string
	Description string
	Required    bool
	Schema      Schema
	// CollectionFormat is Swagger 2.0's multi/csv marker for repeated query
	// parameters; it becomes style+explode in OpenAPI 3.
	CollectionFormat string
	// Range carries an @ApiParam(allowableValues = "range[lo,hi]") constraint.
	// Swagger 2.0 renders it as maxLength/minLength on the parameter, OpenAPI 3
	// as maximum/minimum on its schema.
	Range *Range
	// Style overrides the OpenAPI 3 serialisation style; empty means the
	// default for the parameter's location.
	Style string
	// Explode sets OpenAPI 3's explode flag, used for repeatable parameters.
	Explode bool
	// FromCondition marks a parameter Springfox synthesised from a
	// `params = "name=value"` mapping condition. OpenAPI 3 renders it with
	// allowReserved set; Swagger 2.0 renders its matched value as a default.
	FromCondition bool
	// AllowReserved, when set, writes OpenAPI 3's allowReserved flag
	// explicitly. Springfox emits it only for the parameters its expanded
	// parameter reader produces.
	AllowReserved *bool
	// Ignored marks a parameter annotated @ApiIgnore in the original. It is
	// dropped from a document whose Docket ignores that annotation type.
	Ignored bool
}

// Response describes one documented response code.
type Response struct {
	Code        int
	Description string
	Schema      *Schema
	// Examples maps a media type to a literal example body.
	Examples map[string]string
	// Headers are the response headers declared by @ResponseHeader.
	Headers []Header
}

// Range is an inclusive numeric range constraint. A negative Max means the
// original wrote "range[n,infinity]", which Springfox renders without an upper
// bound.
type Range struct {
	Min int
	Max int
}

// Bounded reports whether the range declares an upper bound.
func (r Range) Bounded() bool { return r.Max >= 0 }

// Header is a documented response header.
type Header struct {
	Name string
	Type Type
}

// SecurityRequirement names a security scheme and the scopes an operation needs.
type SecurityRequirement struct {
	Name   string
	Scopes []string
}

// Operation describes a single method+path pair.
type Operation struct {
	// Name is the Java method name the operation was derived from. The
	// documentation generators turn it into "<Name>Using<METHOD>" and
	// de-duplicate across groups exactly as Springfox does.
	Name string
	// Tag is the tag written on the operation itself. @ApiOperation(tags = …)
	// overrides the controller's own tag here.
	Tag string
	// ListedTag is the controller tag the document's "tags" array advertises,
	// with ListedTagDescription as its @Api description. Set it only when the
	// operation's tag was overridden; otherwise Tag is listed.
	ListedTag            string
	ListedTagDescription string
	Summary              string
	Notes                string
	Consumes             []string
	Produces             []string
	Params               []Parameter
	Responses            []Response
	Security             []SecurityRequirement
	Deprecated           bool
	// Hidden reproduces @ApiOperation(hidden = true): the route is served but
	// never documented.
	Hidden bool
}

// Property is one property of a model.
type Property struct {
	Name        string
	Description string
	Schema      Schema
	// Required marks a property Jackson reports as mandatory, which Springfox
	// surfaces in the model's "required" array.
	Required bool
}

// Model is a documented type. Springfox emits separate request and response
// variants of a model whenever the two differ; ReqName and ResName carry those
// alternative names when the original did so.
type Model struct {
	Name       string
	Title      string
	Properties []Property
	Enum       []string
}

// Registry is the set of models an application can reference from its routes.
type Registry struct {
	models map[string]Model
}

// NewRegistry returns an empty model registry.
func NewRegistry() *Registry { return &Registry{models: map[string]Model{}} }

// Add registers a model, replacing any earlier one with the same name.
func (r *Registry) Add(m Model) { r.models[m.Name] = m }

// Get returns the named model.
func (r *Registry) Get(name string) (Model, bool) {
	m, ok := r.models[name]
	return m, ok
}
