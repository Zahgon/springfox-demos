package main

import (
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// ConfigureResources is springfoxdemo.java.swagger.SpringConfig, the
// @Component WebMvcConfigurer that mounts the Swagger UI: a resource handler at
// /swagger-ui/** and a view controller forwarding /swagger-ui/ to
// /swagger-ui/index.html.
func ConfigureResources(app *web.App) {
	app.AddResourceHandler("/swagger-ui/**", springfox.SwaggerUIResources{})
	app.AddViewController("/swagger-ui/", "forward:/swagger-ui/index.html")
}
