package springfox

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
)

// renderSwagger2 builds the document served at /v2/api-docs.
func (p *Provider) renderSwagger2(g *group) any {
	d := g.docket
	doc := obj().
		set("swagger", "2.0").
		set("info", swagger2Info(d.Info())).
		set("host", p.Host)
	// Springfox reports basePath only when the application has no servlet
	// context path; with a context path the paths themselves carry the prefix.
	if p.app.ContextPath == "" {
		doc.set("basePath", "/")
	}
	doc.set("tags", tagsOf(g))
	doc.set("paths", p.swagger2Paths(g))
	if defs := securityDefinitions(d.securitySchemes); !defs.empty() {
		doc.set("securityDefinitions", defs)
	}
	if defs := p.definitions(g, "#/definitions/", false); defs != nil {
		doc.set("definitions", defs)
	}
	return doc
}

func swagger2Info(info APIInfo) *object {
	return obj().
		set("description", info.Description).
		set("version", info.Version).
		set("title", info.Title).
		set("termsOfService", info.TermsOfServiceURL).
		set("contact", contactObject(info.Contact)).
		set("license", licenseObject(info))
}

func contactObject(c Contact) *object {
	return obj().
		setNonEmpty("name", c.Name).
		setNonEmpty("url", c.URL).
		setNonEmpty("email", c.Email)
}

func licenseObject(info APIInfo) *object {
	return obj().
		setNonEmpty("name", info.License).
		setNonEmpty("url", info.LicenseURL)
}

func (p *Provider) swagger2Paths(g *group) *object {
	paths := obj()
	for _, item := range orderedByPath(g.ops, methodRank) {
		entry := obj()
		path := item[0].path
		if g.docket.urlTemplating {
			path = uriTemplate(path, item[0].op, g.docket.ignoreAnnotatedParams)
		}
		for _, dop := range item {
			// The Swagger 2.0 specification has no `trace` operation, so
			// Springfox drops a TRACE mapping from a v2 document.
			if dop.method == http.MethodTrace {
				continue
			}
			entry.set(strings.ToLower(dop.method), p.swagger2Operation(g, dop))
		}
		if entry.empty() {
			continue
		}
		paths.set(path, entry)
	}
	return paths
}

func (p *Provider) swagger2Operation(g *group, dop documentedOperation) *object {
	op := dop.op
	o := obj().
		set("tags", []string{op.Tag}).
		set("summary", op.Summary).
		setNonEmpty("description", op.Notes).
		set("operationId", dop.id).
		setIf(len(op.Consumes) > 0, "consumes", op.Consumes).
		setIf(len(op.Produces) > 0, "produces", op.Produces)

	// The Swagger 2.0 mapper sorts an operation's parameters by name, ignoring
	// case, and appends the form-data parameters afterwards in the order the
	// multipart reader contributed them. The OpenAPI mapper keeps every
	// parameter in declaration order.
	params := swagger2ParameterOrder(visibleParams(op, g.docket.ignoreAnnotatedParams))
	if len(params) > 0 {
		rendered := make([]*object, 0, len(params))
		for _, prm := range params {
			rendered = append(rendered, swagger2Parameter(prm))
		}
		o.set("parameters", rendered)
	}

	responses := obj()
	for _, r := range mergeResponses(dop.method, op.Responses) {
		responses.set(strconv.Itoa(r.Code), swagger2Response(r))
	}
	o.set("responses", responses)

	if sec := securityRequirements(op, g.docket, dop.path); sec != nil {
		o.set("security", sec)
	}
	o.setIf(op.Deprecated, "deprecated", true)
	return o
}

// swagger2ParameterOrder is the parameter order of a Swagger 2.0 operation.
func swagger2ParameterOrder(params []api.Parameter) []api.Parameter {
	var named, form []api.Parameter
	for _, prm := range params {
		if prm.In == api.InFormData {
			form = append(form, prm)
			continue
		}
		named = append(named, prm)
	}
	sort.SliceStable(named, func(i, j int) bool {
		return strings.ToLower(named[i].Name) < strings.ToLower(named[j].Name)
	})
	return append(named, form...)
}

func swagger2Parameter(prm api.Parameter) *object {
	if prm.In == api.InBody {
		return obj().
			set("in", "body").
			set("name", prm.Name).
			setNonEmpty("description", prm.Description).
			set("required", prm.Required).
			set("schema", swagger2Schema(prm.Schema))
	}
	o := obj().
		set("name", prm.Name).
		set("in", prm.In).
		setNonEmpty("description", prm.Description).
		set("required", prm.Required).
		setIf(prm.Schema.Type != "", "type", string(prm.Schema.Type)).
		setIf(prm.Schema.Default != nil, "default", prm.Schema.Default).
		setNonEmpty("format", prm.Schema.Format)
	schema := withSortedEnums(prm.Schema)
	if schema.Items != nil {
		o.set("items", renderSchema(*schema.Items, "#/definitions/"))
	}
	o.setNonEmpty("collectionFormat", prm.CollectionFormat)
	o.setIf(len(schema.Enum) > 0, "enum", schema.Enum)
	if prm.Range != nil {
		o.setIf(prm.Range.Bounded(), "maxLength", prm.Range.Max)
		o.set("minLength", prm.Range.Min)
	}
	return o
}

func swagger2Response(r api.Response) *object {
	o := obj().set("description", r.Description)
	if len(r.Examples) > 0 {
		examples := obj()
		for _, mt := range sortedKeys(r.Examples) {
			examples.set(mt, r.Examples[mt])
		}
		o.set("examples", examples)
	}
	if r.Schema != nil {
		o.set("schema", swagger2Schema(*r.Schema))
	}
	if len(r.Headers) > 0 {
		headers := obj()
		for _, h := range r.Headers {
			headers.set(h.Name, obj().set("type", string(h.Type)))
		}
		o.set("headers", headers)
	}
	return o
}

func swagger2Schema(s api.Schema) *object {
	return renderSchema(withSortedEnums(s), "#/definitions/")
}

// withSortedEnums sorts the enum values of a schema and of everything nested in
// it. Springfox sorts the allowable values it attaches to a parameter or to a
// response schema, whatever order the Java enum or the @ApiParam declared them
// in.
func withSortedEnums(s api.Schema) api.Schema {
	if len(s.Enum) > 0 {
		enum := make([]string, len(s.Enum))
		copy(enum, s.Enum)
		sort.Strings(enum)
		s.Enum = enum
	}
	if s.Items != nil {
		items := withSortedEnums(*s.Items)
		s.Items = &items
	}
	if s.AdditionalProperties != nil {
		additional := withSortedEnums(*s.AdditionalProperties)
		s.AdditionalProperties = &additional
	}
	return s
}

// renderSchema is shared by both specification versions; only the $ref prefix
// differs between them.
func renderSchema(s api.Schema, refPrefix string) *object {
	if s.Ref != "" {
		return obj().set("$ref", refPrefix+s.Ref)
	}
	o := obj().setIf(s.Type != "", "type", string(s.Type)).
		setNonEmpty("format", s.Format)
	if s.Items != nil {
		o.set("items", renderSchema(*s.Items, refPrefix))
	}
	if s.AdditionalProperties != nil {
		o.set("additionalProperties", renderSchema(*s.AdditionalProperties, refPrefix))
	}
	o.setIf(len(s.Enum) > 0, "enum", s.Enum)
	return o
}

// securityDefinitions renders the Docket's schemes into a Swagger 2.0
// securityDefinitions block, keyed by scheme name.
func securityDefinitions(schemes []SecurityScheme) *object {
	out := obj()
	for _, s := range schemes {
		switch s.Type {
		case SchemeOAuth2:
			scopes := obj()
			for _, sc := range s.Scopes {
				scopes.set(sc.Scope, sc.Description)
			}
			entry := obj().set("type", "oauth2")
			if len(s.GrantTypes) > 0 {
				entry.set("authorizationUrl", s.GrantTypes[0].LoginEndpoint)
				entry.set("flow", s.GrantTypes[0].Type)
			}
			entry.set("scopes", scopes)
			out.set(s.Name, entry)
		case SchemeBasic:
			out.set(s.Name, obj().set("type", "basic"))
		case SchemeAPIKey:
			out.set(s.Name, obj().
				set("type", "apiKey").
				set("name", s.KeyName).
				set("in", s.PassAs))
		}
	}
	return out
}

// securityRequirements renders an operation's `security` block: the
// @Authorization requirements declared on the operation itself if it has any,
// otherwise the Docket's security contexts that apply to the operation's path.
// The requirements are not filtered against the schemes the Docket publishes —
// the original emits a requirement whose scheme has no definition.
func securityRequirements(op api.Operation, d *Docket, path string) []*object {
	var out []*object
	for _, req := range op.Security {
		scopes := req.Scopes
		if scopes == nil {
			scopes = []string{}
		}
		out = append(out, obj().set(req.Name, scopes))
	}
	if len(out) > 0 {
		return out
	}
	for _, ctx := range d.securityContexts {
		if ctx.ForPaths != nil && !ctx.ForPaths(path) {
			continue
		}
		for _, ref := range ctx.References {
			scopes := make([]string, 0, len(ref.Scopes))
			for _, s := range ref.Scopes {
				scopes = append(scopes, s.Scope)
			}
			out = append(out, obj().set(ref.Reference, scopes))
		}
	}
	return out
}

// definitions collects the models a group's operations reference, transitively,
// and renders them sorted by name.
func (p *Provider) definitions(g *group, refPrefix string, titleFirst bool) *object {
	sortEnums := titleFirst || p.UseModelV3
	referenced := map[string]bool{}
	var visit func(s api.Schema)
	visit = func(s api.Schema) {
		if s.Ref != "" {
			if referenced[s.Ref] {
				return
			}
			referenced[s.Ref] = true
			if m, ok := p.app.Models.Get(s.Ref); ok {
				for _, prop := range m.Properties {
					visit(prop.Schema)
				}
			}
			return
		}
		if s.Items != nil {
			visit(*s.Items)
		}
		if s.AdditionalProperties != nil {
			visit(*s.AdditionalProperties)
		}
	}
	for _, dop := range g.candidates {
		for _, prm := range visibleParams(dop.op, g.docket.ignoreAnnotatedParams) {
			visit(prm.Schema)
		}
		for _, r := range dop.op.Responses {
			if r.Schema != nil {
				visit(*r.Schema)
			}
		}
	}
	if len(referenced) == 0 {
		return nil
	}
	names := make([]string, 0, len(referenced))
	for n := range referenced {
		names = append(names, n)
	}
	sort.Strings(names)

	out := obj()
	for _, n := range names {
		m, ok := p.app.Models.Get(n)
		if !ok {
			continue
		}
		out.set(n, renderModel(m, refPrefix, titleFirst, sortEnums))
	}
	return out
}

// renderModel writes one model definition. titleFirst follows the OpenAPI
// mapper, which puts "title" first where the Swagger 2.0 mapper puts it last.
// sortEnums follows springfox.documentation.swagger.v2.use-model-v3: the v3
// model mapper sorts a property's enum values, the legacy v2 mapper keeps them
// in declaration order.
func renderModel(m api.Model, refPrefix string, titleFirst, sortEnums bool) *object {
	title := m.Title
	if title == "" {
		title = m.Name
	}
	var required []string
	for _, prop := range m.Properties {
		if prop.Required {
			required = append(required, prop.Name)
		}
	}

	// The OpenAPI mapper writes title, required, type; the Swagger 2.0 mapper
	// writes type, required, …, title.
	o := obj()
	if titleFirst {
		o.set("title", title)
		o.setIf(len(required) > 0, "required", required)
		o.set("type", "object")
	} else {
		o.set("type", "object")
		o.setIf(len(required) > 0, "required", required)
	}

	props := make([]api.Property, len(m.Properties))
	copy(props, m.Properties)
	sort.SliceStable(props, func(i, j int) bool { return props[i].Name < props[j].Name })

	rendered := obj()
	// A model with no properties omits the "properties" key entirely.
	skipProperties := len(props) == 0
	for _, prop := range props {
		if sortEnums && len(prop.Schema.Enum) > 0 {
			enum := make([]string, len(prop.Schema.Enum))
			copy(enum, prop.Schema.Enum)
			sort.Strings(enum)
			prop.Schema.Enum = enum
		}
		s := renderSchema(prop.Schema, refPrefix)
		if prop.Description != "" && prop.Schema.Ref == "" {
			// The Swagger 2.0 mapper writes type, format, description; the
			// OpenAPI mapper writes type, description, format.
			s = obj().setIf(prop.Schema.Type != "", "type", string(prop.Schema.Type))
			if titleFirst {
				s.set("description", prop.Description).
					setNonEmpty("format", prop.Schema.Format)
			} else {
				s.setNonEmpty("format", prop.Schema.Format).
					set("description", prop.Description)
			}
			if prop.Schema.Items != nil {
				s.set("items", renderSchema(*prop.Schema.Items, refPrefix))
			}
			if prop.Schema.AdditionalProperties != nil {
				s.set("additionalProperties", renderSchema(*prop.Schema.AdditionalProperties, refPrefix))
			}
			s.setIf(len(prop.Schema.Enum) > 0, "enum", prop.Schema.Enum)
		}
		rendered.set(prop.Name, s)
	}
	if !skipProperties {
		o.set("properties", rendered)
	}
	if !titleFirst {
		o.set("title", title)
	}
	return o
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
