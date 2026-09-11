package springfox

import (
	"strconv"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

const oasRefPrefix = "#/components/schemas/"

// renderOpenAPI builds the document served at /v3/api-docs.
func (p *Provider) renderOpenAPI(g *group) any {
	d := g.docket
	doc := obj().
		set("openapi", "3.0.3").
		set("info", openAPIInfo(d.Info()))
	// The servlet stacks report an inferred server URL; the reactive stack does
	// not, because it cannot infer one at documentation time.
	if p.app.Stack != web.StackReactive {
		doc.set("servers", []*object{
			obj().set("url", "http://"+p.Host).set("description", "Inferred Url"),
		})
	}
	doc.set("tags", tagsOf(g))
	doc.set("paths", p.openAPIPaths(g))

	components := obj()
	if defs := p.definitions(g, oasRefPrefix, true); defs != nil {
		components.set("schemas", defs)
	}
	if secSchemes := openAPISecuritySchemes(d.securitySchemes); !secSchemes.empty() {
		components.set("securitySchemes", secSchemes)
	}
	doc.set("components", components)
	return doc
}

func openAPIInfo(info APIInfo) *object {
	return obj().
		set("title", info.Title).
		set("description", info.Description).
		set("termsOfService", info.TermsOfServiceURL).
		set("contact", contactObject(info.Contact)).
		set("license", licenseObject(info)).
		set("version", info.Version)
}

func (p *Provider) openAPIPaths(g *group) *object {
	paths := obj()
	for _, item := range orderedByPath(g.ops, oasMethodRank) {
		entry := obj()
		path := item[0].path
		if g.docket.urlTemplating {
			path = uriTemplate(path, item[0].op, g.docket.ignoreAnnotatedParams)
		}
		for _, dop := range item {
			entry.set(strings.ToLower(dop.method), p.openAPIOperation(g, dop))
		}
		paths.set(path, entry)
	}
	return paths
}

func (p *Provider) openAPIOperation(g *group, dop documentedOperation) *object {
	op := dop.op
	o := obj().
		set("tags", []string{op.Tag}).
		set("summary", op.Summary).
		setNonEmpty("description", op.Notes).
		set("operationId", dop.id)

	params := visibleParams(op, g.docket.ignoreAnnotatedParams)
	var nonBody []*object
	var body *api.Parameter
	for i := range params {
		if params[i].In == api.InBody {
			body = &params[i]
			continue
		}
		nonBody = append(nonBody, openAPIParameter(params[i]))
	}
	if len(nonBody) > 0 {
		o.set("parameters", nonBody)
	}
	if body != nil {
		content := obj()
		for _, mt := range oasContentOrder(op.Consumes) {
			content.set(mt, obj().set("schema", renderSchema(withSortedEnums(body.Schema), oasRefPrefix)))
		}
		o.set("requestBody", obj().set("content", content))
	}

	responses := obj()
	for _, r := range mergeResponses(dop.method, op.Responses) {
		responses.set(strconv.Itoa(r.Code), openAPIResponse(r, op.Produces))
	}
	o.set("responses", responses)

	o.setIf(op.Deprecated, "deprecated", true)
	if sec := securityRequirements(op, g.docket, dop.path); sec != nil {
		o.set("security", sec)
	}
	return o
}

func openAPIParameter(prm api.Parameter) *object {
	style := prm.Style
	if style == "" {
		if prm.In == api.InPath {
			style = "simple"
		} else {
			style = "form"
		}
	}
	o := obj().
		set("name", prm.Name).
		set("in", prm.In).
		setNonEmpty("description", prm.Description).
		set("required", prm.Required).
		set("style", style).
		setIf(prm.Explode, "explode", true).
		setIf(prm.FromCondition, "allowReserved", true)
	if prm.AllowReserved != nil {
		o.set("allowReserved", *prm.AllowReserved)
	}

	schema := obj()
	if prm.Range != nil {
		schema.setIf(prm.Range.Bounded(), "maximum", prm.Range.Max).
			set("exclusiveMaximum", false).
			set("minimum", prm.Range.Min).
			set("exclusiveMinimum", false)
	}
	// The OpenAPI mapper drops the parameter default the Swagger 2.0 mapper
	// keeps.
	schemaValue := withSortedEnums(itemSchema(prm.Schema))
	schemaValue.Default = nil
	inner := renderSchema(schemaValue, oasRefPrefix)
	for _, k := range inner.keys {
		schema.set(k, inner.vals[k])
	}
	o.set("schema", schema)
	return o
}

// itemSchema unwraps a repeated query parameter: Swagger 2.0 describes it as an
// array with a collectionFormat, OpenAPI 3 as the element type plus explode.
func itemSchema(s api.Schema) api.Schema {
	if s.Type == api.TypeArray && s.Items != nil {
		return *s.Items
	}
	return s
}

func openAPIResponse(r api.Response, produces []string) *object {
	o := obj().set("description", r.Description)
	if len(r.Examples) > 0 {
		content := obj()
		for _, mt := range sortedKeys(r.Examples) {
			content.set(mt, obj().set("example", r.Examples[mt]))
		}
		o.set("content", content)
	}
	if r.Schema != nil {
		content := obj()
		for _, mt := range oasContentOrder(produces) {
			content.set(mt, obj().set("schema", renderSchema(withSortedEnums(*r.Schema), oasRefPrefix)))
		}
		o.set("content", content)
	}
	if len(r.Headers) > 0 {
		headers := obj()
		for _, h := range r.Headers {
			headers.set(h.Name, obj().
				set("description", "").
				set("required", true).
				set("schema", obj().set("type", string(h.Type))))
		}
		o.set("headers", headers)
	}
	return o
}

// oasContentOrder is the media-type order Springfox emits inside an OpenAPI
// `content` block. It is the reverse of the Swagger 2.0 `produces` list: the
// mapper collects the types into a set that iterates the other way round. The
// rule was recovered from the original's output, where produces
// ["application/json","application/xml"] becomes content
// {"application/xml": …, "application/json": …}.
func oasContentOrder(mediaTypes []string) []string {
	if len(mediaTypes) == 0 {
		return []string{web.MediaTypeAll}
	}
	out := make([]string, 0, len(mediaTypes))
	for i := len(mediaTypes) - 1; i >= 0; i-- {
		out = append(out, mediaTypes[i])
	}
	return out
}

func openAPISecuritySchemes(schemes []SecurityScheme) *object {
	out := obj()
	for _, s := range schemes {
		switch s.Type {
		case SchemeOAuth2:
			scopes := obj()
			for _, sc := range s.Scopes {
				scopes.set(sc.Scope, sc.Description)
			}
			flow := obj()
			if len(s.GrantTypes) > 0 {
				flow.set("authorizationUrl", s.GrantTypes[0].LoginEndpoint)
			}
			flow.set("scopes", scopes)
			out.set(s.Name, obj().
				set("type", "oauth2").
				set("flows", obj().set("implicit", flow)))
		case SchemeBasic:
			out.set(s.Name, obj().set("type", "http").set("scheme", "basic"))
		case SchemeAPIKey:
			out.set(s.Name, obj().
				set("type", "apiKey").
				set("name", s.KeyName).
				set("in", s.PassAs))
		}
	}
	return out
}
