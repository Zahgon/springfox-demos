package web

// HomeController is springfoxdemo.boot.swagger.web.HomeController.
//
// The Java class is a @Controller, but neither the class nor its single method
// carries a @RequestMapping, so Spring registers no URL for it. Nothing here
// mounts a route either; the component and its view name exist, and no request
// ever reaches them.
type HomeController struct{}

// NewHomeController creates the controller.
func NewHomeController() *HomeController { return &HomeController{} }

// homeViewName is the view the unmapped handler returns.
const homeViewName = "redirect:/swagger-ui/index.html"

// Home is the unmapped handler method.
func (c *HomeController) Home() string { return homeViewName }
