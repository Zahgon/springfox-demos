// Command boot-static-docs is a Spring Boot application with a default Swagger
// 2.0 configuration, used to generate static Asciidoc at build time. It is
// ported from springfoxdemo.staticdocs.Application.
package main

import (
	"time"

	"github.com/springfox/springfox-demos/internal/boot"
	"github.com/springfox/springfox-demos/internal/petstore"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// NewApplication assembles the application context.
//
// The Java class is
// @SpringBootApplication @ComponentScan(basePackageClasses = {SwaggerConfig.class, PetController.class}),
// so the context holds this module's Swagger configuration plus the three
// petstore controllers.
func NewApplication() (*web.App, error) {
	app := web.NewApp("boot-static-docs", web.StackServlet)

	petstore.Register(app, func() int64 { return time.Now().UnixMilli() })
	// BasicErrorController maps error() and errorHtml() over the same methods;
	// this module's documents name the errorHtml variant.
	boot.RegisterErrorController(app, true)

	docs := NewSwaggerConfig(app)
	docs.Mount(springfox.Options{PublishResources: false})

	app.AddResourceHandler("/swagger-ui/**", springfox.SwaggerUIResources{})
	return app, nil
}
