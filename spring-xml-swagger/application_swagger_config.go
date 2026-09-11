package main

import (
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// NewApplicationSwaggerConfig is
// springfoxdemo.xml.swagger.ApplicationSwaggerConfig, the @EnableSwagger2 bean
// that publishes one Docket:
//
//	new Docket(DocumentationType.SWAGGER_2)
//
// with no further customisation, so the group is named "default" and no path
// filtering applies.
func NewApplicationSwaggerConfig(app *web.App) *springfox.Provider {
	docs := springfox.NewProvider(app)
	docs.EnableSpec(springfox.Swagger2)
	docs.AddDocket(petStore())
	return docs
}

// petStore is the Docket bean of the same name.
func petStore() *springfox.Docket { return springfox.NewDocket(springfox.Swagger2) }
