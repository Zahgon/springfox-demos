// Command boot-webmvc is a Spring Boot Web MVC demo with OpenAPI 3.0 API
// documentation enabled, ported from
// io.springfox.demo.bootwebmvc.BootWebmvcApplication.
package main

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/boot"
	"github.com/springfox/springfox-demos/internal/javalang"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// Configuration mirrors src/main/resources/application.properties:
//
//	logging.level.springfox.documentation=DEBUG
//	springfox.documentation.swagger-ui.base-url=/documentation
//	server.servlet.context-path=/mvc
//	springfox.documentation.swagger.v2.use-model-v3=false
const (
	contextPath          = "/mvc"
	swaggerUIBaseURL     = "/documentation"
	documentationLogging = "DEBUG"
	useModelV3           = false
)

// NewApplication assembles the application context. The contextLoads test
// asserts that this returns without an error, exactly as @SpringBootTest
// asserts that the Spring context starts.
func NewApplication() (*web.App, error) {
	app := web.NewApp("boot-webmvc", web.StackServlet)
	app.ContextPath = contextPath
	app.Log.SetLevel("springfox.documentation", web.ParseLevel(documentationLogging))

	// Spring Security: every request is permitted anonymously and CSRF tokens
	// are carried in a cookie that JavaScript can read.
	ConfigureSecurity(app)

	newHelloWorldController().register(app)
	boot.RegisterErrorController(app, false)

	docs := springfox.NewProvider(app)
	docs.UseModelV3 = useModelV3
	// springfox-boot-starter publishes both specification versions over a
	// single implicit group named "default". A group is rendered by whichever
	// specification the client asks for, so one Docket serves both endpoints.
	docs.MountSpec(springfox.Swagger2)
	docs.EnableSpec(springfox.OAS30)
	docs.AddDocket(springfox.NewDocket(springfox.OAS30))
	docs.Mount(springfox.Options{
		BaseURL:               swaggerUIBaseURL,
		SecurityConfiguration: securityConfiguration(),
		PublishResources:      true,
	})

	app.AddResourceHandler(swaggerUIBaseURL+"/swagger-ui/**", springfox.SwaggerUIResources{})
	app.AddViewController(swaggerUIBaseURL+"/swagger-ui/", "forward:"+swaggerUIBaseURL+"/swagger-ui/index.html")
	return app, nil
}

// securityConfiguration is the SecurityConfiguration bean:
//
//	SecurityConfigurationBuilder.builder().enableCsrfSupport(true).build()
func securityConfiguration() springfox.SecurityConfiguration {
	return springfox.NewSecurityConfigurationBuilder().
		EnableCsrfSupport(true).
		Build()
}

// helloWorldController is the nested @RestController @RequestMapping("/hello")
// HelloWorldController.
type helloWorldController struct{}

func newHelloWorldController() *helloWorldController { return &helloWorldController{} }

func (c *helloWorldController) register(app *web.App) {
	stringSchema := api.Schema{Type: api.TypeString}
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/hello",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.hello,
		Op: api.Operation{
			Name: "hello", Tag: "hello-world-controller", Summary: "hello",
			Produces:  []string{web.MediaTypeAll},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &stringSchema}},
		},
	})
	doubleSchema := api.Schema{Type: api.TypeString}
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/hello/double",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.testDouble,
		Op: api.Operation{
			Name: "testDouble", Tag: "hello-world-controller", Summary: "testDouble",
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "test", In: api.InQuery, Description: "test", Required: false,
				Schema: api.Schema{Type: api.TypeNumber, Format: "double", Default: javalang.Double(10)},
			}},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &doubleSchema}},
		},
	})
}

// hello answers GET /hello with a plain string body.
func (c *helloWorldController) hello(*web.Request) (web.ResponseEntity, error) {
	return web.OKText("Hello SpringFox!"), nil
}

// testDouble answers GET /hello/double. The Java handler concatenates a boxed
// Double into a String, so the body carries java.lang.Double.toString's layout
// rather than Go's default float formatting.
func (c *helloWorldController) testDouble(r *web.Request) (web.ResponseEntity, error) {
	count, err := r.QueryDouble("test", 10)
	if err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OKText("Value " + web.JavaDoubleToString(count)), nil
}
