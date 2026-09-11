package integration

import (
	"net/http"
	"strings"
	"testing"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

type foo struct {
	Bar string `json:"bar"`
}

// TestPayloadFromRequestParam covers `.payloadExpression("#requestParams[…][0]")`
// together with `.headerExpression(…, "#pathVariables.…")`.
func TestPayloadFromRequestParam(t *testing.T) {
	app := web.NewApp("test", web.StackServlet)
	NewInboundGateway("/conversions/pathvariable/{upperLower}").
		RequestMapping(http.MethodGet).
		Params("toConvert").
		HeaderExpression("upperLower", "upperLower").
		PayloadFromRequestParam("toConvert").
		ID("toUpperLowerGateway").
		Tag("gateway").
		Handle(func(m Message) (any, error) {
			payload, _ := m.Payload.(string)
			if m.Header("upperLower") == "upper" {
				return strings.ToUpper(payload), nil
			}
			return strings.ToLower(payload), nil
		}).
		Register(app)

	mvc := web.NewMockMvc(app)
	if res := mvc.Perform(web.Get("/conversions/pathvariable/upper?toConvert=Gimli")); res.Body != "GIMLI" {
		t.Errorf("upper = %q, want GIMLI", res.Body)
	}
	if res := mvc.Perform(web.Get("/conversions/pathvariable/lower?toConvert=Gimli")); res.Body != "gimli" {
		t.Errorf("lower = %q, want gimli", res.Body)
	}
	// The comparison against the literal "upper" is case-sensitive.
	if res := mvc.Perform(web.Get("/conversions/pathvariable/UPPER?toConvert=Gimli")); res.Body != "gimli" {
		t.Errorf("UPPER = %q, want gimli", res.Body)
	}
	if res := mvc.Perform(web.Get("/conversions/pathvariable/upper")); res.Status != http.StatusBadRequest {
		t.Errorf("without the required parameter, status = %d, want %d", res.Status, http.StatusBadRequest)
	}
}

// TestRequestPayloadTypeDeserialisesTheBody covers `.requestPayloadType(T.class)`.
func TestRequestPayloadTypeDeserialisesTheBody(t *testing.T) {
	app := web.NewApp("test", web.StackServlet)
	NewInboundGateway("/conversions/lower").
		RequestMapping(http.MethodPost, web.MediaTypeJSON).
		RequestPayloadType(func() any { return &foo{} }, api.Schema{Ref: "Foo"}).
		ID("toLowerGateway").
		Tag("gateway").
		Handle(func(m Message) (any, error) {
			f, ok := m.Payload.(*foo)
			if !ok {
				return nil, ErrClassCast{From: "Foo", To: "String"}
			}
			return &foo{Bar: strings.ToLower(f.Bar)}, nil
		}).
		Register(app)

	res := web.NewMockMvc(app).Perform(web.Post("/conversions/lower").
		ContentType(web.MediaTypeJSON).Content(`{"bar":"Aragorn"}`))
	if res.Body != `{"bar":"aragorn"}` {
		t.Errorf("body = %s, want %s", res.Body, `{"bar":"aragorn"}`)
	}
}

// TestClassCastFailureIsA500 covers the flow whose handler is typed for a
// payload the gateway never produces, which is the defect
// springfox-integration-webflux's toLowerFlow carries.
func TestClassCastFailureIsA500(t *testing.T) {
	app := web.NewApp("test", web.StackReactive)
	NewInboundGateway("/conversions/lower").
		RequestMapping(http.MethodPost, web.MediaTypeJSON).
		RequestPayloadType(func() any { return &foo{} }, api.Schema{Ref: "Foo"}).
		ID("toLowerFlow").
		Tag("gateway").
		Handle(func(m Message) (any, error) {
			payload, ok := m.Payload.(string)
			if !ok {
				return nil, ErrClassCast{From: "Foo", To: "java.lang.String"}
			}
			return strings.ToUpper(payload), nil
		}).
		Register(app)

	res := web.NewMockMvc(app).Perform(web.Post("/conversions/lower").
		ContentType(web.MediaTypeJSON).Content(`{"bar":"Aragorn"}`))
	if res.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
}

// TestClassCastMessage covers the exception text the mistyped flow raises,
// which is what identifies the failure in a log.
func TestClassCastMessage(t *testing.T) {
	err := ErrClassCast{From: "Foo", To: "java.lang.String"}
	want := "class Foo cannot be cast to class java.lang.String"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if _, ok := web.AsHTTPError(err); ok {
		t.Error("a ClassCastException was classified as an HTTP error; it must become a 500")
	}
}

// TestConsumesIsEnforced covers the gateway's requestMapping consumes clause.
func TestConsumesIsEnforced(t *testing.T) {
	app := web.NewApp("test", web.StackServlet)
	NewInboundGateway("/conversions/upper").
		RequestMapping(http.MethodPost, web.MediaTypeTextPlain).
		RequestPayloadString(&api.Schema{Type: api.TypeString}).
		ID("toUpperGateway").Tag("gateway").
		ReplyContentType(web.CharsetTextPlain).
		Handle(func(m Message) (any, error) {
			payload, _ := m.Payload.(string)
			return strings.ToUpper(payload), nil
		}).
		Register(app)

	mvc := web.NewMockMvc(app)
	ok := mvc.Perform(web.Post("/conversions/upper").
		ContentType(web.MediaTypeTextPlain).Content("aragorn"))
	if ok.Body != "ARAGORN" || ok.ContentType != web.CharsetTextPlain {
		t.Errorf("body = %q with %q", ok.Body, ok.ContentType)
	}
	wrong := mvc.Perform(web.Post("/conversions/upper").
		ContentType(web.MediaTypeJSON).Content("aragorn"))
	if wrong.Status != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want %d", wrong.Status, http.StatusUnsupportedMediaType)
	}
}

// TestGatewayDocumentsItsParameters covers the operation metadata a gateway
// contributes: the required request parameters, the path variables its header
// expressions read, and the request body.
func TestGatewayDocumentsItsParameters(t *testing.T) {
	app := web.NewApp("test", web.StackServlet)
	NewInboundGateway("/conversions/pathvariable/{upperLower}").
		RequestMapping(http.MethodGet).
		Params("toConvert").
		HeaderExpression("upperLower", "upperLower").
		PayloadFromRequestParam("toConvert").
		ID("toUpperLowerGateway").Tag("gateway").
		Handle(func(Message) (any, error) { return "", nil }).
		Register(app)

	op := app.Routes()[0].Op
	if op.Name != "toUpperLowerGateway" || op.Summary != "toUpperLowerGateway" {
		t.Errorf("the operation is named %q/%q", op.Name, op.Summary)
	}
	found := map[string]string{}
	for _, p := range op.Params {
		found[p.Name] = p.In
	}
	if found["toConvert"] != api.InQuery || found["upperLower"] != api.InPath {
		t.Errorf("the documented parameters are %v", found)
	}
}
