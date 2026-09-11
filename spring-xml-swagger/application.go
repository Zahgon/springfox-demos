// Command spring-xml-swagger is an XML-configured Spring Web MVC application
// with Swagger 2.0 documentation and the Swagger UI, ported from the
// springfoxdemo.xml.swagger package and from
// src/main/webapp/WEB-INF/dispatcher-servlet.xml. The original is a WAR
// deployed by the Gretty plugin under the context path /spring-xml-swagger; the
// Go port serves the same context path directly.
package main

import "github.com/springfox/springfox-demos/internal/web"

// contextPath is the WAR's deployment context, taken from the module name.
const contextPath = "/spring-xml-swagger"

// NewApplication assembles the application context.
func NewApplication() (*web.App, error) {
	app := web.NewApp("spring-xml-swagger", web.StackWAR)
	app.ContextPath = contextPath
	return NewDispatcherServlet(app)
}
