package main

// ServletInitializer records springfoxdemo.java.swagger.ServletInitializer, the
// AbstractAnnotationConfigDispatcherServletInitializer that bootstraps the WAR:
// AppConfiguration is the only servlet configuration class, the dispatcher is
// mapped at "/", and there is no root configuration.
//
// A Go binary has no servlet container to hand a bootstrapper to, so the values
// it declares are constants here and NewApplication applies them directly.
type ServletInitializer struct{}

// ServletMappings is getServletMappings().
func (ServletInitializer) ServletMappings() []string { return []string{"/"} }

// RootConfigClasses is getRootConfigClasses(), which the original returns null
// from.
func (ServletInitializer) RootConfigClasses() []string { return nil }
