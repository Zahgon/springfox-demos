// Command springfox-integration-webmvc demonstrates Springfox's Spring
// Integration support on a Web MVC project, ported from
// com.escalon.springfox.springintegration.SpringIntegrationWebMvcApplication.
package main

import (
	"net/http"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/boot"
	"github.com/springfox/springfox-demos/internal/integration"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// gatewayTag is the Springfox tag for a Spring Integration HTTP inbound
// gateway on the servlet stack.
const gatewayTag = "http-request-handling-messaging-gateway"

// Foo is the request payload of the toLower flow. Its properties are declared
// in the order Jackson serialises them, and `foo` and `count` carry the field
// initialisers of the Java bean.
type Foo struct {
	Bar   string `json:"bar"`
	Foo   bool   `json:"foo"`
	Count int32  `json:"count"`
}

// NewFoo is the no-argument constructor, which leaves foo false and count 3.
func NewFoo() *Foo { return &Foo{Count: 3} }

// NewFooWithBar is `new Foo(String bar)`. It sets only `bar`, so the values the
// client sent for `foo` and `count` are discarded and the defaults reappear in
// the reply.
func NewFooWithBar(bar string) *Foo {
	f := NewFoo()
	f.Bar = bar
	return f
}

// Baz is the request and response body of the /conversion/controller mapping.
type Baz struct {
	Gnarf string `json:"gnarf"`
}

// NewApplication assembles the application context. The contextLoads test
// asserts that this returns without an error.
func NewApplication() (*web.App, error) {
	app := web.NewApp("springfox-integration-webmvc", web.StackServlet)

	app.Models.Add(api.Model{
		Name: "Foo",
		Properties: []api.Property{
			{Name: "bar", Schema: api.Schema{Type: api.TypeString}},
			{Name: "count", Schema: api.Schema{Type: api.TypeInteger, Format: "int32"}},
			{Name: "foo", Schema: api.Schema{Type: api.TypeBoolean}},
		},
	})
	app.Models.Add(api.Model{
		Name: "Baz",
		Properties: []api.Property{
			{Name: "gnarf", Schema: api.Schema{Type: api.TypeString}},
		},
	})

	boot.RegisterActuator(app, "http://localhost:8080")
	toUpperGetFlow().Register(app)
	toUpperFlow().Register(app)
	toLowerFlow().Register(app)
	registerConvertController(app)
	boot.RegisterErrorController(app, false)

	docs := springfox.NewProvider(app)
	docs.MountSpec(springfox.Swagger2)
	docs.AddDocket(springfox.NewDocket(springfox.Swagger2))
	// @EnableSwagger2WebMvc leaves the resource listing empty: its resource
	// provider never sees the documentation cache.
	docs.Mount(springfox.Options{PublishResources: false})

	app.AddResourceHandler("/swagger-ui/**", springfox.SwaggerUIResources{})
	return app, nil
}

// toUpperGetFlow is the toUpperGetFlow bean: a GET gateway whose payload is the
// `toConvert` request parameter and whose `upperLower` header comes from the
// path variable of the same name.
func toUpperGetFlow() *integration.InboundGateway {
	return integration.NewInboundGateway("/conversions/pathvariable/{upperLower}").
		RequestMapping(http.MethodGet).
		Params("toConvert").
		HeaderExpression("upperLower", "upperLower").
		PayloadFromRequestParam("toConvert").
		ID("toUpperLowerGateway").
		Tag(gatewayTag).
		// The reply carries no request Content-Type to negotiate a charset
		// from, so Spring MVC falls back to ISO-8859-1.
		ReplyContentType(web.CharsetTextPlainLatin1).
		Handle(func(m integration.Message) (any, error) {
			payload, _ := m.Payload.(string)
			if m.Header("upperLower") == "upper" {
				return strings.ToUpper(payload), nil
			}
			return strings.ToLower(payload), nil
		})
}

// toUpperFlow is the toUpperFlow bean: a text/plain POST gateway that
// upper-cases its String payload.
func toUpperFlow() *integration.InboundGateway {
	return integration.NewInboundGateway("/conversions/upper").
		RequestMapping(http.MethodPost, web.MediaTypeTextPlain).
		RequestPayloadString(&api.Schema{Type: api.TypeString}).
		ID("toUpperGateway").
		Tag(gatewayTag).
		ReplyContentType(web.CharsetTextPlain).
		Handle(func(m integration.Message) (any, error) {
			payload, _ := m.Payload.(string)
			return strings.ToUpper(payload), nil
		})
}

// toLowerFlow is the toLowerFlow bean: an application/json POST gateway that
// rebuilds its Foo payload through the one-argument constructor, lower-casing
// `bar` and resetting `foo` and `count` to their field initialisers.
func toLowerFlow() *integration.InboundGateway {
	return integration.NewInboundGateway("/conversions/lower").
		RequestMapping(http.MethodPost, web.MediaTypeJSON).
		RequestPayloadType(func() any { return NewFoo() }, api.Schema{Ref: "Foo"}).
		ID("toLowerGateway").
		Tag(gatewayTag).
		ReplyContentType(web.CharsetJSON).
		Handle(func(m integration.Message) (any, error) {
			foo, ok := m.Payload.(*Foo)
			if !ok {
				return nil, integration.ErrClassCast{From: "Foo", To: "String"}
			}
			return NewFooWithBar(strings.ToLower(foo.Bar)), nil
		})
}

// registerConvertController is the @PostMapping method on the application
// class, which is itself a @RestController here.
func registerConvertController(app *web.App) {
	app.Handle(web.Route{
		Method:   http.MethodPost,
		Pattern:  "/conversion/controller",
		Produces: []string{web.MediaTypeAll},
		Handler:  convert,
		Op: api.Operation{
			Name: "convert", Tag: "spring-integration-web-mvc-application", Summary: "convert",
			Consumes: []string{web.MediaTypeJSON},
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "baz", In: api.InBody, Description: "baz", Required: true,
				Schema: api.Schema{Ref: "Baz"},
			}},
			Responses: []api.Response{{
				Code: http.StatusOK, Description: "OK",
				Schema:   &api.Schema{Ref: "Baz"},
				Examples: map[string]string{web.MediaTypeJSON: `{"gnarf": "dragons"}`},
			}},
		},
	})
}

// convert echoes the request body back, as the Java method does.
func convert(r *web.Request) (web.ResponseEntity, error) {
	var baz Baz
	if err := r.BindJSON(&baz); err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OK(baz).WithContentType(web.MediaTypeJSON), nil
}
