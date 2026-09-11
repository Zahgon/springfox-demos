package main

import "github.com/springfox/springfox-demos/internal/web"

// ConfigureSecurity is io.springfox.demo.bootwebmvc.Security, the
// @EnableWebSecurity WebSecurityConfigurerAdapter:
//
//	http.authorizeRequests().anyRequest().anonymous()
//	    .and().csrf().csrfTokenRepository(CookieCsrfTokenRepository.withHttpOnlyFalse());
//
// Every request is authorised anonymously, so the only observable effect is the
// CSRF token cookie and the 403 that a state-changing request without it gets.
func ConfigureSecurity(app *web.App) {
	app.AddFilter(web.CsrfFilter(app.ContextPath))
}
