package main

// ApplicationInitializer records
// springfoxdemo.xml.swagger.ApplicationInitializer, the WebApplicationInitializer
// that registers the dispatcher servlet under the name "dispatcher", with
// load-on-startup 1, mapped to "/*".
//
// A Go binary has no servlet container to register with, so the values it
// declares are constants here and NewApplication applies them directly.
type ApplicationInitializer struct{}

// dispatcherServletName is the name the servlet is registered under.
const dispatcherServletName = "dispatcher"

// ServletName is the name the servlet is registered under.
func (ApplicationInitializer) ServletName() string { return dispatcherServletName }

// LoadOnStartup is setLoadOnStartup(1).
func (ApplicationInitializer) LoadOnStartup() int { return 1 }

// Mapping is addMapping("/*").
func (ApplicationInitializer) Mapping() string { return "/*" }
