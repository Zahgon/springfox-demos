// Command boot-webflux is a Spring Boot WebFlux demo with OpenAPI 3.0 API
// documentation enabled, ported from
// io.springfox.demo.bootwebflux.BootWebfluxApplication.
package main

import (
	"net/http"
	"strings"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/boot"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// Configuration mirrors src/main/resources/application.properties:
//
//	logging.level.springfox.documentation=DEBUG
//	springfox.documentation.swagger-ui.base-url=/documentation
//	#server.servlet.context-path=/mvc
const (
	swaggerUIBaseURL     = "/documentation"
	documentationLogging = "DEBUG"
)

// NewApplication assembles the application context. The contextLoads test
// asserts that this returns without an error.
func NewApplication() (*web.App, error) {
	app := web.NewApp("boot-webflux", web.StackReactive)
	app.Log.SetLevel("springfox.documentation", web.ParseLevel(documentationLogging))

	newFunctionalHelloWorldController().register(app)
	registerGreetingRoute(app, newGreetingHandler())
	boot.RegisterErrorController(app, false)

	docs := springfox.NewProvider(app)
	docs.MountSpec(springfox.Swagger2)
	docs.EnableSpec(springfox.OAS30)
	docs.AddDocket(springfox.NewDocket(springfox.OAS30))
	docs.Mount(springfox.Options{
		BaseURL:          swaggerUIBaseURL,
		PublishResources: true,
	})

	app.AddResourceHandler(swaggerUIBaseURL+"/swagger-ui/**", springfox.SwaggerUIResources{})
	app.AddViewController(swaggerUIBaseURL+"/swagger-ui/", "forward:"+swaggerUIBaseURL+"/swagger-ui/index.html")
	return app, nil
}

// functionalHelloWorldController is the nested
// @RestController @RequestMapping("/functional-hello") controller.
type functionalHelloWorldController struct{}

func newFunctionalHelloWorldController() *functionalHelloWorldController {
	return &functionalHelloWorldController{}
}

// register mounts the four handlers.
//
// The two `helloPeople` overloads share a method name, so Springfox gives one
// of them an "_1" suffix. Which one gets the plain id depends on the order the
// JVM reports the class's declared methods in, an order the specification
// leaves undefined. The registration order below pins the ids to the ones the
// Java module was observed to emit: /response-flux keeps helloPeopleUsingGET
// and /flux takes helloPeopleUsingGET_1.
func (c *functionalHelloWorldController) register(app *web.App) {
	stringSchema := api.Schema{Type: api.TypeString}
	stringArray := api.Schema{Type: api.TypeArray, Items: &stringSchema}
	// Springfox describes the repeatable `names` parameter as a bare string
	// with explode set, not as an array with a collection format: the reactive
	// parameter reader does not unwrap the varargs/List element type.
	repeatedString := stringSchema

	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/functional-hello/response-mono",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.helloMono,
		Op: api.Operation{
			Name: "helloMono", Tag: "functional-hello-world-controller", Summary: "helloMono",
			Produces: []string{web.MediaTypeAll},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &stringSchema},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/functional-hello/mono",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.helloPerson,
		Op: api.Operation{
			Name: "helloPerson", Tag: "functional-hello-world-controller", Summary: "helloPerson",
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "name", In: api.InQuery, Description: "name", Required: false,
				Schema: stringSchema,
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &stringSchema},
			},
		},
	})
	responseFluxArray := stringArray
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/functional-hello/response-flux",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.helloPeopleRequired,
		Op: api.Operation{
			Name: "helloPeople", Tag: "functional-hello-world-controller", Summary: "helloPeople",
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "names", In: api.InQuery, Description: "names", Required: true,
				Schema: repeatedString, Explode: true,
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &responseFluxArray},
			},
		},
	})
	fluxArray := stringArray
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/functional-hello/flux",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.helloPeopleVarargs,
		Op: api.Operation{
			Name: "helloPeople", Tag: "functional-hello-world-controller", Summary: "helloPeople",
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "names", In: api.InQuery, Description: "names", Required: false,
				Schema: repeatedString, Explode: true,
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &fluxArray},
			},
		},
	})
}

// helloMono answers GET /functional-hello/response-mono.
func (c *functionalHelloWorldController) helloMono(*web.Request) (web.ResponseEntity, error) {
	return web.OKText("Hello SpringFox!").WithContentType(web.CharsetTextPlain), nil
}

// helloPerson answers GET /functional-hello/mono. `name` is a plain String
// parameter, so it is optional and binds to null when absent — which Java's
// string concatenation renders as the four characters "null".
func (c *functionalHelloWorldController) helloPerson(r *web.Request) (web.ResponseEntity, error) {
	name := r.QueryOr("name", "null")
	return web.OKText("Hello " + name + "!").WithContentType(web.CharsetTextPlain), nil
}

// helloPeopleVarargs answers GET /functional-hello/flux. The Java signature
// takes `String... names`, which Spring leaves null when the parameter is
// absent; Flux.fromArray(null) then throws and the request fails with a 500.
func (c *functionalHelloWorldController) helloPeopleVarargs(r *web.Request) (web.ResponseEntity, error) {
	names := r.QueryAll("names")
	if len(names) == 0 {
		return web.ResponseEntity{}, errNullPointer{}
	}
	return fluxOfGreetings(names), nil
}

// helloPeopleRequired answers GET /functional-hello/response-flux. Its
// @RequestParam List<String> is required, so an absent parameter is a 400.
func (c *functionalHelloWorldController) helloPeopleRequired(r *web.Request) (web.ResponseEntity, error) {
	names := r.QueryAll("names")
	if len(names) == 0 {
		return web.ResponseEntity{}, web.ErrBadRequest
	}
	return fluxOfGreetings(names), nil
}

// fluxOfGreetings writes the elements of a Flux<String> the way WebFlux does
// with no `produces` attribute: CharSequenceEncoder.textPlainOnly() is ordered
// ahead of the JSON writer, so the elements are concatenated as plain text with
// no separator between them.
func fluxOfGreetings(names []string) web.ResponseEntity {
	var b strings.Builder
	for _, name := range names {
		b.WriteString("Hello " + name)
	}
	return web.OKText(b.String()).WithContentType(web.CharsetTextPlain)
}

// errNullPointer is java.lang.NullPointerException, which is unmapped and
// therefore surfaces as a 500.
const nullPointerExceptionName = "NullPointerException"

type errNullPointer struct{}

func (errNullPointer) Error() string { return nullPointerExceptionName }

// greetingHandler is the @Component GreetingHandler.
type greetingHandler struct{}

func newGreetingHandler() *greetingHandler { return &greetingHandler{} }

// hello is the handler the router function delegates to.
func (h *greetingHandler) hello(*web.Request) (web.ResponseEntity, error) {
	return web.OKText("Hello, SpringFox!").WithContentType(web.MediaTypeTextPlain), nil
}

// registerGreetingRoute is the RouterFunction bean:
//
//	RouterFunctions.route(RequestPredicates.GET("/hello")
//	    .and(RequestPredicates.accept(MediaType.TEXT_PLAIN)), greetingHandler::hello)
//
// Springfox does not scan router functions, so the route carries no operation
// metadata and never appears in the generated documents.
func registerGreetingRoute(app *web.App, h *greetingHandler) {
	app.Handle(web.Route{
		Method:  http.MethodGet,
		Pattern: "/hello",
		Accept:  []string{web.MediaTypeTextPlain},
		Handler: h.hello,
	})
}
