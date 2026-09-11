// Package integration reproduces the slice of Spring Integration the two
// springfox-integration demos use: an HTTP inbound gateway that turns a request
// into a message, a handler that transforms the payload, and a reply that is
// written back as the response.
//
// The demos exercise the gateway's observable surface only — the request
// mapping, the payload and header expressions, the reply's media type, and what
// happens when a handler is given a payload of the wrong type — so that surface
// is what is reproduced. No channels, adapters or poller machinery are needed
// for it.
package integration

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// Message is the payload/header pair a gateway hands to its handler.
type Message struct {
	Payload any
	Headers map[string]any
}

// Header returns a header value, or nil when it is absent.
func (m Message) Header(name string) any { return m.Headers[name] }

// Handler is the `.handle((p, h) -> …)` lambda of an integration flow.
type Handler func(Message) (any, error)

// PayloadKind selects how the gateway builds the message payload.
type PayloadKind int

const (
	// PayloadBodyString is `.requestPayloadType(String.class)`, or the default
	// for a text/plain gateway: the payload is the request body.
	PayloadBodyString PayloadKind = iota
	// PayloadBodyJSON is `.requestPayloadType(Foo.class)`: the payload is the
	// request body deserialised into the gateway's payload type.
	PayloadBodyJSON
	// PayloadExpression is `.payloadExpression("#requestParams['x'][0]")`: the
	// payload comes from a request parameter.
	PayloadExpression
)

// InboundGateway is Http.inboundGateway / WebFlux.inboundGateway.
type InboundGateway struct {
	path     string
	method   string
	consumes []string
	// params are the `.requestMapping(r -> r.params("x"))` conditions, which
	// Spring Integration treats as required request parameters.
	params []string

	kind PayloadKind
	// payloadParam is the request parameter PayloadExpression reads.
	payloadParam string
	// headerExprs map a header name to the path variable it is taken from.
	headerExprs map[string]string
	// newPayload allocates the value a PayloadBodyJSON gateway deserialises into.
	newPayload func() any

	// id is the gateway's `.id(...)`. When empty the flow supplies the
	// generated endpoint name instead.
	id string
	// tag is the Springfox tag the gateway is documented under.
	tag string
	// bodySchema documents the request body, when the gateway takes one.
	bodySchema *api.Schema
	// replyContentType is the Content-Type of a successful reply.
	replyContentType string

	handler Handler
}

// NewInboundGateway starts an inbound gateway for a path.
func NewInboundGateway(path string) *InboundGateway {
	return &InboundGateway{path: path, method: http.MethodGet, headerExprs: map[string]string{}}
}

// RequestMapping sets the HTTP method and the consumed media types.
func (g *InboundGateway) RequestMapping(method string, consumes ...string) *InboundGateway {
	g.method = method
	g.consumes = consumes
	return g
}

// Params adds required request-parameter conditions.
func (g *InboundGateway) Params(names ...string) *InboundGateway {
	g.params = append(g.params, names...)
	return g
}

// HeaderExpression maps a header name onto a path variable, reproducing
// `.headerExpression("upperLower", "#pathVariables.upperLower")`.
func (g *InboundGateway) HeaderExpression(header, pathVariable string) *InboundGateway {
	g.headerExprs[header] = pathVariable
	return g
}

// PayloadFromRequestParam reproduces
// `.payloadExpression("#requestParams['name'][0]")`.
func (g *InboundGateway) PayloadFromRequestParam(name string) *InboundGateway {
	g.kind = PayloadExpression
	g.payloadParam = name
	return g
}

// RequestPayloadType reproduces `.requestPayloadType(T.class)` for a JSON body.
func (g *InboundGateway) RequestPayloadType(newPayload func() any, schema api.Schema) *InboundGateway {
	g.kind = PayloadBodyJSON
	g.newPayload = newPayload
	g.bodySchema = &schema
	return g
}

// RequestPayloadString reproduces `.requestPayloadType(String.class)`.
func (g *InboundGateway) RequestPayloadString(schema *api.Schema) *InboundGateway {
	g.kind = PayloadBodyString
	g.bodySchema = schema
	return g
}

// ID sets the gateway's `.id(...)`.
func (g *InboundGateway) ID(id string) *InboundGateway { g.id = id; return g }

// Tag sets the Springfox tag the gateway is documented under.
func (g *InboundGateway) Tag(tag string) *InboundGateway { g.tag = tag; return g }

// ReplyContentType pins the Content-Type of the reply.
func (g *InboundGateway) ReplyContentType(ct string) *InboundGateway {
	g.replyContentType = ct
	return g
}

// Handle sets the flow's transformation and completes the gateway.
func (g *InboundGateway) Handle(h Handler) *InboundGateway { g.handler = h; return g }

// Register mounts the gateway on an application.
func (g *InboundGateway) Register(app *web.App) {
	var params []api.Parameter
	for _, name := range g.params {
		params = append(params, api.Parameter{
			Name: name, In: api.InQuery, Description: name, Required: true,
			Schema: api.Schema{Type: api.TypeString},
		})
	}
	for header := range g.headerExprs {
		params = append(params, api.Parameter{
			Name: g.headerExprs[header], In: api.InPath, Description: g.headerExprs[header],
			Required: true, Schema: api.Schema{Type: api.TypeString},
		})
	}
	if g.bodySchema != nil {
		params = append(params, api.Parameter{
			Name: "body", In: api.InBody, Description: "body", Required: true,
			Schema: *g.bodySchema,
		})
	}

	app.Handle(web.Route{
		Method:   g.method,
		Pattern:  g.path,
		Consumes: g.consumes,
		Produces: []string{web.MediaTypeAll},
		Handler:  g.serve,
		Op: api.Operation{
			Name: g.id, Tag: g.tag, Summary: g.id,
			Consumes:  g.consumes,
			Produces:  []string{web.MediaTypeAll},
			Params:    params,
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})
}

// serve builds the message, runs the handler and writes the reply.
func (g *InboundGateway) serve(r *web.Request) (web.ResponseEntity, error) {
	msg := Message{Headers: map[string]any{}}
	for header, pathVar := range g.headerExprs {
		msg.Headers[header] = r.PathVar(pathVar)
	}

	switch g.kind {
	case PayloadExpression:
		v, err := r.RequireQuery(g.payloadParam)
		if err != nil {
			return web.ResponseEntity{}, err
		}
		msg.Payload = v
	case PayloadBodyJSON:
		payload := g.newPayload()
		if err := r.BindJSON(payload); err != nil {
			return web.ResponseEntity{}, err
		}
		msg.Payload = payload
	default:
		body, err := r.BodyText()
		if err != nil {
			return web.ResponseEntity{}, err
		}
		msg.Payload = body
	}

	reply, err := g.handler(msg)
	if err != nil {
		return web.ResponseEntity{}, err
	}
	switch v := reply.(type) {
	case string:
		return web.OKText(v).WithContentType(g.replyContentType), nil
	default:
		return web.OK(v).WithContentType(g.replyContentType), nil
	}
}

// ErrClassCast is java.lang.ClassCastException. A flow whose handler is typed
// for one payload but whose gateway deserialises another throws it at runtime,
// and nothing maps it, so the request fails with a 500.
type ErrClassCast struct{ From, To string }

func (e ErrClassCast) Error() string {
	return "class " + e.From + " cannot be cast to class " + e.To
}
