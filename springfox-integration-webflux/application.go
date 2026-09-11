// Command springfox-integration-webflux demonstrates Springfox's Spring
// Integration support on a WebFlux project, ported from
// com.escalon.springfox.springintegration.SpringIntegrationWebFluxApplication.
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

// gatewayTag is the Springfox tag for a Spring Integration WebFlux inbound
// gateway.
const gatewayTag = "web-flux-inbound-endpoint"

// Foo is the request payload of the toLower flow. This module's Foo is a
// smaller bean than the webmvc module's: it declares only `bar`.
type Foo struct {
	Bar string `json:"bar"`
}

// Baz is the body type of the application class's @PostMapping method. That
// method is never mapped — see convert.
type Baz struct {
	Barf string `json:"barf"`
}

// NewApplication assembles the application context. The contextLoads test
// asserts that this returns without an error.
func NewApplication() (*web.App, error) {
	app := web.NewApp("springfox-integration-webflux", web.StackReactive)

	app.Models.Add(api.Model{
		Name: "Foo",
		Properties: []api.Property{
			{Name: "bar", Schema: api.Schema{Type: api.TypeString}},
		},
	})
	// Springfox cannot unwrap the reactive actuator handler's return type, so
	// it publishes the raw generic signature as a bare model.
	app.Models.Add(api.Model{Name: boot.ReactivePublisherModel})

	boot.RegisterActuator(app, "http://localhost:8080")
	toUpperFlow().Register(app)
	toUpperGetFlow().Register(app)
	toLowerFlow().Register(app)

	docs := springfox.NewProvider(app)
	docs.MountSpec(springfox.Swagger2)
	docs.AddDocket(springfox.NewDocket(springfox.Swagger2))
	// @EnableSwagger2WebFlux leaves the resource listing empty.
	docs.Mount(springfox.Options{PublishResources: false})

	app.AddResourceHandler("/swagger-ui/**", springfox.SwaggerUIResources{})
	return app, nil
}

// endpointName is the identifier Spring Integration generates for a gateway
// that declares no `.id(...)`: the bean method name, the component type, and a
// per-flow ordinal.
func endpointName(flow string) string { return flow + ".webflux:inbound-gateway#0" }

// toUpperFlow is the toUpperFlow bean: a text/plain POST gateway that
// upper-cases its payload.
func toUpperFlow() *integration.InboundGateway {
	return integration.NewInboundGateway("/conversions/upper").
		RequestMapping(http.MethodPost, web.MediaTypeTextPlain).
		ID(endpointName("toUpperFlow")).
		Tag(gatewayTag).
		ReplyContentType(web.MediaTypeTextPlain).
		Handle(func(m integration.Message) (any, error) {
			payload, _ := m.Payload.(string)
			return strings.ToUpper(payload), nil
		})
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
		ID(endpointName("toUpperGetFlow")).
		Tag(gatewayTag).
		ReplyContentType(web.CharsetTextPlain).
		Handle(func(m integration.Message) (any, error) {
			payload, _ := m.Payload.(string)
			if m.Header("upperLower") == "upper" {
				return strings.ToUpper(payload), nil
			}
			return strings.ToLower(payload), nil
		})
}

// toLowerFlow is the toLowerFlow bean. It declares .requestPayloadType(Foo.class)
// and then handles the message with a lambda whose erased parameter type is
// String, so the payload — a Foo — cannot be cast and the flow throws
// ClassCastException at runtime. That is a genuine defect in the original; it
// is observable as a 500 and is reproduced rather than repaired.
func toLowerFlow() *integration.InboundGateway {
	return integration.NewInboundGateway("/conversions/lower").
		RequestMapping(http.MethodPost, web.MediaTypeJSON).
		RequestPayloadType(func() any { return &Foo{} }, api.Schema{Ref: "Foo"}).
		ID(endpointName("toLowerFlow")).
		Tag(gatewayTag).
		Handle(func(m integration.Message) (any, error) {
			payload, ok := m.Payload.(string)
			if !ok {
				return nil, integration.ErrClassCast{
					From: "com.escalon.springfox.springintegration.SpringIntegrationWebFluxApplication$Foo",
					To:   "java.lang.String",
				}
			}
			return strings.ToUpper(payload), nil
		})
}

// convert is the body of the @PostMapping("/conversion/controller") method on
// the application class. Unlike the webmvc module's application, this class
// carries no @RestController or @Controller stereotype, so Spring never
// registers a mapping for it and the path answers 404 — nothing above mounts a
// route. The method is kept for the same reason the Java one is: the demo
// declares it.
func convert(baz Baz) Baz { return baz }
