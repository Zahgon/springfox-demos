package main

import (
	"time"

	"github.com/springfox/springfox-demos/internal/petstore"
	"github.com/springfox/springfox-demos/internal/springfox"
	"github.com/springfox/springfox-demos/internal/web"
)

// NewDispatcherServlet is src/main/webapp/WEB-INF/dispatcher-servlet.xml,
// expressed in Go. The XML declares four things, and each has a line here:
//
//	<mvc:annotation-driven enable-matrix-variables="true"/>
//	<context:component-scan base-package="springfox.petstore.controller"/>
//	<mvc:resources mapping="/swagger-ui/**" location="classpath:/META-INF/resources/webjars/springfox-swagger-ui/"/>
//	<bean name="applicationSwaggerConfig" class="springfoxdemo.xml.swagger.ApplicationSwaggerConfig"/>
func NewDispatcherServlet(app *web.App) (*web.App, error) {
	// <context:component-scan base-package="springfox.petstore.controller"/>
	// is this module's entire API surface.
	petstore.Register(app, func() int64 { return time.Now().UnixMilli() })

	// <bean name="applicationSwaggerConfig" …/>
	docs := NewApplicationSwaggerConfig(app)
	docs.Mount(springfox.Options{PublishResources: true})

	// <mvc:resources mapping="/swagger-ui/**" …/>
	app.AddResourceHandler("/swagger-ui/**", springfox.SwaggerUIResources{})
	return app, nil
}
