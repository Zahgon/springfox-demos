package main

import (
	"strings"

	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// swaggerUIBaseURL is the value of
// @Value("${springfox.documentation.swagger-ui.base-url:}"). The property is
// unset in this module, so the default — the empty string — applies.
const swaggerUIBaseURL = ""

// ConfigureSwaggerUI is springfoxdemo.boot.swagger.SwaggerUiWebMvcConfigurer.
//
// Note the asymmetry the Java class has and this keeps: the resource handler is
// registered under the base URL with one trailing slash trimmed, while the view
// controller uses the raw property value.
func ConfigureSwaggerUI(app *web.App, baseURL string) {
	trimmed := strings.TrimSuffix(baseURL, "/")
	app.AddResourceHandler(trimmed+"/swagger-ui/**", springfox.SwaggerUIResources{})
	app.AddViewController(baseURL+"/swagger-ui/", "forward:"+baseURL+"/swagger-ui/index.html")

	app.AddCorsMapping("/api/pet", "http://editor.swagger.io")
	app.AddCorsMapping("/v2/api-docs.*", "http://editor.swagger.io")
}
