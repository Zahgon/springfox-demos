package main

import (
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// NewSwaggerConfig is springfoxdemo.staticdocs.SwaggerConfig, a @Configuration
// annotated @EnableSwagger2WebMvc with no body: one implicit documentation
// group named "default", covering every mapped route.
func NewSwaggerConfig(app *web.App) *springfox.Provider {
	docs := springfox.NewProvider(app)
	docs.MountSpec(springfox.Swagger2)
	docs.AddDocket(springfox.NewDocket(springfox.Swagger2))
	return docs
}
