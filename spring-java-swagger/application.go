// Command spring-java-swagger is a Java-configured Spring Web MVC application
// with OpenAPI 3.0 documentation and the Swagger UI, ported from the
// springfoxdemo.java.swagger package. The original is a WAR deployed by the
// Gretty plugin under the context path /spring-java-swagger; the Go port serves
// the same context path directly.
package main

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// contextPath is the WAR's deployment context, taken from the module name.
const contextPath = "/spring-java-swagger"

// documentationLogging mirrors src/main/resources/application.properties:
//
//	logging.level.springfox.documentation=DEBUG
const documentationLogging = "DEBUG"

// NewApplication assembles the application context.
func NewApplication() (*web.App, error) {
	app := web.NewApp("spring-java-swagger", web.StackWAR)
	app.ContextPath = contextPath
	app.Log.SetLevel("springfox.documentation", web.ParseLevel(documentationLogging))

	docs := NewAppConfiguration(app)
	docs.Mount(springfox.Options{PublishResources: true})
	ConfigureResources(app)
	return app, nil
}

// NewAppConfiguration is springfoxdemo.java.swagger.AppConfiguration:
// @Configuration @EnableWebMvc @ComponentScan("springfoxdemo.java.swagger")
// @EnableOpenApi. The component scan picks up HelloWorldController and
// SpringConfig; @EnableOpenApi publishes one implicit group named "default"
// over the OpenAPI 3.0 endpoint.
func NewAppConfiguration(app *web.App) *springfox.Provider {
	NewHelloWorldController().Register(app)

	docs := springfox.NewProvider(app)
	docs.EnableSpec(springfox.OAS30)
	docs.AddDocket(springfox.NewDocket(springfox.OAS30))
	return docs
}

// HelloWorldController is springfoxdemo.java.swagger.HelloWorldController.
type HelloWorldController struct{}

// NewHelloWorldController creates the controller.
func NewHelloWorldController() *HelloWorldController { return &HelloWorldController{} }

// Register mounts /hello. The Java mapping is a bare @RequestMapping("/hello")
// with no `method` attribute, so Spring maps it for every HTTP method.
func (c *HelloWorldController) Register(app *web.App) {
	for _, method := range []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodTrace,
	} {
		stringSchema := api.Schema{Type: api.TypeString}
		app.Handle(web.Route{
			Method: method, Pattern: "/hello",
			Produces: []string{web.MediaTypeAll},
			Handler:  c.sayHello,
			Op: api.Operation{
				Name: "sayHello", Tag: "hello-world-controller", Summary: "sayHello",
				Produces:  []string{web.MediaTypeAll},
				Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &stringSchema}},
			},
		})
	}
}

// sayHello answers GET /hello with a plain string body.
func (c *HelloWorldController) sayHello(*web.Request) (web.ResponseEntity, error) {
	return web.OKText("Hello World"), nil
}
